package services

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"
)

// DefaultOpenRouterModel is used when OPENROUTER_MODEL is empty.
const DefaultOpenRouterModel = "google/gemini-flash-1.5"

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
}

type chatResponse struct {
	Model   string `json:"model"`
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
}

// openRouterClient calls OpenRouter's chat completions endpoint. The API key
// only ever travels in the Authorization header.
type openRouterClient struct {
	baseURL string
	apiKey  string
	model   string
	http    *http.Client
}

func newOpenRouterClient(baseURL, apiKey, model string, timeout time.Duration) *openRouterClient {
	return &openRouterClient{
		baseURL: strings.TrimRight(orDefault(baseURL, DefaultOpenRouterBaseURL), "/"),
		apiKey:  strings.TrimSpace(apiKey),
		model:   orDefault(strings.TrimSpace(model), DefaultOpenRouterModel),
		http:    &http.Client{Timeout: timeout},
	}
}

func (c *openRouterClient) configured() bool { return c.apiKey != "" }

// chat returns the first choice's text and the model that produced it.
func (c *openRouterClient) chat(ctx context.Context, messages []chatMessage, maxTokens int, temperature float64) (string, string, error) {
	const service = "openrouter"
	payload, err := json.Marshal(chatRequest{Model: c.model, Messages: messages, MaxTokens: maxTokens, Temperature: temperature})
	if err != nil {
		return "", "", upstreamFormatError(service, "cannot encode request")
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return "", "", upstreamFormatError(service, "cannot build request")
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")

	status, body, err := readUpstream(c.http, service, req)
	if err != nil {
		return "", "", err
	}
	if status != http.StatusOK {
		return "", "", upstreamStatusError(service, status)
	}
	var parsed chatResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", "", upstreamFormatError(service, "response is not valid JSON")
	}
	if len(parsed.Choices) == 0 || strings.TrimSpace(parsed.Choices[0].Message.Content) == "" {
		return "", "", upstreamFormatError(service, "response has no content")
	}
	return strings.TrimSpace(parsed.Choices[0].Message.Content), orDefault(parsed.Model, c.model), nil
}
