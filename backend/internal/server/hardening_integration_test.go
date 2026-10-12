package server_test

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/config"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/routes"
)

func wantSecurityHeaders(t *testing.T, what string, r response, release bool) {
	t.Helper()
	h := r.Header()
	for name, want := range map[string]string{
		"Cache-Control":          "no-store",
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
	} {
		if got := h.Get(name); got != want {
			t.Errorf("%s (status %d): %s = %q, want %q", what, r.Code, name, got, want)
		}
	}
	hsts := h.Get("Strict-Transport-Security")
	if release && hsts != middlewares.HSTSValue {
		t.Errorf("%s: Strict-Transport-Security = %q, want %q", what, hsts, middlewares.HSTSValue)
	}
	if !release && hsts != "" {
		t.Errorf("%s: Strict-Transport-Security = %q outside release mode", what, hsts)
	}
}

// Every response carries the security headers, whoever wrote it: handlers,
// the auth middleware (401), Huma's validation (422), the rate limiter
// (429), Gin's router (404) and the health check.
func TestSecurityHeadersIntegration(t *testing.T) {
	for _, release := range []bool{false, true} {
		mode := "debug"
		if release {
			mode = "release"
		}
		a := newAppWith(t, &routes.Deps{Config: &config.Config{GinMode: mode, TranslationProvider: "mymemory"}})
		anonymous, member := a.client(t), a.client(t)
		member.forwarded = "203.0.113.70"
		member.signup("Mika")

		check := func(what string, r response, status int) {
			t.Helper()
			if r.Code != status {
				t.Fatalf("%s %s = %d, want %d; body: %s", mode, what, r.Code, status, r.Body)
			}
			wantSecurityHeaders(t, mode+" "+what, r, release)
		}
		check("GET /health", anonymous.get("/health"), http.StatusOK)
		check("public 200", anonymous.get("/api/spots"), http.StatusOK)
		check("authenticated 200", member.get("/api/posts"), http.StatusOK)
		check("authenticated profile", member.get("/api/auth/me"), http.StatusOK)
		check("401 from RequireAuth", anonymous.get("/api/posts"), http.StatusUnauthorized)
		check("401 from a handler", anonymous.verifyInvite("WRONG"), http.StatusUnauthorized)
		check("403", anonymous.postSignup(map[string]string{"nickname": "x"}, nil), http.StatusForbidden)
		check("404 unknown route", anonymous.get("/api/no-such-route"), http.StatusNotFound)
		check("404 outside /api", anonymous.get("/nowhere"), http.StatusNotFound)
		check("404 from a handler", member.get("/api/posts/no-such-post"), http.StatusNotFound)
		check("405-ish wrong method", anonymous.send(http.MethodDelete, "/health", "", nil), http.StatusNotFound)
		check("415", anonymous.send(http.MethodPost, "/api/auth/verify-invite", "text/plain", strings.NewReader("{}")), http.StatusUnsupportedMediaType)
		check("422 from Huma", member.postJSON("/api/gps-visits", map[string]any{}), http.StatusUnprocessableEntity)
		check("422 from a service", member.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": " "}), http.StatusUnprocessableEntity)
		check("503", member.postJSON("/api/ai/plan", map[string]string{"message": "hi", "language": "ja"}), http.StatusServiceUnavailable)

		limited := a.client(t)
		limited.remoteAddr = "198.51.100.90:5000"
		for i := 1; i <= 5; i++ {
			limited.verifyInvite("WRONG")
		}
		check("429 from the rate limiter", limited.verifyInvite("WRONG"), http.StatusTooManyRequests)
	}
}

var docPaths = []string{
	"/api/docs", "/api/openapi.json", "/api/openapi.yaml", "/api/openapi-3.0.json", "/api/openapi-3.0.yaml",
	"/api/schemas/ErrorModel.json",
	"/api/docs/auth", "/api/openapi/auth.json", "/api/openapi/auth.yaml", "/api/openapi/auth-3.0.json", "/api/openapi/auth-3.0.yaml",
	"/api/schemas/auth/ErrorModel.json",
}

// Debug mode serves the docs, the OpenAPI documents and the schemas; the
// auth API's only to a member.
func TestAPIDocsInDebugModeIntegration(t *testing.T) {
	a := newAppWith(t, &routes.Deps{Config: &config.Config{GinMode: "debug"}})
	anonymous, member := a.client(t), a.client(t)
	member.signup("Mika")

	for _, path := range docPaths {
		if r := member.get(path); r.Code != http.StatusOK {
			t.Errorf("member GET %s = %d, want 200", path, r.Code)
		}
		want := http.StatusOK
		if strings.Contains(path, "/auth") {
			want = http.StatusUnauthorized
		}
		if r := anonymous.get(path); r.Code != want {
			t.Errorf("anonymous GET %s = %d, want %d", path, r.Code, want)
		}
	}
	// Bodies link to their schema.
	r := member.get("/api/auth/me")
	if _, ok := r.json(t)["$schema"]; !ok {
		t.Errorf("no $schema in a debug mode body: %s", r.Body)
	}
}

// Release mode serves none of them, to anyone, and the API itself works
// exactly as before without the schema links.
func TestAPIDocsDisabledInReleaseModeIntegration(t *testing.T) {
	a := newAppWith(t, &routes.Deps{Config: &config.Config{GinMode: "release"}})
	spotID := a.addSpot(t, "白駒の池", "東信", "自然", true)
	anonymous, member := a.client(t), a.client(t)
	memberID, _ := member.signup("Mika")

	for _, c := range []*client{anonymous, member} {
		for _, path := range docPaths {
			r := c.get(path)
			if r.Code != http.StatusNotFound {
				t.Errorf("release: GET %s = %d, want 404", path, r.Code)
			}
			if body := r.Body.String(); strings.Contains(body, "openapi") || strings.Contains(body, "verify-invite") {
				t.Errorf("release: GET %s describes the API: %s", path, body)
			}
		}
	}

	// Success bodies: same fields, no $schema, no Link header.
	r := member.get("/api/auth/me")
	wantStatus(t, r, http.StatusOK)
	body := r.json(t)
	if _, ok := body["$schema"]; ok {
		t.Errorf("release: body links to a schema that is not served: %s", r.Body)
	}
	if r.Header().Get("Link") != "" {
		t.Errorf("release: Link header = %q", r.Header().Get("Link"))
	}
	if obj(t, body, "member")["id"] != memberID {
		t.Fatalf("release: me = %s", r.Body)
	}

	// Validation still runs and errors keep their shape.
	r = member.postJSON("/api/spots/"+spotID+"/reviews", map[string]any{"stars": 9, "text": "x"})
	wantStatus(t, r, http.StatusUnprocessableEntity)
	problem := r.json(t)
	if problem["title"] != "Unprocessable Entity" || problem["status"] != 422.0 || problem["detail"] != "validation failed" {
		t.Fatalf("release: 422 body = %s", r.Body)
	}
	if errs := list(t, problem, "errors"); len(errs) == 0 || errs[0]["location"] != "body.stars" {
		t.Fatalf("release: 422 errors = %s", r.Body)
	}
	if !strings.HasPrefix(r.Header().Get("Content-Type"), "application/problem+json") {
		t.Errorf("release: 422 Content-Type = %q", r.Header().Get("Content-Type"))
	}
	wantDetail(t, member.get("/api/posts/no-such-post"), http.StatusNotFound, "post not found")
	wantDetail(t, anonymous.get("/api/posts"), http.StatusUnauthorized, "authentication required")
	wantStatus(t, member.postJSON("/api/gps-visits", map[string]any{"place_name": "x", "visited_date": "2026-10-08", "bogus": 1}), http.StatusUnprocessableEntity)

	// Defaults, enums, multipart and JSON bodies all work.
	wantStatus(t, member.postJSON("/api/spots/"+spotID+"/reviews", map[string]any{"stars": 5, "text": "よかった"}), http.StatusOK)
	r = member.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "release で投稿", "spot_id": spotID})
	wantStatus(t, r, http.StatusOK)
	if obj(t, r.json(t), "post")["language"] != "ja" {
		t.Fatalf("release: the language default was not applied: %s", r.Body)
	}
	r = member.get("/api/posts?limit=1")
	wantStatus(t, r, http.StatusOK)
	if r.json(t)["total"] != 1.0 {
		t.Fatalf("release: feed = %s", r.Body)
	}
	wantStatus(t, member.get("/api/rankings?type=bogus"), http.StatusUnprocessableEntity)
	wantStatus(t, anonymous.get("/api/spots"), http.StatusOK)
	wantStatus(t, anonymous.get("/health"), http.StatusOK)
}

// A spot's page returns its newest 100 reviews.
func TestSpotReviewsAreBoundedIntegration(t *testing.T) {
	a, _ := newCommunityApp(t)
	spotID := a.addSpot(t, "松本城", "中信", "歴史", false)
	c := a.client(t)
	memberID, _ := c.signup("Mika")

	reviews := make([]models.Review, 105)
	for i := range reviews {
		reviews[i] = models.Review{MemberID: memberID, SpotID: spotID, Stars: 1 + i%5, Text: fmt.Sprintf("review %03d", i)}
	}
	if err := a.db.Gorm.Omit("Member").Create(&reviews).Error; err != nil {
		t.Fatal(err)
	}
	// Give them distinct, increasing times: review 104 is the newest.
	if err := a.db.Gorm.Exec("UPDATE reviews SET created_at = TIMESTAMPADD(SECOND, CAST(SUBSTRING(text, 8) AS UNSIGNED), '2026-10-01 00:00:00')").Error; err != nil {
		t.Fatal(err)
	}

	r := c.get("/api/spots/" + spotID)
	wantStatus(t, r, http.StatusOK)
	got := list(t, r.json(t), "reviews")
	if len(got) != 100 {
		t.Fatalf("%d reviews returned, want 100", len(got))
	}
	if got[0]["text"] != "review 104" || got[99]["text"] != "review 005" {
		t.Fatalf("not the newest 100: first %v, last %v", got[0]["text"], got[99]["text"])
	}
	// The public list still counts them all.
	if spots := list(t, a.client(t).get("/api/spots").json(t), "spots"); spots[0]["review_count"] != 105.0 {
		t.Fatalf("review_count = %v, want 105", spots[0]["review_count"])
	}
}
