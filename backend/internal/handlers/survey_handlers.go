package handlers

import (
	"context"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
)

// SurveyHandlers serve the anonymous public survey.
type SurveyHandlers struct {
	Surveys *services.SurveyService
}

func (h *SurveyHandlers) SubmitSurvey(ctx context.Context, in *models.SubmitSurveyInput) (*models.SubmitSurveyOutput, error) {
	if err := h.Surveys.Submit(ctx, &in.Body); err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return &models.SubmitSurveyOutput{}, nil
}

func (h *AdminHandlers) GetSurveySummary(ctx context.Context, in *models.GetSurveySummaryInput) (*models.GetSurveySummaryOutput, error) {
	sum, err := h.Surveys.Summary(ctx, middlewares.AdminScopeFrom(ctx), services.SurveySummaryFilter{
		FacilityID: in.FacilityID, From: in.From, To: in.To,
	})
	if err != nil {
		return nil, toHTTPError(ctx, err)
	}
	return &models.GetSurveySummaryOutput{Body: *sum}, nil
}
