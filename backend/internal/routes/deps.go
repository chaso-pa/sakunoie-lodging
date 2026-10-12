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
}
