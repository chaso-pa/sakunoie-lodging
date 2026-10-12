package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// Default base URLs of the external services.
const (
	DefaultOpenRouterBaseURL = "https://openrouter.ai/api/v1"
	DefaultMyMemoryBaseURL   = "https://api.mymemory.translated.net"
	DefaultOpenMeteoBaseURL  = "https://api.open-meteo.com"
)

// ExternalEndpoints overrides where the external services are reached. Empty
// fields mean the real service; tests point them at stub servers.
type ExternalEndpoints struct {
	OpenRouterBaseURL string
	MyMemoryBaseURL   string
	OpenMeteoBaseURL  string
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

// maxExternalResponseBytes bounds how much of an upstream response is read.
const maxExternalResponseBytes = 1 << 20

// ErrUpstream is the cause attached to every failed call to an external
// service. Its text is safe to log.
var ErrUpstream = errors.New("upstream request failed")

// upstreamError describes a transport failure without the request URL.
//
// *url.Error prints the full URL, and for these services the URL carries the
// text being translated or the coordinates of a weather lookup, neither of
// which may reach the logs. Only the underlying cause is kept.
func upstreamError(service string, err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		if urlErr.Timeout() {
			return fmt.Errorf("%s: %w: timeout", service, ErrUpstream)
		}
		err = urlErr.Err
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("%s: %w: timeout", service, ErrUpstream)
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("%s: %w: canceled", service, ErrUpstream)
	}
	return fmt.Errorf("%s: %w: %T", service, ErrUpstream, err)
}

// upstreamStatusError reports an unexpected HTTP status. The response body
// is not included: it is raw upstream text.
func upstreamStatusError(service string, status int) error {
	return fmt.Errorf("%s: %w: status %d", service, ErrUpstream, status)
}

func upstreamFormatError(service, what string) error {
	return fmt.Errorf("%s: %w: %s", service, ErrUpstream, what)
}

// readUpstream sends req and returns the status and at most
// maxExternalResponseBytes of the body.
func readUpstream(client *http.Client, service string, req *http.Request) (int, []byte, error) {
	resp, err := client.Do(req)
	if err != nil {
		return 0, nil, upstreamError(service, err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, maxExternalResponseBytes))
	if err != nil {
		return 0, nil, upstreamError(service, err)
	}
	return resp.StatusCode, body, nil
}
