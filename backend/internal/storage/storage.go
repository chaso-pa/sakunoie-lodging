// Package storage defines how user-supplied images are stored.
package storage

import (
	"context"
	"errors"
	"fmt"
	"io"
)

var (
	// ErrNotConfigured is returned when an image is supplied but no storage
	// backend is wired up.
	ErrNotConfigured = errors.New("image storage is not configured")
	// ErrInvalidImage is returned when the uploaded bytes are not a JPEG, PNG
	// or WebP image that can be decoded. Nothing is stored in that case.
	ErrInvalidImage = errors.New("file is not a supported image (JPEG, PNG or WebP)")
	// ErrImageTooLarge is returned when one uploaded file is over the size
	// allowed for its kind. Nothing is stored in that case.
	ErrImageTooLarge = errors.New("file is too large")
	// ErrUnavailable is returned when the storage backend cannot be reached
	// or refuses the write.
	ErrUnavailable = errors.New("image storage is unavailable")
)

// ImageError says which image of an UploadImages call was rejected. Err is
// ErrInvalidImage or ErrImageTooLarge.
type ImageError struct {
	Index int // 0 based position in the call
	Err   error
}

func (e *ImageError) Error() string { return fmt.Sprintf("image %d: %v", e.Index+1, e.Err) }

func (e *ImageError) Unwrap() error { return e.Err }

// Image kinds, used as the key prefix of stored objects.
const (
	KindAvatar = "avatars"
	KindPost   = "posts"
)

// ImageUploader validates, sanitises (EXIF removal, resize) and stores an
// uploaded image, and returns its public URL.
type ImageUploader interface {
	UploadImage(ctx context.Context, kind string, r io.Reader) (url string, err error)
	// UploadImages stores several images of one kind and returns their URLs
	// in the order given. It is all or nothing: every image is checked
	// before the first is stored, a rejected one is reported as an
	// *ImageError, and when a write fails the images already stored by the
	// call are deleted again.
	UploadImages(ctx context.Context, kind string, images []io.Reader) (urls []string, err error)
	// DeleteImage removes an image previously returned by UploadImage. URLs
	// that do not belong to this storage are ignored.
	DeleteImage(ctx context.Context, url string) error
}

// ObjectStore is the raw object storage behind an ImageUploader.
type ObjectStore interface {
	Put(ctx context.Context, key, contentType string, data []byte) error
	Delete(ctx context.Context, key string) error
	// PublicURL returns the URL browsers fetch the object from.
	PublicURL(key string) string
	// KeyFromURL is the inverse of PublicURL. ok is false for URLs that do
	// not point into this store.
	KeyFromURL(url string) (key string, ok bool)
}

// NotConfigured is the ImageUploader used when no S3 settings are present.
// Every upload fails with ErrNotConfigured.
type NotConfigured struct{}

func (NotConfigured) UploadImage(context.Context, string, io.Reader) (string, error) {
	return "", ErrNotConfigured
}

func (NotConfigured) UploadImages(context.Context, string, []io.Reader) ([]string, error) {
	return nil, ErrNotConfigured
}

func (NotConfigured) DeleteImage(context.Context, string) error { return nil }
