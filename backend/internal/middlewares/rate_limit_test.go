package middlewares

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newLimitedEngine(t *testing.T, perMin int, logs *bytes.Buffer) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	rl := NewRateLimiter(map[string]int{LimitCreate: perMin}, logger)
	r.Use(RequestID(), rl.Middleware())
	r.POST("/api/v1/guest-registrations", func(c *gin.Context) { c.Status(http.StatusCreated) })
	return r
}

func post(r *gin.Engine, remote, xff string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/guest-registrations", nil)
	req.RemoteAddr = remote
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	r.ServeHTTP(w, req)
	return w
}

func TestRateLimit429Envelope(t *testing.T) {
	var logs bytes.Buffer
	r := newLimitedEngine(t, 2, &logs)
	for i := 0; i < 2; i++ {
		if w := post(r, "203.0.113.7:1234", ""); w.Code != http.StatusCreated {
			t.Fatalf("request %d: %d", i, w.Code)
		}
	}
	w := post(r, "203.0.113.7:1234", "")
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("got %d, want 429", w.Code)
	}
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatal(err)
	}
	if m["code"] != "rate_limited" || m["request_id"] != w.Header().Get("X-Request-Id") || len(m) != 4 {
		t.Fatalf("unexpected envelope %s", w.Body.String())
	}
	if w.Header().Get("Retry-After") == "" {
		t.Fatal("missing Retry-After")
	}
	if strings.Contains(logs.String(), "203.0.113.7") {
		t.Fatal("client IP was logged")
	}
	if !strings.Contains(logs.String(), `"route":"/api/v1/guest-registrations"`) {
		t.Fatalf("expected route in WARN log: %s", logs.String())
	}
	// Another client is unaffected.
	if w := post(r, "203.0.113.8:1234", ""); w.Code != http.StatusCreated {
		t.Fatalf("other IP: %d", w.Code)
	}
}

func TestRateLimitXFFTrustBoundary(t *testing.T) {
	var logs bytes.Buffer
	// Untrusted RemoteAddr: a spoofed X-Forwarded-For is ignored, so rotating
	// it does not escape the RemoteAddr bucket.
	r := newLimitedEngine(t, 1, &logs)
	if w := post(r, "198.51.100.1:1", "192.0.2.1"); w.Code != http.StatusCreated {
		t.Fatalf("first: %d", w.Code)
	}
	if w := post(r, "198.51.100.1:1", "192.0.2.2"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("spoofed XFF from untrusted peer must be ignored, got %d", w.Code)
	}

	// Trusted proxy RemoteAddr: X-Forwarded-For is the client key.
	r = newLimitedEngine(t, 1, &logs)
	if w := post(r, "127.0.0.1:1", "192.0.2.1"); w.Code != http.StatusCreated {
		t.Fatalf("first: %d", w.Code)
	}
	if w := post(r, "127.0.0.1:1", "192.0.2.2"); w.Code != http.StatusCreated {
		t.Fatalf("distinct clients behind a trusted proxy share a bucket: %d", w.Code)
	}
	if w := post(r, "127.0.0.1:1", "192.0.2.1"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("same forwarded client must be limited: %d", w.Code)
	}
}

func TestClassifyRoute(t *testing.T) {
	cases := map[string]string{
		"POST /api/v1/guest-registrations":                                 LimitCreate,
		"GET /api/v1/guest-registrations/manage":                           LimitManage,
		"PATCH /api/v1/guest-registrations/manage/guests/:guest_public_id": LimitManage,
		"POST /api/v1/surveys":                                             LimitSurvey,
		"GET /api/v1/facilities/:code/public":                              "",
	}
	for in, want := range cases {
		m, p, _ := strings.Cut(in, " ")
		if got := ClassifyRoute(m, p); got != want {
			t.Errorf("%s: got %q want %q", in, got, want)
		}
	}
}
