package routes

import (
	"net/http"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/handlers"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/middlewares"
	"github.com/danielgtaylor/huma/v2"
)

// SetupInvitationRoutes registers the public invitation check of the
// registration wizard. The admin invitation operations are in
// SetupAdminRoutes.
func SetupInvitationRoutes(api huma.API, h *handlers.InvitationHandlers) {
	huma.Register(api, huma.Operation{
		OperationID: "get-current-invitation",
		Method:      http.MethodGet,
		Path:        "/invitations/current",
		Summary:     "Check the registration invitation",
		Description: "Takes the invitation token as `Authorization: Bearer` and returns only what prefills the wizard: facility code, the host-set stay dates and the expiry. Never the host's note or booking code. No token → 401 invitation_required; unknown, malformed or revoked → 401 invitation_invalid; expired → 410 invitation_expired; already used → 409 invitation_used (the guest uses their edit link). Cache-Control: no-store.",
		Tags:        []string{"Guest registrations"},
		Security:    invitationSecurity,
		Middlewares: huma.Middlewares{middlewares.InvitationToken},
		Errors:      []int{http.StatusUnauthorized, http.StatusConflict, http.StatusGone, http.StatusTooManyRequests},
	}, h.GetCurrentInvitation)
}
