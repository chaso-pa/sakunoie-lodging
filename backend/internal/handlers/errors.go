package handlers

import (
	"context"
	"errors"
	"log/slog"
	"net/http"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
)

var domainErrors = []struct {
	kind    error
	status  int
	message string
}{
	{services.ErrValidation, http.StatusUnprocessableEntity, "validation failed"},
	{services.ErrFacilityUnavailable, http.StatusUnprocessableEntity, "facility is not available"},
	{services.ErrCapacityExceeded, http.StatusConflict, "capacity exceeded"},
	{services.ErrIdempotencyConflict, http.StatusConflict, "Idempotency-Key was already used with a different request"},
	{services.ErrTokenInvalid, http.StatusUnauthorized, "edit link is invalid"},
	{services.ErrTokenExpired, http.StatusGone, "edit link has expired"},
	{services.ErrFieldNotEditable, http.StatusForbidden, "field is not editable with an edit link"},
	{services.ErrNotFound, http.StatusNotFound, "not found"},
	{services.ErrForbidden, http.StatusForbidden, "forbidden"},
	{services.ErrPreconditionFailed, http.StatusPreconditionFailed, "resource changed; reload and retry"},
	{services.ErrUnauthenticated, http.StatusUnauthorized, "sign-in required"},
	{services.ErrNotAuthorized, http.StatusForbidden, "not authorized for the admin console"},
}

// toHTTPError maps a service error to the envelope with a stable code.
// Unknown errors become a generic 500 and are logged by kind only.
func toHTTPError(ctx context.Context, err error) error {
	for _, d := range domainErrors {
		if errors.Is(err, d.kind) {
			env := models.EnvelopeFor(ctx, d.status, d.kind.Error(), d.message)
			for _, f := range services.FieldsOf(err) {
				env.Details = append(env.Details, models.ErrorDetailOut{Location: f.Location, Message: f.Message})
			}
			return env
		}
	}
	if errors.Is(err, context.Canceled) {
		return models.EnvelopeFor(ctx, 499, "client_closed_request", "request cancelled")
	}
	slog.ErrorContext(ctx, "unhandled service error",
		"request_id", utils.RequestID(ctx),
		"error_kind", services.ErrorKind(err),
	)
	return models.EnvelopeFor(ctx, http.StatusInternalServerError, "", "internal error")
}
