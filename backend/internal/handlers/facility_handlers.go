package handlers

import (
	"context"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
)

type FacilityHandlers struct {
	Facilities *services.FacilityService
}

func (h *FacilityHandlers) GetFacilityPublic(ctx context.Context, in *models.GetFacilityPublicInput) (*models.GetFacilityPublicOutput, error) {
	f, err := h.Facilities.GetPublic(ctx, in.Code)
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return &models.GetFacilityPublicOutput{Body: *f}, nil
}
