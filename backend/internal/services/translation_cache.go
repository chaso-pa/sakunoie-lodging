package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// TranslationCache stores finished translations in post_translations, keyed
// by (SHA-256 of the text, source language, target language). The source
// language is part of the key because the same characters translate
// differently depending on the language they are read as.
type TranslationCache struct {
	db *gorm.DB
}

func NewTranslationCache(db *gorm.DB) *TranslationCache {
	return &TranslationCache{db: db}
}

// HashText returns the SHA-256 hex digest used as the cache key.
func HashText(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}

// Get returns the cached translation, or ok=false on a miss.
func (c *TranslationCache) Get(ctx context.Context, text, sourceLang, targetLang string) (translated, provider string, ok bool, err error) {
	var row models.PostTranslation
	err = c.db.WithContext(ctx).
		Where("source_text_hash = ? AND source_lang = ? AND target_lang = ?", HashText(text), sourceLang, targetLang).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", "", false, nil
	}
	if err != nil {
		return "", "", false, fmt.Errorf("read translation cache: %w", err)
	}
	return row.TranslatedText, row.Provider, true, nil
}

// Put stores a translation. A row that already exists for the key is kept,
// so two requests translating the same text at once do not fail.
func (c *TranslationCache) Put(ctx context.Context, text, sourceLang, targetLang, translated, provider string) error {
	row := models.PostTranslation{
		SourceTextHash: HashText(text),
		SourceLang:     sourceLang,
		TargetLang:     targetLang,
		TranslatedText: translated,
		Provider:       provider,
	}
	if err := c.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&row).Error; err != nil {
		return fmt.Errorf("write translation cache: %w", err)
	}
	return nil
}

// TranslationResult is what POST /api/translate returns.
type TranslationResult struct {
	Text     string
	Provider string
	Cached   bool
	// SourceLang is the language the text was translated from, which is the
	// detected one when the request said "auto".
	SourceLang string
}

// TranslationService translates through the cache: a provider is called
// only for text that has not been translated to that language before.
type TranslationService struct {
	translator Translator // nil when no provider is configured
	provider   string
	cache      *TranslationCache
}

func NewTranslationService(translator Translator, provider string, cache *TranslationCache) *TranslationService {
	return &TranslationService{translator: translator, provider: provider, cache: cache}
}

func (s *TranslationService) Translate(ctx context.Context, text, sourceLang, targetLang string) (*TranslationResult, error) {
	text = strings.TrimSpace(text)
	if sourceLang == LangAuto || sourceLang == "" {
		sourceLang = DetectLanguage(text)
	}
	if languageNames[sourceLang] == "" || languageNames[targetLang] == "" {
		return nil, ErrInvalidLanguage
	}
	// Nothing to do: answer with the text itself, without calling out and
	// without filling the cache.
	if text == "" || sourceLang == targetLang {
		return &TranslationResult{Text: text, Provider: ProviderNone, SourceLang: sourceLang}, nil
	}

	translated, provider, ok, err := s.cache.Get(ctx, text, sourceLang, targetLang)
	if err != nil {
		return nil, err
	}
	if ok {
		return &TranslationResult{Text: translated, Provider: provider, Cached: true, SourceLang: sourceLang}, nil
	}

	if s.translator == nil {
		return nil, ErrTranslationUnavailable
	}
	translated, err = s.translator.Translate(ctx, text, sourceLang, targetLang)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrTranslationFailed, err)
	}
	// A failed cache write only costs a repeated translation later.
	if err := s.cache.Put(ctx, text, sourceLang, targetLang, translated, s.provider); err != nil {
		slog.WarnContext(ctx, "translation cache write failed", "error", err)
	}
	return &TranslationResult{Text: translated, Provider: s.provider, SourceLang: sourceLang}, nil
}
