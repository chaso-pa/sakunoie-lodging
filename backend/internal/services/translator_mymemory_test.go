package services

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"
	"unicode/utf8"
)

func TestMyMemoryTranslate(t *testing.T) {
	srv := newStub(t, func(w http.ResponseWriter, r stubRequest) {
		w.Write([]byte(`{"responseData":{"translatedText":"Good morning"},"responseStatus":200}`))
	})
	tr := NewMyMemoryTranslator(srv.URL)

	got, err := tr.Translate(context.Background(), "おはよう", "ja", "en")
	if err != nil {
		t.Fatal(err)
	}
	if got != "Good morning" {
		t.Fatalf("translation = %q", got)
	}
	req := srv.last()
	if req.Method != http.MethodGet || req.Path != "/get" || req.Query["q"] != "おはよう" || req.Query["langpair"] != "ja|en" {
		t.Fatalf("request = %+v", req)
	}

	// Chinese is zh-CN for this API.
	if _, err := tr.Translate(context.Background(), "hello", "en", "zh"); err != nil {
		t.Fatal(err)
	}
	if got := srv.last().Query["langpair"]; got != "en|zh-CN" {
		t.Fatalf("langpair = %q", got)
	}
}

func TestMyMemoryErrors(t *testing.T) {
	const text = "秘密の投稿本文"
	const upstreamText = "MYMEMORY WARNING: YOU USED ALL AVAILABLE FREE TRANSLATIONS FOR TODAY"

	cases := map[string]func(w http.ResponseWriter, r stubRequest){
		// The API reports its own errors with HTTP 200.
		"quota in body": func(w http.ResponseWriter, _ stubRequest) {
			w.Write([]byte(`{"responseData":{"translatedText":"` + upstreamText + `"},"responseStatus":429}`))
		},
		"string status": func(w http.ResponseWriter, _ stubRequest) {
			w.Write([]byte(`{"responseData":{"translatedText":"` + upstreamText + `"},"responseStatus":"403"}`))
		},
		"http 500": func(w http.ResponseWriter, _ stubRequest) {
			http.Error(w, upstreamText, http.StatusInternalServerError)
		},
		"not json": func(w http.ResponseWriter, _ stubRequest) { w.Write([]byte("<html>" + upstreamText + "</html>")) },
		"empty translation": func(w http.ResponseWriter, _ stubRequest) {
			w.Write([]byte(`{"responseData":{"translatedText":""},"responseStatus":200}`))
		},
		"missing status": func(w http.ResponseWriter, _ stubRequest) { w.Write([]byte(`{"responseData":{"translatedText":"x"}}`)) },
	}
	for name, handler := range cases {
		srv := newStub(t, handler)
		_, err := NewMyMemoryTranslator(srv.URL).Translate(context.Background(), text, "ja", "en")
		if !errors.Is(err, ErrUpstream) {
			t.Errorf("%s: err = %v, want ErrUpstream", name, err)
			continue
		}
		assertNoLeak(t, err, text, upstreamText, "MYMEMORY", srv.URL)
	}
}

func TestMyMemoryTimeout(t *testing.T) {
	if translationTimeout != 10*time.Second {
		t.Fatalf("translation timeout = %v, want 10s", translationTimeout)
	}
	release := make(chan struct{})
	srv := newStub(t, func(w http.ResponseWriter, _ stubRequest) { <-release })
	defer close(release)

	tr := NewMyMemoryTranslator(srv.URL)
	if tr.http.Timeout != translationTimeout {
		t.Fatalf("client timeout = %v, want %v", tr.http.Timeout, translationTimeout)
	}
	tr.http.Timeout = 50 * time.Millisecond

	start := time.Now()
	_, err := tr.Translate(context.Background(), "秘密の投稿本文", "ja", "en")
	if !errors.Is(err, ErrUpstream) || !strings.Contains(err.Error(), "timeout") {
		t.Fatalf("err = %v, want a timeout", err)
	}
	if time.Since(start) > 2*time.Second {
		t.Fatal("the call did not time out promptly")
	}
	assertNoLeak(t, err, "秘密の投稿本文", srv.URL)
}

func TestMyMemoryChunksLongText(t *testing.T) {
	srv := newStub(t, func(w http.ResponseWriter, r stubRequest) {
		w.Write([]byte(`{"responseData":{"translatedText":"[` + r.Query["langpair"] + `]"},"responseStatus":200}`))
	})
	sentence := strings.Repeat("あ", 199) + "。"
	text := strings.Repeat(sentence, 5) // 1000 characters

	if _, err := NewMyMemoryTranslator(srv.URL).Translate(context.Background(), text, "ja", "en"); err != nil {
		t.Fatal(err)
	}
	if srv.calls() != 3 {
		t.Fatalf("requests = %d, want 3", srv.calls())
	}
	var sent strings.Builder
	for _, r := range srv.requests {
		if n := utf8.RuneCountInString(r.Query["q"]); n > myMemoryChunkRunes {
			t.Fatalf("a chunk has %d characters, over the limit", n)
		}
		if !strings.HasSuffix(r.Query["q"], "。") {
			t.Fatalf("chunk does not end at a sentence: …%q", r.Query["q"][len(r.Query["q"])-6:])
		}
		sent.WriteString(r.Query["q"])
	}
	if sent.String() != text {
		t.Fatal("the chunks do not add up to the original text")
	}
}

func TestSplitForTranslation(t *testing.T) {
	for _, text := range []string{"", "short", strings.Repeat("x", 450), strings.Repeat("x", 451), strings.Repeat("word ", 300), "line one\n" + strings.Repeat("y", 600)} {
		chunks := splitForTranslation(text, 450)
		if strings.Join(chunks, "") != text {
			t.Fatalf("chunks of %d characters do not rejoin", len(text))
		}
		for _, c := range chunks {
			if utf8.RuneCountInString(c) > 450 || c == "" {
				t.Fatalf("bad chunk of %d characters", utf8.RuneCountInString(c))
			}
		}
	}
}

func TestDetectLanguage(t *testing.T) {
	cases := map[string]string{
		"静かな森の空気":         "ja",
		"カフェ":             "ja",
		"安静的森林":           "zh",
		"조용한 숲":           "ko",
		"A peaceful pond": "en",
		"":                "en",
	}
	for text, want := range cases {
		if got := DetectLanguage(text); got != want {
			t.Errorf("DetectLanguage(%q) = %s, want %s", text, got, want)
		}
	}
}
