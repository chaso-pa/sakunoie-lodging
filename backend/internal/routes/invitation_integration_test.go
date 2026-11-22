package routes_test

import (
	"encoding/json"
	"maps"
	"net/http"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
	"github.com/google/uuid"
)

// The invitation endpoints over HTTP: admin issue/list/revoke with scope,
// the public check (minimal, no-store, stable codes) and the create gate.
func TestIntegrationHTTPInvitations(t *testing.T) {
	h := newHarness(t)
	testutil.GrantAdmin(t, h.db, "uid-a", services.RoleOwner, testutil.FacilitySaku)
	admin := map[string]string{"Authorization": "Bearer tok-a"}
	saku := strconv.FormatUint(testutil.FacilityID(t, h.db, testutil.FacilitySaku), 10)
	hono := strconv.FormatUint(testutil.FacilityID(t, h.db, testutil.FacilityHonolulu), 10)

	issue := func(facility string) *httpResult {
		id, _ := strconv.ParseUint(facility, 10, 64)
		return result(h.do(http.MethodPost, "/api/v1/admin/invitations", map[string]any{
			"facility_id": id, "check_in_date": "2026-12-01", "check_out_date": "2026-12-03",
			"booking_reference": "HM-TEST-1", "note": "Test Guest Note",
		}, admin))
	}

	// Unauthenticated and another facility.
	if w := h.do(http.MethodPost, "/api/v1/admin/invitations", map[string]any{}, nil); w.Code != http.StatusUnauthorized {
		t.Fatalf("no admin token: %d", w.Code)
	}
	if r := issue(hono); r.code != http.StatusForbidden || r.body["code"] != "forbidden" {
		t.Fatalf("other facility: %d %v", r.code, r.body)
	}

	r := issue(saku)
	if r.code != http.StatusCreated || r.cacheControl != "no-store" {
		t.Fatalf("issue: %d %v", r.code, r.body)
	}
	token, _ := r.body["invitation_token"].(string)
	inv := r.body["invitation"].(map[string]any)
	publicID := inv["public_id"].(string)
	if len(token) != 43 || r.body["invitation_url_path"] != "/stays/saku-no-ie/registration" || inv["status"] != "active" {
		t.Fatalf("issue body: %v", r.body)
	}

	// Public check: minimal fields only, no-store.
	bearer := map[string]string{"Authorization": "Bearer " + token}
	w := h.do(http.MethodGet, "/api/v1/invitations/current", nil, bearer)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("current: %d %s", w.Code, w.Body.String())
	}
	cur := decode(t, w.Body.Bytes())
	if strings.Join(slices.Sorted(maps.Keys(cur)), ",") != "check_in_date,check_out_date,expires_at,facility_code" ||
		strings.Contains(w.Body.String(), "Test Guest Note") || strings.Contains(w.Body.String(), "HM-TEST-1") {
		t.Fatalf("current body: %s", w.Body.String())
	}
	for hdr, code := range map[string]string{"": "invitation_required", "Bearer nope": "invitation_invalid"} {
		w := h.do(http.MethodGet, "/api/v1/invitations/current", nil, map[string]string{"Authorization": hdr})
		if w.Code != http.StatusUnauthorized || envelope(t, w)["code"] != code || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("current %q: %d %s", hdr, w.Code, w.Body.String())
		}
	}

	// List: scoped; never a token or hash.
	w = h.do(http.MethodGet, "/api/v1/admin/invitations?facility_id="+saku+"&status=active", nil, admin)
	list := decode(t, w.Body.Bytes())
	if w.Code != http.StatusOK || list["total"] != float64(1) || strings.Contains(w.Body.String(), token) ||
		strings.Contains(w.Body.String(), "hash") || strings.Contains(w.Body.String(), "token") {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	if w := h.do(http.MethodGet, "/api/v1/admin/invitations?facility_id="+hono, nil, admin); w.Code != http.StatusForbidden {
		t.Fatalf("list other facility: %d", w.Code)
	}
	if w := h.do(http.MethodGet, "/api/v1/admin/invitations?status=bogus", nil, admin); w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("bad status: %d", w.Code)
	}

	// Create with the invitation as Bearer, then it is used (409).
	body := testutil.CreateBody(testutil.FacilitySaku, "2026-12-01", "2026-12-03", 1)
	key := "test-" + uuid.NewString()
	if w := h.do(http.MethodPost, "/api/v1/guest-registrations", body, createHeaders(key, token)); w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	if w := h.do(http.MethodPost, "/api/v1/guest-registrations", body, createHeaders(key, token)); w.Code != http.StatusOK {
		t.Fatalf("same-key replay: %d %s", w.Code, w.Body.String())
	}
	w = h.do(http.MethodPost, "/api/v1/guest-registrations", body, createHeaders("test-"+uuid.NewString(), token))
	if w.Code != http.StatusConflict || envelope(t, w)["code"] != "invitation_used" {
		t.Fatalf("second key: %d %s", w.Code, w.Body.String())
	}
	w = h.do(http.MethodGet, "/api/v1/invitations/current", nil, bearer)
	if w.Code != http.StatusConflict || envelope(t, w)["code"] != "invitation_used" {
		t.Fatalf("current used: %d %s", w.Code, w.Body.String())
	}
	w = h.do(http.MethodPost, "/api/v1/admin/invitations/"+publicID+"/revoke", nil, admin)
	if w.Code != http.StatusConflict || envelope(t, w)["code"] != "invitation_used" {
		t.Fatalf("revoke used: %d %s", w.Code, w.Body.String())
	}

	// Dates mismatch → 422 invitation_dates_mismatch; revoked → 401 invitation_invalid.
	r = issue(saku)
	token2 := r.body["invitation_token"].(string)
	publicID2 := r.body["invitation"].(map[string]any)["public_id"].(string)
	shifted := testutil.CreateBody(testutil.FacilitySaku, "2026-12-02", "2026-12-03", 1)
	w = h.do(http.MethodPost, "/api/v1/guest-registrations", shifted, createHeaders("test-"+uuid.NewString(), token2))
	if w.Code != http.StatusUnprocessableEntity || envelope(t, w)["code"] != "invitation_dates_mismatch" {
		t.Fatalf("mismatch: %d %s", w.Code, w.Body.String())
	}
	w = h.do(http.MethodPost, "/api/v1/admin/invitations/"+publicID2+"/revoke", nil, admin)
	if w.Code != http.StatusOK || decode(t, w.Body.Bytes())["status"] != "revoked" {
		t.Fatalf("revoke: %d %s", w.Code, w.Body.String())
	}
	w = h.do(http.MethodPost, "/api/v1/guest-registrations", body, createHeaders("test-"+uuid.NewString(), token2))
	if w.Code != http.StatusUnauthorized || envelope(t, w)["code"] != "invitation_invalid" {
		t.Fatalf("revoked create: %d %s", w.Code, w.Body.String())
	}

	// Another facility's invitation: 404 on revoke.
	other := h.invite(t, testutil.FacilityHonolulu, "2026-12-01", "2026-12-03")
	var otherID string
	h.db.Raw("SELECT public_id FROM registration_invitations WHERE facility_id = ?", hono).Scan(&otherID)
	if w := h.do(http.MethodPost, "/api/v1/admin/invitations/"+otherID+"/revoke", nil, admin); w.Code != http.StatusNotFound {
		t.Fatalf("revoke other facility: %d", w.Code)
	}

	// No token anywhere in the logs.
	for _, secret := range []string{token, token2, other} {
		if strings.Contains(h.logs.String(), secret) {
			t.Fatal("logs contain an invitation token")
		}
	}
}

type httpResult struct {
	code         int
	cacheControl string
	body         map[string]any
}

func result(w interface {
	Result() *http.Response
}) *httpResult {
	res := w.Result()
	var m map[string]any
	_ = json.NewDecoder(res.Body).Decode(&m)
	return &httpResult{code: res.StatusCode, cacheControl: res.Header.Get("Cache-Control"), body: m}
}
