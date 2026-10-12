package services

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestWeatherLabel(t *testing.T) {
	// The prototype's table (docs/pages/sns.html, weatherLabel).
	cases := map[string][]int{
		"晴れ ☀️":      {0},
		"晴れ時々くもり 🌤️": {1, 2},
		"くもり ☁️":     {3, 4, 10, 100, -1},
		"霧 🌫️":       {45, 48},
		"小雨 🌦️":      {51, 53, 55, 56, 57},
		"雨 🌧️":       {61, 63, 65, 66, 67, 80, 81, 82},
		"雪 ❄️":       {71, 73, 75, 77, 85, 86},
		"雷雨 ⛈️":      {95, 96, 99},
	}
	for want, codes := range cases {
		for _, code := range codes {
			label, icon := WeatherLabel(code)
			if got := label + " " + icon; got != want {
				t.Errorf("WeatherLabel(%d) = %q, want %q", code, got, want)
			}
		}
	}
}

func TestWeatherCurrent(t *testing.T) {
	srv := newStub(t, func(w http.ResponseWriter, _ stubRequest) {
		w.Write([]byte(`{"current":{"time":"2026-10-12T20:00","temperature_2m":13.6,"weather_code":61}}`))
	})
	s := NewWeatherService(srv.URL)

	w, err := s.Current(context.Background(), 36.248812345, 138.476954321)
	if err != nil {
		t.Fatal(err)
	}
	if w.Label != "雨" || w.Icon != "🌧️" || w.Code != 61 || w.Temperature != 14 || w.Source != "open-meteo" {
		t.Fatalf("weather = %+v", w)
	}
	req := srv.last()
	if req.Path != "/v1/forecast" || req.Query["current"] != "temperature_2m,weather_code" || req.Query["timezone"] != "Asia/Tokyo" {
		t.Fatalf("request = %+v", req)
	}
	// The position leaves the server rounded to two decimals.
	if req.Query["latitude"] != "36.25" || req.Query["longitude"] != "138.48" {
		t.Fatalf("coordinates sent = %s, %s", req.Query["latitude"], req.Query["longitude"])
	}

	// A nearby position within the cache lifetime makes no second call.
	if _, err := s.Current(context.Background(), 36.2512, 138.4761); err != nil {
		t.Fatal(err)
	}
	if srv.calls() != 1 {
		t.Fatalf("calls = %d, want 1 (cached)", srv.calls())
	}
	// Somewhere else does.
	if _, err := s.Current(context.Background(), 35.0, 135.0); err != nil {
		t.Fatal(err)
	}
	if srv.calls() != 2 {
		t.Fatalf("calls = %d, want 2", srv.calls())
	}
	// After the lifetime the entry is refreshed.
	s.now = func() time.Time { return time.Now().Add(weatherCacheTTL + time.Second) }
	if _, err := s.Current(context.Background(), 36.25, 138.48); err != nil {
		t.Fatal(err)
	}
	if srv.calls() != 3 {
		t.Fatalf("calls = %d, want 3 (expired)", srv.calls())
	}
}

func TestWeatherErrors(t *testing.T) {
	const upstreamText = "Cannot initialize WeatherVariable from invalid String value"
	cases := map[string]func(w http.ResponseWriter, r stubRequest){
		"http 500": func(w http.ResponseWriter, _ stubRequest) {
			http.Error(w, upstreamText, http.StatusInternalServerError)
		},
		"http 400": func(w http.ResponseWriter, _ stubRequest) {
			http.Error(w, `{"error":true,"reason":"`+upstreamText+`"}`, http.StatusBadRequest)
		},
		"not json":   func(w http.ResponseWriter, _ stubRequest) { w.Write([]byte(upstreamText)) },
		"no current": func(w http.ResponseWriter, _ stubRequest) { w.Write([]byte(`{"latitude":36.25}`)) },
		"no code":    func(w http.ResponseWriter, _ stubRequest) { w.Write([]byte(`{"current":{"temperature_2m":13.6}}`)) },
		"no temp":    func(w http.ResponseWriter, _ stubRequest) { w.Write([]byte(`{"current":{"weather_code":3}}`)) },
		"null fields": func(w http.ResponseWriter, _ stubRequest) {
			w.Write([]byte(`{"current":{"temperature_2m":null,"weather_code":null}}`))
		},
	}
	for name, handler := range cases {
		srv := newStub(t, handler)
		s := NewWeatherService(srv.URL)
		_, err := s.Current(context.Background(), 35.123456, 139.654321)
		if !errors.Is(err, ErrWeatherUnavailable) {
			t.Errorf("%s: err = %v, want ErrWeatherUnavailable", name, err)
			continue
		}
		assertNoLeak(t, err, upstreamText, "35.12", "139.65", srv.URL)
		// A failure is not cached.
		if _, err := s.Current(context.Background(), 35.123456, 139.654321); err == nil || srv.calls() != 2 {
			t.Errorf("%s: failure was cached (calls = %d)", name, srv.calls())
		}
	}
}

func TestWeatherTimeout(t *testing.T) {
	if weatherTimeout != 5*time.Second {
		t.Fatalf("weather timeout = %v, want 5s", weatherTimeout)
	}
	release := make(chan struct{})
	srv := newStub(t, func(w http.ResponseWriter, _ stubRequest) { <-release })
	defer close(release)

	s := NewWeatherService(srv.URL)
	if s.http.Timeout != weatherTimeout {
		t.Fatalf("client timeout = %v, want %v", s.http.Timeout, weatherTimeout)
	}
	s.http.Timeout = 50 * time.Millisecond

	_, err := s.Current(context.Background(), 35.123456, 139.654321)
	if !errors.Is(err, ErrWeatherUnavailable) || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("err = %v, want a timeout", err)
	}
	assertNoLeak(t, err, "35.12", "139.65", srv.URL)
}
