package routes_test

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/config"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/routes"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// syncBuffer is a goroutine-safe log sink.
type syncBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *syncBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *syncBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

type harness struct {
	db     *gorm.DB
	engine *gin.Engine
	logs   *syncBuffer
}

func newHarness(t *testing.T) *harness {
	t.Helper()
	db := testutil.RequireDB(t)
	gin.SetMode(gin.TestMode)
	logs := &syncBuffer{}
	logger := slog.New(slog.NewJSONHandler(logs, &slog.HandlerOptions{Level: slog.LevelDebug}))
	prev := slog.Default()
	slog.SetDefault(logger)
	t.Cleanup(func() { slog.SetDefault(prev) })

	clock := testutil.NewClock(time.Date(2026, 11, 20, 3, 0, 0, 0, time.UTC))
	cipher := testutil.PassportCipher()
	tokens := services.NewTokenService(clock)
	idem := services.NewIdempotencyService(db, clock, testutil.TestHMACKey, logger)
	cfg := &config.Config{
		AppEnv:                config.EnvTest,
		TrustedProxies:        []string{"127.0.0.1", "::1"},
		RateLimitCreatePerMin: 1000,
		RateLimitManagePerMin: 1000,
		RateLimitSurveyPerMin: 1000,
	}
	engine, err := routes.NewEngine(cfg, db, routes.Deps{
		Facilities:    services.NewFacilityService(db),
		Registrations: services.NewRegistrationService(db, clock, tokens, idem, cipher),
		Manage:        services.NewGuestManageService(db, clock, tokens, cipher),
		AdminAuth:     services.NewAdminAuthService(db, testutil.FakeVerifier{"tok-a": "uid-a", "tok-none": "uid-none"}),
		Admin:         services.NewAdminRegistrationService(db, clock, tokens, cipher),
		Surveys:       services.NewSurveyService(db, clock, 2),
	}, logger)
	if err != nil {
		t.Fatal(err)
	}
	return &harness{db: db, engine: engine, logs: logs}
}

func (h *harness) do(method, path string, body any, headers map[string]string) *httptest.ResponseRecorder {
	var rd *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rd = bytes.NewReader(b)
	} else {
		rd = bytes.NewReader(nil)
	}
	req := httptest.NewRequest(method, path, rd)
	req.RemoteAddr = "192.0.2.10:1234"
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	w := httptest.NewRecorder()
	h.engine.ServeHTTP(w, req)
	return w
}

func envelope(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("not JSON: %s", w.Body.String())
	}
	if len(m) != 4 || m["request_id"] != w.Header().Get("X-Request-Id") || m["request_id"] == "" {
		t.Fatalf("bad envelope (%d): %s", w.Code, w.Body.String())
	}
	if w.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("missing X-Content-Type-Options: nosniff (%d)", w.Code)
	}
	return m
}

const (
	secretPassport = "TESTPP9876"
	secretName     = "Test Redaction Person"
)

func foreignRep(passport string) models.GuestInput {
	g := testutil.ForeignGuest(secretName, passport)
	g.IsRepresentative = true
	return g
}

// Pre-mortem P2: PII never reaches logs, error bodies, audit rows or the
// idempotency table.
func TestNoPIIInLogsOrErrors(t *testing.T) {
	h := newHarness(t)
	key := func() map[string]string { return map[string]string{"Idempotency-Key": "test-" + uuid.NewString()} }

	// 1. Huma schema validation failure (pattern) on the passport.
	bad := testutil.CreateBody(testutil.FacilitySaku, "2026-12-01", "2026-12-03", 1, foreignRep(secretPassport+"!"))
	w := h.do(http.MethodPost, "/api/v1/guest-registrations", bad, key())
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("pattern failure: %d %s", w.Code, w.Body.String())
	}
	envelope(t, w)
	bodies := w.Body.String()

	// 2. Service validation failure carrying the passport and name.
	noNat := foreignRep(secretPassport)
	noNat.Nationality = nil
	w = h.do(http.MethodPost, "/api/v1/guest-registrations",
		testutil.CreateBody(testutil.FacilitySaku, "2026-12-01", "2026-12-03", 1, noNat), key())
	if w.Code != http.StatusUnprocessableEntity || envelope(t, w)["code"] != "validation_error" {
		t.Fatalf("service validation: %d %s", w.Code, w.Body.String())
	}
	bodies += w.Body.String()

	// 3. Successful create, then manage GET.
	w = h.do(http.MethodPost, "/api/v1/guest-registrations",
		testutil.CreateBody(testutil.FacilitySaku, "2026-12-01", "2026-12-03", 2, foreignRep(secretPassport)), key())
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	if w.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("create response must be no-store")
	}
	var created models.CreateGuestRegistrationResult
	_ = json.Unmarshal(w.Body.Bytes(), &created)
	token := *created.EditToken
	bodies += w.Body.String()

	w = h.do(http.MethodGet, "/api/v1/guest-registrations/manage", nil, map[string]string{"Authorization": "Bearer " + token})
	if w.Code != http.StatusOK {
		t.Fatalf("manage: %d %s", w.Code, w.Body.String())
	}
	bodies += w.Body.String()

	for _, secret := range []string{secretPassport, secretName} {
		if strings.Contains(bodies, secret) {
			t.Errorf("response bodies contain %q", secret)
		}
	}
	logs := h.logs.String()
	for _, secret := range []string{secretPassport, secretName, token, "Testland"} {
		if strings.Contains(logs, secret) {
			t.Errorf("logs contain %q", secret)
		}
	}
	if !strings.Contains(logs, `"route":"/api/v1/guest-registrations"`) {
		t.Fatalf("request log missing: %s", logs)
	}
	for _, q := range []string{
		"SELECT COUNT(*) FROM audit_logs WHERE CONCAT_WS('|', action, actor_type, IFNULL(changed_fields_json,'')) LIKE ?",
		"SELECT COUNT(*) FROM api_idempotency_keys WHERE CONCAT_WS('|', HEX(key_hash), HEX(request_hash), IFNULL(response_json,''), key_hash, request_hash) LIKE ?",
		"SELECT COUNT(*) FROM guests WHERE passport_number_ciphertext LIKE ?",
	} {
		for _, secret := range []string{secretPassport, secretName} {
			var n int64
			if err := h.db.Raw(q, "%"+secret+"%").Scan(&n).Error; err != nil {
				t.Fatal(err)
			}
			if n != 0 {
				t.Errorf("%q found by %s", secret, q)
			}
		}
	}
}

func TestIntegrationHTTPManageFlow(t *testing.T) {
	h := newHarness(t)
	body := testutil.CreateBody(testutil.FacilitySaku, "2026-12-01", "2026-12-03", 3)
	keyHdr := map[string]string{"Idempotency-Key": "test-" + uuid.NewString()}

	// Missing Idempotency-Key → 422 envelope.
	w := h.do(http.MethodPost, "/api/v1/guest-registrations", body, nil)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("no key: %d", w.Code)
	}
	envelope(t, w)

	w = h.do(http.MethodPost, "/api/v1/guest-registrations", body, keyHdr)
	if w.Code != http.StatusCreated {
		t.Fatalf("create: %d %s", w.Code, w.Body.String())
	}
	var created models.CreateGuestRegistrationResult
	_ = json.Unmarshal(w.Body.Bytes(), &created)

	// Replay → 200; different body → 409 idempotency_conflict.
	w = h.do(http.MethodPost, "/api/v1/guest-registrations", body, keyHdr)
	if w.Code != http.StatusOK {
		t.Fatalf("replay: %d", w.Code)
	}
	changed := testutil.CreateBody(testutil.FacilitySaku, "2026-12-01", "2026-12-04", 3)
	w = h.do(http.MethodPost, "/api/v1/guest-registrations", changed, keyHdr)
	if w.Code != http.StatusConflict || envelope(t, w)["code"] != "idempotency_conflict" {
		t.Fatalf("conflict: %d %s", w.Code, w.Body.String())
	}
	// Capacity → 409 capacity_exceeded.
	over := testutil.CreateBody(testutil.FacilitySaku, "2026-12-01", "2026-12-03", 5)
	w = h.do(http.MethodPost, "/api/v1/guest-registrations", over, map[string]string{"Idempotency-Key": "test-" + uuid.NewString()})
	if w.Code != http.StatusConflict || envelope(t, w)["code"] != "capacity_exceeded" {
		t.Fatalf("capacity: %d %s", w.Code, w.Body.String())
	}

	auth := map[string]string{"Authorization": "Bearer " + *created.EditToken}
	// No token / bad token → 401 token_invalid, no-store.
	for _, hdr := range []map[string]string{nil, {"Authorization": "Bearer nope"}} {
		w = h.do(http.MethodGet, "/api/v1/guest-registrations/manage", nil, hdr)
		if w.Code != http.StatusUnauthorized || envelope(t, w)["code"] != "token_invalid" || w.Header().Get("Cache-Control") != "no-store" {
			t.Fatalf("unauthenticated: %d %s %q", w.Code, w.Body.String(), w.Header().Get("Cache-Control"))
		}
	}

	w = h.do(http.MethodGet, "/api/v1/guest-registrations/manage", nil, auth)
	if w.Code != http.StatusOK || w.Header().Get("Cache-Control") != "no-store" || w.Header().Get("ETag") == "" {
		t.Fatalf("get: %d %v", w.Code, w.Header())
	}
	etag := w.Header().Get("ETag")

	// PATCH without If-Match → 422 (required header).
	w = h.do(http.MethodPatch, "/api/v1/guest-registrations/manage", map[string]any{"booking_source": "direct"}, auth)
	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("patch without If-Match: %d", w.Code)
	}
	// Forbidden field → 403 field_not_editable.
	w = h.do(http.MethodPatch, "/api/v1/guest-registrations/manage", map[string]any{"check_in_date": "2026-12-02"},
		map[string]string{"Authorization": auth["Authorization"], "If-Match": etag})
	if w.Code != http.StatusForbidden || envelope(t, w)["code"] != "field_not_editable" {
		t.Fatalf("forbidden field: %d %s", w.Code, w.Body.String())
	}

	w = h.do(http.MethodPost, "/api/v1/guest-registrations/manage/guests",
		map[string]any{"guests": []models.CompanionInput{testutil.Companion("Test Guest B"), testutil.Companion("Test Guest C")}}, auth)
	if w.Code != http.StatusOK {
		t.Fatalf("add: %d %s", w.Code, w.Body.String())
	}
	var sum models.ManageSummary
	_ = json.Unmarshal(w.Body.Bytes(), &sum)
	if sum.RegisteredCount != 3 || !sum.IsComplete {
		t.Fatalf("summary %+v", sum)
	}
	w = h.do(http.MethodPost, "/api/v1/guest-registrations/manage/guests",
		map[string]any{"guests": []models.CompanionInput{testutil.Companion("Test Guest D")}}, auth)
	if w.Code != http.StatusConflict || envelope(t, w)["code"] != "capacity_exceeded" {
		t.Fatalf("over planned: %d %s", w.Code, w.Body.String())
	}

	// Stale ETag → 412 precondition_failed.
	w = h.do(http.MethodPatch, "/api/v1/guest-registrations/manage", map[string]any{"booking_source": "direct"},
		map[string]string{"Authorization": auth["Authorization"], "If-Match": etag})
	if w.Code != http.StatusPreconditionFailed || envelope(t, w)["code"] != "precondition_failed" {
		t.Fatalf("stale etag: %d %s", w.Code, w.Body.String())
	}

	// Wildcard and weak tags never match: a write must name its version.
	for _, im := range []string{"*", "W/" + sum.Version, `W/"` + sum.Version + `"`} {
		w = h.do(http.MethodPatch, "/api/v1/guest-registrations/manage", map[string]any{"booking_source": "direct"},
			map[string]string{"Authorization": auth["Authorization"], "If-Match": im})
		if w.Code != http.StatusPreconditionFailed || envelope(t, w)["code"] != "precondition_failed" {
			t.Fatalf("If-Match %q: %d %s", im, w.Code, w.Body.String())
		}
	}

	// Unknown routes return the envelope too.
	w = h.do(http.MethodGet, "/api/v1/nope", nil, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("no route: %d", w.Code)
	}
	envelope(t, w)

	// Public facility endpoint.
	w = h.do(http.MethodGet, "/api/v1/facilities/saku-no-ie/public", nil, nil)
	if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"terms_version":"2026-11-01"`) || strings.Contains(w.Body.String(), `"id"`) {
		t.Fatalf("facility: %d %s", w.Code, w.Body.String())
	}
	w = h.do(http.MethodGet, "/api/v1/facilities/unknown/public", nil, nil)
	if w.Code != http.StatusNotFound || envelope(t, w)["code"] != "not_found" {
		t.Fatalf("unknown facility: %d", w.Code)
	}

	// /ready pings the DB.
	w = h.do(http.MethodGet, "/ready", nil, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("ready: %d", w.Code)
	}
}
