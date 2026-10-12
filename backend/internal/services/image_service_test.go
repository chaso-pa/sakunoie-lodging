package services

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
)

// A 1x1 lossless WebP.
var tinyWebP = []byte("RIFF\x1a\x00\x00\x00WEBPVP8L\x0d\x00\x00\x00\x2f\x00\x00\x00\x10\x07\x10\x11\x11\x88\x88\xfe\x07\x00")

func decodeJPEG(t *testing.T, data []byte) image.Image {
	t.Helper()
	img, format, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("output does not decode: %v", err)
	}
	if format != "jpeg" {
		t.Fatalf("output format = %s, want jpeg", format)
	}
	return img
}

func TestDetectImageType(t *testing.T) {
	cases := map[string]struct {
		data []byte
		want string
	}{
		"jpeg":       {testutil.PlainJPEG(t, 4, 4), "image/jpeg"},
		"png":        {testutil.PlainPNG(t, 4, 4), "image/png"},
		"webp":       {tinyWebP, "image/webp"},
		"gif":        {[]byte("GIF89a\x01\x00\x01\x00"), ""},
		"svg":        {[]byte(`<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script></svg>`), ""},
		"elf":        {[]byte("\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00"), ""},
		"windows":    {[]byte("MZ\x90\x00\x03\x00\x00\x00"), ""},
		"html":       {[]byte("<!doctype html><html></html>"), ""},
		"riff wav":   {[]byte("RIFF\x24\x00\x00\x00WAVEfmt "), ""},
		"empty":      {nil, ""},
		"two bytes":  {[]byte{0xFF, 0xD8}, ""},
		"plain text": {[]byte("hello"), ""},
	}
	for name, c := range cases {
		if got := DetectImageType(c.data); got != c.want {
			t.Errorf("%s: DetectImageType = %q, want %q", name, got, c.want)
		}
	}
}

func TestSanitizeImageRemovesExifAndGPS(t *testing.T) {
	// 40x20 with Orientation=6: displayed rotated, so 20x40.
	src := testutil.JPEGWithGPSExif(t, 40, 20)
	if !testutil.HasExif(src) {
		t.Fatal("source has no EXIF")
	}

	out, err := SanitizeImage(src, 1400, 0)
	if err != nil {
		t.Fatal(err)
	}
	if testutil.HasExif(out) {
		t.Fatal("EXIF survived sanitising")
	}
	for _, marker := range testutil.JPEGSegments(out) {
		// Only the tables and frame headers Go's encoder writes: no APPn
		// (0xE0-0xEF) and no comment (0xFE) segment at all.
		if (marker >= 0xE0 && marker <= 0xEF) || marker == 0xFE {
			t.Fatalf("output has metadata segment 0xFF%02X", marker)
		}
	}

	img := decodeJPEG(t, out)
	if b := img.Bounds(); b.Dx() != 20 || b.Dy() != 40 {
		t.Fatalf("size = %dx%d, want 20x40 (EXIF orientation applied before it is dropped)", b.Dx(), b.Dy())
	}
	// Rotated 90° clockwise, the red left half ends up on top.
	r, _, b, _ := img.At(10, 5).RGBA()
	if r>>8 < 150 || b>>8 > 100 {
		t.Fatalf("top pixel is not red (r=%d b=%d): orientation was not applied", r>>8, b>>8)
	}
}

func TestSanitizeImageResizes(t *testing.T) {
	out, err := SanitizeImage(testutil.PlainJPEG(t, 2800, 100), 1400, 0)
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeJPEG(t, out).Bounds(); b.Dx() != 1400 || b.Dy() != 50 {
		t.Fatalf("size = %dx%d, want 1400x50", b.Dx(), b.Dy())
	}

	// Narrower images are not scaled up.
	out, err = SanitizeImage(testutil.PlainJPEG(t, 300, 200), 1400, 0)
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeJPEG(t, out).Bounds(); b.Dx() != 300 || b.Dy() != 200 {
		t.Fatalf("size = %dx%d, want 300x200", b.Dx(), b.Dy())
	}

	// Both bounds, as for avatars.
	out, err = SanitizeImage(testutil.PlainJPEG(t, 400, 1024), 512, 512)
	if err != nil {
		t.Fatal(err)
	}
	if b := decodeJPEG(t, out).Bounds(); b.Dx() != 200 || b.Dy() != 512 {
		t.Fatalf("size = %dx%d, want 200x512", b.Dx(), b.Dy())
	}
}

func TestSanitizeImageConvertsPNGAndWebP(t *testing.T) {
	// A fully transparent PNG becomes white, not black.
	transparent := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	if err := png.Encode(&buf, transparent); err != nil {
		t.Fatal(err)
	}
	out, err := SanitizeImage(buf.Bytes(), 1400, 0)
	if err != nil {
		t.Fatal(err)
	}
	r, g, b, _ := decodeJPEG(t, out).At(4, 4).RGBA()
	if r>>8 < 240 || g>>8 < 240 || b>>8 < 240 {
		t.Fatalf("transparent pixel became (%d,%d,%d), want white", r>>8, g>>8, b>>8)
	}

	out, err = SanitizeImage(tinyWebP, 1400, 0)
	if err != nil {
		t.Fatalf("webp: %v", err)
	}
	if b := decodeJPEG(t, out).Bounds(); b.Dx() != 1 || b.Dy() != 1 {
		t.Fatalf("webp size = %dx%d, want 1x1", b.Dx(), b.Dy())
	}
}

// hugePNGHeader is a valid PNG signature and IHDR claiming 30000x30000
// pixels, with nothing after it.
func hugePNGHeader() []byte {
	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:], 30000)
	binary.BigEndian.PutUint32(ihdr[4:], 30000)
	ihdr[8], ihdr[9] = 8, 6 // 8 bit RGBA
	var buf bytes.Buffer
	buf.WriteString("\x89PNG\r\n\x1a\n")
	binary.Write(&buf, binary.BigEndian, uint32(len(ihdr)))
	chunk := append([]byte("IHDR"), ihdr...)
	buf.Write(chunk)
	binary.Write(&buf, binary.BigEndian, crc32.ChecksumIEEE(chunk))
	return buf.Bytes()
}

func TestSanitizeImageRejects(t *testing.T) {
	jpegBytes := testutil.PlainJPEG(t, 16, 16)
	cases := map[string][]byte{
		"empty":                    nil,
		"gif":                      []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;"),
		"svg":                      []byte(`<svg xmlns="http://www.w3.org/2000/svg"/>`),
		"executable":               []byte("\x7fELF\x02\x01\x01\x00\x00\x00\x00\x00\x00\x00\x00\x00"),
		"jpeg magic, garbage body": append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, []byte("this is not a jpeg at all")...),
		"truncated jpeg":           jpegBytes[:len(jpegBytes)/3],
		"png magic only":           []byte("\x89PNG\r\n\x1a\n"),
		"decompression bomb":       hugePNGHeader(),
		"script with jpeg magic":   append([]byte{0xFF, 0xD8, 0xFF}, []byte("<script>alert(1)</script>")...),
	}
	for name, data := range cases {
		if _, err := SanitizeImage(data, 1400, 0); !errors.Is(err, storage.ErrInvalidImage) {
			t.Errorf("%s: err = %v, want ErrInvalidImage", name, err)
		}
	}
}

// memoryStore is an in-memory storage.ObjectStore.
type memoryStore struct {
	objects map[string][]byte
	types   map[string]string
	putErr  error
	puts    int
}

func newMemoryStore() *memoryStore {
	return &memoryStore{objects: map[string][]byte{}, types: map[string]string{}}
}

func (m *memoryStore) Put(_ context.Context, key, contentType string, data []byte) error {
	m.puts++
	if m.putErr != nil {
		return m.putErr
	}
	m.objects[key] = data
	m.types[key] = contentType
	return nil
}

func (m *memoryStore) Delete(_ context.Context, key string) error {
	delete(m.objects, key)
	return nil
}

func (m *memoryStore) PublicURL(key string) string { return "http://storage.test/bucket/" + key }

func (m *memoryStore) KeyFromURL(url string) (string, bool) {
	key, ok := strings.CutPrefix(url, "http://storage.test/bucket/")
	return key, ok && key != ""
}

func TestImageServiceUpload(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	s := NewImageService(store)

	url, err := s.UploadImage(ctx, storage.KindPost, bytes.NewReader(testutil.JPEGWithGPSExif(t, 64, 32)))
	if err != nil {
		t.Fatal(err)
	}
	key, ok := store.KeyFromURL(url)
	if !ok || !strings.HasPrefix(key, "posts/") || !strings.HasSuffix(key, ".jpg") {
		t.Fatalf("url = %q", url)
	}
	stored := store.objects[key]
	if stored == nil || store.types[key] != "image/jpeg" {
		t.Fatalf("object not stored as image/jpeg: %v", store.types)
	}
	if testutil.HasExif(stored) {
		t.Fatal("stored object still has EXIF")
	}

	// Avatars are bounded on both sides.
	big := image.NewRGBA(image.Rect(0, 0, 900, 1800))
	for i := range big.Pix {
		big.Pix[i] = 0xFF
	}
	big.Set(1, 1, color.RGBA{R: 1, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, big, nil); err != nil {
		t.Fatal(err)
	}
	url, err = s.UploadImage(ctx, storage.KindAvatar, &buf)
	if err != nil {
		t.Fatal(err)
	}
	key, _ = store.KeyFromURL(url)
	if b := decodeJPEG(t, store.objects[key]).Bounds(); b.Dx() != 256 || b.Dy() != 512 {
		t.Fatalf("avatar size = %dx%d, want 256x512", b.Dx(), b.Dy())
	}

	// Deleting by URL removes the object; foreign URLs are ignored.
	if err := s.DeleteImage(ctx, url); err != nil {
		t.Fatal(err)
	}
	if _, still := store.objects[key]; still {
		t.Fatal("object was not deleted")
	}
	if err := s.DeleteImage(ctx, "https://elsewhere.example/x.jpg"); err != nil {
		t.Fatal(err)
	}
}

func TestImageServiceNeverStoresRejectedFiles(t *testing.T) {
	ctx := context.Background()
	store := newMemoryStore()
	s := NewImageService(store)

	for name, data := range map[string][]byte{
		"empty":      nil,
		"gif":        []byte("GIF89a\x01\x00\x01\x00"),
		"executable": []byte("\x7fELF\x02\x01\x01\x00"),
		"fake jpeg":  append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, []byte("not a real jpeg")...),
		"too large":  append(testutil.PlainJPEG(t, 8, 8), make([]byte, MaxImageBytes)...),
	} {
		if _, err := s.UploadImage(ctx, storage.KindPost, bytes.NewReader(data)); !errors.Is(err, storage.ErrInvalidImage) {
			t.Errorf("%s: err = %v, want ErrInvalidImage", name, err)
		}
	}
	if store.puts != 0 || len(store.objects) != 0 {
		t.Fatalf("rejected files reached the store (%d puts)", store.puts)
	}

	store.putErr = errors.New("connection refused")
	if _, err := s.UploadImage(ctx, storage.KindPost, bytes.NewReader(testutil.PlainJPEG(t, 8, 8))); !errors.Is(err, storage.ErrUnavailable) {
		t.Fatalf("err = %v, want ErrUnavailable", err)
	}
}
