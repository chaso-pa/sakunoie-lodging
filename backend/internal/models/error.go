package models

import (
	"context"
	"errors"
	"net/http"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"github.com/danielgtaylor/huma/v2"
)

// ErrorEnvelope is the only error body of the API (contract §5):
// {code, message, details, request_id}. It never carries input values.
type ErrorEnvelope struct {
	Status    int              `json:"-"`
	Code      string           `json:"code" doc:"Stable machine-readable error code; clients localize by this" example:"validation_error"`
	Message   string           `json:"message" doc:"English summary for developers; not shown to guests verbatim"`
	Details   []ErrorDetailOut `json:"details" nullable:"false" doc:"Field-level problems (may be empty)"`
	RequestID string           `json:"request_id" doc:"Matches the X-Request-Id response header"`
}

// ErrorDetailOut is one field-level problem. Huma's ErrorDetail.Value is
// deliberately dropped: it echoes the submitted value (e.g. a passport number).
type ErrorDetailOut struct {
	Location string `json:"location" doc:"Where the problem is, e.g. body.guests[1].passport_number"`
	Message  string `json:"message" doc:"What is wrong"`
}

// Error formats only code and message, so logging the error is PII-safe.
func (e *ErrorEnvelope) Error() string { return e.Code + ": " + e.Message }

// GetStatus implements huma.StatusError.
func (e *ErrorEnvelope) GetStatus() int { return e.Status }

// ContentType implements huma.ContentTypeFilter: always plain JSON, never
// application/problem+json.
func (e *ErrorEnvelope) ContentType(string) string { return "application/json" }

// CodeForStatus is the default code when a domain error does not set one.
func CodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthenticated"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusMethodNotAllowed:
		return "method_not_allowed"
	case http.StatusNotAcceptable:
		return "not_acceptable"
	case http.StatusConflict:
		return "conflict"
	case http.StatusGone:
		return "gone"
	case http.StatusPreconditionFailed:
		return "precondition_failed"
	case http.StatusRequestEntityTooLarge:
		return "payload_too_large"
	case http.StatusUnsupportedMediaType:
		return "unsupported_media_type"
	case http.StatusUnprocessableEntity:
		return "validation_error"
	case http.StatusPreconditionRequired:
		return "precondition_required"
	case http.StatusTooManyRequests:
		return "rate_limited"
	case http.StatusServiceUnavailable:
		return "unavailable"
	}
	if status >= 500 {
		return "internal_error"
	}
	return "error"
}

// NewEnvelope is the single error constructor, installed as both
// huma.NewError (ctx == nil) and huma.NewErrorWithContext. Only the location
// and message of ErrorDetailer errors are kept; other errors are dropped from
// the body because their text may contain input values.
func NewEnvelope(ctx huma.Context, status int, msg string, errs ...error) huma.StatusError {
	env := &ErrorEnvelope{
		Status:  status,
		Code:    CodeForStatus(status),
		Message: msg,
		Details: []ErrorDetailOut{},
	}
	if status >= 500 && msg == "" {
		env.Message = "internal error"
	}
	for _, err := range errs {
		var d huma.ErrorDetailer
		if err == nil || !errors.As(err, &d) {
			continue
		}
		detail := d.ErrorDetail()
		if detail == nil {
			continue
		}
		env.Details = append(env.Details, ErrorDetailOut{Location: detail.Location, Message: detail.Message})
	}
	if ctx != nil {
		env.RequestID = utils.RequestID(ctx.Context())
	}
	return env
}

// InstallErrorConstructors routes every Huma error (validation, the
// ctx-less Error4xx helpers and OpenAPI schema generation) through
// NewEnvelope. cmd/server, cmd/openapi and tests all call it.
func InstallErrorConstructors() {
	huma.NewError = func(status int, msg string, errs ...error) huma.StatusError {
		return NewEnvelope(nil, status, msg, errs...)
	}
	huma.NewErrorWithContext = NewEnvelope
}

// RequestIDTransformer stamps request_id onto error envelopes created
// without a context (e.g. huma.Error404NotFound returned by a handler).
func RequestIDTransformer(ctx huma.Context, _ string, v any) (any, error) {
	if env, ok := v.(*ErrorEnvelope); ok {
		if env.RequestID == "" {
			env.RequestID = utils.RequestID(ctx.Context())
		}
		if env.Details == nil {
			env.Details = []ErrorDetailOut{}
		}
	}
	return v, nil
}

// EnvelopeFor builds an envelope outside Huma (raw Gin middleware).
func EnvelopeFor(ctx context.Context, status int, code, msg string) *ErrorEnvelope {
	if code == "" {
		code = CodeForStatus(status)
	}
	return &ErrorEnvelope{Status: status, Code: code, Message: msg, Details: []ErrorDetailOut{}, RequestID: utils.RequestID(ctx)}
}
