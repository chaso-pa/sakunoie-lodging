package middlewares

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestRequestLoggerUsesRouteTemplate(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var buf bytes.Buffer
	r := gin.New()
	r.Use(RequestID(), RequestLogger(slog.New(slog.NewJSONHandler(&buf, nil))))
	r.GET("/api/v1/admin/guest-registrations/:id", func(c *gin.Context) { c.Status(http.StatusOK) })

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/guest-registrations/abc-secret?q=Test%20Guest", nil)
	req.Header.Set("Authorization", "Bearer secret-token")
	r.ServeHTTP(httptest.NewRecorder(), req)

	out := buf.String()
	for _, leak := range []string{"abc-secret", "Test", "q=", "secret-token", "192.0.2"} {
		if strings.Contains(out, leak) {
			t.Fatalf("log contains %q: %s", leak, out)
		}
	}
	for _, want := range []string{`"route":"/api/v1/admin/guest-registrations/:id"`, `"status":200`, `"actor_type":"admin"`, `"request_id":"`, `"latency_ms":`} {
		if !strings.Contains(out, want) {
			t.Fatalf("log missing %s: %s", want, out)
		}
	}
}
