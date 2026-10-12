package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"unicode/utf8"
)

// myMemoryChunkRunes is how much text one request carries. The free API
// refuses queries over 500 characters; the prototype sent at most 450.
const myMemoryChunkRunes = 450

// mymemory's codes for our languages, where they differ.
var myMemoryLangCodes = map[string]string{"zh": "zh-CN"}

// MyMemoryTranslator uses the free, keyless MyMemory API.
type MyMemoryTranslator struct {
	baseURL string
	http    *http.Client
}

func NewMyMemoryTranslator(baseURL string) *MyMemoryTranslator {
	return &MyMemoryTranslator{
		baseURL: strings.TrimRight(orDefault(baseURL, DefaultMyMemoryBaseURL), "/"),
		http:    &http.Client{Timeout: translationTimeout},
	}
}

func (t *MyMemoryTranslator) Translate(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	var out strings.Builder
	for _, chunk := range splitForTranslation(text, myMemoryChunkRunes) {
		if strings.TrimSpace(chunk) == "" {
			out.WriteString(chunk)
			continue
		}
		translated, err := t.translateChunk(ctx, chunk, sourceLang, targetLang)
		if err != nil {
			return "", err
		}
		out.WriteString(translated)
		// Line breaks separate chunks and the API drops trailing ones.
		if strings.HasSuffix(chunk, "\n") && !strings.HasSuffix(translated, "\n") {
			out.WriteString("\n")
		}
	}
	return strings.TrimSpace(out.String()), nil
}

type myMemoryResponse struct {
	ResponseData struct {
		TranslatedText string `json:"translatedText"`
	} `json:"responseData"`
	// A number on success, but a string such as "403" on some errors.
	ResponseStatus json.RawMessage `json:"responseStatus"`
}

func (t *MyMemoryTranslator) translateChunk(ctx context.Context, text, sourceLang, targetLang string) (string, error) {
	const service = "mymemory"
	query := url.Values{}
	query.Set("q", text)
	query.Set("langpair", myMemoryLang(sourceLang)+"|"+myMemoryLang(targetLang))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, t.baseURL+"/get?"+query.Encode(), nil)
	if err != nil {
		return "", upstreamFormatError(service, "cannot build request")
	}

	status, body, err := readUpstream(t.http, service, req)
	if err != nil {
		return "", err
	}
	if status != http.StatusOK {
		return "", upstreamStatusError(service, status)
	}
	var parsed myMemoryResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return "", upstreamFormatError(service, "response is not valid JSON")
	}
	// The API answers HTTP 200 with its own status inside the body, and puts
	// the error message where the translation would be (quota exhausted,
	// invalid language pair and so on).
	if code := myMemoryStatus(parsed.ResponseStatus); code != http.StatusOK {
		return "", upstreamStatusError(service, code)
	}
	translated := parsed.ResponseData.TranslatedText
	if strings.TrimSpace(translated) == "" {
		return "", upstreamFormatError(service, "response has no translation")
	}
	return translated, nil
}

func myMemoryLang(lang string) string {
	if code, ok := myMemoryLangCodes[lang]; ok {
		return code
	}
	return lang
}

func myMemoryStatus(raw json.RawMessage) int {
	code, err := strconv.Atoi(strings.Trim(strings.TrimSpace(string(raw)), `"`))
	if err != nil {
		return 0
	}
	return code
}

// splitForTranslation cuts text into pieces of at most limit runes whose
// concatenation is the original text. It prefers to cut after a line break,
// then after a sentence end, then after a space.
func splitForTranslation(text string, limit int) []string {
	var chunks []string
	for utf8.RuneCountInString(text) > limit {
		runes := []rune(text)
		cut := limit
		for _, seps := range []string{"\n", "。！？.!?", " 、,"} {
			if i := lastIndexAnyRune(runes[:limit], seps); i >= 0 {
				cut = i + 1
				break
			}
		}
		chunks = append(chunks, string(runes[:cut]))
		text = string(runes[cut:])
	}
	if text != "" {
		chunks = append(chunks, text)
	}
	return chunks
}

func lastIndexAnyRune(runes []rune, chars string) int {
	for i := len(runes) - 1; i >= 0; i-- {
		if strings.ContainsRune(chars, runes[i]) {
			return i
		}
	}
	return -1
}
