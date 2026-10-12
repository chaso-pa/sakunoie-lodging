package handlers

import (
	"context"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
)

// GpsVisitHandlers holds the footprint endpoints, all behind RequireAuth.
// Which coordinates a response may carry is decided in the service.
type GpsVisitHandlers struct {
	visits *services.GpsVisitService
}

func NewGpsVisitHandlers(visits *services.GpsVisitService) *GpsVisitHandlers {
	return &GpsVisitHandlers{visits: visits}
}

func (h *GpsVisitHandlers) ListMyVisits(ctx context.Context, _ *struct{}) (*models.ListGpsVisitsOutput, error) {
	visits, err := h.visits.ListMine(ctx, middlewares.GetMemberID(ctx))
	if err != nil {
		return nil, serviceError(ctx, "list own visits", err)
	}
	resp := &models.ListGpsVisitsOutput{}
	resp.Body.Visits = visits
	return resp, nil
}

func (h *GpsVisitHandlers) ListPublicVisits(ctx context.Context, _ *struct{}) (*models.ListGpsVisitsOutput, error) {
	visits, err := h.visits.ListPublic(ctx, middlewares.GetMemberID(ctx))
	if err != nil {
		return nil, serviceError(ctx, "list public visits", err)
	}
	resp := &models.ListGpsVisitsOutput{}
	resp.Body.Visits = visits
	return resp, nil
}

func (h *GpsVisitHandlers) CreateVisit(ctx context.Context, input *models.CreateGpsVisitInput) (*models.GpsVisitOutput, error) {
	b := input.Body
	visit, err := h.visits.Create(ctx, middlewares.GetMemberID(ctx), services.CreateGpsVisitParams{
		SpotID:      b.SpotID,
		PlaceName:   b.PlaceName,
		Emoji:       b.Emoji,
		Area:        b.Area,
		VisitedDate: b.VisitedDate,
		Weather:     b.Weather,
		Temperature: b.Temperature,
		Note:        b.Note,
		IsPublic:    b.IsPublic,
		ExactPublic: b.ExactPublic,
		Latitude:    b.Latitude,
		Longitude:   b.Longitude,
	})
	if err != nil {
		return nil, serviceError(ctx, "create visit", err)
	}
	resp := &models.GpsVisitOutput{}
	resp.Body.Visit = *visit
	return resp, nil
}

func (h *GpsVisitHandlers) DeleteVisit(ctx context.Context, input *models.GpsVisitIDInput) (*models.OKOutput, error) {
	if err := h.visits.Delete(ctx, middlewares.GetMemberID(ctx), input.ID); err != nil {
		return nil, serviceError(ctx, "delete visit", err)
	}
	return models.NewOKOutput(), nil
}

func (h *GpsVisitHandlers) UpdateVisitVisibility(ctx context.Context, input *models.UpdateGpsVisitVisibilityInput) (*models.GpsVisitOutput, error) {
	visit, err := h.visits.SetVisibility(ctx, middlewares.GetMemberID(ctx), input.ID, input.Body.IsPublic, input.Body.ExactPublic)
	if err != nil {
		return nil, serviceError(ctx, "update visit visibility", err)
	}
	resp := &models.GpsVisitOutput{}
	resp.Body.Visit = *visit
	return resp, nil
}
