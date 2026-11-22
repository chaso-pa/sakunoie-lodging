package middlewares

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"github.com/danielgtaylor/huma/v2"
)

type adminScopeKey struct{}

// AdminAuthenticator resolves a Firebase ID token to an AdminScope
// (services.AdminAuthService).
type AdminAuthenticator interface {
	Authenticate(ctx context.Context, idToken string) (*services.AdminScope, error)
}

// AdminAuth is the Huma middleware of every admin operation. It runs before
// input parsing, sets Cache-Control: no-store, and answers 401
// (unauthenticated: missing, invalid or expired token) or 403
// (not_authorized: unknown uid or no owned facility) with the error
// envelope. On success the AdminScope is put in the context.
func AdminAuth(api huma.API, auth AdminAuthenticator) func(huma.Context, func(huma.Context)) {
	return func(ctx huma.Context, next func(huma.Context)) {
		ctx.SetHeader("Cache-Control", "no-store")
		scope, err := auth.Authenticate(ctx.Context(), bearerToken(ctx))
		if err != nil {
			env := adminAuthError(ctx.Context(), err)
			if env.Status == http.StatusUnauthorized {
				ctx.SetHeader("WWW-Authenticate", "Bearer")
			}
			ctx.SetHeader("Content-Type", "application/json")
			ctx.SetStatus(env.Status)
			_ = api.Marshal(ctx.BodyWriter(), "application/json", env)
			return
		}
		next(huma.WithValue(ctx, adminScopeKey{}, *scope))
	}
}

func adminAuthError(ctx context.Context, err error) *models.ErrorEnvelope {
	switch {
	case errors.Is(err, services.ErrUnauthenticated):
		return models.EnvelopeFor(ctx, http.StatusUnauthorized, "unauthenticated", "sign-in required")
	case errors.Is(err, services.ErrNotAuthorized):
		return models.EnvelopeFor(ctx, http.StatusForbidden, "not_authorized", "not authorized for the admin console")
	}
	slog.ErrorContext(ctx, "admin authorization failed",
		"request_id", utils.RequestID(ctx), "error_kind", services.ErrorKind(err))
	return models.EnvelopeFor(ctx, http.StatusInternalServerError, "", "internal error")
}

// AdminScopeFrom returns the scope placed by AdminAuth. A missing scope is
// the zero value, which every admin service method rejects.
func AdminScopeFrom(ctx context.Context) services.AdminScope {
	s, _ := ctx.Value(adminScopeKey{}).(services.AdminScope)
	return s
}
