// Package server assembles the Gin engine: middleware, the two Huma APIs and
// route registration. cmd/server and the integration tests share it.
package server

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/routes"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
)

const apiPrefix = "/api"

// NewRouter builds the engine. Serve it wrapped in the session manager:
// http.ListenAndServe(addr, deps.Sessions.LoadAndSave(router)). Without that
// wrapper no handler can read the session.
func NewRouter(deps *routes.Deps) (*gin.Engine, error) {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	r := gin.New()
	// gin.Logger and gin.Recovery are not used: the first logs query strings
	// (search text, coordinates) and the second dumps request headers,
	// including the session cookie.
	r.Use(middlewares.RequestID(), middlewares.RequestLogger(logger), middlewares.Recovery(logger))

	// Trust only the Next.js proxy on localhost. Gin trusts every proxy by
	// default, in which case ClientIP() returns the left-most X-Forwarded-For
	// value, which the client controls, and per-IP rate limits can be dodged
	// by forging the header. Add the load balancer's address here in
	// production.
	if err := r.SetTrustedProxies([]string{"127.0.0.1", "::1"}); err != nil {
		return nil, err
	}

	limits := middlewares.NewRateLimiters()
	// 5 consecutive failed restores block the client for 15 minutes.
	transferLockout := middlewares.NewFailureLockout(5, 15*time.Minute)
	maxBody := middlewares.MaxBodySize(middlewares.MaxUploadBytes)

	// Group middleware must be attached before any route is registered on
	// the group: Gin copies the chain at registration time.
	publicGroup := r.Group(apiPrefix)
	publicGroup.Use(
		maxBody,
		middlewares.ForRoute(http.MethodPost, apiPrefix+"/auth/verify-invite", limits.Invite),
		middlewares.ForRoute(http.MethodPost, apiPrefix+"/auth/transfer", transferLockout.Middleware()),
		middlewares.ForRoute(http.MethodPost, apiPrefix+"/auth/transfer", limits.Transfer),
	)

	// Member-keyed limiters come after RequireAuth, which supplies their key.
	authGroup := r.Group(apiPrefix)
	authGroup.Use(
		maxBody,
		middlewares.RequireAuth(deps.Sessions, deps.DB),
		middlewares.ForRoute(http.MethodPost, apiPrefix+"/ai/plan", limits.AI),
		middlewares.ForRoute(http.MethodPost, apiPrefix+"/translate", limits.Translate),
		middlewares.ForRoute(http.MethodPost, apiPrefix+"/posts", limits.Post),
	)

	// Two Huma APIs on the same /api prefix: public and auth-required. Their
	// OpenAPI, docs and schema paths differ so the routes do not collide.
	// Operation paths are relative to the group ("/auth/me" → /api/auth/me).
	publicAPI := humagin.NewWithGroup(r, publicGroup, apiConfig("Sakunoie Public API", ""))
	authAPI := humagin.NewWithGroup(r, authGroup, apiConfig("Sakunoie Auth API", "/auth"))

	routes.SetupStaticRoutes(r) // /health (raw Gin)
	routes.SetupAuthRoutes(publicAPI, authAPI, deps)
	routes.SetupSpotRoutes(publicAPI, authAPI, deps)
	routes.SetupPostRoutes(publicAPI, authAPI, deps)
	routes.SetupGpsVisitRoutes(publicAPI, authAPI, deps)
	routes.SetupMemberRoutes(publicAPI, authAPI, deps)
	routes.SetupExternalRoutes(publicAPI, authAPI, deps)

	return r, nil
}

// apiConfig returns Huma's default config with the documentation endpoints
// moved under suffix: /api/docs{suffix}, /api/openapi{suffix}.json and
// /api/schemas{suffix}/{schema}.
func apiConfig(title, suffix string) huma.Config {
	cfg := huma.DefaultConfig(title, "1.0.0")
	cfg.Servers = []*huma.Server{{URL: apiPrefix}}
	cfg.DocsPath = "/docs" + suffix
	cfg.OpenAPIPath = "/openapi" + suffix
	cfg.SchemasPath = "/schemas" + suffix
	return cfg
}
