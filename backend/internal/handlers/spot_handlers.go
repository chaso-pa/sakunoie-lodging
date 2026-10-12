package handlers

import (
	"context"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
)

type SpotHandlers struct {
	spots *services.SpotService
}

func NewSpotHandlers(spots *services.SpotService) *SpotHandlers {
	return &SpotHandlers{spots: spots}
}

func (h *SpotHandlers) ListSpots(ctx context.Context, input *models.ListSpotsInput) (*models.ListSpotsOutput, error) {
	spots, err := h.spots.List(ctx, input.Area)
	if err != nil {
		return nil, serviceError(ctx, "list spots", err)
	}
	resp := &models.ListSpotsOutput{}
	resp.Body.Spots = spots
	return resp, nil
}

func (h *SpotHandlers) GetSpot(ctx context.Context, input *models.GetSpotInput) (*models.GetSpotOutput, error) {
	detail, err := h.spots.Get(ctx, input.ID)
	if err != nil {
		return nil, serviceError(ctx, "get spot", err)
	}
	resp := &models.GetSpotOutput{}
	resp.Body.Spot = detail.Spot
	resp.Body.Reviews = detail.Reviews
	resp.Body.Posts = detail.Posts
	return resp, nil
}

func (h *SpotHandlers) ListAiSpots(ctx context.Context, input *models.ListAiSpotsInput) (*models.ListAiSpotsOutput, error) {
	spots, err := h.spots.ListAiSpots(ctx, services.AiSpotFilter{
		Genre:     input.Genre,
		Purpose:   input.Purpose,
		Weather:   input.Weather,
		Duration:  input.Duration,
		Transport: input.Transport,
	})
	if err != nil {
		return nil, serviceError(ctx, "list ai spots", err)
	}
	resp := &models.ListAiSpotsOutput{}
	resp.Body.Spots = spots
	return resp, nil
}

// CreateReview is registered behind RequireAuth.
func (h *SpotHandlers) CreateReview(ctx context.Context, input *models.CreateReviewInput) (*models.CreateReviewOutput, error) {
	review, err := h.spots.CreateReview(ctx, middlewares.GetMemberID(ctx), input.ID, input.Body.Stars, input.Body.Text)
	if err != nil {
		return nil, serviceError(ctx, "create review", err)
	}
	resp := &models.CreateReviewOutput{}
	resp.Body.Review = *review
	return resp, nil
}
