package routes

import (
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/handlers"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
	"github.com/danielgtaylor/huma/v2"
)

// SetupAuthRoutes registers /api/auth/*. Paths are relative to the /api
// group both APIs are mounted on.
func SetupAuthRoutes(publicAPI, authAPI huma.API, deps *Deps) {
	h := handlers.NewAuthHandlers(deps.Sessions, services.NewAuthService(deps.DB, deps.Config.InviteCode, deps.Images))

	huma.Post(publicAPI, "/auth/verify-invite", h.VerifyInvite, withTag("auth"))
	huma.Post(publicAPI, "/auth/signup", h.Signup, withTag("auth"))
	huma.Post(publicAPI, "/auth/transfer", h.Transfer, withTag("auth"))
	huma.Post(publicAPI, "/auth/logout", h.Logout, withTag("auth"))

	huma.Get(authAPI, "/auth/me", h.Me, withTag("auth"))
}

func withTag(tag string) func(*huma.Operation) {
	return func(o *huma.Operation) {
		o.Tags = append(o.Tags, tag)
	}
}
