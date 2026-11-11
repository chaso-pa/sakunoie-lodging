package routes

import (
	"net/http"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/handlers"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/middlewares"
	"github.com/danielgtaylor/huma/v2"
)

var editTokenSecurity = []map[string][]string{{"editToken": {}}}

func SetupGuestRegistrationRoutes(api huma.API, reg *handlers.GuestRegistrationHandlers, manage *handlers.GuestManageHandlers) {
	huma.Register(api, huma.Operation{
		OperationID:   "create-guest-registration",
		Method:        http.MethodPost,
		Path:          "/guest-registrations",
		Summary:       "Submit a guest registration",
		Description:   "Creates one registration with 1..N guests in one transaction. Requires Idempotency-Key. 201 on create; 200 on an idempotent replay of the same body, which returns a new additional edit token, or edit_token=null with code=edit_link_unavailable when 3 links are active or the host reissued the link. Same key with a different body → 409 idempotency_conflict.",
		Tags:          []string{"Guest registrations"},
		DefaultStatus: http.StatusCreated,
		Responses: map[string]*huma.Response{
			"200": {
				Description: "Idempotent replay (same Idempotency-Key and body)",
				Content: map[string]*huma.MediaType{"application/json": {
					Schema: &huma.Schema{Ref: "#/components/schemas/CreateGuestRegistrationResult"},
				}},
			},
		},
		Middlewares: huma.Middlewares{middlewares.NoStore},
		Errors:      []int{http.StatusBadRequest, http.StatusConflict, http.StatusUnprocessableEntity, http.StatusTooManyRequests},
	}, reg.CreateGuestRegistration)

	manageOp := func(op huma.Operation) huma.Operation {
		op.Tags = []string{"Guest manage"}
		op.Security = editTokenSecurity
		op.Middlewares = huma.Middlewares{middlewares.EditToken}
		return op
	}

	huma.Register(api, manageOp(huma.Operation{
		OperationID: "get-manage-registration",
		Method:      http.MethodGet,
		Path:        "/guest-registrations/manage",
		Summary:     "Get the registration for an edit token",
		Description: "Minimal masked summary. Passport numbers are never returned (last 4 only). Cache-Control: no-store.",
		Errors:      []int{http.StatusUnauthorized, http.StatusGone, http.StatusTooManyRequests},
	}), manage.GetManageRegistration)

	huma.Register(api, manageOp(huma.Operation{
		OperationID: "patch-manage-registration",
		Method:      http.MethodPatch,
		Path:        "/guest-registrations/manage",
		Summary:     "Correct low-risk registration fields",
		Description: "Allowlist: estimated_arrival_time, booking_source, booking_reference, planned_guest_count (>= registered, <= facility max_guests). Requires If-Match.",
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusConflict, http.StatusGone,
			http.StatusPreconditionFailed, http.StatusUnprocessableEntity, http.StatusTooManyRequests},
	}), manage.PatchManageRegistration)

	huma.Register(api, manageOp(huma.Operation{
		OperationID: "add-manage-companions",
		Method:      http.MethodPost,
		Path:        "/guest-registrations/manage/guests",
		Summary:     "Add companions",
		Description: "Appends companions transactionally; registered + new must not exceed planned_guest_count (409 capacity_exceeded).",
		Errors: []int{http.StatusUnauthorized, http.StatusConflict, http.StatusGone,
			http.StatusUnprocessableEntity, http.StatusTooManyRequests},
	}), manage.AddCompanions)

	huma.Register(api, manageOp(huma.Operation{
		OperationID: "patch-manage-guest",
		Method:      http.MethodPatch,
		Path:        "/guest-registrations/manage/guests/{guest_public_id}",
		Summary:     "Correct a companion",
		Description: "Companions only: full_name, name_kana, address, phone, email. The representative and identity fields → 403 field_not_editable. Requires If-Match.",
		Errors: []int{http.StatusUnauthorized, http.StatusForbidden, http.StatusNotFound, http.StatusGone,
			http.StatusPreconditionFailed, http.StatusUnprocessableEntity, http.StatusTooManyRequests},
	}), manage.PatchManageGuest)
}
