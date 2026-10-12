package handlers

import (
	"context"
	"errors"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/danielgtaylor/huma/v2"
)

// serviceError maps an error from the service layer to an HTTP error.
// Anything it does not recognise is logged and answered with a bare 500.
func serviceError(ctx context.Context, op string, err error) error {
	var validation *services.ValidationError
	switch {
	case errors.As(err, &validation):
		return huma.Error422UnprocessableEntity(validation.Message)
	case errors.Is(err, services.ErrInvalidNickname), errors.Is(err, services.ErrInvalidLanguage):
		return huma.Error422UnprocessableEntity(err.Error())
	case errors.Is(err, services.ErrPostNotFound):
		return huma.Error404NotFound("post not found")
	case errors.Is(err, services.ErrSpotNotFound):
		return huma.Error404NotFound("spot not found")
	case errors.Is(err, services.ErrVisitNotFound):
		return huma.Error404NotFound("visit not found")
	case errors.Is(err, services.ErrForbidden):
		return huma.Error403Forbidden("you can only change your own content")
	case errors.Is(err, services.ErrMemberNotFound):
		return huma.Error401Unauthorized("authentication required")
	case errors.Is(err, storage.ErrInvalidImage):
		return huma.Error422UnprocessableEntity("file is not a supported image (JPEG, PNG or WebP)")
	case errors.Is(err, storage.ErrNotConfigured):
		return huma.Error503ServiceUnavailable("image storage is not configured")
	case errors.Is(err, storage.ErrUnavailable):
		logFailure(ctx, op, err)
		return huma.Error503ServiceUnavailable("image storage is unavailable")
	}
	return internalError(ctx, op, err)
}
