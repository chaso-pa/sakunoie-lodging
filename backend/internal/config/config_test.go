package config

import (
	"strings"
	"testing"
	"time"
)

func TestInviteCodeWarnings(t *testing.T) {
	cases := map[string]int{
		ExampleInviteCode:        1, // the example value; it is exactly 12 characters
		"SHORT-CODE":             1,
		"abcdefghijk":            1, // 11 characters
		"abcdefghijkl":           0, // 12 characters
		"k3Vq-91xT-Zp0d-Lm7R-aa": 0,
		"  padded  ":             1,
	}
	for code, want := range cases {
		cfg := &Config{InviteCode: code}
		if got := cfg.InviteCodeWarnings(); len(got) != want {
			t.Errorf("InviteCodeWarnings(%q) = %q, want %d warnings", code, got, want)
		}
	}
}

func TestPlaceSettings(t *testing.T) {
	t.Setenv("GORM_DATABASE_URL", "user:pw@tcp(localhost:3306)/db")
	t.Setenv("INVITE_CODE", "a-long-random-invite-code")
	for _, key := range []string{"PLACE_PROVIDER", "PLACE_USER_AGENT", "PLACE_UPSTREAM_MIN_INTERVAL_MS", "NOMINATIM_BASE_URL", "PHOTON_BASE_URL",
		"GOOGLE_MAPS_API_KEY", "GOOGLE_PLACES_BASE_URL", "GOOGLE_GEOCODING_BASE_URL"} {
		t.Setenv(key, "")
	}

	// Nothing set: the OpenStreetMap services, spaced as their policy asks.
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.PlaceProvider != "" || cfg.PlaceMinInterval != 1100*time.Millisecond || cfg.GoogleMapsAPIKey != "" {
		t.Fatalf("defaults = %+v", cfg)
	}
	// The public services want a User-Agent of the deployment's own.
	if w := cfg.PlaceWarnings(); len(w) != 1 || !strings.Contains(w[0], "PLACE_USER_AGENT") {
		t.Fatalf("warnings = %q", w)
	}

	t.Setenv("PLACE_USER_AGENT", "tabinote/1.0 (ops@example.test)")
	t.Setenv("PLACE_UPSTREAM_MIN_INTERVAL_MS", "0")
	cfg, err = Load()
	if err != nil || cfg.PlaceMinInterval != 0 || cfg.PlaceUserAgent != "tabinote/1.0 (ops@example.test)" {
		t.Fatalf("cfg = %+v, err = %v", cfg, err)
	}
	// No spacing against a public service is a policy violation.
	if w := cfg.PlaceWarnings(); len(w) != 1 || !strings.Contains(w[0], "PLACE_UPSTREAM_MIN_INTERVAL_MS") {
		t.Fatalf("warnings = %q", w)
	}
	// Against one's own instances it is fine.
	t.Setenv("NOMINATIM_BASE_URL", "http://nominatim.internal:8080")
	t.Setenv("PHOTON_BASE_URL", "http://photon.internal:2322")
	if cfg, _ = Load(); len(cfg.PlaceWarnings()) != 0 || cfg.NominatimBaseURL != "http://nominatim.internal:8080" || cfg.PhotonBaseURL != "http://photon.internal:2322" {
		t.Fatalf("cfg = %+v, warnings = %q", cfg, cfg.PlaceWarnings())
	}

	t.Setenv("PLACE_PROVIDER", "google")
	t.Setenv("GOOGLE_MAPS_API_KEY", "AIza-test")
	t.Setenv("PLACE_UPSTREAM_MIN_INTERVAL_MS", "250")
	cfg, err = Load()
	if err != nil || cfg.PlaceProvider != "google" || cfg.GoogleMapsAPIKey != "AIza-test" || cfg.PlaceMinInterval != 250*time.Millisecond || len(cfg.PlaceWarnings()) != 0 {
		t.Fatalf("cfg = %+v, err = %v", cfg, err)
	}

	for _, bad := range []string{"fast", "-1", "60001", "1.5"} {
		t.Setenv("PLACE_UPSTREAM_MIN_INTERVAL_MS", bad)
		if _, err := Load(); err == nil {
			t.Errorf("PLACE_UPSTREAM_MIN_INTERVAL_MS=%q was accepted", bad)
		}
	}
}
