package config

import (
	"errors"
	"os"
	"strings"
	"unicode/utf8"

	"github.com/joho/godotenv"
)

// ExampleInviteCode is the value shipped in .env.example. Running in release
// mode with it unchanged logs a warning at startup.
const ExampleInviteCode = "SAKU-26-ABCD"

type Config struct {
	GormDatabaseURL string
	InviteCode      string
	Port            string
	GinMode         string

	S3Endpoint  string
	S3Bucket    string
	S3AccessKey string
	S3SecretKey string
	S3UseSSL    bool
	S3PublicURL string

	OpenRouterAPIKey    string
	OpenRouterModel     string
	TranslationProvider string
}

// Load reads configuration from the environment, after loading .env from the
// working directory when one exists.
func Load() (*Config, error) {
	_ = godotenv.Load()

	cfg := &Config{
		GormDatabaseURL:     os.Getenv("GORM_DATABASE_URL"),
		InviteCode:          os.Getenv("INVITE_CODE"),
		Port:                getenv("PORT", "8080"),
		GinMode:             getenv("GIN_MODE", "debug"),
		S3Endpoint:          os.Getenv("S3_ENDPOINT"),
		S3Bucket:            os.Getenv("S3_BUCKET"),
		S3AccessKey:         os.Getenv("S3_ACCESS_KEY"),
		S3SecretKey:         os.Getenv("S3_SECRET_KEY"),
		S3UseSSL:            os.Getenv("S3_USE_SSL") == "true",
		S3PublicURL:         os.Getenv("S3_PUBLIC_URL"),
		OpenRouterAPIKey:    os.Getenv("OPENROUTER_API_KEY"),
		OpenRouterModel:     os.Getenv("OPENROUTER_MODEL"),
		TranslationProvider: os.Getenv("TRANSLATION_PROVIDER"),
	}

	if cfg.GormDatabaseURL == "" {
		return nil, errors.New("GORM_DATABASE_URL is required")
	}
	if cfg.InviteCode == "" {
		return nil, errors.New("INVITE_CODE is required")
	}
	return cfg, nil
}

// MinInviteCodeLength is the length below which a release-mode server warns
// about INVITE_CODE. The code is one secret shared by all guests, and without
// a reverse proxy forwarding client addresses its guessing rate is bounded
// only by a shared ceiling (see internal/middlewares/rate_limit.go), so its
// length is what protects it.
const MinInviteCodeLength = 12

// InviteCodeWarnings returns what is wrong with INVITE_CODE for production
// use. The caller logs them in release mode; none of them stops the server.
func (c *Config) InviteCodeWarnings() []string {
	var warnings []string
	if c.InviteCode == ExampleInviteCode {
		warnings = append(warnings, "INVITE_CODE is set to the example value. Change it for production.")
	}
	if utf8.RuneCountInString(strings.TrimSpace(c.InviteCode)) < MinInviteCodeLength {
		warnings = append(warnings, "INVITE_CODE is shorter than 12 characters. Use a long random code: it is the only thing that keeps the invite from being guessed.")
	}
	return warnings
}

// IsRelease reports whether the server runs in Gin release mode.
func (c *Config) IsRelease() bool {
	return c.GinMode == "release"
}

func getenv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
