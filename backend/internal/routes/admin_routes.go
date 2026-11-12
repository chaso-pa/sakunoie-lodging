package routes

import (
	"net/http"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/handlers"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/middlewares"
	"github.com/danielgtaylor/huma/v2"
)

var adminSecurity = []map[string][]string{{"adminBearer": {}}}

// SetupAdminRoutes registers /admin operations. Each one runs the AdminAuth
// middleware (401/403 before input parsing) and is scoped server-side.
// The export route is registered before /{id}, and {id} is format uuid.
func SetupAdminRoutes(api huma.API, h *handlers.AdminHandlers, auth middlewares.AdminAuthenticator) {
	authMW := middlewares.AdminAuth(api, auth)
	adminOp := func(op huma.Operation) huma.Operation {
		op.Tags = []string{"Admin"}
		op.Security = adminSecurity
		op.Middlewares = huma.Middlewares{authMW}
		op.Errors = append([]int{http.StatusUnauthorized, http.StatusForbidden}, op.Errors...)
		return op
	}

	huma.Register(api, adminOp(huma.Operation{
		OperationID: "list-admin-facilities",
		Method:      http.MethodGet,
		Path:        "/admin/facilities",
		Summary:     "List the facilities this admin is authorized for",
		Description: "Facilities with an owner row in admin_facility_access for the signed-in admin.",
	}), h.ListAdminFacilities)

	huma.Register(api, adminOp(huma.Operation{
		OperationID: "list-admin-guest-registrations",
		Method:      http.MethodGet,
		Path:        "/admin/guest-registrations",
		Summary:     "List registrations (ledger)",
		Description: "Scoped, paginated ledger (50 per page, newest check-in first) with metrics over the whole filter. facility_id outside the admin's facilities → 403.",
		Errors:      []int{http.StatusUnprocessableEntity},
	}), h.ListAdminRegistrations)

	huma.Register(api, adminOp(huma.Operation{
		OperationID: "export-admin-guest-registrations",
		Method:      http.MethodGet,
		Path:        "/admin/guest-registrations/export",
		Summary:     "Export registrations as CSV",
		Description: "UTF-8 CSV with BOM, one row per guest. Formula-leading cells (= + - @, full-width variants, tab, CR and other control characters) are prefixed with a single quote. Passport numbers are exported as the last 4 characters only. Writes an audit row with the filters and counts, never the content.",
		Responses: map[string]*huma.Response{
			"200": {
				Description: `CSV file (Content-Disposition: attachment; filename="registrations-<code>-<from|start>-<to|end>.csv")`,
				Content:     map[string]*huma.MediaType{"text/csv": {Schema: &huma.Schema{Type: "string"}}},
			},
		},
		Errors: []int{http.StatusUnprocessableEntity},
	}), h.ExportAdminRegistrations)

	huma.Register(api, adminOp(huma.Operation{
		OperationID: "get-admin-guest-registration",
		Method:      http.MethodGet,
		Path:        "/admin/guest-registrations/{id}",
		Summary:     "Get a registration",
		Description: "Scoped detail; a registration of another facility is 404. Address, phone and email are masked unless reveal=true (audited). Passport numbers: last 4 only.",
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}), h.GetAdminRegistration)

	huma.Register(api, adminOp(huma.Operation{
		OperationID: "patch-admin-guest-registration",
		Method:      http.MethodPatch,
		Path:        "/admin/guest-registrations/{id}",
		Summary:     "Correct a registration (host)",
		Description: "Controlled corrections of dates, counts, booking fields and guests (including the representative and passport numbers, re-encrypted). planned_guest_count above the current facility max_guests needs capacity_override. Requires If-Match. Changed field names are audited.",
		Errors: []int{http.StatusNotFound, http.StatusConflict, http.StatusPreconditionFailed,
			http.StatusUnprocessableEntity},
	}), h.PatchAdminRegistration)

	huma.Register(api, adminOp(huma.Operation{
		OperationID: "verify-admin-guest-registration-identity",
		Method:      http.MethodPost,
		Path:        "/admin/guest-registrations/{id}/verify-identity",
		Summary:     "Record the in-person identity check",
		Description: "Sets identity_verified_at/by for the signed-in admin and writes an audit row. Idempotent: an already verified registration is returned unchanged.",
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}), h.VerifyIdentity)

	huma.Register(api, adminOp(huma.Operation{
		OperationID: "reissue-admin-guest-registration-edit-link",
		Method:      http.MethodPost,
		Path:        "/admin/guest-registrations/{id}/reissue-edit-link",
		Summary:     "Reissue the guest edit link",
		Description: "Revokes every earlier edit link and returns a new token once, valid until the end of the current checkout day in the facility time zone. Writes an audit row.",
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity},
	}), h.ReissueEditLink)

	huma.Register(api, adminOp(huma.Operation{
		OperationID: "get-admin-survey-summary",
		Method:      http.MethodGet,
		Path:        "/admin/surveys/summary",
		Summary:     "Aggregate survey results",
		Description: "Aggregates only, for the current survey version: per-question answered counts, rating mean and 1-5 distribution, and option counts. Free-text answers are counted, never returned. A question answered by fewer than min_responses (SURVEY_MIN_RESPONSES) responses has mean and counts null. facility_id outside the admin's facilities → 403. from/to filter the submission day (UTC).",
		Errors:      []int{http.StatusUnprocessableEntity},
	}), h.GetSurveySummary)
}
