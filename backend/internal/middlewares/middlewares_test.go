package middlewares

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func serve(r http.Handler, method, path, remoteAddr string, header http.Header) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, path, strings.NewReader(""))
	req.RemoteAddr = remoteAddr
	for k, v := range header {
		req.Header[k] = v
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

func TestGetMemberIDWithoutAuth(t *testing.T) {
	if id := GetMemberID(context.Background()); id != "" {
		t.Fatalf("got %q, want empty", id)
	}
}

func TestForRouteMatchesOnlyItsRoute(t *testing.T) {
	r := gin.New()
	g := r.Group("/api")
	g.Use(ForRoute(http.MethodPost, "/api/items/:id", func(c *gin.Context) {
		c.Header("X-Matched", "yes")
		c.Next()
	}))
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	g.POST("/items/:id", ok)
	g.GET("/items/:id", ok)
	g.POST("/other", ok)

	cases := []struct {
		method, path string
		matched      bool
	}{
		{http.MethodPost, "/api/items/42", true},
		{http.MethodGet, "/api/items/42", false},
		{http.MethodPost, "/api/other", false},
	}
	for _, c := range cases {
		w := serve(r, c.method, c.path, "127.0.0.1:1", nil)
		if w.Code != http.StatusOK {
			t.Errorf("%s %s = %d, want 200", c.method, c.path, w.Code)
		}
		if got := w.Header().Get("X-Matched") == "yes"; got != c.matched {
			t.Errorf("%s %s matched = %v, want %v", c.method, c.path, got, c.matched)
		}
	}
}

func TestMaxBodySize(t *testing.T) {
	r := gin.New()
	r.Use(MaxBodySize(10))
	r.POST("/", func(c *gin.Context) {
		if _, err := c.GetRawData(); err != nil {
			c.Status(http.StatusRequestEntityTooLarge)
			return
		}
		c.Status(http.StatusOK)
	})

	post := func(body string) int {
		w := httptest.NewRecorder()
		r.ServeHTTP(w, httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body)))
		return w.Code
	}
	if code := post("0123456789"); code != http.StatusOK {
		t.Errorf("body at the limit = %d, want 200", code)
	}
	if code := post("0123456789X"); code != http.StatusRequestEntityTooLarge {
		t.Errorf("body over the limit = %d, want the read to fail", code)
	}
}

// fakeAuth stands in for RequireAuth: it takes the member id from a header.
func fakeAuth(c *gin.Context) {
	ctx := context.WithValue(c.Request.Context(), MemberIDKey, c.GetHeader("X-Test-Member"))
	c.Request = c.Request.WithContext(ctx)
	c.Next()
}

// Members get separate buckets, and the same member's AI and translate
// limits are counted independently.
func TestMemberKeyedLimitsAreSeparate(t *testing.T) {
	limits := NewRateLimiters()
	r := gin.New()
	r.Use(fakeAuth)
	ok := func(c *gin.Context) { c.Status(http.StatusOK) }
	r.POST("/ai", limits.AI, ok)
	r.POST("/translate", limits.Translate, ok)

	call := func(path, member string) int {
		// Same address for everyone: the key must be the member, not the IP.
		return serve(r, http.MethodPost, path, "127.0.0.1:1", http.Header{"X-Test-Member": {member}}).Code
	}

	for i := 1; i <= 10; i++ {
		if code := call("/ai", "member-a"); code != http.StatusOK {
			t.Fatalf("member-a AI request %d = %d, want 200", i, code)
		}
	}
	if code := call("/ai", "member-a"); code != http.StatusTooManyRequests {
		t.Fatalf("member-a AI request 11 = %d, want 429", code)
	}
	if code := call("/ai", "member-b"); code != http.StatusOK {
		t.Errorf("member-b AI = %d, want 200: another member's limit must not apply", code)
	}
	if code := call("/translate", "member-a"); code != http.StatusOK {
		t.Errorf("member-a translate = %d, want 200: the AI limit must not spill into translate", code)
	}
}

func TestRateLimitResponseBody(t *testing.T) {
	limits := NewRateLimiters()
	r := gin.New()
	r.POST("/", limits.Transfer, func(c *gin.Context) { c.Status(http.StatusOK) })

	var w *httptest.ResponseRecorder
	for range 4 {
		w = serve(r, http.MethodPost, "/", "198.51.100.7:1", nil)
	}
	if w.Code != http.StatusTooManyRequests {
		t.Fatalf("4th request = %d, want 429", w.Code)
	}
	if body := w.Body.String(); !strings.Contains(body, `"status":429`) || !strings.Contains(body, `"detail":"rate limit exceeded"`) {
		t.Errorf("unexpected 429 body: %s", body)
	}
}

func TestFailureLockout(t *testing.T) {
	now := time.Date(2026, 10, 12, 12, 0, 0, 0, time.UTC)
	lockout := NewFailureLockout(5, 15*time.Minute)
	lockout.now = func() time.Time { return now }

	status := http.StatusUnauthorized
	r := gin.New()
	r.POST("/", lockout.Middleware(), func(c *gin.Context) { c.Status(status) })
	call := func(addr string) int {
		return serve(r, http.MethodPost, "/", addr, nil).Code
	}
	const alice, bob = "198.51.100.1:1", "198.51.100.2:1"

	// A success in the middle resets the streak.
	for range 4 {
		call(alice)
	}
	status = http.StatusOK
	if code := call(alice); code != http.StatusOK {
		t.Fatalf("got %d, want 200", code)
	}
	status = http.StatusUnauthorized

	for i := 1; i <= 5; i++ {
		if code := call(alice); code != http.StatusUnauthorized {
			t.Fatalf("failure %d = %d, want 401", i, code)
		}
	}
	if code := call(alice); code != http.StatusTooManyRequests {
		t.Fatalf("after 5 consecutive failures = %d, want 429", code)
	}
	// Even a correct attempt is refused while blocked.
	status = http.StatusOK
	if code := call(alice); code != http.StatusTooManyRequests {
		t.Fatalf("blocked client = %d, want 429", code)
	}
	if code := call(bob); code != http.StatusOK {
		t.Errorf("other client = %d, want 200", code)
	}

	now = now.Add(15*time.Minute - time.Second)
	if code := call(alice); code != http.StatusTooManyRequests {
		t.Errorf("just before the block ends = %d, want 429", code)
	}
	now = now.Add(2 * time.Second)
	if code := call(alice); code != http.StatusOK {
		t.Errorf("after the block ends = %d, want 200", code)
	}
}
