package middlewares

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func newLimitedEngine(t *testing.T, perMin int, logs *bytes.Buffer) *gin.Engine {
	return newLimitedEngineHeader(t, perMin, logs, "X-Real-IP")
}

func newLimitedEngineHeader(t *testing.T, perMin int, logs *bytes.Buffer, header string) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := ConfigureClientIP(r, []string{"127.0.0.1", "::1"}, header); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewJSONHandler(logs, nil))
	rl := NewRateLimiter(map[string]int{LimitCreate: perMin}, logger)
	r.Use(RequestID(), rl.Middleware())
	r.POST("/api/v1/guest-registrations", func(c *gin.Context) { c.Status(http.StatusCreated) })
	return r
}

func post(r *gin.Engine, remote, xff string) *httptest.ResponseRecorder {
	return postHeaders(r, remote, xff, "")
}

func postHeaders(r *gin.Engine, remote, xff, realIP string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/guest-registrations", nil)
	req.RemoteAddr = remote
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	if realIP != "" {
		req.Header.Set("X-Real-IP", realIP)
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

func TestRateLimitClientIPTrustBoundary(t *testing.T) {
	var logs bytes.Buffer

	// Untrusted peer: neither X-Forwarded-For nor X-Real-IP is honoured, so
	// rotating them does not escape the RemoteAddr bucket.
	r := newLimitedEngine(t, 1, &logs)
	if w := postHeaders(r, "198.51.100.1:1", "192.0.2.1", "192.0.2.1"); w.Code != http.StatusCreated {
		t.Fatalf("first: %d", w.Code)
	}
	if w := postHeaders(r, "198.51.100.1:1", "192.0.2.2", "192.0.2.2"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("X-Forwarded-For/X-Real-IP from an untrusted peer must be ignored, got %d", w.Code)
	}

	// Trusted hop (the Next rewrite forwards the client's XFF unchanged): a
	// spoofed X-Forwarded-For must not change the bucket.
	r = newLimitedEngine(t, 1, &logs)
	if w := postHeaders(r, "127.0.0.1:1", "192.0.2.1", "203.0.113.50"); w.Code != http.StatusCreated {
		t.Fatalf("first: %d", w.Code)
	}
	if w := postHeaders(r, "127.0.0.1:1", "192.0.2.2", "203.0.113.50"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("rotating X-Forwarded-For through a trusted hop escaped the limit: %d", w.Code)
	}

	// Trusted hop: X-Real-IP (overwritten by the edge) is the client key.
	r = newLimitedEngine(t, 1, &logs)
	if w := postHeaders(r, "127.0.0.1:1", "", "192.0.2.1"); w.Code != http.StatusCreated {
		t.Fatalf("first: %d", w.Code)
	}
	if w := postHeaders(r, "127.0.0.1:1", "", "192.0.2.2"); w.Code != http.StatusCreated {
		t.Fatalf("distinct X-Real-IP clients behind a trusted hop share a bucket: %d", w.Code)
	}
	if w := postHeaders(r, "127.0.0.1:1", "", "192.0.2.1"); w.Code != http.StatusTooManyRequests {
		t.Fatalf("same X-Real-IP client must be limited: %d", w.Code)
	}

	// CLIENT_IP_HEADER=CF-Connecting-IP (Cloudflare Tunnel; cloudflared is the trusted hop).
	cf := func(r *gin.Engine, remote, xff, realIP, cfIP string) int {
		w := httptest.NewRecorder()
		req := httptest.NewRequest(http.MethodPost, "/api/v1/guest-registrations", nil)
		req.RemoteAddr = remote
		for k, v := range map[string]string{"X-Forwarded-For": xff, "X-Real-IP": realIP, "CF-Connecting-IP": cfIP} {
			if v != "" {
				req.Header.Set(k, v)
			}
		}
		r.ServeHTTP(w, req)
		return w.Code
	}

	// Trusted hop: CF-Connecting-IP is the client key; X-Real-IP is no longer read.
	r = newLimitedEngineHeader(t, 1, &logs, "CF-Connecting-IP")
	if code := cf(r, "127.0.0.1:1", "", "203.0.113.50", "192.0.2.1"); code != http.StatusCreated {
		t.Fatalf("first: %d", code)
	}
	if code := cf(r, "127.0.0.1:1", "", "203.0.113.50", "192.0.2.2"); code != http.StatusCreated {
		t.Fatalf("distinct CF-Connecting-IP clients behind a trusted hop share a bucket: %d", code)
	}
	if code := cf(r, "127.0.0.1:1", "", "203.0.113.51", "192.0.2.1"); code != http.StatusTooManyRequests {
		t.Fatalf("same CF-Connecting-IP client must be limited even with another X-Real-IP: %d", code)
	}

	// Untrusted peer: a client-sent CF-Connecting-IP is ignored (RemoteAddr bucket).
	r = newLimitedEngineHeader(t, 1, &logs, "CF-Connecting-IP")
	if code := cf(r, "198.51.100.1:1", "", "", "192.0.2.1"); code != http.StatusCreated {
		t.Fatalf("first: %d", code)
	}
	if code := cf(r, "198.51.100.1:1", "", "", "192.0.2.2"); code != http.StatusTooManyRequests {
		t.Fatalf("CF-Connecting-IP from an untrusted peer must be ignored, got %d", code)
	}

	// Trusted hop: a spoofed X-Forwarded-For neither changes the bucket nor
	// replaces a missing CF-Connecting-IP.
	r = newLimitedEngineHeader(t, 1, &logs, "CF-Connecting-IP")
	if code := cf(r, "127.0.0.1:1", "192.0.2.1", "", "192.0.2.9"); code != http.StatusCreated {
		t.Fatalf("first: %d", code)
	}
	if code := cf(r, "127.0.0.1:1", "192.0.2.2", "", "192.0.2.9"); code != http.StatusTooManyRequests {
		t.Fatalf("rotating X-Forwarded-For escaped the CF-Connecting-IP bucket: %d", code)
	}
	if code := cf(r, "127.0.0.1:1", "192.0.2.3", "", ""); code != http.StatusCreated {
		t.Fatalf("first request keyed on the peer: %d", code)
	}
	if code := cf(r, "127.0.0.1:1", "192.0.2.4", "", ""); code != http.StatusTooManyRequests {
		t.Fatalf("without CF-Connecting-IP, X-Forwarded-For must not become the key: %d", code)
	}
}

func TestRateLimitBucketCap(t *testing.T) {
	var logs bytes.Buffer
	rl := NewRateLimiter(map[string]int{LimitCreate: 1}, slog.New(slog.NewJSONHandler(&logs, nil)))
	rl.maxBuckets = 10
	for i := 0; i < 1000; i++ {
		rl.allow(LimitCreate, fmt.Sprintf("192.0.2.%d", i))
		if n := len(rl.buckets); n > rl.maxBuckets {
			t.Fatalf("after %d clients: %d buckets, cap %d", i+1, n, rl.maxBuckets)
		}
	}
	if !strings.Contains(logs.String(), "bucket cap reached") {
		t.Fatal("expected a WARN when the cap is reached")
	}
	// The newest client keeps its bucket (inserted after eviction).
	if rl.allow(LimitCreate, "192.0.2.999") {
		t.Fatal("the most recent client must still be limited")
	}
}

func TestSecurityHeadersNosniff(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(SecurityHeaders())
	r.NoRoute(NoRoute)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/nope", nil))
	if got := w.Header().Get("X-Content-Type-Options"); got != "nosniff" {
		t.Fatalf("X-Content-Type-Options = %q", got)
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
