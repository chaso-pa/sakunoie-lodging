package handlers

import (
	"context"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
)

type GuestManageHandlers struct {
	Manage *services.GuestManageService
}

func manageOutput(s *models.ManageSummary) *models.ManageOutput {
	return &models.ManageOutput{ETag: `"` + s.Version + `"`, CacheControl: "no-store", Body: *s}
}

func (h *GuestManageHandlers) GetManageRegistration(ctx context.Context, _ *models.GetManageInput) (*models.ManageOutput, error) {
	s, err := h.Manage.Summary(ctx, middlewares.EditTokenFrom(ctx))
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return manageOutput(s), nil
}

func (h *GuestManageHandlers) PatchManageRegistration(ctx context.Context, in *models.PatchManageRegistrationInput) (*models.ManageOutput, error) {
	s, err := h.Manage.PatchRegistration(ctx, middlewares.EditTokenFrom(ctx), in.IfMatch, &in.Body)
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return manageOutput(s), nil
}

func (h *GuestManageHandlers) AddCompanions(ctx context.Context, in *models.AddCompanionsInput) (*models.ManageOutput, error) {
	s, err := h.Manage.AddCompanions(ctx, middlewares.EditTokenFrom(ctx), in.Body.Guests)
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return manageOutput(s), nil
}

func (h *GuestManageHandlers) PatchManageGuest(ctx context.Context, in *models.PatchManageGuestInput) (*models.ManageOutput, error) {
	s, err := h.Manage.PatchGuest(ctx, middlewares.EditTokenFrom(ctx), in.IfMatch, in.GuestPublicID, &in.Body)
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return manageOutput(s), nil
}
