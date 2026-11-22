package routes

import (
	"net/http"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/handlers"
	"github.com/danielgtaylor/huma/v2"
)

func SetupFacilityRoutes(api huma.API, h *handlers.FacilityHandlers) {
	huma.Register(api, huma.Operation{
		OperationID: "get-facility-public",
		Method:      http.MethodGet,
		Path:        "/facilities/{code}/public",
		Summary:     "Get public facility information",
		Description: "Public name, capacity, time zone and current terms/privacy versions of an active facility. Never returns internal IDs.",
		Tags:        []string{"Facilities"},
		Security:    publicSecurity,
		Errors:      []int{http.StatusNotFound, http.StatusUnprocessableEntity, http.StatusTooManyRequests},
	}, h.GetFacilityPublic)
}
