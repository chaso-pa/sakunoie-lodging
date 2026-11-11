package routes

import (
	"log/slog"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/config"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/handlers"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humagin"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

const (
	APITitle   = "Saku no Ie API"
	APIVersion = "1.0.0"
	APIPrefix  = "/api/v1"
)

// Deps are the services behind the routes. cmd/openapi passes zero values:
// handlers are only registered, never called.
type Deps struct {
	Facilities    *services.FacilityService
	Registrations *services.RegistrationService
	Manage        *services.GuestManageService
	AdminAuth     *services.AdminAuthService
	Admin         *services.AdminRegistrationService
	Surveys       *services.SurveyService
}

// publicSecurity marks an operation as intentionally unauthenticated
// (`security: []` in the spec).
var publicSecurity = []map[string][]string{}

// HumaConfig is the single API configuration used by the server, the
// OpenAPI exporter and the tests.
func HumaConfig(docsEnabled bool) huma.Config {
	cfg := huma.DefaultConfig(APITitle, APIVersion)
	// Drop the default $schema link transformer: the envelope must have
	// exactly {code,message,details,request_id}.
	cfg.CreateHooks = nil
	cfg.Transformers = []huma.Transformer{models.RequestIDTransformer}
	cfg.Info.Description = "Guest registration API for Saku no Ie. Errors use the envelope {code, message, details, request_id}."
	// Same origin: the web app proxies /api/* to this server (paths already carry /api/v1).
	cfg.Servers = []*huma.Server{{URL: "/", Description: "Same origin as the web app"}}
	cfg.Components.SecuritySchemes = map[string]*huma.SecurityScheme{
		"adminBearer": {Type: "http", Scheme: "bearer", BearerFormat: "JWT", Description: "Firebase ID token (admin endpoints)"},
		"editToken":   {Type: "http", Scheme: "bearer", Description: "Guest edit token from the manage URL fragment"},
	}
	if !docsEnabled {
		cfg.DocsPath = ""
		cfg.OpenAPIPath = ""
		cfg.SchemasPath = ""
	}
	return cfg
}

// NewAPI installs the error constructors, creates the Huma API on r and
// registers every /api/v1 operation.
func NewAPI(r *gin.Engine, docsEnabled bool, deps Deps) huma.API {
	models.InstallErrorConstructors()
	api := humagin.New(r, HumaConfig(docsEnabled))
	Register(api, deps)
	return api
}

// Register adds all operations under /api/v1.
func Register(api huma.API, deps Deps) {
	v1 := huma.NewGroup(api, APIPrefix)
	SetupFacilityRoutes(v1, &handlers.FacilityHandlers{Facilities: deps.Facilities})
	SetupGuestRegistrationRoutes(v1,
		&handlers.GuestRegistrationHandlers{Registrations: deps.Registrations},
		&handlers.GuestManageHandlers{Manage: deps.Manage})
	SetupSurveyRoutes(v1, &handlers.SurveyHandlers{Surveys: deps.Surveys})
	SetupAdminRoutes(v1, &handlers.AdminHandlers{Registrations: deps.Admin, Surveys: deps.Surveys}, deps.AdminAuth)
}

// NewEngine builds the Gin engine with the middleware chain used in
// production: request ID → recovery → request log → rate limit.
func NewEngine(cfg *config.Config, db *gorm.DB, deps Deps, logger *slog.Logger) (*gin.Engine, error) {
	r := gin.New()
	if err := r.SetTrustedProxies(cfg.TrustedProxies); err != nil {
		return nil, err
	}
	limiter := middlewares.NewRateLimiter(map[string]int{
		middlewares.LimitCreate: cfg.RateLimitCreatePerMin,
		middlewares.LimitManage: cfg.RateLimitManagePerMin,
		middlewares.LimitSurvey: cfg.RateLimitSurveyPerMin,
	}, logger)
	r.Use(middlewares.RequestID(), middlewares.Recovery(logger), middlewares.RequestLogger(logger), limiter.Middleware())
	r.NoRoute(middlewares.NoRoute)
	SetupStaticRoutes(r, db)
	NewAPI(r, cfg.DocsEnabled(), deps)
	return r, nil
}

// OpenAPIYAML renders the contract with docs enabled and nil services.
func OpenAPIYAML() ([]byte, error) {
	gin.SetMode(gin.ReleaseMode)
	api := NewAPI(gin.New(), true, Deps{})
	return api.OpenAPI().YAML()
}
