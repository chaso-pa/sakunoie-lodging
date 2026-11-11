package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/config"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/database"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/routes"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"github.com/gin-gonic/gin"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)
	if err := run(logger); err != nil {
		logger.Error("server stopped", "error", err.Error())
		os.Exit(1)
	}
}

func run(logger *slog.Logger) error {
	cfg, err := config.Load(os.Getenv)
	if err != nil {
		return err
	}
	if cfg.AppEnv == config.EnvDevelopment {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	db, err := database.Open(cfg.DatabaseURL, logger)
	if err != nil {
		return err
	}
	if err := database.Ping(context.Background(), db, 5*time.Second); err != nil {
		return errors.New("database: ping failed")
	}

	cipher, err := utils.NewPassportCipher(cfg.PassportKeys, cfg.PassportActiveKeyID)
	if err != nil {
		return err
	}
	clock := services.SystemClock{}
	tokens := services.NewTokenService(clock)
	idem := services.NewIdempotencyService(db, clock, cfg.IdempotencyHMACKey, logger)
	verifier, err := services.NewFirebaseVerifier(context.Background(), cfg.FirebaseProjectID)
	if err != nil {
		return err
	}
	deps := routes.Deps{
		Facilities:    services.NewFacilityService(db),
		Registrations: services.NewRegistrationService(db, clock, tokens, idem, cipher),
		Manage:        services.NewGuestManageService(db, clock, tokens, cipher),
		AdminAuth:     services.NewAdminAuthService(db, verifier),
		Admin:         services.NewAdminRegistrationService(db, clock, tokens, cipher),
		Surveys:       services.NewSurveyService(db, clock, cfg.SurveyMinResponses),
	}

	models.InstallErrorConstructors()
	engine, err := routes.NewEngine(cfg, db, deps, logger)
	if err != nil {
		return err
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()
	janitorDone := idem.StartJanitor(ctx)

	srv := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           engine,
		ReadHeaderTimeout: 10 * time.Second,
	}
	errCh := make(chan error, 1)
	go func() {
		logger.Info("server starting", "port", cfg.Port, "app_env", cfg.AppEnv, "docs", cfg.DocsEnabled(),
			"firebase_emulator", cfg.FirebaseEmulatorHost != "")
		errCh <- srv.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		logger.Info("shutting down")
	}
	stop()
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}
	<-janitorDone
	if sqlDB, err := db.DB(); err == nil {
		_ = sqlDB.Close()
	}
	return nil
}
