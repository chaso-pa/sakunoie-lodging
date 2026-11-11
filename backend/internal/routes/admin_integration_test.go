package routes_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
	"github.com/google/uuid"
)

func (h *harness) create(t *testing.T, facility string, planned int) map[string]any {
	t.Helper()
	body := testutil.CreateBody(facility, "2026-12-01", "2026-12-03", planned)
	w := h.do(http.MethodPost, "/api/v1/guest-registrations", body, map[string]string{"Idempotency-Key": "test-" + uuid.NewString()})
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var m map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	return m
}

func decode(t *testing.T, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatalf("not JSON: %s", b)
	}
	return m
}

func TestIntegrationHTTPAdmin(t *testing.T) {
	h := newHarness(t)
	testutil.GrantAdmin(t, h.db, "uid-a", services.RoleOwner, testutil.FacilitySaku)
	own := h.create(t, testutil.FacilitySaku, 1)
	other := h.create(t, testutil.FacilityHonolulu, 1)
	ownID, otherID := own["public_id"].(string), other["public_id"].(string)
	auth := map[string]string{"Authorization": "Bearer tok-a"}
	hono := strconv.FormatUint(testutil.FacilityID(t, h.db, testutil.FacilityHonolulu), 10)
	saku := strconv.FormatUint(testutil.FacilityID(t, h.db, testutil.FacilitySaku), 10)

	// 401 without or with a bad token, 403 for an unknown uid; all before
	// input validation, all as the error envelope, never cached.
	for name, hdr := range map[string]map[string]string{
		"missing": nil,
		"bad":     {"Authorization": "Bearer forged"},
		"basic":   {"Authorization": "Basic dG9rLWE="},
	} {
		w := h.do(http.MethodGet, "/api/v1/admin/guest-registrations/not-a-uuid", nil, hdr)
		if w.Code != http.StatusUnauthorized || envelope(t, w)["code"] != "unauthenticated" ||
			w.Header().Get("WWW-Authenticate") != "Bearer" || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("%s token: %d %s", name, w.Code, w.Body.String())
		}
	}
	w := h.do(http.MethodGet, "/api/v1/admin/facilities", nil, map[string]string{"Authorization": "Bearer tok-none"})
	if w.Code != http.StatusForbidden || envelope(t, w)["code"] != "not_authorized" {
		t.Fatalf("unknown uid: %d %s", w.Code, w.Body.String())
	}

	w = h.do(http.MethodGet, "/api/v1/admin/facilities", nil, auth)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("facilities: %d %s", w.Code, w.Body.String())
	}
	if facs := decode(t, w.Body.Bytes())["facilities"].([]any); len(facs) != 1 {
		t.Fatalf("facilities = %v", facs)
	}

	// Isolation through HTTP: query parameters cannot widen the scope.
	w = h.do(http.MethodGet, "/api/v1/admin/guest-registrations?facility_id="+hono, nil, auth)
	if w.Code != http.StatusForbidden || envelope(t, w)["code"] != "forbidden" {
		t.Fatalf("list other facility: %d %s", w.Code, w.Body.String())
	}
	w = h.do(http.MethodGet, "/api/v1/admin/guest-registrations/"+otherID, nil, auth)
	if w.Code != http.StatusNotFound || envelope(t, w)["code"] != "not_found" {
		t.Fatalf("detail other facility: %d %s", w.Code, w.Body.String())
	}
	w = h.do(http.MethodGet, "/api/v1/admin/guest-registrations/export?facility_id="+hono, nil, auth)
	if w.Code != http.StatusForbidden {
		t.Fatalf("export other facility: %d %s", w.Code, w.Body.String())
	}
	for _, path := range []string{"/verify-identity", "/reissue-edit-link"} {
		w = h.do(http.MethodPost, "/api/v1/admin/guest-registrations/"+otherID+path, nil, auth)
		if w.Code != http.StatusNotFound {
			t.Fatalf("%s other facility: %d", path, w.Code)
		}
	}
	w = h.do(http.MethodGet, "/api/v1/admin/guest-registrations/not-a-uuid", nil, auth)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("non-uuid id: %d", w.Code)
	}

	w = h.do(http.MethodGet, "/api/v1/admin/guest-registrations?q=Test&from=2026-12-01&to=2026-12-31", nil, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	list := decode(t, w.Body.Bytes())
	if list["total"] != float64(1) || list["metrics"].(map[string]any)["incomplete"] != float64(0) {
		t.Fatalf("list = %v", list)
	}

	// Detail → PATCH with the ETag → verify → reissue.
	w = h.do(http.MethodGet, "/api/v1/admin/guest-registrations/"+ownID, nil, auth)
	etag := w.Header().Get("ETag")
	if w.Code != http.StatusOK || etag == "" || strings.Contains(w.Body.String(), "Test Town") {
		t.Fatalf("detail: %d etag=%q %s", w.Code, etag, w.Body.String())
	}
	w = h.do(http.MethodPatch, "/api/v1/admin/guest-registrations/"+ownID, map[string]any{"booking_source": "direct"}, auth)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("patch without If-Match: %d", w.Code)
	}
	w = h.do(http.MethodPatch, "/api/v1/admin/guest-registrations/"+ownID, map[string]any{"booking_source": "direct"},
		map[string]string{"Authorization": "Bearer tok-a", "If-Match": etag})
	if w.Code != http.StatusOK || decode(t, w.Body.Bytes())["booking_source"] != "direct" || w.Header().Get("ETag") == etag {
		t.Fatalf("patch: %d %s", w.Code, w.Body.String())
	}
	w = h.do(http.MethodPatch, "/api/v1/admin/guest-registrations/"+ownID, map[string]any{"booking_source": "airbnb"},
		map[string]string{"Authorization": "Bearer tok-a", "If-Match": etag})
	if w.Code != http.StatusPreconditionFailed {
		t.Fatalf("stale patch: %d", w.Code)
	}
	w = h.do(http.MethodPost, "/api/v1/admin/guest-registrations/"+ownID+"/verify-identity", nil, auth)
	if w.Code != http.StatusOK || decode(t, w.Body.Bytes())["identity_verified"] != true {
		t.Fatalf("verify: %d %s", w.Code, w.Body.String())
	}
	oldToken := own["edit_token"].(string)
	w = h.do(http.MethodPost, "/api/v1/admin/guest-registrations/"+ownID+"/reissue-edit-link", nil, auth)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("reissue: %d %s", w.Code, w.Body.String())
	}
	newToken := decode(t, w.Body.Bytes())["edit_token"].(string)
	w = h.do(http.MethodGet, "/api/v1/guest-registrations/manage", nil, map[string]string{"Authorization": "Bearer " + oldToken})
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("old token after reissue: %d", w.Code)
	}
	w = h.do(http.MethodGet, "/api/v1/guest-registrations/manage", nil, map[string]string{"Authorization": "Bearer " + newToken})
	if w.Code != http.StatusOK {
		t.Fatalf("new token: %d", w.Code)
	}

	// CSV download.
	w = h.do(http.MethodGet, "/api/v1/admin/guest-registrations/export?facility_id="+saku+"&from=2026-12-01&to=2026-12-01", nil, auth)
	if w.Code != http.StatusOK || w.Header().Get("Content-Type") != "text/csv; charset=utf-8" ||
		w.Header().Get("Content-Disposition") != `attachment; filename="registrations-saku-no-ie-2026-12-01-2026-12-01.csv"` ||
		w.Header().Get("Cache-Control") != "no-store" || !bytes.HasPrefix(w.Body.Bytes(), []byte("\xEF\xBB\xBFfacility_code,")) ||
		!strings.Contains(w.Body.String(), ownID) {
		t.Fatalf("export: %d %v\n%s", w.Code, w.Header(), w.Body.String())
	}
	w = h.do(http.MethodGet, "/api/v1/admin/guest-registrations/export", nil, auth)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("export without facility_id: %d", w.Code)
	}

	// Logs: route templates only; never a token or the guest's data.
	logs := h.logs.String()
	for _, secret := range []string{"tok-a", "Bearer", oldToken, newToken, "Test Guest", ownID} {
		if strings.Contains(logs, secret) {
			t.Fatalf("log contains %q", secret)
		}
	}
	if !strings.Contains(logs, `"route":"/api/v1/admin/guest-registrations/:id"`) || !strings.Contains(logs, `"actor_type":"admin"`) {
		t.Fatalf("admin requests not logged with the route template:\n%s", logs)
	}
}
