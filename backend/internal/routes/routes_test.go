package routes

import (
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestDocsGating(t *testing.T) {
	gin.SetMode(gin.TestMode)
	paths := []string{"/docs", "/openapi.yaml", "/openapi.json", "/schemas/ErrorEnvelope.json"}

	off := gin.New()
	NewAPI(off, false, Deps{})
	for _, p := range paths {
		w := httptest.NewRecorder()
		off.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if w.Code != http.StatusNotFound {
			t.Errorf("production %s: %d, want 404", p, w.Code)
		}
	}

	on := gin.New()
	NewAPI(on, true, Deps{})
	for _, p := range paths {
		w := httptest.NewRecorder()
		on.ServeHTTP(w, httptest.NewRequest(http.MethodGet, p, nil))
		if w.Code != http.StatusOK {
			t.Errorf("development %s: %d, want 200", p, w.Code)
		}
	}
	w := httptest.NewRecorder()
	on.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/openapi.yaml", nil))
	committed, err := os.ReadFile("../../api/openapi.yaml")
	if err != nil {
		t.Fatal(err)
	}
	if w.Body.String() != string(committed) {
		t.Error("served /openapi.yaml differs from the committed api/openapi.yaml")
	}
}
