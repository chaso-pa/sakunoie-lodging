package testutil

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"gorm.io/gorm"
)

// FakeVerifier maps a bearer token to a Firebase uid (services.TokenVerifier).
type FakeVerifier map[string]string

func (f FakeVerifier) VerifyIDToken(_ context.Context, token string) (string, error) {
	if uid, ok := f[token]; ok {
		return uid, nil
	}
	return "", errors.New("fake verifier: invalid token")
}

// GrantAdmin creates (or reuses) the admin user for uid and grants role on
// each facility code. It returns the admin_users.id.
func GrantAdmin(t testing.TB, db *gorm.DB, uid, role string, facilityCodes ...string) uint64 {
	t.Helper()
	now := time.Now().UTC().Truncate(time.Microsecond)
	var u entities.AdminUser
	err := db.Where("firebase_uid = ?", uid).Take(&u).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		u = entities.AdminUser{FirebaseUID: uid, CreatedAt: now, UpdatedAt: now}
		err = db.Create(&u).Error
	}
	if err != nil {
		t.Fatalf("grant admin: %v", err)
	}
	for _, code := range facilityCodes {
		var f entities.Facility
		if err := db.Where("code = ?", code).Take(&f).Error; err != nil {
			t.Fatalf("grant admin: facility %s: %v", code, err)
		}
		if err := db.Create(&entities.AdminFacilityAccess{AdminUserID: u.ID, FacilityID: f.ID, Role: role, CreatedAt: now}).Error; err != nil {
			t.Fatalf("grant admin: access: %v", err)
		}
	}
	return u.ID
}

// FacilityID returns the ID of a seeded facility.
func FacilityID(t testing.TB, db *gorm.DB, code string) uint64 {
	t.Helper()
	var f entities.Facility
	if err := db.Where("code = ?", code).Take(&f).Error; err != nil {
		t.Fatalf("facility %s: %v", code, err)
	}
	return f.ID
}
