package services

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// stub is an HTTP server standing in for an external service. It records
// every request it receives.
type stub struct {
	*httptest.Server
	mu       sync.Mutex
	requests []stubRequest
}

type stubRequest struct {
	Method string
	Path   string
	Query  map[string]string
	Auth   string
	Body   []byte
}

func newStub(t *testing.T, handler func(w http.ResponseWriter, r stubRequest)) *stub {
	t.Helper()
	s := &stub{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		req := stubRequest{Method: r.Method, Path: r.URL.Path, Auth: r.Header.Get("Authorization"), Body: body, Query: map[string]string{}}
		for k := range r.URL.Query() {
			req.Query[k] = r.URL.Query().Get(k)
		}
		s.mu.Lock()
		s.requests = append(s.requests, req)
		s.mu.Unlock()
		handler(w, req)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *stub) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

func (s *stub) last() stubRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests[len(s.requests)-1]
}

// assertNoLeak fails when err's text contains anything that must not reach
// a log line or a response.
func assertNoLeak(t *testing.T, err error, secrets ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, secret := range secrets {
		if secret != "" && strings.Contains(err.Error(), secret) {
			t.Fatalf("error text leaks %q: %v", secret, err)
		}
	}
}

func chatReply(content, model string) string {
	b, _ := json.Marshal(map[string]any{
		"model":   model,
		"choices": []map[string]any{{"message": map[string]string{"role": "assistant", "content": content}}},
	})
	return string(b)
}

func decodeChat(t *testing.T, body []byte) chatRequest {
	t.Helper()
	var req chatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		t.Fatalf("request body is not a chat request: %s", body)
	}
	return req
}

func TestUpstreamErrorDropsURL(t *testing.T) {
	// A request to a closed port fails with a *url.Error naming the URL.
	srv := httptest.NewServer(http.NotFoundHandler())
	url := srv.URL
	srv.Close()

	req, _ := http.NewRequest(http.MethodGet, url+"/get?q=secret+post+body&latitude=35.123456", nil)
	_, _, err := readUpstream(&http.Client{Timeout: time.Second}, "svc", req)
	if !errors.Is(err, ErrUpstream) {
		t.Fatalf("err = %v", err)
	}
	assertNoLeak(t, err, "secret", "35.123456", url, "127.0.0.1")

	if got := upstreamError("svc", context.DeadlineExceeded).Error(); !strings.Contains(got, "timeout") {
		t.Fatalf("timeout not reported: %s", got)
	}
}
