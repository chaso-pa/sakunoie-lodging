package session_test

import (
	"context"
	"io"
	"net/http"
	"net/http/cookiejar"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/session"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

type valueOutput struct {
	Body struct {
		Value string `json:"value"`
	}
}

// The session written by request A must be readable by request B through the
// cookie, both from a raw Gin handler and from a Huma handler's ctx, when the
// engine is wrapped as sessionManager.LoadAndSave(r).
func TestSessionRoundTripIntegration(t *testing.T) {
	db := testutil.NewDB(t, "session")
	sm := session.New(db.Sessions, false)

	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/set", func(c *gin.Context) {
		sm.Put(c.Request.Context(), "greeting", "hello")
		c.Status(http.StatusNoContent)
	})
	r.GET("/api/get", func(c *gin.Context) {
		c.String(http.StatusOK, sm.GetString(c.Request.Context(), "greeting"))
	})
	api := humagin.NewWithGroup(r, r.Group("/api"), huma.DefaultConfig("test", "1.0.0"))
	huma.Get(api, "/huma-get", func(ctx context.Context, _ *struct{}) (*valueOutput, error) {
		out := &valueOutput{}
		out.Body.Value = sm.GetString(ctx, "greeting")
		return out, nil
	})

	srv := httptest.NewServer(sm.LoadAndSave(r))
	defer srv.Close()
	jar, _ := cookiejar.New(nil)
	client := &http.Client{Jar: jar}

	resp, err := client.Post(srv.URL+"/api/set", "application/json", nil)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	cookies := resp.Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != "sakunoie_session" || !c.HttpOnly || c.Secure || c.SameSite != http.SameSiteLaxMode || c.Path != "/api" {
		t.Errorf("unexpected cookie attributes: %+v", c)
	}
	if len(c.Value) != 43 {
		t.Errorf("token length %d, want 43 (sessions.token is CHAR(43))", len(c.Value))
	}
	if days := time.Until(c.Expires).Hours() / 24; days < 29.9 || days > 30.1 {
		t.Errorf("cookie expires in %.2f days, want 30", days)
	}

	for _, path := range []string{"/api/get", "/api/huma-get"} {
		resp, err := client.Get(srv.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		if !strings.Contains(string(body), "hello") {
			t.Errorf("GET %s = %q, want the value stored by the first request", path, body)
		}
	}

	// A client without the cookie sees nothing.
	resp, err = http.Get(srv.URL + "/api/get")
	if err != nil {
		t.Fatal(err)
	}
	body, _ := io.ReadAll(resp.Body)
	resp.Body.Close()
	if string(body) != "" {
		t.Errorf("cookieless request read %q", body)
	}

	// The row's expiry is compared with UTC_TIMESTAMP() by the store, so it
	// must be stored in UTC whatever loc the application DSN uses.
	var seconds float64
	if err := db.Sessions.QueryRow("SELECT TIMESTAMPDIFF(SECOND, UTC_TIMESTAMP(), expiry) FROM sessions WHERE token = ?", c.Value).Scan(&seconds); err != nil {
		t.Fatal(err)
	}
	if want := session.Lifetime.Seconds(); seconds < want-120 || seconds > want+120 {
		t.Errorf("stored expiry is %.0fs from now (UTC), want about %.0fs", seconds, want)
	}

	if !session.New(db.Sessions, true).Cookie.Secure {
		t.Error("cookie must be Secure when secure=true")
	}
}
