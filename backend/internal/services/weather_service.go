package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	weatherTimeout  = 5 * time.Second
	weatherCacheTTL = 10 * time.Minute
	weatherSource   = "open-meteo"

	// Saku City, the default when the client sends no position. The
	// prototype always used this point.
	SakuLatitude  = 36.2488
	SakuLongitude = 138.4769
)

// ErrWeatherUnavailable means open-meteo could not be reached or answered
// with something unusable.
var ErrWeatherUnavailable = errors.New("weather service temporarily unavailable")

// Weather is the current weather at a position.
type Weather struct {
	Label       string  // Japanese label, as stored on posts ("晴れ" etc.)
	Icon        string  // emoji
	Code        int     // WMO weather code
	Temperature float64 // °C, rounded to a whole number
	Source      string
}

// WeatherLabel maps a WMO weather code to the prototype's label and emoji
// (weatherLabel in docs/pages/sns.html). Unknown codes read as cloudy.
func WeatherLabel(code int) (label, icon string) {
	switch code {
	case 0:
		return "晴れ", "☀️"
	case 1, 2:
		return "晴れ時々くもり", "🌤️"
	case 3:
		return "くもり", "☁️"
	case 45, 48:
		return "霧", "🌫️"
	case 51, 53, 55, 56, 57:
		return "小雨", "🌦️"
	case 61, 63, 65, 66, 67, 80, 81, 82:
		return "雨", "🌧️"
	case 71, 73, 75, 77, 85, 86:
		return "雪", "❄️"
	case 95, 96, 99:
		return "雷雨", "⛈️"
	}
	return "くもり", "☁️"
}

// WeatherService reads the current weather from open-meteo.
type WeatherService struct {
	baseURL string
	http    *http.Client
	now     func() time.Time

	mu    sync.Mutex
	cache map[string]weatherCacheEntry
}

type weatherCacheEntry struct {
	weather Weather
	expires time.Time
}

func NewWeatherService(baseURL string) *WeatherService {
	return &WeatherService{
		baseURL: strings.TrimRight(orDefault(baseURL, DefaultOpenMeteoBaseURL), "/"),
		http:    &http.Client{Timeout: weatherTimeout},
		now:     time.Now,
		cache:   map[string]weatherCacheEntry{},
	}
}

type openMeteoResponse struct {
	Current *struct {
		Temperature *float64 `json:"temperature_2m"`
		WeatherCode *float64 `json:"weather_code"`
	} `json:"current"`
}

// Current returns the weather at the position.
//
// The position is rounded to two decimals (about 1km) before it leaves the
// server: weather does not vary below that, the exact position of a member
// is not handed to a third party, and nearby requests share a cache entry.
func (s *WeatherService) Current(ctx context.Context, latitude, longitude float64) (*Weather, error) {
	const service = "open-meteo"
	lat := strconv.FormatFloat(math.Round(latitude*100)/100, 'f', 2, 64)
	lon := strconv.FormatFloat(math.Round(longitude*100)/100, 'f', 2, 64)
	cacheKey := lat + "," + lon

	s.mu.Lock()
	entry, ok := s.cache[cacheKey]
	s.mu.Unlock()
	if ok && s.now().Before(entry.expires) {
		w := entry.weather
		return &w, nil
	}

	query := url.Values{}
	query.Set("latitude", lat)
	query.Set("longitude", lon)
	query.Set("current", "temperature_2m,weather_code")
	query.Set("timezone", "Asia/Tokyo")
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.baseURL+"/v1/forecast?"+query.Encode(), nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWeatherUnavailable, upstreamFormatError(service, "cannot build request"))
	}
	status, body, err := readUpstream(s.http, service, req)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWeatherUnavailable, err)
	}
	if status != http.StatusOK {
		return nil, fmt.Errorf("%w: %w", ErrWeatherUnavailable, upstreamStatusError(service, status))
	}
	var parsed openMeteoResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return nil, fmt.Errorf("%w: %w", ErrWeatherUnavailable, upstreamFormatError(service, "response is not valid JSON"))
	}
	if parsed.Current == nil || parsed.Current.Temperature == nil || parsed.Current.WeatherCode == nil {
		return nil, fmt.Errorf("%w: %w", ErrWeatherUnavailable, upstreamFormatError(service, "response has no current weather"))
	}

	code := int(*parsed.Current.WeatherCode)
	label, icon := WeatherLabel(code)
	weather := Weather{
		Label:       label,
		Icon:        icon,
		Code:        code,
		Temperature: math.Round(*parsed.Current.Temperature),
		Source:      weatherSource,
	}

	s.mu.Lock()
	for key, e := range s.cache {
		if !s.now().Before(e.expires) {
			delete(s.cache, key)
		}
	}
	s.cache[cacheKey] = weatherCacheEntry{weather: weather, expires: s.now().Add(weatherCacheTTL)}
	s.mu.Unlock()

	return &weather, nil
}
