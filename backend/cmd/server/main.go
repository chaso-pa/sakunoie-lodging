package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/config"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/database"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/logging"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/routes"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/server"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/session"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}
	gin.SetMode(cfg.GinMode)
	logger := logging.Setup(os.Stdout, slog.LevelInfo)
	if cfg.IsRelease() {
		for _, warning := range cfg.InviteCodeWarnings() {
			log.Println("WARNING: " + warning)
		}
	}

	db, err := database.Open(cfg.GormDatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect database: ", err)
	}
	sessionDB, err := database.OpenSessionDB(cfg.GormDatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect session database: ", err)
	}
	sessionManager := session.New(sessionDB, cfg.IsRelease())

	// Routes are registered in server.NewRouter (internal/server/router.go).
	r, err := server.NewRouter(&routes.Deps{
		DB:       db,
		Sessions: sessionManager,
		Config:   cfg,
		Images:   newImageUploader(cfg),
		Logger:   logger,
	})
	if err != nil {
		log.Fatal("Failed to build router: ", err)
	}

	// scs wraps the whole engine so every Gin middleware and Huma handler
	// sees the session on the request context. Do not use r.Run().
	log.Printf("Server starting on port %s", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, sessionManager.LoadAndSave(r)); err != nil {
		log.Fatal("Failed to start server: ", err)
	}
}

// newImageUploader builds the S3 backed uploader. Without S3 settings image
// uploads answer 503 and everything else keeps working.
//
// The bucket and its public-read policy are created here when missing. If
// the store is down at startup the server still starts, and the same check
// runs again before the first upload.
func newImageUploader(cfg *config.Config) storage.ImageUploader {
	store, err := storage.NewS3Store(storage.S3Config{
		Endpoint:  cfg.S3Endpoint,
		Bucket:    cfg.S3Bucket,
		AccessKey: cfg.S3AccessKey,
		SecretKey: cfg.S3SecretKey,
		UseSSL:    cfg.S3UseSSL,
		PublicURL: cfg.S3PublicURL,
	})
	if err != nil {
		if errors.Is(err, storage.ErrNotConfigured) {
			slog.Warn("S3 settings are missing: image uploads are disabled")
		} else {
			slog.Error("cannot build the S3 client: image uploads are disabled", "error", err)
		}
		return storage.NotConfigured{}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := store.EnsureBucket(ctx); err != nil {
		slog.Warn("image bucket is not ready yet; it is checked again on the first upload", "error", err)
	}
	return services.NewImageService(store)
}
