package storage_test

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
	"github.com/google/uuid"
)

func fetch(t *testing.T, url string) (int, []byte, string) {
	t.Helper()
	resp, err := http.Get(url)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, body, resp.Header.Get("Content-Type")
}

// Upload, anonymous read, delete, against the real S3 API (MinIO): this is
// where a signature or path-style mismatch would show.
func TestS3StoreIntegration(t *testing.T) {
	s3 := testutil.RequireS3(t)
	bucket := "sakunoie-test-" + strings.ReplaceAll(uuid.NewString(), "-", "")[:16]
	cfg := storage.S3Config{
		Endpoint:  s3.Endpoint,
		Bucket:    bucket,
		AccessKey: s3.AccessKey,
		SecretKey: s3.SecretKey,
		PublicURL: s3.Endpoint + "/" + bucket,
	}
	store, err := storage.NewS3Store(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	t.Cleanup(func() {
		if err := store.RemoveBucket(context.Background()); err != nil {
			t.Logf("test bucket %s was not removed: %v", bucket, err)
		}
	})

	// The bucket does not exist yet: EnsureBucket creates it with the
	// public-read policy, and is safe to repeat.
	if err := store.EnsureBucket(ctx); err != nil {
		t.Fatalf("EnsureBucket: %v", err)
	}
	if err := store.EnsureBucket(ctx); err != nil {
		t.Fatalf("EnsureBucket again: %v", err)
	}
	// A second client finds the bucket and its policy in place.
	again, _ := storage.NewS3Store(cfg)
	if err := again.EnsureBucket(ctx); err != nil {
		t.Fatalf("EnsureBucket on an existing bucket: %v", err)
	}

	key := "posts/" + uuid.NewString() + ".jpg"
	data := []byte("\xff\xd8\xff object body")
	if err := store.Put(ctx, key, "image/jpeg", data); err != nil {
		t.Fatalf("Put: %v", err)
	}

	url := store.PublicURL(key)
	if got, ok := store.KeyFromURL(url); !ok || got != key {
		t.Fatalf("KeyFromURL(%q) = %q, %v", url, got, ok)
	}
	if _, ok := store.KeyFromURL("https://elsewhere.example/" + key); ok {
		t.Fatal("KeyFromURL accepted a foreign URL")
	}

	// Anyone can read the object without credentials...
	status, body, contentType := fetch(t, url)
	if status != http.StatusOK || !bytes.Equal(body, data) || contentType != "image/jpeg" {
		t.Fatalf("anonymous GET = %d %q (%s)", status, body, contentType)
	}
	// ...but cannot list the bucket.
	if status, _, _ := fetch(t, cfg.PublicURL+"/"); status != http.StatusForbidden {
		t.Fatalf("anonymous bucket listing = %d, want 403", status)
	}

	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if status, _, _ := fetch(t, url); status != http.StatusNotFound {
		t.Fatalf("GET after delete = %d, want 404", status)
	}
	// Deleting a missing object is not an error.
	if err := store.Delete(ctx, key); err != nil {
		t.Fatalf("Delete of a missing object: %v", err)
	}
}

func TestNewS3StoreNotConfigured(t *testing.T) {
	if _, err := storage.NewS3Store(storage.S3Config{Bucket: "b"}); err != storage.ErrNotConfigured {
		t.Fatalf("err = %v, want ErrNotConfigured", err)
	}
}
