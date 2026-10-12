package handlers

import (
	"context"
	"errors"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
	"github.com/danielgtaylor/huma/v2"
)

// ExternalHandlers holds the endpoints backed by outside services:
// translation, the AI assistant and the weather. Their error bodies are
// fixed strings; the upstream cause only goes to the log, already stripped
// of URLs and upstream text by the services.
type ExternalHandlers struct {
	translation *services.TranslationService
	ai          *services.AIService
	weather     *services.WeatherService
}

func NewExternalHandlers(translation *services.TranslationService, ai *services.AIService, weather *services.WeatherService) *ExternalHandlers {
	return &ExternalHandlers{translation: translation, ai: ai, weather: weather}
}

func (h *ExternalHandlers) Translate(ctx context.Context, input *models.TranslateInput) (*models.TranslateOutput, error) {
	result, err := h.translation.Translate(ctx, input.Body.Text, input.Body.SourceLang, input.Body.TargetLang)
	if err != nil {
		switch {
		case errors.Is(err, services.ErrTranslationUnavailable):
			return nil, huma.Error503ServiceUnavailable("translation service unavailable")
		case errors.Is(err, services.ErrTranslationFailed):
			logFailure(ctx, "translate", err)
			return nil, huma.Error502BadGateway("translation failed")
		}
		return nil, serviceError(ctx, "translate", err)
	}
	resp := &models.TranslateOutput{}
	resp.Body.TranslatedText = result.Text
	resp.Body.Provider = result.Provider
	resp.Body.Cached = result.Cached
	resp.Body.SourceLang = result.SourceLang
	return resp, nil
}

func (h *ExternalHandlers) AIPlan(ctx context.Context, input *models.AIPlanInput) (*models.AIPlanOutput, error) {
	b := input.Body
	answer, model, err := h.ai.Plan(ctx, services.AIPlanRequest{
		Message:   b.Message,
		Language:  b.Language,
		Purpose:   b.Purpose,
		Genre:     b.Genre,
		Weather:   b.Weather,
		Duration:  b.Duration,
		Transport: b.Transport,
	})
	if err != nil {
		switch {
		case errors.Is(err, services.ErrAINotConfigured):
			return nil, huma.Error503ServiceUnavailable("AI service not configured")
		case errors.Is(err, services.ErrInvalidAIRequest):
			return nil, huma.Error422UnprocessableEntity(err.Error())
		case errors.Is(err, services.ErrAIUnavailable):
			logFailure(ctx, "ai plan", err)
			return nil, huma.Error502BadGateway("AI service temporarily unavailable")
		}
		return nil, internalError(ctx, "ai plan", err)
	}
	resp := &models.AIPlanOutput{}
	resp.Body.Answer = answer
	resp.Body.Model = model
	return resp, nil
}

func (h *ExternalHandlers) GetWeather(ctx context.Context, input *models.WeatherInput) (*models.WeatherOutput, error) {
	weather, err := h.weather.Current(ctx, input.Latitude, input.Longitude)
	if err != nil {
		if errors.Is(err, services.ErrWeatherUnavailable) {
			logFailure(ctx, "get weather", err)
			return nil, huma.Error502BadGateway("weather service temporarily unavailable")
		}
		return nil, internalError(ctx, "get weather", err)
	}
	resp := &models.WeatherOutput{}
	resp.Body.Weather = weather.Label
	resp.Body.Temperature = weather.Temperature
	resp.Body.Icon = weather.Icon
	resp.Body.Code = weather.Code
	resp.Body.Source = weather.Source
	return resp, nil
}
