// Package logging sets up structured logging with slog and carries the
// request id through the request context.
//
// What must never be logged: API keys, the invite code, transfer codes,
// session tokens, post or review text, and coordinates. The request logger
// in internal/middlewares therefore records the route template rather than
// the URL (query strings carry search text and coordinates), and services
// strip request URLs from upstream errors before returning them.
package logging

import (
	"context"
	"io"
	"log/slog"
)

type contextKey string

const (
	requestIDKey contextKey = "request_id"
	memberIDKey  contextKey = "log_member_id"
)

// WithRequestID returns ctx carrying the request id.
func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

// RequestID returns the request id on ctx, or "".
func RequestID(ctx context.Context) string {
	id, _ := ctx.Value(requestIDKey).(string)
	return id
}

// WithMemberID returns ctx carrying the authenticated member id, so every
// log line written with that context names the member.
func WithMemberID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, memberIDKey, id)
}

func memberID(ctx context.Context) string {
	id, _ := ctx.Value(memberIDKey).(string)
	return id
}

// contextHandler adds request_id and member_id from the context to every
// record logged through slog's *Context functions.
type contextHandler struct {
	slog.Handler
}

func (h contextHandler) Handle(ctx context.Context, r slog.Record) error {
	if id := RequestID(ctx); id != "" {
		r.AddAttrs(slog.String("request_id", id))
	}
	if id := memberID(ctx); id != "" {
		r.AddAttrs(slog.String("member_id", id))
	}
	return h.Handler.Handle(ctx, r)
}

func (h contextHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return contextHandler{h.Handler.WithAttrs(attrs)}
}

func (h contextHandler) WithGroup(name string) slog.Handler {
	return contextHandler{h.Handler.WithGroup(name)}
}

// New returns a JSON logger writing to w.
func New(w io.Writer, level slog.Level) *slog.Logger {
	return slog.New(contextHandler{slog.NewJSONHandler(w, &slog.HandlerOptions{Level: level})})
}

// Setup makes a JSON logger on w the process default and returns it.
func Setup(w io.Writer, level slog.Level) *slog.Logger {
	logger := New(w, level)
	slog.SetDefault(logger)
	return logger
}
