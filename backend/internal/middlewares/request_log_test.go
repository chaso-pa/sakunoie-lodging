package middlewares

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/logging"
	"github.com/gin-gonic/gin"
)

func newLoggedEngine(logs *bytes.Buffer) *gin.Engine {
	gin.SetMode(gin.TestMode)
	logger := logging.New(logs, slog.LevelDebug)
	r := gin.New()
	r.Use(RequestID(), RequestLogger(logger), Recovery(logger))
	return r
}

func TestRequestLoggerLogsRouteNotURL(t *testing.T) {
	var logs bytes.Buffer
	r := newLoggedEngine(&logs)
	r.GET("/api/weather/:id", func(c *gin.Context) {
		if logging.RequestID(c.Request.Context()) == "" {
			t.Error("no request id on the handler's context")
		}
		c.String(http.StatusOK, "ok")
	})

	req := httptest.NewRequest(http.MethodGet, "/api/weather/abc?latitude=35.123456&q=secret-search", nil)
	req.Header.Set("Cookie", "sakunoie_session=SESSIONTOKEN")
	req.Header.Set("Authorization", "Bearer APIKEY")
	req.Header.Set("X-Request-Id", "client-chosen")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	id := w.Header().Get(RequestIDHeader)
	if id == "" || id == "client-chosen" {
		t.Fatalf("X-Request-Id = %q", id)
	}
	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("log is not one JSON line: %s", logs.String())
	}
	if entry["msg"] != "request" || entry["method"] != "GET" || entry["route"] != "/api/weather/:id" ||
		entry["status"] != 200.0 || entry["request_id"] != id || entry["level"] != "INFO" {
		t.Fatalf("log entry = %v", entry)
	}
	for _, secret := range []string{"35.123456", "latitude", "secret-search", "SESSIONTOKEN", "APIKEY", "/api/weather/abc"} {
		if strings.Contains(logs.String(), secret) {
			t.Errorf("the log contains %q: %s", secret, logs.String())
		}
	}

	// A path that matches no route is not logged either.
	logs.Reset()
	r.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/api/transfer/ABCD-EFGH-IJKL", nil))
	if !strings.Contains(logs.String(), `"route":"unmatched"`) || strings.Contains(logs.String(), "ABCD-EFGH-IJKL") {
		t.Fatalf("unmatched request log = %s", logs.String())
	}
}

func TestRecoveryDoesNotDumpTheRequest(t *testing.T) {
	var logs bytes.Buffer
	r := newLoggedEngine(&logs)
	r.GET("/boom", func(*gin.Context) { panic("kaboom") })

	req := httptest.NewRequest(http.MethodGet, "/boom?latitude=35.123456", nil)
	req.Header.Set("Cookie", "sakunoie_session=SESSIONTOKEN")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusInternalServerError {
		t.Fatalf("status = %d, want 500", w.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body["detail"] != "internal server error" {
		t.Fatalf("body = %s", w.Body)
	}
	if strings.Contains(w.Body.String(), "kaboom") {
		t.Fatal("the panic value is in the response")
	}
	out := logs.String()
	if !strings.Contains(out, "panic recovered") || !strings.Contains(out, "kaboom") || !strings.Contains(out, `"status":500`) {
		t.Fatalf("log = %s", out)
	}
	for _, secret := range []string{"SESSIONTOKEN", "35.123456", "Cookie"} {
		if strings.Contains(out, secret) {
			t.Errorf("the log contains %q", secret)
		}
	}
}
