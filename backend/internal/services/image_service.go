package services

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder
	"io"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp" // registers the WebP decoder
)

const (
	// MaxImageBytes matches the API's request body cap.
	MaxImageBytes = 10 << 20
	// maxPostImageWidth is the width post photos are scaled down to.
	maxPostImageWidth = 1400
	// maxAvatarSize bounds both sides of an avatar.
	maxAvatarSize = 512
	// maxImagePixels rejects decompression bombs before any pixel is decoded.
	maxImagePixels = 50_000_000
	jpegQuality    = 85

	storedImageContentType = "image/jpeg"
)

// DetectImageType returns the MIME type of data judged by its leading bytes,
// or "" when it is not JPEG, PNG or WebP. The Content-Type header and the
// file name are never consulted.
func DetectImageType(data []byte) string {
	switch {
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return "image/jpeg"
	case len(data) >= 8 && bytes.Equal(data[:8], []byte("\x89PNG\r\n\x1a\n")):
		return "image/png"
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return "image/webp"
	}
	return ""
}

// SanitizeImage validates data and returns it re-encoded as a JPEG no wider
// than maxWidth and no taller than maxHeight (0 means unbounded).
//
// The result is built from the decoded pixels only, so no metadata of the
// original survives: EXIF (including GPS position), XMP, ICC profiles and
// comments are all dropped. The EXIF orientation is applied to the pixels
// first so photos do not come out rotated.
func SanitizeImage(data []byte, maxWidth, maxHeight int) ([]byte, error) {
	if DetectImageType(data) == "" {
		return nil, storage.ErrInvalidImage
	}
	cfg, _, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width <= 0 || cfg.Height <= 0 {
		return nil, storage.ErrInvalidImage
	}
	if int64(cfg.Width)*int64(cfg.Height) > maxImagePixels {
		return nil, storage.ErrInvalidImage
	}

	img, err := imaging.Decode(bytes.NewReader(data), imaging.AutoOrientation(true))
	if err != nil {
		return nil, storage.ErrInvalidImage
	}

	bounds := img.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if maxWidth > 0 && w > maxWidth {
		h = max(h*maxWidth/w, 1)
		w = maxWidth
	}
	if maxHeight > 0 && h > maxHeight {
		w = max(w*maxHeight/h, 1)
		h = maxHeight
	}
	if w != bounds.Dx() || h != bounds.Dy() {
		img = imaging.Resize(img, w, h, imaging.Lanczos)
	}

	// JPEG has no alpha channel: put transparent images on white instead of
	// letting the encoder turn transparency black.
	if o, ok := img.(interface{ Opaque() bool }); !ok || !o.Opaque() {
		bg := imaging.New(w, h, color.White)
		img = imaging.Overlay(bg, img, image.Point{}, 1)
	}

	var out bytes.Buffer
	if err := jpeg.Encode(&out, img, &jpeg.Options{Quality: jpegQuality}); err != nil {
		return nil, fmt.Errorf("encode image: %w", err)
	}
	return out.Bytes(), nil
}

// ImageService is the storage.ImageUploader used in production: it checks
// and sanitises an upload, then writes the result to the object store. A
// rejected upload never reaches the store.
type ImageService struct {
	store storage.ObjectStore
}

func NewImageService(store storage.ObjectStore) *ImageService {
	return &ImageService{store: store}
}

func (s *ImageService) UploadImage(ctx context.Context, kind string, r io.Reader) (string, error) {
	data, err := io.ReadAll(io.LimitReader(r, MaxImageBytes+1))
	if err != nil {
		return "", fmt.Errorf("read image: %w", err)
	}
	if len(data) == 0 || len(data) > MaxImageBytes {
		return "", storage.ErrInvalidImage
	}

	maxWidth, maxHeight := maxPostImageWidth, 0
	if kind == storage.KindAvatar {
		maxWidth, maxHeight = maxAvatarSize, maxAvatarSize
	}
	clean, err := SanitizeImage(data, maxWidth, maxHeight)
	if err != nil {
		return "", err
	}

	key := kind + "/" + uuid.NewString() + ".jpg"
	if err := s.store.Put(ctx, key, storedImageContentType, clean); err != nil {
		return "", fmt.Errorf("%w: %v", storage.ErrUnavailable, err)
	}
	return s.store.PublicURL(key), nil
}

func (s *ImageService) DeleteImage(ctx context.Context, url string) error {
	key, ok := s.store.KeyFromURL(url)
	if !ok {
		return nil
	}
	return s.store.Delete(ctx, key)
}
