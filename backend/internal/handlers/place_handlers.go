package handlers

import (
	"context"
	"errors"
	"net/http"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
	"github.com/danielgtaylor/huma/v2"
)

// PlaceHandlers holds place search and spot registration, all behind
// RequireAuth.
//
// Search text and positions arrive in the request body, never in the URL.
// Nothing here logs them: a failure is logged with the provider's own
// sanitised error, which names a status or a cause and nothing else.
type PlaceHandlers struct {
	places *services.PlaceService
}

func NewPlaceHandlers(places *services.PlaceService) *PlaceHandlers {
	return &PlaceHandlers{places: places}
}

// placeBusyRetryAfter is the Retry-After, in seconds, of a busy answer:
// long enough for the queue of upstream calls to drain.
const placeBusyRetryAfter = "3"

// placeError maps the provider errors to fixed responses. The upstream
// cause only goes to the log.
func placeError(ctx context.Context, op string, err error) error {
	switch {
	case errors.Is(err, services.ErrPlaceNotConfigured):
		return huma.Error503ServiceUnavailable("place search is not configured")
	case errors.Is(err, services.ErrPlaceBusy):
		// Retry-After is what tells this 503 from the "not configured"
		// one: only this one is worth trying again.
		return huma.ErrorWithHeaders(
			huma.Error503ServiceUnavailable("place search is busy, try again shortly"),
			http.Header{"Retry-After": []string{placeBusyRetryAfter}},
		)
	case errors.Is(err, services.ErrPlaceNotFound):
		return huma.Error404NotFound("place not found")
	case errors.Is(err, services.ErrPlaceUnavailable):
		logFailure(ctx, op, err)
		return huma.Error502BadGateway("place search temporarily unavailable")
	}
	return serviceError(ctx, op, err)
}

func (h *PlaceHandlers) SearchPlaces(ctx context.Context, input *models.SearchPlacesInput) (*models.PlaceListOutput, error) {
	b := input.Body
	found, err := h.places.Search(ctx, middlewares.GetMemberID(ctx), services.PlaceSearchParams{
		Query:     b.Query,
		Language:  b.Language,
		Latitude:  b.Latitude,
		Longitude: b.Longitude,
		Limit:     b.Limit,
	})
	if err != nil {
		return nil, placeError(ctx, "search places", err)
	}
	return placeListOutput(found), nil
}

func (h *PlaceHandlers) NearbyPlaces(ctx context.Context, input *models.NearbyPlacesInput) (*models.PlaceListOutput, error) {
	b := input.Body
	found, err := h.places.Nearby(ctx, middlewares.GetMemberID(ctx), services.PlaceNearbyParams{
		Latitude:  b.Latitude,
		Longitude: b.Longitude,
		RadiusM:   b.RadiusM,
		Category:  b.Category,
		Language:  b.Language,
		Limit:     b.Limit,
	})
	if err != nil {
		return nil, placeError(ctx, "nearby places", err)
	}
	return placeListOutput(found), nil
}

func (h *PlaceHandlers) ReversePlace(ctx context.Context, input *models.ReversePlaceInput) (*models.ReversePlaceOutput, error) {
	b := input.Body
	result, meta, err := h.places.Reverse(ctx, middlewares.GetMemberID(ctx), b.Latitude, b.Longitude, b.Language)
	if err != nil {
		return nil, placeError(ctx, "reverse place", err)
	}
	resp := &models.ReversePlaceOutput{}
	resp.Body.Result = result
	resp.Body.Provider = meta.Provider
	resp.Body.Attribution = meta.Attribution
	return resp, nil
}

func (h *PlaceHandlers) RegisterSpot(ctx context.Context, input *models.RegisterSpotInput) (*models.RegisterSpotOutput, error) {
	b := input.Body
	spot, created, err := h.places.Register(ctx, middlewares.GetMemberID(ctx), services.RegisterSpotParams{
		Source:     b.Source,
		ExternalID: b.ExternalID,
		Language:   b.Language,
		Name:       b.Name,
		Address:    b.Address,
		Latitude:   b.Latitude,
		Longitude:  b.Longitude,
		Category:   b.Category,
		Area:       b.Area,
	})
	if err != nil {
		return nil, placeError(ctx, "register spot", err)
	}
	resp := &models.RegisterSpotOutput{}
	resp.Body.Spot = *spot
	resp.Body.Created = created
	return resp, nil
}

func placeListOutput(found *services.PlaceResults) *models.PlaceListOutput {
	resp := &models.PlaceListOutput{}
	resp.Body.Results = found.Results
	if resp.Body.Results == nil {
		resp.Body.Results = []models.PlaceResult{}
	}
	resp.Body.Provider = found.Provider
	resp.Body.Attribution = found.Attribution
	return resp
}
