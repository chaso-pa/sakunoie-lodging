package services_test

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Fixed "today" for integration tests: stays are in December 2026.
var baseNow = time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC)

type env struct {
	db     *gorm.DB
	clock  *testutil.Clock
	tokens *services.TokenService
	idem   *services.IdempotencyService
	reg    *services.RegistrationService
	manage *services.GuestManageService
	admin  *services.AdminRegistrationService
}

func setup(t *testing.T) *env {
	t.Helper()
	db := testutil.RequireDB(t)
	clock := testutil.NewClock(baseNow)
	cipher := testutil.PassportCipher()
	tokens := services.NewTokenService(clock)
	idem := services.NewIdempotencyService(db, clock, testutil.TestHMACKey, slog.New(slog.NewTextHandler(io.Discard, nil)))
	return &env{
		db:     db,
		clock:  clock,
		tokens: tokens,
		idem:   idem,
		reg:    services.NewRegistrationService(db, clock, tokens, idem, cipher),
		manage: services.NewGuestManageService(db, clock, tokens, cipher),
		admin:  services.NewAdminRegistrationService(db, clock, tokens, cipher),
	}
}

func newKey() string { return "test-key-" + uuid.NewString() }

func (e *env) mustCreate(t *testing.T, key string, body *models.CreateGuestRegistrationBody) *services.CreateResult {
	t.Helper()
	res, err := e.reg.Create(context.Background(), key, body)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	return res
}

func sakuBody(planned int, guests ...models.GuestInput) *models.CreateGuestRegistrationBody {
	return testutil.CreateBody(testutil.FacilitySaku, "2026-12-01", "2026-12-03", planned, guests...)
}

func expectErr(t *testing.T, err, want error) {
	t.Helper()
	if !errors.Is(err, want) {
		t.Fatalf("got error %v, want %v", err, want)
	}
}

func (e *env) registration(t *testing.T, publicID string) entities.GuestRegistration {
	t.Helper()
	var r entities.GuestRegistration
	if err := e.db.Where("public_id = ?", publicID).Take(&r).Error; err != nil {
		t.Fatal(err)
	}
	return r
}

func (e *env) activeTokens(t *testing.T, regID uint64) int64 {
	t.Helper()
	var n int64
	if err := e.db.Model(&entities.RegistrationAccessToken{}).
		Where("registration_id = ? AND revoked_at IS NULL", regID).Count(&n).Error; err != nil {
		t.Fatal(err)
	}
	return n
}

func (e *env) reissue(t *testing.T, publicID string) string {
	t.Helper()
	var raw string
	err := e.db.Transaction(func(tx *gorm.DB) error {
		var reg entities.GuestRegistration
		if err := tx.Where("public_id = ?", publicID).Take(&reg).Error; err != nil {
			return err
		}
		var f entities.Facility
		if err := tx.Where("id = ?", reg.FacilityID).Take(&f).Error; err != nil {
			return err
		}
		var err error
		raw, _, err = e.tokens.Reissue(tx, &f, &reg)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	return raw
}

func strp(s string) *string { return &s }
func intp(i int) *int       { return &i }
