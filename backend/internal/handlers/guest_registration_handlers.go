package handlers

import (
	"context"
	"net/http"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
)

type GuestRegistrationHandlers struct {
	Registrations *services.RegistrationService
}

func (h *GuestRegistrationHandlers) CreateGuestRegistration(ctx context.Context, in *models.CreateGuestRegistrationInput) (*models.CreateGuestRegistrationOutput, error) {
	res, err := h.Registrations.Create(ctx, in.IdempotencyKey, &in.Body)
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	status := http.StatusCreated
	if res.Replayed {
		status = http.StatusOK
	}
	return &models.CreateGuestRegistrationOutput{Status: status, CacheControl: "no-store", Body: res.Body}, nil
}
