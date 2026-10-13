package handlers

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/danielgtaylor/huma/v2"
)

// Codes that tell a client why the photos of a post were refused. They are
// the message of the single entry in the 422's "errors" list, whose location
// is body.media, or body.media[i] (0 based) for one photo; "detail" carries
// the sentence for people. A request body over the cap never gets this far:
// it is refused while the form is read, with "request body too large".
const (
	CodeTooManyPhotos    = "too_many_photos"
	CodePhotoTooLarge    = "photo_too_large"
	CodeUnsupportedImage = "unsupported_image"
)

func uploadError(detail, code, location string, value any) error {
	return huma.NewError(http.StatusUnprocessableEntity, detail,
		&huma.ErrorDetail{Message: code, Location: location, Value: value})
}

// serviceError maps an error from the service layer to an HTTP error.
// Anything it does not recognise is logged and answered with a bare 500.
func serviceError(ctx context.Context, op string, err error) error {
	var validation *services.ValidationError
	var image *storage.ImageError
	switch {
	case errors.Is(err, services.ErrTooManyPhotos):
		return uploadError(fmt.Sprintf("at most %d photos are allowed", services.MaxPostPhotos),
			CodeTooManyPhotos, "body.media", services.MaxPostPhotos)
	case errors.As(err, &image):
		location := fmt.Sprintf("body.media[%d]", image.Index)
		if errors.Is(err, storage.ErrImageTooLarge) {
			return uploadError(fmt.Sprintf("photo %d is too large (at most %dMB each)", image.Index+1, services.MaxPostPhotoBytes>>20),
				CodePhotoTooLarge, location, services.MaxPostPhotoBytes)
		}
		return uploadError(fmt.Sprintf("photo %d is not a supported image (JPEG, PNG or WebP)", image.Index+1),
			CodeUnsupportedImage, location, nil)
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
	case errors.Is(err, storage.ErrImageTooLarge):
		return huma.Error422UnprocessableEntity("file is too large")
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
