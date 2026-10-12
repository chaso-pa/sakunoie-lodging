package services

import (
	"context"
	"errors"
	"regexp"
	"time"
)

// Translator turns text from one supported language into another.
type Translator interface {
	Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error)
}

// Translation provider names, as set in TRANSLATION_PROVIDER and stored in
// post_translations.provider.
const (
	ProviderMyMemory   = "mymemory"
	ProviderOpenRouter = "openrouter"
	// ProviderNone marks a result that needed no translation.
	ProviderNone = "none"
)

// translationTimeout bounds one call to a translation provider.
const translationTimeout = 10 * time.Second

// LangAuto asks the server to detect the source language.
const LangAuto = "auto"

var (
	// ErrTranslationUnavailable means no usable provider is configured.
	ErrTranslationUnavailable = errors.New("translation service unavailable")
	// ErrTranslationFailed means the provider was called and failed.
	ErrTranslationFailed = errors.New("translation failed")
)

var languageNames = map[string]string{
	"ja": "Japanese",
	"en": "English",
	"ko": "Korean",
	"zh": "Simplified Chinese",
}

var (
	hangulPattern = regexp.MustCompile(`[가-힣]`)
	kanaPattern   = regexp.MustCompile(`[ぁ-ゖァ-ヺー]`)
	hanPattern    = regexp.MustCompile(`[一-龯]`)
)

// DetectLanguage guesses the language of text from its script, the same way
// the prototype's detectTextLanguage does: Hangul means Korean, kana means
// Japanese, Han characters alone mean Chinese, anything else English.
func DetectLanguage(text string) string {
	switch {
	case hangulPattern.MatchString(text):
		return "ko"
	case kanaPattern.MatchString(text):
		return "ja"
	case hanPattern.MatchString(text):
		return "zh"
	}
	return "en"
}
