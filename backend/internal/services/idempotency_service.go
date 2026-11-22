package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"log/slog"
	"strconv"
	"strings"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

// Idempotency rows live 24 h. The row stores hashes and a lookup pointer
// only: key_hash = SHA-256(Idempotency-Key), request_hash =
// HMAC-SHA256(IDEMPOTENCY_HMAC_KEY, canonical DTO JSON) (a plain hash would
// be an unsalted digest of PII), response_json = {"registration_public_id"}.
const (
	IdempotencyTTL      = 24 * time.Hour
	janitorBatchSize    = 1000
	janitorInterval     = time.Hour
	idempotencyKeyIndex = "key_hash"
)

type IdempotencyService struct {
	db      *gorm.DB
	clock   Clock
	hmacKey []byte
	logger  *slog.Logger
}

func NewIdempotencyService(db *gorm.DB, clock Clock, hmacKey []byte, logger *slog.Logger) *IdempotencyService {
	return &IdempotencyService{db: db, clock: clock, hmacKey: hmacKey, logger: logger}
}

type idempotencyPointer struct {
	RegistrationPublicID string `json:"registration_public_id"`
}

// KeyHash hashes the client's Idempotency-Key.
func (s *IdempotencyService) KeyHash(key string) []byte {
	h := sha256.Sum256([]byte(key))
	return h[:]
}

// RequestHash is the HMAC of the canonical JSON of the parsed DTO
// (encoding/json emits struct fields in declaration order).
func (s *IdempotencyService) RequestHash(dto any) ([]byte, error) {
	b, err := json.Marshal(dto)
	if err != nil {
		return nil, err
	}
	m := hmac.New(sha256.New, s.hmacKey)
	m.Write(b)
	return m.Sum(nil), nil
}

func pointerJSON(publicID string) (*string, error) {
	b, err := json.Marshal(idempotencyPointer{RegistrationPublicID: publicID})
	if err != nil {
		return nil, err
	}
	s := string(b)
	return &s, nil
}

// insertKey inserts the key row inside the create transaction. A concurrent
// request with the same key blocks on the InnoDB unique-index lock until
// this transaction ends, then gets 1062 (commit) or proceeds (rollback).
func (s *IdempotencyService) insertKey(tx *gorm.DB, keyHash, reqHash []byte, publicID string) error {
	ptr, err := pointerJSON(publicID)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	status := uint16(201)
	return tx.Create(&entities.APIIdempotencyKey{
		KeyHash:        keyHash,
		RequestHash:    reqHash,
		ResponseJSON:   ptr,
		ResponseStatus: &status,
		CreatedAt:      now,
		ExpiresAt:      now.Add(IdempotencyTTL),
	}).Error
}

// overwriteKey reuses an expired key row for a new registration.
func (s *IdempotencyService) overwriteKey(tx *gorm.DB, row *entities.APIIdempotencyKey, reqHash []byte, publicID string) error {
	ptr, err := pointerJSON(publicID)
	if err != nil {
		return err
	}
	now := s.clock.Now()
	return tx.Model(row).Updates(map[string]any{
		"request_hash":    reqHash,
		"response_json":   ptr,
		"response_status": 201,
		"created_at":      now,
		"expires_at":      now.Add(IdempotencyTTL),
	}).Error
}

// isDuplicateIdempotencyKey reports a 1062 on the key_hash unique index.
func isDuplicateIdempotencyKey(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1062 && strings.Contains(me.Message, "'"+idempotencyKeyIndex+"'")
}

// DeleteExpired removes expired rows in batches and returns the count.
func (s *IdempotencyService) DeleteExpired(ctx context.Context) (int64, error) {
	var total int64
	now := s.clock.Now()
	for {
		res := s.db.WithContext(ctx).Exec("DELETE FROM api_idempotency_keys WHERE expires_at < ? LIMIT ?", now, janitorBatchSize)
		if res.Error != nil {
			return total, res.Error
		}
		total += res.RowsAffected
		if res.RowsAffected < janitorBatchSize {
			return total, nil
		}
	}
}

// StartJanitor deletes expired rows at startup and hourly until ctx ends.
// The returned channel closes when the goroutine has stopped.
func (s *IdempotencyService) StartJanitor(ctx context.Context) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		ticker := time.NewTicker(janitorInterval)
		defer ticker.Stop()
		for {
			n, err := s.DeleteExpired(ctx)
			if err != nil && ctx.Err() == nil {
				s.logger.Error("idempotency janitor failed", "error_kind", ErrorKind(err))
			} else if err == nil {
				s.logger.Info("idempotency janitor", "deleted", n)
			}
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
			}
		}
	}()
	return done
}

// ErrorKind describes an error for logs without its message, which for
// MySQL errors can quote stored values (e.g. "Duplicate entry '...'").
func ErrorKind(err error) string {
	var me *mysql.MySQLError
	if errors.As(err, &me) {
		return "mysql_" + strconv.Itoa(int(me.Number))
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "context"
	}
	return "other"
}
