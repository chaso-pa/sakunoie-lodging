package routes_test

import (
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
)

func TestIntegrationHTTPSurvey(t *testing.T) {
	h := newHarness(t) // SURVEY_MIN_RESPONSES = 2
	testutil.GrantAdmin(t, h.db, "uid-a", services.RoleOwner, testutil.FacilitySaku)
	auth := map[string]string{"Authorization": "Bearer tok-a"}
	secret := "Synthetic survey comment 7731"

	post := func(body map[string]any) int {
		t.Helper()
		return h.do(http.MethodPost, "/api/v1/surveys", body, nil).Code
	}
	valid := func() map[string]any {
		return map[string]any{"facility_code": testutil.FacilitySaku, "survey_version": "v1",
			"answers": map[string]any{"q01": 4, "q08": []string{"quiet"}, "q10": secret}}
	}
	if c := post(valid()); c != http.StatusAccepted {
		t.Fatalf("submit: %d", c)
	}
	unknown := valid()
	unknown["facility_code"] = "no-such-facility"
	w := h.do(http.MethodPost, "/api/v1/surveys", unknown, nil)
	if w.Code != http.StatusUnprocessableEntity || envelope(t, w)["code"] != "facility_unavailable" {
		t.Fatalf("unknown facility: %d %s", w.Code, w.Body.String())
	}
	bad := valid()
	bad["answers"] = map[string]any{"q01": 6}
	w = h.do(http.MethodPost, "/api/v1/surveys", bad, nil)
	if w.Code != http.StatusUnprocessableEntity || envelope(t, w)["code"] != "validation_error" ||
		!strings.Contains(w.Body.String(), "body.answers.q01") {
		t.Fatalf("invalid rating: %d %s", w.Code, w.Body.String())
	}
	hp := valid()
	hp["website"] = "spam"
	if c := post(hp); c != http.StatusAccepted {
		t.Fatalf("honeypot: %d", c)
	}
	if n := testutil.CountRows(t, h.db, "survey_responses"); n != 1 {
		t.Fatalf("rows = %d", n)
	}

	// Admin summary: auth, scope, suppression below 2, aggregates only.
	if w := h.do(http.MethodGet, "/api/v1/admin/surveys/summary", nil, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("summary without token: %d", w.Code)
	}
	hono := strconv.FormatUint(testutil.FacilityID(t, h.db, testutil.FacilityHonolulu), 10)
	w = h.do(http.MethodGet, "/api/v1/admin/surveys/summary?facility_id="+hono, nil, auth)
	if w.Code != http.StatusForbidden || envelope(t, w)["code"] != "forbidden" {
		t.Fatalf("summary other facility: %d %s", w.Code, w.Body.String())
	}
	w = h.do(http.MethodGet, "/api/v1/admin/surveys/summary", nil, auth)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" ||
		!strings.Contains(w.Body.String(), `"suppressed":true`) || strings.Contains(w.Body.String(), `"suppressed":false`) {
		t.Fatalf("suppressed summary: %d %s", w.Code, w.Body.String())
	}
	if c := post(valid()); c != http.StatusAccepted {
		t.Fatalf("second submit: %d", c)
	}
	w = h.do(http.MethodGet, "/api/v1/admin/surveys/summary", nil, auth)
	sum := decode(t, w.Body.Bytes())
	q01 := sum["questions"].([]any)[0].(map[string]any)
	if w.Code != http.StatusOK || sum["total"] != float64(2) || q01["suppressed"] != false || q01["mean"] != float64(4) {
		t.Fatalf("summary: %d %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), secret) || strings.Contains(h.logs.String(), secret) {
		t.Fatal("free text leaked into the summary or the logs")
	}
	if !strings.Contains(h.logs.String(), `"route":"/api/v1/surveys"`) {
		t.Fatal("survey requests not logged with the route template")
	}
}
