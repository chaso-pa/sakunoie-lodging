package handlers

import (
	"context"
	"errors"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/session"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/danielgtaylor/huma/v2"
)

func statusOf(t *testing.T, err error) int {
	t.Helper()
	var se huma.StatusError
	if !errors.As(err, &se) {
		t.Fatalf("error %v is not a Huma status error", err)
	}
	return se.GetStatus()
}

// Signup must refuse a session that has not passed verify-invite, before it
// looks at the form or touches the database (the service here has none).
func TestSignupRequiresInviteVerified(t *testing.T) {
	sm := session.NewInMemory()
	h := NewAuthHandlers(sm, services.NewAuthService(nil, "SAKU-26-ABCD", storage.NotConfigured{}))
	ctx, err := sm.Load(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}

	_, err = h.Signup(ctx, &models.SignupInput{})
	if got := statusOf(t, err); got != 403 {
		t.Fatalf("status = %d, want 403", got)
	}
}

func TestVerifyInviteSetsSessionFlag(t *testing.T) {
	sm := session.NewInMemory()
	h := NewAuthHandlers(sm, services.NewAuthService(nil, "SAKU-26-ABCD", storage.NotConfigured{}))
	ctx, err := sm.Load(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}

	bad := &models.VerifyInviteInput{}
	bad.Body.Code = "WRONG"
	if _, err := h.VerifyInvite(ctx, bad); statusOf(t, err) != 401 {
		t.Fatal("wrong code must be 401")
	}
	if sm.GetBool(ctx, session.KeyInviteVerified) {
		t.Fatal("a wrong code must not set invite_verified")
	}

	good := &models.VerifyInviteInput{}
	good.Body.Code = "saku-26-abcd"
	out, err := h.VerifyInvite(ctx, good)
	if err != nil || !out.Body.Valid {
		t.Fatalf("out = %+v, err = %v", out, err)
	}
	if !sm.GetBool(ctx, session.KeyInviteVerified) {
		t.Fatal("invite_verified was not set")
	}
}
