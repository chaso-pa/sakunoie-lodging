package services

import (
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
)

func TestInvitationStatus(t *testing.T) {
	exp := time.Date(2026, 12, 3, 14, 59, 59, 999999000, time.UTC)
	ts := exp.Add(-time.Hour)
	for name, c := range map[string]struct {
		inv  entities.RegistrationInvitation
		now  time.Time
		want string
	}{
		"active at the last microsecond": {entities.RegistrationInvitation{ExpiresAt: exp}, exp, models.InvitationActive},
		"expired one microsecond later":  {entities.RegistrationInvitation{ExpiresAt: exp}, exp.Add(time.Microsecond), models.InvitationExpired},
		"revoked wins over expired":      {entities.RegistrationInvitation{ExpiresAt: exp, RevokedAt: &ts}, exp.Add(time.Hour), models.InvitationRevoked},
		"used wins over everything":      {entities.RegistrationInvitation{ExpiresAt: exp, UsedAt: &ts, RevokedAt: &ts}, exp.Add(time.Hour), models.InvitationUsed},
	} {
		if got := invitationStatus(&c.inv, c.now); got != c.want {
			t.Errorf("%s: got %q want %q", name, got, c.want)
		}
	}
}

func TestCheckInvitationUsable(t *testing.T) {
	exp := time.Date(2026, 12, 3, 14, 59, 59, 999999000, time.UTC)
	used := exp.Add(-time.Hour)
	if err := checkInvitationUsable(&entities.RegistrationInvitation{ExpiresAt: exp}, exp); err != nil {
		t.Fatalf("at expiry: %v", err)
	}
	if err := checkInvitationUsable(&entities.RegistrationInvitation{ExpiresAt: exp}, exp.Add(time.Microsecond)); err != ErrInvitationExpired {
		t.Fatalf("after expiry: %v", err)
	}
	if err := checkInvitationUsable(&entities.RegistrationInvitation{ExpiresAt: exp, UsedAt: &used}, exp); err != ErrInvitationUsed {
		t.Fatalf("used: %v", err)
	}
}

func TestFindInvitationRejectsBadTokensWithoutDB(t *testing.T) {
	// Both checks run before any query, so a nil DB is never touched.
	if _, err := findInvitation(nil, " ", false); err != ErrInvitationRequired {
		t.Fatalf("blank: %v", err)
	}
	if _, err := findInvitation(nil, "short", false); err != ErrInvitationInvalid {
		t.Fatalf("malformed: %v", err)
	}
}
