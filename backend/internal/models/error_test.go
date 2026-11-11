package models_test

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

type passportInput struct {
	Body struct {
		Passport string `json:"passport" pattern:"^[A-Z0-9]{6,20}$"`
	}
}

func newTestEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	models.InstallErrorConstructors()
	r := gin.New()
	r.Use(middlewares.RequestID())
	cfg := huma.DefaultConfig("t", "1")
	cfg.CreateHooks = nil
	cfg.Transformers = []huma.Transformer{models.RequestIDTransformer}
	api := humagin.New(r, cfg)
	huma.Post(api, "/validate", func(ctx context.Context, in *passportInput) (*struct{}, error) {
		return nil, nil
	})
	huma.Get(api, "/missing", func(ctx context.Context, _ *struct{}) (*struct{}, error) {
		return nil, huma.Error404NotFound("x")
	})
	return r
}

func decode(t *testing.T, w *httptest.ResponseRecorder) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("body is not JSON: %s", w.Body.String())
	}
	return m
}

func TestValidationEnvelopeHasNoValue(t *testing.T) {
	r := newTestEngine(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/validate", strings.NewReader(`{"passport":"testpp9876!"}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d: %s", w.Code, w.Body.String())
	}
	if strings.Contains(w.Body.String(), "testpp9876") {
		t.Fatalf("body echoes the submitted value: %s", w.Body.String())
	}
	m := decode(t, w)
	keys := []string{}
	for k := range m {
		keys = append(keys, k)
	}
	if len(m) != 4 || m["code"] != "validation_error" || m["message"] == nil {
		t.Fatalf("unexpected envelope keys %v: %s", keys, w.Body.String())
	}
	if m["request_id"] == "" || m["request_id"] != w.Header().Get("X-Request-Id") {
		t.Fatalf("request_id %v != header %q", m["request_id"], w.Header().Get("X-Request-Id"))
	}
	details := m["details"].([]any)
	if len(details) == 0 {
		t.Fatal("expected details")
	}
	for _, d := range details {
		dm := d.(map[string]any)
		if _, ok := dm["value"]; ok || len(dm) != 2 || dm["location"] == nil || dm["message"] == nil {
			t.Fatalf("detail must be exactly {location,message}: %v", dm)
		}
	}
	if ct := w.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("content-type %q", ct)
	}
}

func TestCtxLessErrorGetsRequestID(t *testing.T) {
	r := newTestEngine(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.Header.Set("X-Request-Id", "test-request-0001")
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status %d", w.Code)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("content-type %q, want application/json", ct)
	}
	m := decode(t, w)
	if m["request_id"] != "test-request-0001" || m["code"] != "not_found" {
		t.Fatalf("unexpected envelope: %s", w.Body.String())
	}
	if d, ok := m["details"].([]any); !ok || len(d) != 0 {
		t.Fatalf("details must be an empty array: %s", w.Body.String())
	}
}

func TestEnvelopeLogIsValueFree(t *testing.T) {
	models.InstallErrorConstructors()
	err := huma.NewError(http.StatusUnprocessableEntity, "validation failed",
		&huma.ErrorDetail{Location: "body.guests[0].passport_number", Message: "bad", Value: "TESTPP9876"})
	var buf bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buf, nil))
	logger.Error("request failed", "error", err, "error_text", err.Error())
	b, _ := json.Marshal(err)
	if strings.Contains(buf.String(), "TESTPP9876") || strings.Contains(string(b), "TESTPP9876") {
		t.Fatalf("value leaked: log=%s body=%s", buf.String(), b)
	}
}

func TestInvalidInboundRequestIDIsReplaced(t *testing.T) {
	r := newTestEngine(t)
	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/missing", nil)
	req.Header.Set("X-Request-Id", "bad id with spaces")
	r.ServeHTTP(w, req)
	if got := w.Header().Get("X-Request-Id"); got == "" || strings.Contains(got, " ") {
		t.Fatalf("request id %q", got)
	}
}
