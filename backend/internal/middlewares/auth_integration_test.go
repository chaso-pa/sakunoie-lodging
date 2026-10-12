package middlewares_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/session"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
	"github.com/gin-gonic/gin"
)

func TestRequireAuthIntegration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	db := testutil.NewDB(t, "middlewares")
	sm := session.New(db.Sessions, false)

	member := models.Member{Nickname: "Mika", PreferredLanguage: "ja", TransferCodeHash: "hash-1", SessionVersion: 3}
	if err := db.Gorm.Create(&member).Error; err != nil {
		t.Fatal(err)
	}

	r := gin.New()
	// Test-only login: writes whatever member id and version the query asks for.
	r.POST("/login", func(c *gin.Context) {
		ctx := c.Request.Context()
		sm.Put(ctx, session.KeyMemberID, c.Query("member"))
		version := 3
		if c.Query("version") == "stale" {
			version = 2
		}
		sm.Put(ctx, session.KeySessionVersion, version)
		c.Status(http.StatusNoContent)
	})
	protected := r.Group("/api")
	protected.Use(middlewares.RequireAuth(sm, db.Gorm))
	protected.GET("/whoami", func(c *gin.Context) {
		// Read it the way a Huma handler does: from the request context.
		c.String(http.StatusOK, middlewares.GetMemberID(c.Request.Context()))
	})
	handler := sm.LoadAndSave(r)

	do := func(method, target, cookie string) *httptest.ResponseRecorder {
		req := httptest.NewRequest(method, target, nil)
		if cookie != "" {
			req.AddCookie(&http.Cookie{Name: session.CookieName, Value: cookie})
		}
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, req)
		return w
	}
	login := func(query string) string {
		for _, c := range do(http.MethodPost, "/login?"+query, "").Result().Cookies() {
			if c.Name == session.CookieName {
				return c.Value
			}
		}
		t.Fatal("login set no session cookie")
		return ""
	}
	sessionRows := func(token string) (n int) {
		if err := db.Sessions.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", token).Scan(&n); err != nil {
			t.Fatal(err)
		}
		return n
	}

	t.Run("no session", func(t *testing.T) {
		w := do(http.MethodGet, "/api/whoami", "")
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "authentication required") {
			t.Fatalf("got %d %s", w.Code, w.Body)
		}
	})

	t.Run("valid session", func(t *testing.T) {
		token := login("member=" + member.ID)
		w := do(http.MethodGet, "/api/whoami", token)
		if w.Code != http.StatusOK || w.Body.String() != member.ID {
			t.Fatalf("got %d %q, want 200 with the member id on the request context", w.Code, w.Body)
		}
		// Sliding window: the cookie is reissued with a fresh expiry.
		if len(w.Result().Cookies()) != 1 {
			t.Error("authenticated request did not refresh the session cookie")
		}
	})

	t.Run("stale session version", func(t *testing.T) {
		token := login("member=" + member.ID + "&version=stale")
		w := do(http.MethodGet, "/api/whoami", token)
		if w.Code != http.StatusUnauthorized || !strings.Contains(w.Body.String(), "session expired") {
			t.Fatalf("got %d %s", w.Code, w.Body)
		}
		if sessionRows(token) != 0 {
			t.Error("stale session was not destroyed")
		}
	})

	t.Run("member no longer exists", func(t *testing.T) {
		token := login("member=deleted-member")
		if w := do(http.MethodGet, "/api/whoami", token); w.Code != http.StatusUnauthorized {
			t.Fatalf("got %d", w.Code)
		}
		if sessionRows(token) != 0 {
			t.Error("orphaned session was not destroyed")
		}
	})
}
