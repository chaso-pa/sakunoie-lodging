package services

import "strings"

// NewTranslator picks the provider named by TRANSLATION_PROVIDER.
//
// It returns nil and ErrTranslationUnavailable when the name is empty or
// unknown, or when it is "openrouter" and no API key is set. mymemory needs
// no key and is always available.
func NewTranslator(provider, openRouterAPIKey, openRouterModel string, endpoints ExternalEndpoints) (Translator, string, error) {
	switch strings.ToLower(strings.TrimSpace(provider)) {
	case ProviderMyMemory:
		return NewMyMemoryTranslator(endpoints.MyMemoryBaseURL), ProviderMyMemory, nil
	case ProviderOpenRouter:
		t, err := NewOpenRouterTranslator(endpoints.OpenRouterBaseURL, openRouterAPIKey, openRouterModel)
		if err != nil {
			return nil, "", ErrTranslationUnavailable
		}
		return t, ProviderOpenRouter, nil
	}
	return nil, "", ErrTranslationUnavailable
}
