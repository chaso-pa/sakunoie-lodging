package services

import (
	"errors"
	"strings"
)

// Domain errors. internal/handlers maps them to HTTP status + stable code.
var (
	ErrValidation          = errors.New("validation_error")
	ErrFacilityUnavailable = errors.New("facility_unavailable")
	ErrCapacityExceeded    = errors.New("capacity_exceeded")
	ErrIdempotencyConflict = errors.New("idempotency_conflict")
	ErrTokenInvalid        = errors.New("token_invalid")
	ErrTokenExpired        = errors.New("token_expired")
	ErrFieldNotEditable    = errors.New("field_not_editable")
	ErrNotFound            = errors.New("not_found")
	ErrForbidden           = errors.New("forbidden")
	ErrPreconditionFailed  = errors.New("precondition_failed")
	ErrUnauthenticated     = errors.New("unauthenticated")
	ErrNotAuthorized       = errors.New("not_authorized")
)

// FieldError locates a problem without echoing the submitted value.
type FieldError struct {
	Location string
	Message  string
}

// DomainError wraps a sentinel with field details.
type DomainError struct {
	Kind   error
	Fields []FieldError
}

func (e *DomainError) Error() string {
	locs := make([]string, len(e.Fields))
	for i, f := range e.Fields {
		locs[i] = f.Location
	}
	return e.Kind.Error() + " [" + strings.Join(locs, ",") + "]"
}

func (e *DomainError) Unwrap() error { return e.Kind }

func newDomainError(kind error, fields ...FieldError) error {
	return &DomainError{Kind: kind, Fields: fields}
}

// FieldsOf returns the field details of err, if any.
func FieldsOf(err error) []FieldError {
	var de *DomainError
	if errors.As(err, &de) {
		return de.Fields
	}
	return nil
}
