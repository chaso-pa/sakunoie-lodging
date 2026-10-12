package routes

import (
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/handlers"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
	"github.com/danielgtaylor/huma/v2"
)

// SetupSpotRoutes registers the spot endpoints.
//
// Only what carries no member content is public: the spot list (spot data
// and aggregate counts) and the static AI spot list. A spot's page returns
// members' posts, nicknames and reviews, which are for guests only, so it
// needs a session like the review endpoint.
func SetupSpotRoutes(publicAPI, authAPI huma.API, deps *Deps) {
	h := handlers.NewSpotHandlers(services.NewSpotService(deps.DB))

	huma.Get(publicAPI, "/spots", h.ListSpots, withTag("spots"))
	huma.Get(publicAPI, "/ai-spots", h.ListAiSpots, withTag("spots"))

	huma.Get(authAPI, "/spots/{id}", h.GetSpot, withTag("spots"))
	huma.Post(authAPI, "/spots/{id}/reviews", h.CreateReview, withTag("spots"))
}

// SetupPostRoutes registers the feed, likes, bookmarks and rankings.
func SetupPostRoutes(_, authAPI huma.API, deps *Deps) {
	h := handlers.NewPostHandlers(services.NewPostService(deps.DB, deps.Images))

	huma.Get(authAPI, "/posts", h.ListPosts, withTag("posts"))
	huma.Get(authAPI, "/posts/bookmarked", h.ListBookmarkedPosts, withTag("posts"))
	huma.Get(authAPI, "/posts/mine", h.ListMyPosts, withTag("posts"))
	huma.Get(authAPI, "/posts/{id}", h.GetPost, withTag("posts"))
	huma.Post(authAPI, "/posts", h.CreatePost, withTag("posts"))
	huma.Delete(authAPI, "/posts/{id}", h.DeletePost, withTag("posts"))
	huma.Post(authAPI, "/posts/{id}/like", h.ToggleLike, withTag("posts"))
	huma.Post(authAPI, "/posts/{id}/bookmark", h.ToggleBookmark, withTag("posts"))

	huma.Get(authAPI, "/rankings", h.GetRankings, withTag("rankings"))
}

// SetupGpsVisitRoutes registers the footprint endpoints.
func SetupGpsVisitRoutes(_, authAPI huma.API, deps *Deps) {
	h := handlers.NewGpsVisitHandlers(services.NewGpsVisitService(deps.DB))

	huma.Get(authAPI, "/gps-visits", h.ListMyVisits, withTag("gps-visits"))
	huma.Get(authAPI, "/gps-visits/public", h.ListPublicVisits, withTag("gps-visits"))
	huma.Post(authAPI, "/gps-visits", h.CreateVisit, withTag("gps-visits"))
	huma.Delete(authAPI, "/gps-visits/{id}", h.DeleteVisit, withTag("gps-visits"))
	huma.Patch(authAPI, "/gps-visits/{id}/visibility", h.UpdateVisitVisibility, withTag("gps-visits"))
}

// SetupMemberRoutes registers the profile update. There is deliberately no
// endpoint that edits the host or business page.
func SetupMemberRoutes(_, authAPI huma.API, deps *Deps) {
	h := handlers.NewMemberHandlers(services.NewMemberService(deps.DB, deps.Images))

	huma.Patch(authAPI, "/members/me", h.UpdateProfile, withTag("members"))
}

// SetupExternalRoutes registers translation, the AI assistant and weather.
func SetupExternalRoutes(_, authAPI huma.API, deps *Deps) {
	cfg := deps.Config
	// A missing or unknown provider leaves translator nil: the endpoint then
	// answers 503 for anything that is not already cached.
	translator, provider, _ := services.NewTranslator(cfg.TranslationProvider, cfg.OpenRouterAPIKey, cfg.OpenRouterModel, deps.External)
	h := handlers.NewExternalHandlers(
		services.NewTranslationService(translator, provider, services.NewTranslationCache(deps.DB)),
		services.NewAIService(deps.External.OpenRouterBaseURL, cfg.OpenRouterAPIKey, cfg.OpenRouterModel),
		services.NewWeatherService(deps.External.OpenMeteoBaseURL),
	)

	huma.Post(authAPI, "/translate", h.Translate, withTag("translate"))
	huma.Post(authAPI, "/ai/plan", h.AIPlan, withTag("ai"))
	huma.Get(authAPI, "/weather", h.GetWeather, withTag("weather"))
}
