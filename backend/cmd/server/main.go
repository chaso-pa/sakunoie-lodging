package main

import (
	"context"
	"errors"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
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

// HTTP server timeouts.
//
// readTimeout covers the whole request including its body, so it has to fit
// a 10MB photo sent over a slow phone connection: 3 minutes is about 0.5Mbps.
// writeTimeout runs from the end of the request headers to the end of the
// response, so it includes that upload as well as the handler's own work,
// the slowest of which is the AI call with its 30 second limit.
// shutdownTimeout is longer than that AI call, so a request in flight at
// shutdown can still be answered.
const (
	readHeaderTimeout = 10 * time.Second
	readTimeout       = 3 * time.Minute
	writeTimeout      = 4 * time.Minute
	idleTimeout       = 2 * time.Minute
	shutdownTimeout   = 40 * time.Second
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal("Failed to load config: ", err)
	}
	gin.SetMode(cfg.GinMode)
	logger := logging.Setup(os.Stdout, slog.LevelInfo)
	if cfg.IsRelease() {
		for _, warning := range append(cfg.InviteCodeWarnings(), cfg.PlaceWarnings()...) {
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
	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           sessionManager.LoadAndSave(r),
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}

	// Stop on SIGINT and SIGTERM: no new connections, and requests already
	// running get shutdownTimeout to finish.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	serveErr := make(chan error, 1)
	go func() {
		log.Printf("Server starting on port %s", cfg.Port)
		serveErr <- srv.ListenAndServe()
	}()

	select {
	case err := <-serveErr:
		log.Fatal("Failed to start server: ", err)
	case <-ctx.Done():
	}
	stop() // a second signal kills the process the usual way

	log.Println("Shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Shutdown did not finish cleanly: %v", err)
		_ = srv.Close()
	}
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	_ = sessionDB.Close()
	log.Println("Server stopped")
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
