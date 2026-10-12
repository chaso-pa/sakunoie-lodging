package services

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
)

// countingTranslator answers "<target>:<text>" and counts its calls.
type countingTranslator struct {
	mu    sync.Mutex
	calls int
	err   error
}

func (c *countingTranslator) Translate(_ context.Context, text, _, targetLang string) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.calls++
	if c.err != nil {
		return "", c.err
	}
	return targetLang + ":" + text, nil
}

func TestTranslationServiceCacheIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	provider := &countingTranslator{}
	s := NewTranslationService(provider, ProviderMyMemory, NewTranslationCache(db))

	// Miss: the provider is called and the result stored.
	first, err := s.Translate(ctx, "静かな森", "ja", "en")
	if err != nil {
		t.Fatal(err)
	}
	if first.Text != "en:静かな森" || first.Cached || first.Provider != ProviderMyMemory || first.SourceLang != "ja" {
		t.Fatalf("first = %+v", first)
	}
	// Hit: same text and pair, no second call.
	second, err := s.Translate(ctx, "  静かな森  ", "ja", "en")
	if err != nil {
		t.Fatal(err)
	}
	if second.Text != first.Text || !second.Cached || second.Provider != ProviderMyMemory {
		t.Fatalf("second = %+v", second)
	}
	if provider.calls != 1 {
		t.Fatalf("provider calls = %d, want 1", provider.calls)
	}

	// Another target, and the same characters read as another source
	// language, are separate entries.
	if r, _ := s.Translate(ctx, "静かな森", "ja", "ko"); r == nil || r.Cached {
		t.Fatalf("other target = %+v", r)
	}
	if r, _ := s.Translate(ctx, "静かな森", "zh", "en"); r == nil || r.Cached {
		t.Fatalf("other source = %+v", r)
	}
	if provider.calls != 3 {
		t.Fatalf("provider calls = %d, want 3", provider.calls)
	}

	// "auto" detects the language, and then shares the cache entry.
	auto, err := s.Translate(ctx, "静かな森", LangAuto, "en")
	if err != nil {
		t.Fatal(err)
	}
	if !auto.Cached || auto.SourceLang != "ja" {
		t.Fatalf("auto = %+v, want the cached ja→en entry", auto)
	}

	// Same source and target: the text comes back, nothing is called or stored.
	same, err := s.Translate(ctx, "Hello", "en", "en")
	if err != nil {
		t.Fatal(err)
	}
	if same.Text != "Hello" || same.Cached || same.Provider != ProviderNone {
		t.Fatalf("same language = %+v", same)
	}
	if same, _ := s.Translate(ctx, "Hello there", LangAuto, "en"); same == nil || same.Provider != ProviderNone {
		t.Fatalf("auto, same language = %+v", same)
	}
	if provider.calls != 3 {
		t.Fatalf("provider was called for a same-language request")
	}
	var rows int64
	db.Model(&models.PostTranslation{}).Count(&rows)
	if rows != 3 {
		t.Fatalf("cache rows = %d, want 3", rows)
	}

	if _, err := s.Translate(ctx, "x", "fr", "en"); !errors.Is(err, ErrInvalidLanguage) {
		t.Fatalf("unsupported language: err = %v", err)
	}
}

func TestTranslationCacheDuplicateInsertIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	cache := NewTranslationCache(db)

	if err := cache.Put(ctx, "text", "ja", "en", "first", ProviderMyMemory); err != nil {
		t.Fatal(err)
	}
	// The same key again is not an error and keeps the first row.
	if err := cache.Put(ctx, "text", "ja", "en", "second", ProviderOpenRouter); err != nil {
		t.Fatalf("duplicate insert: %v", err)
	}
	got, provider, ok, err := cache.Get(ctx, "text", "ja", "en")
	if err != nil || !ok || got != "first" || provider != ProviderMyMemory {
		t.Fatalf("Get = %q, %q, %v, %v", got, provider, ok, err)
	}
	if _, _, ok, _ := cache.Get(ctx, "text", "ja", "ko"); ok {
		t.Fatal("hit for a pair that was never stored")
	}
	var rows int64
	db.Model(&models.PostTranslation{}).Count(&rows)
	if rows != 1 {
		t.Fatalf("rows = %d, want 1", rows)
	}

	// Concurrent first translations of one text all succeed.
	provider2 := &countingTranslator{}
	s := NewTranslationService(provider2, ProviderMyMemory, cache)
	var wg sync.WaitGroup
	errs := make(chan error, 8)
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if _, err := s.Translate(ctx, "同時に翻訳", "ja", "en"); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		t.Fatalf("concurrent translate: %v", err)
	}
	db.Model(&models.PostTranslation{}).Count(&rows)
	if rows != 2 {
		t.Fatalf("rows = %d, want 2", rows)
	}
}

func TestTranslationServiceFailuresIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	cache := NewTranslationCache(db)

	// No provider: 503 for new text, but cached text is still served.
	unconfigured := NewTranslationService(nil, "", cache)
	if _, err := unconfigured.Translate(ctx, "こんにちは", "ja", "en"); !errors.Is(err, ErrTranslationUnavailable) {
		t.Fatalf("err = %v, want ErrTranslationUnavailable", err)
	}
	if err := cache.Put(ctx, "こんにちは", "ja", "en", "Hello", ProviderMyMemory); err != nil {
		t.Fatal(err)
	}
	r, err := unconfigured.Translate(ctx, "こんにちは", "ja", "en")
	if err != nil || !r.Cached || r.Text != "Hello" {
		t.Fatalf("cached without provider = %+v, %v", r, err)
	}
	if r, err := unconfigured.Translate(ctx, "Hello", "en", "en"); err != nil || r.Provider != ProviderNone {
		t.Fatalf("same language without provider = %+v, %v", r, err)
	}

	// A provider failure is ErrTranslationFailed and is not cached.
	failing := &countingTranslator{err: upstreamStatusError("mymemory", 429)}
	s := NewTranslationService(failing, ProviderMyMemory, cache)
	if _, err := s.Translate(ctx, "さようなら", "ja", "en"); !errors.Is(err, ErrTranslationFailed) {
		t.Fatalf("err = %v, want ErrTranslationFailed", err)
	}
	if _, _, ok, _ := cache.Get(ctx, "さようなら", "ja", "en"); ok {
		t.Fatal("a failed translation was cached")
	}
	failing.err = nil
	if r, err := s.Translate(ctx, "さようなら", "ja", "en"); err != nil || r.Cached {
		t.Fatalf("retry = %+v, %v", r, err)
	}
}
