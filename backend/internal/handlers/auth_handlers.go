package handlers

import (
	"context"
	"errors"
	"io"
	"log/slog"

	"github.com/alexedwards/scs/v2"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/session"
	"github.com/danielgtaylor/huma/v2"
)

// AuthHandlers holds the auth endpoints. In every handler ctx is the HTTP
// request context, which scs's LoadAndSave has already loaded the session
// into, so the session manager is called with ctx directly.
type AuthHandlers struct {
	sessions *scs.SessionManager
	auth     *services.AuthService
}

func NewAuthHandlers(sessions *scs.SessionManager, auth *services.AuthService) *AuthHandlers {
	return &AuthHandlers{sessions: sessions, auth: auth}
}

func (h *AuthHandlers) VerifyInvite(ctx context.Context, input *models.VerifyInviteInput) (*models.VerifyInviteOutput, error) {
	if !h.auth.VerifyInvite(input.Body.Code) {
		return nil, huma.Error401Unauthorized("invalid invite code")
	}
	h.sessions.Put(ctx, session.KeyInviteVerified, true)

	resp := &models.VerifyInviteOutput{}
	resp.Body.Valid = true
	return resp, nil
}

func (h *AuthHandlers) Signup(ctx context.Context, input *models.SignupInput) (*models.SignupOutput, error) {
	if !h.sessions.GetBool(ctx, session.KeyInviteVerified) {
		return nil, huma.Error403Forbidden("invite code not verified")
	}

	data := input.RawBody.Data()
	var avatar io.Reader
	if data.Avatar.IsSet {
		defer data.Avatar.Close()
		avatar = data.Avatar
	}

	member, transferCode, err := h.auth.Signup(ctx, data.Nickname, data.Language, avatar)
	if err != nil {
		return nil, serviceError(ctx, "signup", err)
	}

	// New token on privilege change, against session fixation.
	if err := h.sessions.RenewToken(ctx); err != nil {
		return nil, internalError(ctx, "renew session token", err)
	}
	h.sessions.Put(ctx, session.KeyMemberID, member.ID)
	h.sessions.Put(ctx, session.KeySessionVersion, member.SessionVersion)
	h.sessions.Remove(ctx, session.KeyInviteVerified)

	resp := &models.SignupOutput{}
	resp.Body.Member = models.NewMemberResponse(member)
	resp.Body.TransferCode = transferCode
	return resp, nil
}

func (h *AuthHandlers) Transfer(ctx context.Context, input *models.TransferInput) (*models.TransferOutput, error) {
	member, err := h.auth.RestoreByTransferCode(ctx, input.Body.TransferCode)
	if err != nil {
		if errors.Is(err, services.ErrInvalidTransferCode) {
			return nil, huma.Error401Unauthorized("invalid transfer code")
		}
		return nil, internalError(ctx, "transfer", err)
	}

	if err := h.sessions.RenewToken(ctx); err != nil {
		return nil, internalError(ctx, "renew session token", err)
	}
	// member.SessionVersion was just incremented, so sessions on the previous
	// device no longer match and are rejected by RequireAuth.
	h.sessions.Put(ctx, session.KeyMemberID, member.ID)
	h.sessions.Put(ctx, session.KeySessionVersion, member.SessionVersion)
	h.sessions.Remove(ctx, session.KeyInviteVerified)

	resp := &models.TransferOutput{}
	resp.Body.Member = models.NewMemberResponse(member)
	return resp, nil
}

func (h *AuthHandlers) Logout(ctx context.Context, _ *struct{}) (*models.LogoutOutput, error) {
	if err := h.sessions.Destroy(ctx); err != nil {
		return nil, internalError(ctx, "destroy session", err)
	}
	resp := &models.LogoutOutput{}
	resp.Body.OK = true
	return resp, nil
}

// Me is registered behind RequireAuth, so the member id is always present.
func (h *AuthHandlers) Me(ctx context.Context, _ *struct{}) (*models.MeOutput, error) {
	member, err := h.auth.GetMember(ctx, middlewares.GetMemberID(ctx))
	if err != nil {
		if errors.Is(err, services.ErrMemberNotFound) {
			return nil, huma.Error401Unauthorized("authentication required")
		}
		return nil, internalError(ctx, "get member", err)
	}
	resp := &models.MeOutput{}
	resp.Body.Member = models.NewMemberResponse(member)
	return resp, nil
}

// internalError logs the cause and returns a 500 that reveals none of it.
func internalError(ctx context.Context, op string, err error) error {
	logFailure(ctx, op, err)
	return huma.Error500InternalServerError("internal server error")
}

func logFailure(ctx context.Context, op string, err error) {
	slog.ErrorContext(ctx, "handler failed", "op", op, "error", err)
}
