package testutil

import (
	"net/http"
	"os"
	"testing"
	"time"
)

// S3Settings describes the S3 compatible store used by integration tests:
// by default the MinIO container from the repository's docker-compose.yml.
// TEST_S3_ENDPOINT, TEST_S3_ACCESS_KEY and TEST_S3_SECRET_KEY override it.
type S3Settings struct {
	Endpoint  string
	AccessKey string
	SecretKey string
}

// RequireS3 returns the test store's settings. When it is unreachable the
// test is skipped, or failed if REQUIRE_TEST_S3=1 or REQUIRE_TEST_DB=1 is
// set.
func RequireS3(t testing.TB) S3Settings {
	t.Helper()
	s := S3Settings{
		Endpoint:  envOr("TEST_S3_ENDPOINT", "http://127.0.0.1:9000"),
		AccessKey: envOr("TEST_S3_ACCESS_KEY", "minioadmin"),
		SecretKey: envOr("TEST_S3_SECRET_KEY", "minioadmin"),
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get(s.Endpoint + "/minio/health/live")
	if err == nil {
		resp.Body.Close()
		return s
	}
	msg := "S3 store is not reachable at " + s.Endpoint + ". Start it with `docker compose up -d minio` at the repository root, or set TEST_S3_ENDPOINT."
	if os.Getenv("REQUIRE_TEST_S3") == "1" || os.Getenv("REQUIRE_TEST_DB") == "1" {
		t.Fatal(msg)
	}
	t.Skip("skipping integration test: " + msg)
	return s
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
