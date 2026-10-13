package config

import (
	"errors"
	"os"
	"strconv"
	"strings"
	"time"
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

	// Place search (docs/spot-search.md).
	//
	// PlaceProvider is "osm" (also when empty) or "google". The base URLs
	// are empty for the public services. PlaceMinInterval spaces out the
	// calls to each OpenStreetMap service; zero disables the spacing, which
	// Load never produces unless PLACE_UPSTREAM_MIN_INTERVAL_MS says 0.
	PlaceProvider          string
	PlaceUserAgent         string
	PlaceMinInterval       time.Duration
	NominatimBaseURL       string
	PhotonBaseURL          string
	GoogleMapsAPIKey       string
	GooglePlacesBaseURL    string
	GoogleGeocodingBaseURL string
}

// DefaultPlaceMinInterval is the spacing between calls to each public
// OpenStreetMap service: Nominatim's usage policy allows the whole
// application at most one request per second.
const DefaultPlaceMinInterval = 1100 * time.Millisecond

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

		PlaceProvider:          os.Getenv("PLACE_PROVIDER"),
		PlaceUserAgent:         os.Getenv("PLACE_USER_AGENT"),
		PlaceMinInterval:       DefaultPlaceMinInterval,
		NominatimBaseURL:       os.Getenv("NOMINATIM_BASE_URL"),
		PhotonBaseURL:          os.Getenv("PHOTON_BASE_URL"),
		GoogleMapsAPIKey:       os.Getenv("GOOGLE_MAPS_API_KEY"),
		GooglePlacesBaseURL:    os.Getenv("GOOGLE_PLACES_BASE_URL"),
		GoogleGeocodingBaseURL: os.Getenv("GOOGLE_GEOCODING_BASE_URL"),
	}
	if raw := strings.TrimSpace(os.Getenv("PLACE_UPSTREAM_MIN_INTERVAL_MS")); raw != "" {
		ms, err := strconv.Atoi(raw)
		if err != nil || ms < 0 || ms > 60000 {
			return nil, errors.New("PLACE_UPSTREAM_MIN_INTERVAL_MS must be a number of milliseconds between 0 and 60000")
		}
		cfg.PlaceMinInterval = time.Duration(ms) * time.Millisecond
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

// PlaceWarnings returns what a production deployment should fix in the
// place search settings. The caller logs them in release mode.
func (c *Config) PlaceWarnings() []string {
	provider := strings.ToLower(strings.TrimSpace(c.PlaceProvider))
	if provider != "" && provider != "osm" {
		return nil
	}
	var warnings []string
	public := c.NominatimBaseURL == "" || c.PhotonBaseURL == ""
	if public && strings.TrimSpace(c.PlaceUserAgent) == "" {
		warnings = append(warnings, "PLACE_USER_AGENT is not set. The public OpenStreetMap services require a User-Agent that identifies the application; set one that names this site and a contact.")
	}
	if public && c.PlaceMinInterval < time.Second {
		warnings = append(warnings, "PLACE_UPSTREAM_MIN_INTERVAL_MS is below 1000 while a public OpenStreetMap service is in use. Nominatim allows at most one request per second.")
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
