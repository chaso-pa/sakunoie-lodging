// Package config loads typed settings from the environment and refuses to
// start on unsafe or incomplete configuration (plan Decision A, §5.7).
package config

import (
	"encoding/base64"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
)

const (
	EnvDevelopment = "development"
	EnvTest        = "test"
	EnvProduction  = "production"
)

// Config holds the API server settings. Secrets are kept as decoded bytes and
// are never formatted into errors or logs.
type Config struct {
	AppEnv         string
	Port           string
	DatabaseURL    string // GORM_DATABASE_URL
	TrustedProxies []string

	PassportKeys        map[byte][]byte
	PassportActiveKeyID byte
	IdempotencyHMACKey  []byte

	FirebaseProjectID    string
	FirebaseEmulatorHost string

	RateLimitCreatePerMin int
	RateLimitManagePerMin int
	RateLimitSurveyPerMin int
	SurveyMinResponses    int
}

// DocsEnabled reports whether /docs, /schemas and /openapi.* are served.
func (c *Config) DocsEnabled() bool { return c.AppEnv != EnvProduction }

// Load reads the configuration through getenv (os.Getenv in production).
func Load(getenv func(string) string) (*Config, error) {
	c := &Config{
		AppEnv:               strings.TrimSpace(getenv("APP_ENV")),
		Port:                 orDefault(getenv("PORT"), "8080"),
		DatabaseURL:          getenv("GORM_DATABASE_URL"),
		FirebaseProjectID:    strings.TrimSpace(getenv("FIREBASE_PROJECT_ID")),
		FirebaseEmulatorHost: strings.TrimSpace(getenv("FIREBASE_AUTH_EMULATOR_HOST")),
	}

	switch c.AppEnv {
	case EnvDevelopment, EnvTest, EnvProduction:
	case "":
		return nil, errors.New("config: APP_ENV is required (development|test|production)")
	default:
		return nil, errors.New("config: APP_ENV must be one of development|test|production")
	}

	if c.FirebaseEmulatorHost != "" {
		if c.AppEnv == EnvProduction {
			return nil, errors.New("config: refusing to start: FIREBASE_AUTH_EMULATOR_HOST is set with APP_ENV=production")
		}
		if !strings.HasPrefix(c.FirebaseProjectID, "demo-") {
			return nil, errors.New("config: refusing to start: FIREBASE_AUTH_EMULATOR_HOST requires a demo- FIREBASE_PROJECT_ID")
		}
	}

	if c.DatabaseURL == "" {
		return nil, errors.New("config: GORM_DATABASE_URL is required")
	}

	rawKeys := getenv("PASSPORT_ENC_KEYS")
	if strings.TrimSpace(rawKeys) == "" {
		return nil, errors.New("config: PASSPORT_ENC_KEYS is required")
	}
	keys, err := utils.ParsePassportKeys(rawKeys)
	if err != nil {
		return nil, fmt.Errorf("config: PASSPORT_ENC_KEYS: %w", err)
	}
	c.PassportKeys = keys
	active, err := strconv.ParseUint(strings.TrimSpace(getenv("PASSPORT_ENC_ACTIVE_KEY_ID")), 10, 8)
	if err != nil {
		return nil, errors.New("config: PASSPORT_ENC_ACTIVE_KEY_ID is required (integer 1-255)")
	}
	if _, ok := keys[byte(active)]; !ok {
		return nil, errors.New("config: PASSPORT_ENC_ACTIVE_KEY_ID does not match any PASSPORT_ENC_KEYS id")
	}
	c.PassportActiveKeyID = byte(active)

	hmacRaw := strings.TrimSpace(getenv("IDEMPOTENCY_HMAC_KEY"))
	if hmacRaw == "" {
		return nil, errors.New("config: IDEMPOTENCY_HMAC_KEY is required")
	}
	hmacKey, err := base64.StdEncoding.DecodeString(hmacRaw)
	if err != nil || len(hmacKey) < 32 {
		return nil, errors.New("config: IDEMPOTENCY_HMAC_KEY must be base64 of at least 32 bytes")
	}
	c.IdempotencyHMACKey = hmacKey

	c.TrustedProxies = splitList(orDefault(getenv("TRUSTED_PROXIES"), "127.0.0.1,::1"))

	for _, f := range []struct {
		name string
		dst  *int
		def  int
	}{
		{"RATE_LIMIT_CREATE_PER_MIN", &c.RateLimitCreatePerMin, 10},
		{"RATE_LIMIT_MANAGE_PER_MIN", &c.RateLimitManagePerMin, 60},
		{"RATE_LIMIT_SURVEY_PER_MIN", &c.RateLimitSurveyPerMin, 10},
		{"SURVEY_MIN_RESPONSES", &c.SurveyMinResponses, 5},
	} {
		v, err := positiveInt(getenv(f.name), f.def)
		if err != nil {
			return nil, fmt.Errorf("config: %s: %w", f.name, err)
		}
		*f.dst = v
	}
	return c, nil
}

func orDefault(v, def string) string {
	if strings.TrimSpace(v) == "" {
		return def
	}
	return strings.TrimSpace(v)
}

func splitList(s string) []string {
	var out []string
	for p := range strings.SplitSeq(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func positiveInt(s string, def int) (int, error) {
	if strings.TrimSpace(s) == "" {
		return def, nil
	}
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v < 1 {
		return 0, errors.New("must be a positive integer")
	}
	return v, nil
}
