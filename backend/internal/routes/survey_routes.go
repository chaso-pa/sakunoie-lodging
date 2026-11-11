package routes

import (
	"net/http"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/handlers"
	"github.com/danielgtaylor/huma/v2"
)

// SetupSurveyRoutes registers the public survey. The path must stay
// POST /api/v1/surveys: middlewares.ClassifyRoute applies the survey rate
// limit to it. The admin summary is registered in SetupAdminRoutes.
func SetupSurveyRoutes(api huma.API, h *handlers.SurveyHandlers) {
	huma.Register(api, huma.Operation{
		OperationID:   "submit-survey",
		Method:        http.MethodPost,
		Path:          "/surveys",
		Summary:       "Submit an anonymous post-stay survey",
		Description:   "Stores the answers with the facility and survey version only; no guest, registration or booking linkage. submitted_at and created_at are stored truncated to the UTC day. Answers are validated against the versioned definition (unknown question, out-of-range rating, invalid option or over-length text → 422). Unknown or inactive facility → 422 facility_unavailable. A filled honeypot field is accepted (202) and discarded.",
		Tags:          []string{"Surveys"},
		DefaultStatus: http.StatusAccepted,
		Errors:        []int{http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusTooManyRequests},
	}, h.SubmitSurvey)
}
