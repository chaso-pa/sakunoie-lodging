package services_test

import (
	"context"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
)

func surveyBody(facility string, answers map[string]any) *models.SubmitSurveyBody {
	return &models.SubmitSurveyBody{FacilityCode: facility, SurveyVersion: "v1", Answers: answers}
}

func TestIntegrationSurveyAnonymousSchema(t *testing.T) {
	e := setup(t)
	var cols []string
	if err := e.db.Raw("SELECT column_name FROM information_schema.columns WHERE table_schema = DATABASE() AND table_name = 'survey_responses' ORDER BY column_name").
		Scan(&cols).Error; err != nil {
		t.Fatal(err)
	}
	if strings.Join(cols, ",") != "answers_json,created_at,facility_id,id,submitted_at,survey_version" {
		t.Fatalf("survey_responses columns = %v (no registration, guest, booking or identity linkage allowed)", cols)
	}
}

func TestIntegrationSurveySubmit(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.clock.Set(time.Date(2026, 11, 20, 23, 59, 58, 123456000, time.UTC))
	svc := services.NewSurveyService(e.db, e.clock, 5)

	if err := svc.Submit(ctx, surveyBody(testutil.FacilitySaku, map[string]any{"q01": 5.0, "q10": "Synthetic comment"})); err != nil {
		t.Fatal(err)
	}
	var rows []entities.SurveyResponse
	if err := e.db.Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	day := time.Date(2026, 11, 20, 0, 0, 0, 0, time.UTC)
	if len(rows) != 1 || !rows[0].SubmittedAt.Equal(day) || !rows[0].CreatedAt.Equal(day) ||
		rows[0].SurveyVersion != "v1" || rows[0].AnswersJSON != `{"q01":5,"q10":"Synthetic comment"}` ||
		rows[0].FacilityID != testutil.FacilityID(t, e.db, testutil.FacilitySaku) {
		t.Fatalf("stored = %+v", rows)
	}

	// Honeypot: accepted, nothing stored (even with invalid content).
	hp := surveyBody("no-such-facility", map[string]any{"q99": "spam"})
	hp.Website = "https://spam.example.test"
	if err := svc.Submit(ctx, hp); err != nil {
		t.Fatalf("honeypot: %v", err)
	}

	// Unknown and inactive facilities → 422 facility_unavailable.
	expectErr(t, svc.Submit(ctx, surveyBody("no-such-facility", map[string]any{"q01": 4.0})), services.ErrFacilityUnavailable)
	if err := e.db.Model(&entities.Facility{}).Where("code = ?", testutil.FacilityHonolulu).Update("is_active", false).Error; err != nil {
		t.Fatal(err)
	}
	expectErr(t, svc.Submit(ctx, surveyBody(testutil.FacilityHonolulu, map[string]any{"q01": 4.0})), services.ErrFacilityUnavailable)

	// Validation and version.
	expectErr(t, svc.Submit(ctx, surveyBody(testutil.FacilitySaku, map[string]any{"q01": 7.0})), services.ErrValidation)
	old := surveyBody(testutil.FacilitySaku, map[string]any{"q01": 4.0})
	old.SurveyVersion = "v0"
	expectErr(t, svc.Submit(ctx, old), services.ErrValidation)

	if n := testutil.CountRows(t, e.db, "survey_responses"); n != 1 {
		t.Fatalf("survey_responses = %d, want 1", n)
	}
}

func TestIntegrationSurveySummaryScopeAndSuppression(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	svc := services.NewSurveyService(e.db, e.clock, 5)
	scope := e.adminScope(t) // saku-no-ie only
	saku := testutil.FacilityID(t, e.db, testutil.FacilitySaku)
	hono := testutil.FacilityID(t, e.db, testutil.FacilityHonolulu)

	submit := func(facility string, answers map[string]any) {
		t.Helper()
		if err := svc.Submit(ctx, surveyBody(facility, answers)); err != nil {
			t.Fatal(err)
		}
	}
	for i := range 4 {
		submit(testutil.FacilitySaku, map[string]any{"q01": float64(i%5 + 1), "q05": "business", "q10": "Synthetic text"})
	}
	for range 3 {
		submit(testutil.FacilityHonolulu, map[string]any{"q01": 1.0})
	}

	// Out-of-scope facility → 403; empty scope fails closed.
	_, err := svc.Summary(ctx, scope, services.SurveySummaryFilter{FacilityID: hono})
	expectErr(t, err, services.ErrForbidden)
	_, err = svc.Summary(ctx, services.AdminScope{}, services.SurveySummaryFilter{})
	expectErr(t, err, services.ErrNotAuthorized)
	_, err = svc.Summary(ctx, scope, services.SurveySummaryFilter{From: "2026-12-02", To: "2026-12-01"})
	expectErr(t, err, services.ErrValidation)

	// 4 responses < 5: every cell suppressed; only the scope's rows count.
	sum, err := svc.Summary(ctx, scope, services.SurveySummaryFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if sum.Total != 4 || sum.MinResponses != 5 || sum.SurveyVersion != "v1" {
		t.Fatalf("summary = %+v", sum)
	}
	for _, q := range sum.Questions {
		if !q.Suppressed || q.Mean != nil || q.Counts != nil {
			t.Fatalf("%s not suppressed: %+v", q.Code, q)
		}
	}

	// The fifth response lifts suppression for q01 and q05 only.
	submit(testutil.FacilitySaku, map[string]any{"q01": 5.0, "q05": "business", "q10": "Synthetic text"})
	sum, err = svc.Summary(ctx, scope, services.SurveySummaryFilter{FacilityID: saku})
	if err != nil {
		t.Fatal(err)
	}
	i01 := slices.IndexFunc(sum.Questions, func(q models.SurveyQuestionSummary) bool { return q.Code == "q01" })
	i02 := slices.IndexFunc(sum.Questions, func(q models.SurveyQuestionSummary) bool { return q.Code == "q02" })
	i10 := slices.IndexFunc(sum.Questions, func(q models.SurveyQuestionSummary) bool { return q.Code == "q10" })
	if q := sum.Questions[i01]; sum.Total != 5 || q.Suppressed || q.Mean == nil || *q.Mean != 3 {
		t.Fatalf("q01 = %+v (total %d)", q, sum.Total)
	}
	if q := sum.Questions[i02]; !q.Suppressed || q.Answered != 0 {
		t.Fatalf("q02 = %+v", q)
	}
	if q := sum.Questions[i10]; q.Answered != 5 || q.Counts != nil {
		t.Fatalf("q10 = %+v", q)
	}

	// Date filter on the UTC submission day.
	today := baseNow.Format("2006-01-02")
	sum, err = svc.Summary(ctx, scope, services.SurveySummaryFilter{From: today, To: today})
	if err != nil || sum.Total != 5 {
		t.Fatalf("today: %+v %v", sum, err)
	}
	sum, err = svc.Summary(ctx, scope, services.SurveySummaryFilter{From: baseNow.AddDate(0, 0, 1).Format("2006-01-02")})
	if err != nil || sum.Total != 0 {
		t.Fatalf("tomorrow: %+v %v", sum, err)
	}
}
