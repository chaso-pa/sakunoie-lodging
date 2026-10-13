package routes

import (
	"log/slog"

	"github.com/alexedwards/scs/v2"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/config"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"gorm.io/gorm"
)

// Deps is what route setup functions build their services and handlers from.
// Add shared dependencies (external API clients and the like) here.
type Deps struct {
	DB       *gorm.DB
	Sessions *scs.SessionManager
	Config   *config.Config
	Images   storage.ImageUploader
	// External overrides the base URLs of OpenRouter, MyMemory and
	// open-meteo. The zero value means the real services.
	External services.ExternalEndpoints
	// Logger receives the request log. nil means slog.Default().
	Logger *slog.Logger
	// Places is place search and the reading of stored spots. Leave it nil:
	// server.NewRouter builds it from Config and External (NewPlaces).
	Places *services.PlaceService
}

// NewPlaces builds the place service from the configuration. The base URLs
// in deps.External, which tests set, win over the configured ones.
//
// A provider that cannot be used (unknown name, google without a key, a
// malformed base URL) is logged here once and leaves the place endpoints
// answering 503; nothing else is affected.
func NewPlaces(deps *Deps) *services.PlaceService {
	cfg := deps.Config
	providers, err := services.NewPlaceProviders(services.PlaceProviderConfig{
		Provider: cfg.PlaceProvider,
		OSM: services.OSMPlaceOptions{
			NominatimBaseURL: orDefault(deps.External.NominatimBaseURL, cfg.NominatimBaseURL),
			PhotonBaseURL:    orDefault(deps.External.PhotonBaseURL, cfg.PhotonBaseURL),
			UserAgent:        cfg.PlaceUserAgent,
			MinInterval:      cfg.PlaceMinInterval,
		},
		Google: services.GooglePlaceOptions{
			APIKey:           cfg.GoogleMapsAPIKey,
			PlacesBaseURL:    orDefault(deps.External.GooglePlacesBaseURL, cfg.GooglePlacesBaseURL),
			GeocodingBaseURL: orDefault(deps.External.GoogleGeocodingBaseURL, cfg.GoogleGeocodingBaseURL),
		},
	})
	if err != nil {
		slog.Warn("place search is disabled", "error", err)
	}
	return services.NewPlaceService(deps.DB, providers)
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}
