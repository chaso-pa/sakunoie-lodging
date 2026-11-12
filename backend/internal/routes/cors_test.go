package routes_test

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/config"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/routes"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	testWebOrigin   = "http://127.0.0.1:3100"
	foreignOrigin   = "https://evil.example"
	allowedMethods  = "GET, POST, PATCH, OPTIONS"
	allowedHeaders  = "Authorization, Content-Type, Idempotency-Key, If-Match, X-Request-ID"
	exposedHeaders  = "ETag, X-Request-ID, Retry-After, Content-Disposition"
	createPath      = "/api/v1/guest-registrations"
	adminDetailPath = "/api/v1/admin/guest-registrations/00000000-0000-0000-0000-000000000000"
)

// countingVerifier records every admin token verification and rejects all.
type countingVerifier struct{ calls atomic.Int32 }

func (v *countingVerifier) VerifyIDToken(context.Context, string) (string, error) {
	v.calls.Add(1)
	return "", errors.New("rejected")
}

// newCORSEngine builds the production engine without a DB: only paths that
// answer before any service call are exercised. The create limit is 1/min.
func newCORSEngine(t *testing.T, origins []string) (*gin.Engine, *countingVerifier) {
	t.Helper()
	gin.SetMode(gin.TestMode)
	v := &countingVerifier{}
	cfg := &config.Config{
		AppEnv:                config.EnvTest,
		TrustedProxies:        []string{"127.0.0.1", "::1"},
		ClientIPHeader:        "X-Real-IP",
		CORSOrigins:           origins,
		RateLimitCreatePerMin: 1,
		RateLimitManagePerMin: 1,
		RateLimitSurveyPerMin: 1,
	}
	engine, err := routes.NewEngine(cfg, nil, routes.Deps{AdminAuth: services.NewAdminAuthService(nil, v)},
		slog.New(slog.NewJSONHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}
	return engine, v
}

func corsDo(engine *gin.Engine, method, path, body string, headers map[string]string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	req.RemoteAddr = "192.0.2.20:1234"
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	engine.ServeHTTP(w, req)
	return w
}

func preflight(engine *gin.Engine, origin, path, method, reqHeaders string) *httptest.ResponseRecorder {
	return corsDo(engine, http.MethodOptions, path, "", map[string]string{
		"Origin":                         origin,
		"Access-Control-Request-Method":  method,
		"Access-Control-Request-Headers": reqHeaders,
	})
}

func assertAllowedOrigin(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	h := w.Header()
	if got := h.Get("Access-Control-Allow-Origin"); got != testWebOrigin {
		t.Fatalf("Access-Control-Allow-Origin = %q (%d)", got, w.Code)
	}
	if got := h.Get("Access-Control-Expose-Headers"); got != exposedHeaders {
		t.Fatalf("Access-Control-Expose-Headers = %q", got)
	}
	if !strings.Contains(strings.Join(h.Values("Vary"), ","), "Origin") {
		t.Fatalf("Vary = %v", h.Values("Vary"))
	}
	if h.Get("Access-Control-Allow-Credentials") != "" {
		t.Fatal("credentials must never be allowed")
	}
	if h.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatal("missing nosniff")
	}
}

func assertNoCORS(t *testing.T, w *httptest.ResponseRecorder) {
	t.Helper()
	for _, k := range []string{"Access-Control-Allow-Origin", "Access-Control-Allow-Methods",
		"Access-Control-Allow-Headers", "Access-Control-Expose-Headers", "Access-Control-Allow-Credentials"} {
		if v := w.Header().Get(k); v != "" {
			t.Fatalf("%s = %q, want none", k, v)
		}
	}
}

func corsEnvelope(t *testing.T, w *httptest.ResponseRecorder, code string) {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("not JSON (%d): %s", w.Code, w.Body.String())
	}
	if m["code"] != code || len(m) != 4 || m["request_id"] != w.Header().Get("X-Request-Id") {
		t.Fatalf("envelope (%d): %s", w.Code, w.Body.String())
	}
}

func TestCORSPreflightAllowedOrigin(t *testing.T) {
	engine, v := newCORSEngine(t, []string{"https://other.example", testWebOrigin})
	cases := []struct{ path, method, headers string }{
		{createPath, "POST", "content-type,idempotency-key"},
		{"/api/v1/guest-registrations/manage", "PATCH", "authorization,content-type,if-match"},
		{"/api/v1/guest-registrations/manage", "GET", "authorization"},
		// Admin endpoint preflight with Authorization: answered before AdminAuth.
		{adminDetailPath, "PATCH", "authorization,content-type,if-match"},
		{"/api/v1/admin/facilities", "GET", "authorization,x-request-id"},
	}
	for _, c := range cases {
		w := preflight(engine, testWebOrigin, c.path, c.method, c.headers)
		if w.Code != http.StatusNoContent || w.Body.Len() != 0 {
			t.Fatalf("%s %s preflight: %d %q", c.method, c.path, w.Code, w.Body.String())
		}
		h := w.Header()
		if h.Get("Access-Control-Allow-Origin") != testWebOrigin || h.Get("Access-Control-Allow-Methods") != allowedMethods ||
			h.Get("Access-Control-Allow-Headers") != allowedHeaders || h.Get("Access-Control-Max-Age") != "600" {
			t.Fatalf("%s %s preflight headers: %v", c.method, c.path, h)
		}
		if h.Get("Access-Control-Allow-Credentials") != "" || h.Get("X-Content-Type-Options") != "nosniff" {
			t.Fatalf("preflight headers: %v", h)
		}
	}
	if n := v.calls.Load(); n != 0 {
		t.Fatalf("a preflight reached admin auth (%d verifications)", n)
	}
}

func TestCORSPreflightNeverRateLimited(t *testing.T) {
	engine, _ := newCORSEngine(t, []string{testWebOrigin})
	for i := 0; i < 5; i++ {
		if w := preflight(engine, testWebOrigin, createPath, "POST", "content-type"); w.Code != http.StatusNoContent {
			t.Fatalf("preflight %d: %d", i, w.Code)
		}
	}
	// The single create token (1/min) is still unused: the request reaches Huma
	// (422), and the error envelope carries ACAO so the browser can read it.
	w := corsDo(engine, http.MethodPost, createPath, `{}`, map[string]string{"Origin": testWebOrigin})
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("first POST after preflights: %d, want 422 (preflights must not count)", w.Code)
	}
	assertAllowedOrigin(t, w)
	corsEnvelope(t, w, "validation_error")
	// The next one is limited; the 429 envelope and Retry-After stay readable.
	w = corsDo(engine, http.MethodPost, createPath, `{}`, map[string]string{"Origin": testWebOrigin})
	if w.Code != http.StatusTooManyRequests || w.Header().Get("Retry-After") == "" {
		t.Fatalf("second POST: %d", w.Code)
	}
	assertAllowedOrigin(t, w)
	corsEnvelope(t, w, "rate_limited")
}

func TestCORSAdminErrorCarriesACAO(t *testing.T) {
	engine, v := newCORSEngine(t, []string{testWebOrigin})
	w := corsDo(engine, http.MethodGet, adminDetailPath, "", map[string]string{
		"Origin": testWebOrigin, "Authorization": "Bearer not-a-firebase-token",
	})
	if w.Code != http.StatusUnauthorized || v.calls.Load() != 1 {
		t.Fatalf("admin GET: %d, %d verifications", w.Code, v.calls.Load())
	}
	assertAllowedOrigin(t, w)
	corsEnvelope(t, w, "unauthenticated")
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("Cache-Control = %q", w.Header().Get("Cache-Control"))
	}

	// Unknown route (404 envelope) for an allowed origin.
	w = corsDo(engine, http.MethodGet, "/api/v1/nope", "", map[string]string{"Origin": testWebOrigin})
	if w.Code != http.StatusNotFound {
		t.Fatalf("unknown route: %d", w.Code)
	}
	assertAllowedOrigin(t, w)
	corsEnvelope(t, w, "not_found")
}

func TestCORSDisallowedOrigin(t *testing.T) {
	engine, v := newCORSEngine(t, []string{testWebOrigin})
	for _, origin := range []string{foreignOrigin, "http://127.0.0.1:3101", "https://127.0.0.1:3100", "null"} {
		w := preflight(engine, origin, adminDetailPath, "PATCH", "authorization")
		if w.Code != http.StatusNoContent {
			t.Fatalf("%s preflight: %d", origin, w.Code)
		}
		assertNoCORS(t, w)
		if w.Header().Get("Access-Control-Max-Age") != "" {
			t.Fatal("no Max-Age for a disallowed origin")
		}
		w = corsDo(engine, http.MethodPost, createPath, `{}`, map[string]string{"Origin": origin})
		assertNoCORS(t, w)
	}
	if v.calls.Load() != 0 {
		t.Fatal("a disallowed preflight reached admin auth")
	}
	// Same-origin / non-browser requests (no Origin) get no CORS headers, only Vary.
	w := corsDo(engine, http.MethodGet, "/api/v1/nope", "", nil)
	assertNoCORS(t, w)
	if !strings.Contains(strings.Join(w.Header().Values("Vary"), ","), "Origin") {
		t.Fatal("Vary: Origin missing")
	}
}

func TestCORSDisabledWithoutOrigins(t *testing.T) {
	engine, _ := newCORSEngine(t, nil)
	w := preflight(engine, testWebOrigin, createPath, "POST", "content-type")
	assertNoCORS(t, w)
	if w.Code == http.StatusNoContent {
		t.Fatal("CORS disabled: a preflight must not be answered as allowed")
	}
	w = corsDo(engine, http.MethodPost, createPath, `{}`, map[string]string{"Origin": testWebOrigin})
	assertNoCORS(t, w)
}

// Successful cross-origin GET, POST and PATCH (compose DB): each response
// carries ACAO for the allowed origin and keeps no-store and nosniff.
func TestIntegrationHTTPCORS(t *testing.T) {
	h := newHarness(t)
	origin := map[string]string{"Origin": testWebOrigin}

	w := h.do(http.MethodGet, "/api/v1/facilities/"+testutil.FacilitySaku+"/public", nil, origin)
	if w.Code != http.StatusOK {
		t.Fatalf("facility: %d", w.Code)
	}
	assertAllowedOrigin(t, w)

	body := testutil.CreateBody(testutil.FacilitySaku, "2026-12-01", "2026-12-03", 2)
	w = h.do(http.MethodPost, createPath, body, map[string]string{"Origin": testWebOrigin, "Idempotency-Key": "test-" + uuid.NewString()})
	if w.Code != http.StatusCreated || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	assertAllowedOrigin(t, w)
	var created models.CreateGuestRegistrationResult
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	auth := map[string]string{"Origin": testWebOrigin, "Authorization": "Bearer " + *created.EditToken}
	w = h.do(http.MethodGet, "/api/v1/guest-registrations/manage", nil, auth)
	if w.Code != http.StatusOK || w.Header().Get("ETag") == "" || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("manage get: %d", w.Code)
	}
	assertAllowedOrigin(t, w)

	auth["If-Match"] = w.Header().Get("ETag")
	w = h.do(http.MethodPatch, "/api/v1/guest-registrations/manage", map[string]any{"booking_source": "direct"}, auth)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("manage patch: %d %s", w.Code, w.Body.String())
	}
	assertAllowedOrigin(t, w)

	// The stale ETag now fails: the 412 envelope is readable cross-origin.
	w = h.do(http.MethodPatch, "/api/v1/guest-registrations/manage", map[string]any{"booking_source": "airbnb"}, auth)
	if w.Code != http.StatusPreconditionFailed || envelope(t, w)["code"] != "precondition_failed" {
		t.Fatalf("stale patch: %d %s", w.Code, w.Body.String())
	}
	assertAllowedOrigin(t, w)

	// A foreign origin gets the same answer without CORS headers.
	w = h.do(http.MethodGet, "/api/v1/facilities/"+testutil.FacilitySaku+"/public", nil, map[string]string{"Origin": foreignOrigin})
	if w.Code != http.StatusOK {
		t.Fatalf("facility (foreign origin): %d", w.Code)
	}
	assertNoCORS(t, w)
}
