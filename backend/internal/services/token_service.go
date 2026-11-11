package services

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MaxActiveTokens caps the edit links a registration can have at once
// (Decision D1: idempotent replays mint additional tokens).
const MaxActiveTokens = 3

const tokenBytes = 32

// TokenService issues and resolves guest edit tokens. Only SHA-256 hashes
// are stored; the raw token exists only in the create/reissue response.
type TokenService struct {
	clock Clock
}

func NewTokenService(clock Clock) *TokenService { return &TokenService{clock: clock} }

// hashToken decodes a raw base64url token and returns its SHA-256.
func hashToken(raw string) ([]byte, bool) {
	b, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil || len(b) != tokenBytes {
		return nil, false
	}
	h := sha256.Sum256(b)
	return h[:], true
}

// tokenExpiry is the end of the checkout day in the facility time zone.
func tokenExpiry(f *entities.Facility, reg *entities.GuestRegistration) (time.Time, error) {
	return utils.EditTokenExpiry(utils.CivilDateFromDB(reg.CheckOutDate), f.Timezone)
}

// issue inserts a new token row and returns the raw token.
func (s *TokenService) issue(tx *gorm.DB, regID uint64, expiresAt time.Time) (string, error) {
	b := make([]byte, tokenBytes)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	h := sha256.Sum256(b)
	row := entities.RegistrationAccessToken{
		RegistrationID: regID,
		TokenHash:      h[:],
		ExpiresAt:      expiresAt,
		CreatedAt:      s.clock.Now(),
	}
	if err := tx.Create(&row).Error; err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// IssueAdditional mints another token for an idempotent replay. It returns
// ok=false (no token) when MaxActiveTokens are active, or when any token of
// the registration was ever revoked: only an admin reissue revokes, and a
// replay must not undo the host's revocation.
func (s *TokenService) IssueAdditional(tx *gorm.DB, f *entities.Facility, reg *entities.GuestRegistration) (raw string, expiresAt time.Time, ok bool, err error) {
	var revoked int64
	if err = tx.Model(&entities.RegistrationAccessToken{}).
		Where("registration_id = ? AND revoked_at IS NOT NULL", reg.ID).
		Count(&revoked).Error; err != nil {
		return "", time.Time{}, false, err
	}
	if revoked > 0 {
		return "", time.Time{}, false, nil
	}
	var active int64
	if err = tx.Model(&entities.RegistrationAccessToken{}).
		Where("registration_id = ? AND revoked_at IS NULL AND expires_at >= ?", reg.ID, s.clock.Now()).
		Count(&active).Error; err != nil {
		return "", time.Time{}, false, err
	}
	if active >= MaxActiveTokens {
		return "", time.Time{}, false, nil
	}
	expiresAt, err = tokenExpiry(f, reg)
	if err != nil {
		return "", time.Time{}, false, err
	}
	raw, err = s.issue(tx, reg.ID, expiresAt)
	if err != nil {
		return "", time.Time{}, false, err
	}
	return raw, expiresAt, true, nil
}

// Reissue revokes every unrevoked token of the registration and issues a
// new one with expiry recomputed from the current checkout date (admin).
func (s *TokenService) Reissue(tx *gorm.DB, f *entities.Facility, reg *entities.GuestRegistration) (string, time.Time, error) {
	now := s.clock.Now()
	if err := tx.Model(&entities.RegistrationAccessToken{}).
		Where("registration_id = ? AND revoked_at IS NULL", reg.ID).
		Update("revoked_at", now).Error; err != nil {
		return "", time.Time{}, err
	}
	expiresAt, err := tokenExpiry(f, reg)
	if err != nil {
		return "", time.Time{}, err
	}
	raw, err := s.issue(tx, reg.ID, expiresAt)
	return raw, expiresAt, err
}

// Resolve authenticates a raw token inside tx and returns the token row and
// its (not soft-deleted) registration. Unknown, malformed, revoked and
// soft-deleted → ErrTokenInvalid (401). Expired → ErrTokenExpired (410).
// It records last_used_at.
func (s *TokenService) Resolve(tx *gorm.DB, raw string) (*entities.RegistrationAccessToken, *entities.GuestRegistration, error) {
	h, ok := hashToken(raw)
	if !ok {
		return nil, nil, ErrTokenInvalid
	}
	var tok entities.RegistrationAccessToken
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("token_hash = ?", h).Take(&tok).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrTokenInvalid
	}
	if err != nil {
		return nil, nil, err
	}
	if tok.RevokedAt != nil {
		return nil, nil, ErrTokenInvalid
	}
	var reg entities.GuestRegistration
	err = tx.Scopes(notDeleted).Where("id = ?", tok.RegistrationID).Take(&reg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil, ErrTokenInvalid
	}
	if err != nil {
		return nil, nil, err
	}
	now := s.clock.Now()
	if now.After(tok.ExpiresAt) {
		return nil, nil, ErrTokenExpired
	}
	if err := tx.Model(&tok).Update("last_used_at", now).Error; err != nil {
		return nil, nil, err
	}
	return &tok, &reg, nil
}
