package services

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	_ "image/png" // registers the PNG decoder
	"io"
	"log/slog"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/disintegration/imaging"
	"github.com/google/uuid"
	_ "golang.org/x/image/webp" // registers the WebP decoder
)

const (
	// MaxImageBytes matches the API's request body cap.
	MaxImageBytes = 10 << 20
	// MaxPostPhotoBytes caps one photo of a post. A post carries up to
	// MaxPostPhotos of them inside the same 10MB request body, and the app
	// sends photos scaled to 1600px, which stay far below this.
	MaxPostPhotoBytes = 4 << 20
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
	clean, err := prepareImage(kind, r)
	if err != nil {
		return "", err
	}
	return s.put(ctx, kind, clean)
}

// UploadImages checks and re-encodes every image before it stores the
// first, so a file that is refused never leaves the others in the store.
//
// The images are handled one after another, not in parallel: decoding a
// phone photo takes tens of megabytes, and a small server should hold one
// decoded picture at a time, not five. What is kept until the writes is the
// re-encoded JPEGs, a few hundred kilobytes each.
func (s *ImageService) UploadImages(ctx context.Context, kind string, images []io.Reader) ([]string, error) {
	clean := make([][]byte, len(images))
	for i, r := range images {
		data, err := prepareImage(kind, r)
		if err != nil {
			if errors.Is(err, storage.ErrInvalidImage) || errors.Is(err, storage.ErrImageTooLarge) {
				return nil, &storage.ImageError{Index: i, Err: err}
			}
			return nil, err
		}
		clean[i] = data
	}

	urls := make([]string, 0, len(clean))
	for _, data := range clean {
		url, err := s.put(ctx, kind, data)
		if err != nil {
			s.removeStored(ctx, urls)
			return nil, err
		}
		urls = append(urls, url)
	}
	return urls, nil
}

// prepareImage reads one upload and returns it sanitised for its kind.
func prepareImage(kind string, r io.Reader) ([]byte, error) {
	maxBytes, maxWidth, maxHeight := MaxImageBytes, maxPostImageWidth, 0
	switch kind {
	case storage.KindAvatar:
		maxWidth, maxHeight = maxAvatarSize, maxAvatarSize
	case storage.KindPost:
		maxBytes = MaxPostPhotoBytes
	}

	data, err := io.ReadAll(io.LimitReader(r, int64(maxBytes)+1))
	if err != nil {
		return nil, fmt.Errorf("read image: %w", err)
	}
	if len(data) == 0 {
		return nil, storage.ErrInvalidImage
	}
	if len(data) > maxBytes {
		return nil, storage.ErrImageTooLarge
	}
	return SanitizeImage(data, maxWidth, maxHeight)
}

func (s *ImageService) put(ctx context.Context, kind string, clean []byte) (string, error) {
	key := kind + "/" + uuid.NewString() + ".jpg"
	if err := s.store.Put(ctx, key, storedImageContentType, clean); err != nil {
		return "", fmt.Errorf("%w: %v", storage.ErrUnavailable, err)
	}
	return s.store.PublicURL(key), nil
}

// removeStored deletes the images a failed UploadImages call had already
// written. Like PostService.removeImages it outlives a cancelled request,
// and an object it cannot delete is logged and left as an orphan.
func (s *ImageService) removeStored(ctx context.Context, urls []string) {
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), cleanupTimeout)
	defer cancel()
	for _, url := range urls {
		if err := s.DeleteImage(ctx, url); err != nil {
			slog.WarnContext(ctx, "stored image was not deleted", "url", url, "error", err)
		}
	}
}

func (s *ImageService) DeleteImage(ctx context.Context, url string) error {
	key, ok := s.store.KeyFromURL(url)
	if !ok {
		return nil
	}
	return s.store.Delete(ctx, key)
}
