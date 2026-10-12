package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
)

// S3Config holds the settings of an S3 compatible store (MinIO locally).
type S3Config struct {
	Endpoint  string // e.g. http://localhost:9000
	Bucket    string
	AccessKey string
	SecretKey string
	UseSSL    bool   // scheme used when Endpoint has none
	PublicURL string // base URL of the bucket as browsers see it
}

// S3Store is an ObjectStore on an S3 compatible service.
type S3Store struct {
	client    *s3.Client
	bucket    string
	publicURL string

	mu    sync.Mutex
	ready bool
}

// NewS3Store builds the client. It makes no network call; EnsureBucket does.
func NewS3Store(cfg S3Config) (*S3Store, error) {
	if cfg.Endpoint == "" || cfg.Bucket == "" || cfg.AccessKey == "" || cfg.SecretKey == "" {
		return nil, ErrNotConfigured
	}
	endpoint := strings.TrimRight(cfg.Endpoint, "/")
	if !strings.Contains(endpoint, "://") {
		scheme := "http://"
		if cfg.UseSSL {
			scheme = "https://"
		}
		endpoint = scheme + endpoint
	}
	publicURL := strings.TrimRight(cfg.PublicURL, "/")
	if publicURL == "" {
		publicURL = endpoint + "/" + cfg.Bucket
	}

	client := s3.New(s3.Options{
		// MinIO ignores the region but the signer needs one.
		Region:       "us-east-1",
		BaseEndpoint: aws.String(endpoint),
		// MinIO serves buckets by path, not by virtual host.
		UsePathStyle: true,
		Credentials:  credentials.NewStaticCredentialsProvider(cfg.AccessKey, cfg.SecretKey, ""),
		// Newer SDKs add a CRC32 trailer to every upload by default, which
		// not every S3 compatible server accepts.
		RequestChecksumCalculation: aws.RequestChecksumCalculationWhenRequired,
		ResponseChecksumValidation: aws.ResponseChecksumValidationWhenRequired,
	})
	return &S3Store{client: client, bucket: cfg.Bucket, publicURL: publicURL}, nil
}

// EnsureBucket creates the bucket when it is missing and, when the bucket
// has no policy at all, attaches one that lets anyone read (not list) its
// objects. A bucket that already carries a policy is left alone. It is
// called at startup and again before an upload if it has not succeeded yet,
// so the server can start while the store is down.
func (s *S3Store) EnsureBucket(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ready {
		return nil
	}

	_, err := s.client.HeadBucket(ctx, &s3.HeadBucketInput{Bucket: &s.bucket})
	if err != nil {
		if !isNotFound(err) {
			return fmt.Errorf("check bucket: %w", err)
		}
		if _, err := s.client.CreateBucket(ctx, &s3.CreateBucketInput{Bucket: &s.bucket}); err != nil {
			var owned *types.BucketAlreadyOwnedByYou
			if !errors.As(err, &owned) {
				return fmt.Errorf("create bucket: %w", err)
			}
		}
	}

	_, err = s.client.GetBucketPolicy(ctx, &s3.GetBucketPolicyInput{Bucket: &s.bucket})
	if err != nil {
		if apiErrorCode(err) != "NoSuchBucketPolicy" {
			return fmt.Errorf("read bucket policy: %w", err)
		}
		policy, err := publicReadPolicy(s.bucket)
		if err != nil {
			return err
		}
		if _, err := s.client.PutBucketPolicy(ctx, &s3.PutBucketPolicyInput{Bucket: &s.bucket, Policy: &policy}); err != nil {
			return fmt.Errorf("set bucket policy: %w", err)
		}
	}

	s.ready = true
	return nil
}

func (s *S3Store) Put(ctx context.Context, key, contentType string, data []byte) error {
	if err := s.EnsureBucket(ctx); err != nil {
		return err
	}
	_, err := s.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket:        &s.bucket,
		Key:           &key,
		Body:          bytes.NewReader(data),
		ContentLength: aws.Int64(int64(len(data))),
		ContentType:   &contentType,
		// Object keys are random and never rewritten.
		CacheControl: aws.String("public, max-age=31536000, immutable"),
	})
	if err != nil {
		return fmt.Errorf("put object: %w", err)
	}
	return nil
}

// Delete removes the object. A missing object is not an error.
func (s *S3Store) Delete(ctx context.Context, key string) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: &s.bucket, Key: &key}); err != nil {
		return fmt.Errorf("delete object: %w", err)
	}
	return nil
}

func (s *S3Store) PublicURL(key string) string {
	return s.publicURL + "/" + key
}

func (s *S3Store) KeyFromURL(url string) (string, bool) {
	key, ok := strings.CutPrefix(url, s.publicURL+"/")
	if !ok || key == "" {
		return "", false
	}
	return key, true
}

func publicReadPolicy(bucket string) (string, error) {
	policy := map[string]any{
		"Version": "2012-10-17",
		"Statement": []map[string]any{{
			"Sid":       "PublicReadObjects",
			"Effect":    "Allow",
			"Principal": map[string]any{"AWS": []string{"*"}},
			"Action":    []string{"s3:GetObject"},
			"Resource":  []string{"arn:aws:s3:::" + bucket + "/*"},
		}},
	}
	b, err := json.Marshal(policy)
	if err != nil {
		return "", fmt.Errorf("build bucket policy: %w", err)
	}
	return string(b), nil
}

func apiErrorCode(err error) string {
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		return apiErr.ErrorCode()
	}
	return ""
}

func isNotFound(err error) bool {
	var notFound *types.NotFound
	if errors.As(err, &notFound) {
		return true
	}
	switch apiErrorCode(err) {
	case "NotFound", "NoSuchBucket":
		return true
	}
	return false
}

// RemoveBucket deletes the bucket, which must be empty. Tests use it to
// clean up after themselves.
func (s *S3Store) RemoveBucket(ctx context.Context) error {
	if _, err := s.client.DeleteBucket(ctx, &s3.DeleteBucketInput{Bucket: &s.bucket}); err != nil {
		return fmt.Errorf("delete bucket: %w", err)
	}
	return nil
}
