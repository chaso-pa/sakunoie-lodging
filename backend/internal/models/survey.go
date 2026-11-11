package models

// Survey responses are anonymous: the request carries a facility code, the
// survey version and the answers, and nothing that links to a guest,
// registration or booking.

type SubmitSurveyBody struct {
	FacilityCode  string         `json:"facility_code" pattern:"^[a-z0-9-]{1,64}$" doc:"Facility code (resolved server-side to an active facility; unknown or inactive → 422 facility_unavailable)"`
	SurveyVersion string         `json:"survey_version" minLength:"1" maxLength:"64" doc:"Version of the question set the answers belong to (currently v1)"`
	Answers       map[string]any `json:"answers" maxProperties:"20" doc:"Answers keyed by question code (q01..q10). rating_1_5: integer 1-5; single_choice: option code; multi_choice: array of distinct option codes; text: string up to maxLength characters. Omit unanswered questions."`
	Website       string         `json:"website,omitempty" maxLength:"200" doc:"Honeypot; leave empty. A non-empty value is accepted with 202 and discarded"`
}

type SubmitSurveyInput struct {
	Body SubmitSurveyBody
}

// SubmitSurveyOutput is an empty 202 Accepted.
type SubmitSurveyOutput struct{}

type GetSurveySummaryInput struct {
	FacilityID uint64 `query:"facility_id" doc:"Restrict to one authorized facility (403 if not authorized). Omitted: all authorized facilities"`
	From       string `query:"from" format:"date" doc:"Earliest submission date (inclusive, UTC YYYY-MM-DD; submissions are stored per UTC day)"`
	To         string `query:"to" format:"date" doc:"Latest submission date (inclusive, UTC YYYY-MM-DD)"`
}

type SurveyCount struct {
	Value string `json:"value" doc:"Option code, or the rating 1-5 as a string"`
	Count int    `json:"count"`
}

type SurveyQuestionSummary struct {
	Code       string        `json:"code"`
	Type       string        `json:"type" enum:"rating_1_5,single_choice,multi_choice,text"`
	Answered   int           `json:"answered" doc:"Responses that answered this question (for text: non-empty answers; the text itself is never returned)"`
	Suppressed bool          `json:"suppressed" doc:"True when answered < min_responses; mean and counts are then null"`
	Mean       *float64      `json:"mean" doc:"rating_1_5 only: mean rounded to 2 decimals"`
	Counts     []SurveyCount `json:"counts" doc:"rating_1_5: counts for 1..5; single/multi_choice: counts per option in definition order; null for text"`
}

type SurveySummary struct {
	SurveyVersion string                  `json:"survey_version" doc:"Only responses of this (current) version are aggregated"`
	From          *string                 `json:"from" format:"date"`
	To            *string                 `json:"to" format:"date"`
	Total         int                     `json:"total" doc:"Responses matching the filter"`
	MinResponses  int                     `json:"min_responses" doc:"Small-cell threshold (SURVEY_MIN_RESPONSES)"`
	Questions     []SurveyQuestionSummary `json:"questions" nullable:"false"`
}

type GetSurveySummaryOutput struct {
	Body SurveySummary
}
