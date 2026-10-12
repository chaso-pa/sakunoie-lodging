package middlewares

import (
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func newProxiedEngine(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		t.Fatal(err)
	}
	return r
}

func TestHasClientAddress(t *testing.T) {
	r := newProxiedEngine(t)
	var got bool
	r.GET("/", func(c *gin.Context) { got = HasClientAddress(c) })

	cases := []struct {
		name       string
		remoteAddr string
		forwarded  string
		want       bool
	}{
		{"proxy on this host, nothing forwarded", "127.0.0.1:4000", "", false},
		{"IPv6 loopback proxy", "[::1]:4000", "", false},
		// Next.js dev reports its own loopback peer in this form.
		{"proxy forwarding IPv4-mapped loopback", "127.0.0.1:4000", "::ffff:127.0.0.1", false},
		{"proxy forwarding loopback", "127.0.0.1:4000", "127.0.0.1", false},
		{"IPv4-mapped loopback peer", "[::ffff:127.0.0.1]:4000", "", false},
		{"proxy forwarding a client", "127.0.0.1:4000", "203.0.113.9", true},
		{"proxy forwarding an IPv6 client", "127.0.0.1:4000", "2001:db8::9", true},
		{"direct public client", "198.51.100.7:4000", "", true},
		// An untrusted peer cannot make itself look like the shared case.
		{"public client forging loopback", "198.51.100.7:4000", "127.0.0.1", true},
		{"private network client", "10.0.0.5:4000", "", true},
	}
	for _, c := range cases {
		header := http.Header{}
		if c.forwarded != "" {
			header.Set("X-Forwarded-For", c.forwarded)
		}
		serve(r, http.MethodGet, "/", c.remoteAddr, header)
		if got != c.want {
			t.Errorf("%s: HasClientAddress = %v, want %v", c.name, got, c.want)
		}
	}
}

// Without a client address the invite and transfer limits are one shared,
// higher ceiling; with one they are per client as before.
func TestSharedAddressLimits(t *testing.T) {
	const loopback, public = "127.0.0.1:4000", "198.51.100.7:4000"
	for _, c := range []struct {
		name               string
		pick               func(*RateLimiters) gin.HandlerFunc
		perClient, ceiling int
	}{
		{"invite", func(l *RateLimiters) gin.HandlerFunc { return l.Invite }, InviteLimitPerClient, InviteLimitShared},
		{"transfer", func(l *RateLimiters) gin.HandlerFunc { return l.Transfer }, TransferLimitPerClient, TransferLimitShared},
	} {
		r := newProxiedEngine(t)
		r.POST("/", c.pick(NewRateLimiters()), func(ctx *gin.Context) { ctx.Status(http.StatusOK) })
		call := func(addr string) int { return serve(r, http.MethodPost, "/", addr, nil).Code }

		for i := 1; i <= c.ceiling; i++ {
			if code := call(loopback); code != http.StatusOK {
				t.Fatalf("%s: shared request %d = %d, want 200", c.name, i, code)
			}
		}
		if code := call(loopback); code != http.StatusTooManyRequests {
			t.Fatalf("%s: shared request %d = %d, want 429", c.name, c.ceiling+1, code)
		}
		// The exhausted shared bucket does not touch clients with an address.
		for i := 1; i <= c.perClient; i++ {
			if code := call(public); code != http.StatusOK {
				t.Fatalf("%s: per-client request %d = %d, want 200", c.name, i, code)
			}
		}
		if code := call(public); code != http.StatusTooManyRequests {
			t.Fatalf("%s: per-client request %d = %d, want 429", c.name, c.perClient+1, code)
		}
	}
	if InviteLimitPerClient != 5 || TransferLimitPerClient != 3 || InviteLimitShared != 30 || TransferLimitShared != 30 {
		t.Fatal("the limits changed; update the comment in rate_limit.go, README.md and .env.example with them")
	}
}

func TestFailureLockoutSkipsSharedAddress(t *testing.T) {
	lockout := NewFailureLockout(5, 15*time.Minute)
	r := newProxiedEngine(t)
	r.POST("/", lockout.Middleware(), func(c *gin.Context) { c.Status(http.StatusUnauthorized) })

	for i := 1; i <= 20; i++ {
		if code := serve(r, http.MethodPost, "/", "127.0.0.1:4000", nil).Code; code != http.StatusUnauthorized {
			t.Fatalf("failure %d from the shared address = %d, want 401 (no lockout)", i, code)
		}
	}
	if len(lockout.entries) != 0 {
		t.Fatalf("the shared address was counted: %v", lockout.entries)
	}
	// A client with an address is still locked out after 5.
	for i := 1; i <= 5; i++ {
		serve(r, http.MethodPost, "/", "198.51.100.7:4000", nil)
	}
	if code := serve(r, http.MethodPost, "/", "198.51.100.7:4000", nil).Code; code != http.StatusTooManyRequests {
		t.Fatalf("6th attempt from a public address = %d, want 429", code)
	}
}
