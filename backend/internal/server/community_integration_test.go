package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	_ "image/jpeg"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/config"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/logging"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/routes"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
)

// memStore is an in-memory storage.ObjectStore, so the real image pipeline
// (services.ImageService) runs in these tests without MinIO.
type memStore struct {
	mu      sync.Mutex
	objects map[string][]byte
}

func newMemStore() *memStore { return &memStore{objects: map[string][]byte{}} }

const memStoreBase = "http://storage.test/bucket/"

func (m *memStore) Put(_ context.Context, key, _ string, data []byte) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.objects[key] = data
	return nil
}

func (m *memStore) Delete(_ context.Context, key string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.objects, key)
	return nil
}

func (m *memStore) PublicURL(key string) string { return memStoreBase + key }

func (m *memStore) KeyFromURL(url string) (string, bool) {
	key, ok := strings.CutPrefix(url, memStoreBase)
	return key, ok && key != ""
}

func (m *memStore) get(url string) []byte {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.objects[strings.TrimPrefix(url, memStoreBase)]
}

func (m *memStore) count() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.objects)
}

// newCommunityApp builds the app with the real image service on a memStore.
func newCommunityApp(t *testing.T) (*app, *memStore) {
	t.Helper()
	store := newMemStore()
	return newApp(t, services.NewImageService(store)), store
}

type filePart struct {
	field       string
	contentType string
	data        []byte
}

func (c *client) multipart(method, path string, fields map[string]string, files ...filePart) response {
	c.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	for i, f := range files {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", fmt.Sprintf(`form-data; name=%q; filename="upload%d.jpg"`, f.field, i))
		h.Set("Content-Type", f.contentType)
		part, _ := mw.CreatePart(h)
		part.Write(f.data)
	}
	mw.Close()
	return c.send(method, path, mw.FormDataContentType(), &buf)
}

func (c *client) sendJSON(method, path string, body any) response {
	c.t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		c.t.Fatal(err)
	}
	return c.send(method, path, "application/json", bytes.NewReader(b))
}

func (c *client) del(path string) response {
	return c.send(http.MethodDelete, path, "", nil)
}

// obj and list read nested JSON without a type assertion at every step.
func obj(t *testing.T, v any, keys ...string) map[string]any {
	t.Helper()
	for _, k := range keys {
		m, ok := v.(map[string]any)
		if !ok {
			t.Fatalf("not an object at %q: %v", k, v)
		}
		v = m[k]
	}
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("not an object at %v: %v", keys, v)
	}
	return m
}

func list(t *testing.T, v map[string]any, key string) []map[string]any {
	t.Helper()
	raw, ok := v[key].([]any)
	if !ok {
		t.Fatalf("%q is not an array: %v", key, v[key])
	}
	out := make([]map[string]any, len(raw))
	for i, item := range raw {
		out[i] = item.(map[string]any)
	}
	return out
}

func wantDetail(t *testing.T, r response, status int, detail string) {
	t.Helper()
	wantStatus(t, r, status)
	if got := r.json(t)["detail"]; got != detail {
		t.Fatalf("detail = %v, want %q", got, detail)
	}
}

func (a *app) addSpot(t *testing.T, name, area, category string, sample bool) string {
	t.Helper()
	s := models.Spot{Name: name, Emoji: "🌲", Area: area, Category: category, Description: name + "の説明", IsSample: sample}
	if err := a.db.Gorm.Create(&s).Error; err != nil {
		t.Fatal(err)
	}
	return s.ID
}

// stubService is an HTTP server standing in for an external API.
type stubService struct {
	*httptest.Server
	calls atomic.Int64
}

func newStubService(t *testing.T, handler http.HandlerFunc) *stubService {
	t.Helper()
	s := &stubService{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		s.calls.Add(1)
		handler(w, r)
	}))
	t.Cleanup(s.Close)
	return s
}

// Every route registered on the auth API, by method and path.
var protectedRoutes = []struct{ method, path string }{
	{http.MethodGet, "/api/auth/me"},
	{http.MethodGet, "/api/posts"},
	{http.MethodGet, "/api/posts/bookmarked"},
	{http.MethodGet, "/api/posts/mine"},
	{http.MethodGet, "/api/posts/{id}"},
	{http.MethodPost, "/api/posts"},
	{http.MethodDelete, "/api/posts/{id}"},
	{http.MethodPost, "/api/posts/{id}/like"},
	{http.MethodPost, "/api/posts/{id}/bookmark"},
	{http.MethodGet, "/api/rankings"},
	{http.MethodGet, "/api/spots/{id}"},
	{http.MethodPost, "/api/spots/{id}/reviews"},
	{http.MethodGet, "/api/gps-visits"},
	{http.MethodGet, "/api/gps-visits/public"},
	{http.MethodPost, "/api/gps-visits"},
	{http.MethodDelete, "/api/gps-visits/{id}"},
	{http.MethodPatch, "/api/gps-visits/{id}/visibility"},
	{http.MethodPatch, "/api/members/me"},
	{http.MethodPost, "/api/translate"},
	{http.MethodPost, "/api/ai/plan"},
	{http.MethodGet, "/api/weather"},
	{http.MethodPost, "/api/places/search"},
	{http.MethodPost, "/api/places/nearby"},
	{http.MethodPost, "/api/places/reverse"},
	{http.MethodPost, "/api/spots"},
}

func TestProtectedRoutesRequireAuthIntegration(t *testing.T) {
	a, store := newCommunityApp(t)
	spotID := a.addSpot(t, "白駒の池", "東信", "自然", true)
	anonymous := a.client(t)

	for _, route := range protectedRoutes {
		path := strings.ReplaceAll(route.path, "{id}", spotID)
		r := anonymous.send(route.method, path, "application/json", strings.NewReader(`{}`))
		if r.Code != http.StatusUnauthorized {
			t.Errorf("%s %s without a session = %d, want 401; body: %s", route.method, route.path, r.Code, r.Body)
			continue
		}
		if r.json(t)["detail"] != "authentication required" {
			t.Errorf("%s %s: body = %s", route.method, route.path, r.Body)
		}
	}
	// A multipart post without a session is refused before anything is stored.
	r := anonymous.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "x"},
		filePart{"media", "image/jpeg", testutil.PlainJPEG(t, 8, 8)})
	wantStatus(t, r, http.StatusUnauthorized)
	if store.count() != 0 {
		t.Fatal("an unauthenticated upload was stored")
	}

	// The list above is complete: it matches the auth API's own OpenAPI.
	member := a.client(t)
	member.signup("Mika")
	r = member.get("/api/openapi/auth.json")
	wantStatus(t, r, http.StatusOK)
	var doc struct {
		Paths map[string]map[string]any `json:"paths"`
	}
	if err := json.Unmarshal(r.Body.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	var documented, listed []string
	for path, ops := range doc.Paths {
		for method := range ops {
			documented = append(documented, strings.ToUpper(method)+" /api"+path)
		}
	}
	for _, route := range protectedRoutes {
		listed = append(listed, route.method+" "+route.path)
	}
	sort.Strings(documented)
	sort.Strings(listed)
	if strings.Join(documented, "\n") != strings.Join(listed, "\n") {
		t.Fatalf("protectedRoutes is out of date.\nregistered:\n%s\n\nlisted:\n%s", strings.Join(documented, "\n"), strings.Join(listed, "\n"))
	}

	// What needs no login: spot data with aggregate counts and the static
	// AI spot list. Neither carries anything a member wrote.
	for _, path := range []string{"/health", "/api/spots", "/api/ai-spots"} {
		wantStatus(t, anonymous.get(path), http.StatusOK)
	}
	wantStatus(t, member.postJSON("/api/spots/"+spotID+"/reviews", map[string]any{"stars": 5, "text": "会員だけの口コミ"}), http.StatusOK)
	wantStatus(t, member.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "会員だけの投稿", "spot_id": spotID}), http.StatusOK)
	for _, path := range []string{"/api/spots", "/api/ai-spots", "/api/spots/" + spotID, "/api/openapi.json"} {
		body := anonymous.get(path).Body.String()
		for _, memberContent := range []string{"会員だけ", "Mika"} {
			if strings.Contains(body, memberContent) {
				t.Errorf("GET %s shows member content (%q) without a session", path, memberContent)
			}
		}
	}
	if spots := list(t, anonymous.get("/api/spots").json(t), "spots"); spots[0]["review_count"] != 1.0 || spots[0]["post_count"] != 1.0 {
		t.Fatalf("public spot list = %v", spots)
	}
}

// By decision there is no endpoint that edits the host / business page.
func TestNoBusinessUpdateEndpointIntegration(t *testing.T) {
	a, _ := newCommunityApp(t)
	c := a.client(t)
	c.signup("Mika")
	for _, method := range []string{http.MethodGet, http.MethodPatch, http.MethodPut, http.MethodPost, http.MethodDelete} {
		for _, path := range []string{"/api/business", "/api/host", "/api/business/me"} {
			if r := c.send(method, path, "application/json", strings.NewReader(`{}`)); r.Code != http.StatusNotFound {
				t.Errorf("%s %s = %d, want 404", method, path, r.Code)
			}
		}
	}
}

// Create with a photo → another member sees it → like → save → both persist
// → rankings and own lists → only the owner can delete → 404 afterwards.
func TestPostFlowIntegration(t *testing.T) {
	a, store := newCommunityApp(t)
	spotID := a.addSpot(t, "白駒の池", "東信", "自然", true)
	alice, bob := a.client(t), a.client(t)
	aliceID, _ := alice.signup("Alice")
	bobID, _ := bob.signup("Bob")

	r := alice.multipart(http.MethodPost, "/api/posts", map[string]string{
		"caption": "朝の苔の森\n静かで気持ちいい", "spot_id": spotID, "language": "ja", "weather": "晴れ",
		"temperature": "13", "visited_at": "2026-10-08", "tags": "自然, 朝さんぽ",
	}, filePart{"media", "image/jpeg", testutil.JPEGWithGPSExif(t, 1600, 800)})
	wantStatus(t, r, http.StatusOK)
	post := obj(t, r.json(t), "post")
	postID := post["id"].(string)
	if post["caption"] != "朝の苔の森\n静かで気持ちいい" || post["weather"] != "晴れ" || post["temperature"] != 13.0 ||
		post["visited_at"] != "2026-10-08" || post["language"] != "ja" || post["area"] != "東信" ||
		post["is_sample"] != false || post["is_mine"] != true || post["like_count"] != 0.0 {
		t.Fatalf("created post = %v", post)
	}
	if tags := fmt.Sprint(post["tags"]); tags != "[自然 朝さんぽ]" {
		t.Fatalf("tags = %s", tags)
	}
	if spot := obj(t, post, "spot"); spot["id"] != spotID || spot["name"] != "白駒の池" || spot["emoji"] != "🌲" || spot["area"] != "東信" {
		t.Fatalf("spot = %v", spot)
	}
	if member := obj(t, post, "member"); member["id"] != aliceID || member["nickname"] != "Alice" {
		t.Fatalf("member = %v", member)
	}
	media := list(t, post, "media")
	if len(media) != 1 || media[0]["type"] != "photo" {
		t.Fatalf("media = %v", media)
	}
	mediaURL := media[0]["url"].(string)

	// The stored object is a JPEG without EXIF, rotated upright and scaled.
	stored := store.get(mediaURL)
	if stored == nil {
		t.Fatalf("no object stored for %s", mediaURL)
	}
	if testutil.HasExif(stored) {
		t.Fatal("the stored photo still carries EXIF")
	}
	if services.DetectImageType(stored) != "image/jpeg" {
		t.Fatal("the stored photo is not a JPEG")
	}

	// Bob sees it in the feed and by id.
	r = bob.get("/api/posts")
	wantStatus(t, r, http.StatusOK)
	feed := list(t, r.json(t), "posts")
	if len(feed) != 1 || feed[0]["id"] != postID || r.json(t)["total"] != 1.0 {
		t.Fatalf("bob's feed = %s", r.Body)
	}
	if feed[0]["is_mine"] != false || feed[0]["is_liked"] != false || feed[0]["is_bookmarked"] != false {
		t.Fatalf("bob's view = %v", feed[0])
	}
	if got := list(t, feed[0], "media")[0]["url"]; got != mediaURL {
		t.Fatalf("media url in the feed = %v", got)
	}

	// Like and save.
	r = bob.postJSON("/api/posts/"+postID+"/like", nil)
	wantStatus(t, r, http.StatusOK)
	if body := r.json(t); body["liked"] != true || body["like_count"] != 1.0 {
		t.Fatalf("like = %s", r.Body)
	}
	r = bob.postJSON("/api/posts/"+postID+"/bookmark", nil)
	wantStatus(t, r, http.StatusOK)
	if r.json(t)["bookmarked"] != true {
		t.Fatalf("bookmark = %s", r.Body)
	}

	// A fresh request shows both persisted, for bob only.
	r = bob.get("/api/posts/" + postID)
	wantStatus(t, r, http.StatusOK)
	if p := obj(t, r.json(t), "post"); p["is_liked"] != true || p["is_bookmarked"] != true || p["like_count"] != 1.0 {
		t.Fatalf("after like and save, bob sees %v", p)
	}
	if p := obj(t, alice.get("/api/posts/"+postID).json(t), "post"); p["is_liked"] != false || p["is_bookmarked"] != false || p["like_count"] != 1.0 {
		t.Fatalf("alice sees %v", p)
	}

	// Saved posts and own posts.
	if saved := list(t, bob.get("/api/posts/bookmarked").json(t), "posts"); len(saved) != 1 || saved[0]["id"] != postID {
		t.Fatalf("bob's saved posts = %v", saved)
	}
	if saved := list(t, alice.get("/api/posts/bookmarked").json(t), "posts"); len(saved) != 0 {
		t.Fatalf("alice's saved posts = %v", saved)
	}
	if mine := list(t, alice.get("/api/posts/mine").json(t), "posts"); len(mine) != 1 || mine[0]["id"] != postID {
		t.Fatalf("alice's posts = %v", mine)
	}
	if mine := list(t, bob.get("/api/posts/mine").json(t), "posts"); len(mine) != 0 {
		t.Fatalf("bob's posts = %v", mine)
	}

	// Rankings: 1 like + 1 save*3 = 4 for the spot, 1 for the post.
	r = bob.get("/api/rankings?type=spots")
	wantStatus(t, r, http.StatusOK)
	if items := list(t, r.json(t), "items"); len(items) != 1 || items[0]["id"] != spotID || items[0]["rank"] != 1.0 ||
		items[0]["score"] != 4.0 || items[0]["name"] != "白駒の池" || items[0]["is_sample"] != true || items[0]["post_count"] != 1.0 {
		t.Fatalf("spot ranking = %s", r.Body)
	}
	r = bob.get("/api/rankings?type=posts")
	wantStatus(t, r, http.StatusOK)
	if items := list(t, r.json(t), "items"); len(items) != 1 || items[0]["id"] != postID || items[0]["score"] != 1.0 ||
		items[0]["name"] != "朝の苔の森" || items[0]["detail"] != "白駒の池 · Alice" {
		t.Fatalf("post ranking = %s", r.Body)
	}
	wantStatus(t, bob.get("/api/rankings?type=members"), http.StatusUnprocessableEntity)

	// The spot page lists the post.
	r = bob.get("/api/spots/" + spotID)
	wantStatus(t, r, http.StatusOK)
	if posts := list(t, r.json(t), "posts"); len(posts) != 1 || posts[0]["id"] != postID || posts[0]["like_count"] != 1.0 {
		t.Fatalf("spot posts = %s", r.Body)
	}

	// Unlike: the toggle goes back.
	r = bob.postJSON("/api/posts/"+postID+"/like", nil)
	if body := r.json(t); body["liked"] != false || body["like_count"] != 0.0 {
		t.Fatalf("unlike = %s", r.Body)
	}
	bob.postJSON("/api/posts/"+postID+"/like", nil)

	// Bob cannot delete Alice's post.
	r = bob.del("/api/posts/" + postID)
	wantStatus(t, r, http.StatusForbidden)
	wantStatus(t, alice.get("/api/posts/"+postID), http.StatusOK)
	if store.get(mediaURL) == nil {
		t.Fatal("the photo was deleted by a refused request")
	}

	// Alice can. Likes, bookmarks, media rows and the stored photo go too.
	r = alice.del("/api/posts/" + postID)
	wantStatus(t, r, http.StatusOK)
	if r.json(t)["ok"] != true {
		t.Fatalf("delete = %s", r.Body)
	}
	wantDetail(t, alice.get("/api/posts/"+postID), http.StatusNotFound, "post not found")
	wantStatus(t, alice.del("/api/posts/"+postID), http.StatusNotFound)
	wantStatus(t, bob.postJSON("/api/posts/"+postID+"/like", nil), http.StatusNotFound)
	wantStatus(t, bob.postJSON("/api/posts/"+postID+"/bookmark", nil), http.StatusNotFound)
	if store.count() != 0 {
		t.Fatalf("%d objects left in storage after the post was deleted", store.count())
	}
	for _, model := range []any{&models.Like{}, &models.Bookmark{}, &models.PostMedia{}, &models.PostTag{}} {
		var n int64
		a.db.Gorm.Model(model).Count(&n)
		if n != 0 {
			t.Fatalf("%T has %d rows left", model, n)
		}
	}
	// Listings that pointed at it still work.
	for _, path := range []string{"/api/posts", "/api/posts/bookmarked", "/api/posts/mine", "/api/rankings?type=posts", "/api/rankings?type=spots", "/api/spots/" + spotID} {
		wantStatus(t, bob.get(path), http.StatusOK)
	}
	if saved := list(t, bob.get("/api/posts/bookmarked").json(t), "posts"); len(saved) != 0 {
		t.Fatalf("bob's saved posts after the delete = %v", saved)
	}
	_ = bobID
}

func TestPostUploadRejectionsIntegration(t *testing.T) {
	a, store := newCommunityApp(t)
	c := a.client(t)
	c.signup("Mika")
	jpeg := testutil.PlainJPEG(t, 8, 8)

	assertNothingCreated := func(name string) {
		t.Helper()
		var posts int64
		a.db.Gorm.Model(&models.Post{}).Count(&posts)
		if posts != 0 || store.count() != 0 {
			t.Fatalf("%s: left %d posts and %d stored objects", name, posts, store.count())
		}
	}

	// Content decides, not the declared type: a script labelled image/jpeg.
	r := c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "x"},
		filePart{"media", "image/jpeg", []byte("#!/bin/sh\necho pwned\n")})
	wantStatus(t, r, http.StatusUnprocessableEntity)
	assertNothingCreated("script as image/jpeg")

	// JPEG magic bytes followed by garbage.
	r = c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "x"},
		filePart{"media", "image/jpeg", append([]byte{0xFF, 0xD8, 0xFF, 0xE0}, []byte("<script>alert(1)</script>")...)})
	wantStatus(t, r, http.StatusUnprocessableEntity)
	assertNothingCreated("fake jpeg")

	// A GIF, honestly labelled and labelled as PNG.
	gif := []byte("GIF89a\x01\x00\x01\x00\x80\x00\x00\xff\xff\xff\x00\x00\x00,\x00\x00\x00\x00\x01\x00\x01\x00\x00\x02\x02D\x01\x00;")
	for _, contentType := range []string{"image/gif", "image/png"} {
		r = c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "x"}, filePart{"media", contentType, gif})
		wantStatus(t, r, http.StatusUnprocessableEntity)
		assertNothingCreated("gif as " + contentType)
	}

	// More than one photo.
	r = c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "x"},
		filePart{"media", "image/jpeg", jpeg}, filePart{"media", "image/jpeg", jpeg})
	wantStatus(t, r, http.StatusUnprocessableEntity)
	if !strings.Contains(r.Body.String(), "only one was expected") {
		t.Fatalf("two photos: body = %s", r.Body)
	}
	assertNothingCreated("two photos")

	// Over 10MB.
	r = c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "x"},
		filePart{"media", "image/jpeg", append(jpeg, make([]byte, 11<<20)...)})
	wantStatus(t, r, http.StatusUnprocessableEntity)
	if !strings.Contains(r.Body.String(), "request body too large") {
		t.Fatalf("11MB upload: body = %s", r.Body)
	}
	assertNothingCreated("11MB upload")

	// Neither caption nor photo, and bad fields.
	wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"weather": "晴れ"}), http.StatusUnprocessableEntity)
	wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "x", "language": "fr"}), http.StatusUnprocessableEntity)
	wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "x", "spot_id": "nope"}), http.StatusUnprocessableEntity)
	wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": strings.Repeat("あ", 2001)}), http.StatusUnprocessableEntity)
	assertNothingCreated("invalid fields")

	// PNG and photo-only posts are accepted and stored as JPEG.
	r = c.multipart(http.MethodPost, "/api/posts", nil, filePart{"media", "image/png", testutil.PlainPNG(t, 8, 8)})
	wantStatus(t, r, http.StatusOK)
	url := list(t, obj(t, r.json(t), "post"), "media")[0]["url"].(string)
	if services.DetectImageType(store.get(url)) != "image/jpeg" {
		t.Fatal("the PNG was not stored as a JPEG")
	}
}

// Sample rows are labelled so the UI can mark them.
func TestSampleFlagIntegration(t *testing.T) {
	a, _ := newCommunityApp(t)
	spotID := a.addSpot(t, "佐久の家", "東信", "宿泊", true)
	realSpot := a.addSpot(t, "新しいカフェ", "東信", "カフェ", false)
	c := a.client(t)
	memberID, _ := c.signup("Mika")

	samplePost := models.Post{MemberID: memberID, SpotID: &spotID, Caption: "サンプル投稿", Language: "ja", IsSample: true}
	sampleReview := models.Review{MemberID: memberID, SpotID: spotID, Stars: 5, Text: "サンプル口コミ", IsSample: true}
	if err := a.db.Gorm.Omit("Member", "Spot", "Media", "Tags").Create(&samplePost).Error; err != nil {
		t.Fatal(err)
	}
	if err := a.db.Gorm.Omit("Member").Create(&sampleReview).Error; err != nil {
		t.Fatal(err)
	}
	wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "本物の投稿", "spot_id": realSpot}), http.StatusOK)

	flags := map[string]any{}
	for _, p := range list(t, c.get("/api/posts").json(t), "posts") {
		flags[p["caption"].(string)] = p["is_sample"]
	}
	if flags["サンプル投稿"] != true || flags["本物の投稿"] != false {
		t.Fatalf("post flags = %v", flags)
	}
	for _, s := range list(t, c.get("/api/spots").json(t), "spots") {
		if want := s["id"] == spotID; s["is_sample"] != want {
			t.Fatalf("spot %v is_sample = %v", s["name"], s["is_sample"])
		}
	}
	detail := c.get("/api/spots/" + spotID).json(t)
	if obj(t, detail, "spot")["is_sample"] != true || list(t, detail, "reviews")[0]["is_sample"] != true || list(t, detail, "posts")[0]["is_sample"] != true {
		t.Fatalf("spot detail flags = %v", detail)
	}
	// A member's own review is not a sample.
	r := c.postJSON("/api/spots/"+spotID+"/reviews", map[string]any{"stars": 4, "text": "よかった"})
	wantStatus(t, r, http.StatusOK)
	if obj(t, r.json(t), "review")["is_sample"] != false {
		t.Fatalf("review = %s", r.Body)
	}
	if items := list(t, c.get("/api/rankings?type=posts").json(t), "items"); len(items) != 2 {
		t.Fatalf("post ranking = %v", items)
	}
}

func TestReviewsIntegration(t *testing.T) {
	a, _ := newCommunityApp(t)
	spotID := a.addSpot(t, "松本城", "中信", "歴史", false)
	alice, bob := a.client(t), a.client(t)
	aliceID, _ := alice.signup("Alice")
	bob.signup("Bob")

	r := alice.postJSON("/api/spots/"+spotID+"/reviews", map[string]any{"stars": 5, "text": "  城下町も楽しい  "})
	wantStatus(t, r, http.StatusOK)
	review := obj(t, r.json(t), "review")
	if review["stars"] != 5.0 || review["text"] != "城下町も楽しい" || review["id"] == "" || review["created_at"] == nil ||
		obj(t, review, "member")["id"] != aliceID || obj(t, review, "member")["nickname"] != "Alice" {
		t.Fatalf("review = %v", review)
	}
	wantStatus(t, bob.postJSON("/api/spots/"+spotID+"/reviews", map[string]any{"stars": 2, "text": "混んでいた"}), http.StatusOK)

	// Persisted: a fresh request shows both, newest first.
	r = bob.get("/api/spots/" + spotID)
	wantStatus(t, r, http.StatusOK)
	reviews := list(t, r.json(t), "reviews")
	if len(reviews) != 2 || reviews[0]["text"] != "混んでいた" || reviews[1]["text"] != "城下町も楽しい" {
		t.Fatalf("reviews = %v", reviews)
	}
	spots := list(t, a.client(t).get("/api/spots?area=中信").json(t), "spots")
	if len(spots) != 1 || spots[0]["review_count"] != 2.0 || spots[0]["avg_stars"] != 3.5 || spots[0]["category"] != "歴史" || spots[0]["description"] != "松本城の説明" {
		t.Fatalf("spot list = %v", spots)
	}
	if other := list(t, a.client(t).get("/api/spots?area=北信").json(t), "spots"); len(other) != 0 {
		t.Fatalf("area filter = %v", other)
	}

	for name, body := range map[string]map[string]any{
		"no stars":  {"text": "x"},
		"0 stars":   {"stars": 0, "text": "x"},
		"6 stars":   {"stars": 6, "text": "x"},
		"no text":   {"stars": 5, "text": ""},
		"blank":     {"stars": 5, "text": "   "},
		"long text": {"stars": 5, "text": strings.Repeat("あ", 1001)},
	} {
		if r := alice.postJSON("/api/spots/"+spotID+"/reviews", body); r.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s = %d, want 422", name, r.Code)
		}
	}
	wantDetail(t, alice.postJSON("/api/spots/no-such-spot/reviews", map[string]any{"stars": 5, "text": "x"}), http.StatusNotFound, "spot not found")
	wantDetail(t, alice.get("/api/spots/no-such-spot"), http.StatusNotFound, "spot not found")
	wantStatus(t, a.client(t).get("/api/spots/"+spotID), http.StatusUnauthorized)
}

func TestAiSpotsIntegration(t *testing.T) {
	a, _ := newCommunityApp(t)
	rows := []models.AiSpot{
		{Name: "遠い森", Purposes: `["hiking","scenic"]`, Area: "佐久穂町", Kind: "森", DriveMin: 65, TransitMin: 120, StayMin: 120, Genres: `["自然・絶景","穴場スポット"]`, Weathers: `["sunny"]`, Description: "d", MapQuery: "q", SearchURL: "https://example.test/1"},
		{Name: "近いカフェ", Purposes: `["rainyday","family"]`, Area: "佐久市", Kind: "カフェ", DriveMin: 15, TransitMin: 40, StayMin: 45, Genres: `["穴場スポット","カフェ・グルメ"]`, Weathers: `["any","sunny","rainy"]`, Description: "d", MapQuery: "q", SearchURL: "https://example.test/2"},
		{Name: "美術館", Purposes: `["rainyday","japan"]`, Area: "佐久市", Kind: "屋内", DriveMin: 20, TransitMin: 30, StayMin: 60, Genres: `["歴史・文化"]`, Weathers: `["any","rainy"]`, Description: "d", MapQuery: "q", SearchURL: "https://example.test/3"},
		{Name: "温泉", Purposes: `["onsen"]`, Area: "軽井沢町", Kind: "温泉", DriveMin: 40, TransitMin: 70, StayMin: 100, Genres: `["自然・絶景"]`, Weathers: `["any","sunny","rainy"]`, Description: "d", MapQuery: "q", SearchURL: "https://example.test/4"},
	}
	for i := range rows {
		if err := a.db.Gorm.Create(&rows[i]).Error; err != nil {
			t.Fatal(err)
		}
	}
	c := a.client(t) // no login needed
	names := func(query string) string {
		t.Helper()
		r := c.get("/api/ai-spots" + query)
		wantStatus(t, r, http.StatusOK)
		var out []string
		for _, s := range list(t, r.json(t), "spots") {
			out = append(out, s["name"].(string))
		}
		return strings.Join(out, ",")
	}

	// Defaults: every genre, by car, half a day → the three nearest.
	if got := names(""); got != "近いカフェ,美術館,温泉" {
		t.Fatalf("default = %s", got)
	}
	if got := names("?duration=full"); got != "近いカフェ,美術館,温泉,遠い森" {
		t.Fatalf("full day = %s", got)
	}
	if got := names("?duration=short&transport=transit"); got != "美術館,近いカフェ" {
		t.Fatalf("short by transit = %s", got)
	}
	if got := names("?purpose=rainyday&weather=rainy"); got != "近いカフェ,美術館" {
		t.Fatalf("rainy day = %s", got)
	}
	if got := names("?genre=" + "歴史・文化"); got != "美術館" {
		t.Fatalf("genre = %s", got)
	}
	// Nothing matches genre + purpose: the genre is dropped, as in the prototype.
	if got := names("?purpose=onsen&genre=カフェ・グルメ"); got != "温泉" {
		t.Fatalf("relaxed genre = %s", got)
	}
	// Nothing matches purpose + weather either: only the purpose is kept.
	if got := names("?purpose=hiking&weather=rainy"); got != "遠い森" {
		t.Fatalf("relaxed weather = %s", got)
	}
	wantStatus(t, c.get("/api/ai-spots?purpose=shopping"), http.StatusUnprocessableEntity)

	spot := list(t, c.get("/api/ai-spots?duration=short").json(t), "spots")[0]
	for _, key := range []string{"id", "name", "area", "kind", "drive_min", "transit_min", "stay_min", "description", "map_query", "search_url"} {
		if _, ok := spot[key]; !ok {
			t.Errorf("ai spot is missing %q", key)
		}
	}
	for _, internal := range []string{"purposes", "genres", "weathers"} {
		if _, ok := spot[internal]; ok {
			t.Errorf("ai spot exposes %q", internal)
		}
	}
}

// Coordinates that must never reach another member, and a pair that may.
const (
	secretLat, secretLon = 35.123456, 139.654321
	sharedLat, sharedLon = 36.271828, 138.314159
)

func containsCoordinate(body string, values ...float64) bool {
	for _, v := range values {
		full := fmt.Sprintf("%.6f", v)
		// Also catch a rounded or truncated form.
		if strings.Contains(body, full) || strings.Contains(body, full[:len(full)-3]) {
			return true
		}
	}
	return false
}

// Another member's visit: invisible unless public, and without coordinates
// unless its owner published the exact position.
func TestGpsVisitPrivacyIntegration(t *testing.T) {
	a, _ := newCommunityApp(t)
	spotID := a.addSpot(t, "白駒の池", "東信", "自然", true)
	alice, bob := a.client(t), a.client(t)
	aliceID, _ := alice.signup("Alice")
	bob.signup("Bob")

	create := func(body map[string]any) map[string]any {
		t.Helper()
		r := alice.postJSON("/api/gps-visits", body)
		wantStatus(t, r, http.StatusOK)
		return obj(t, r.json(t), "visit")
	}
	// Public, exact position left at its default: not public.
	plain := create(map[string]any{"spot_id": spotID, "visited_date": "2026-10-08", "weather": "晴れ", "temperature": 13,
		"note": "苔がきれい", "is_public": true, "latitude": secretLat, "longitude": secretLon})
	if plain["is_public"] != true || plain["exact_public"] != false {
		t.Fatalf("public visit = %v", plain)
	}
	if plain["latitude"] != secretLat || plain["longitude"] != secretLon || plain["is_mine"] != true ||
		plain["place_name"] != "白駒の池" || plain["visited_date"] != "2026-10-08" || plain["spot_id"] != spotID {
		t.Fatalf("alice's own visit = %v", plain)
	}
	private := create(map[string]any{"place_name": "自宅のそば", "visited_date": "2026-10-07", "note": "ないしょ",
		"is_public": false, "exact_public": true, "latitude": secretLat, "longitude": secretLon})
	exact := create(map[string]any{"place_name": "展望台", "visited_date": "2026-10-06",
		"is_public": true, "exact_public": true, "latitude": sharedLat, "longitude": sharedLon})
	// A visit recorded after the browser refused the position.
	noPosition := create(map[string]any{"place_name": "カフェ", "visited_date": "2026-10-05", "is_public": true, "exact_public": false})
	if noPosition["latitude"] != nil || noPosition["longitude"] != nil {
		t.Fatalf("visit without a position = %v", noPosition)
	}

	// Alice always sees her own, all four, with coordinates.
	mine := list(t, alice.get("/api/gps-visits").json(t), "visits")
	if len(mine) != 4 {
		t.Fatalf("alice's visits = %d, want 4", len(mine))
	}
	for _, v := range mine {
		if v["id"] != noPosition["id"] && (v["latitude"] == nil || v["longitude"] == nil) {
			t.Fatalf("alice's own visit has no coordinates: %v", v)
		}
	}

	// Bob's own list is empty; his public list has three of Alice's.
	if got := list(t, bob.get("/api/gps-visits").json(t), "visits"); len(got) != 0 {
		t.Fatalf("bob's own visits = %v", got)
	}
	r := bob.get("/api/gps-visits/public")
	wantStatus(t, r, http.StatusOK)
	seen := map[any]map[string]any{}
	for _, v := range list(t, r.json(t), "visits") {
		seen[v["id"]] = v
	}
	if len(seen) != 3 {
		t.Fatalf("bob sees %d public visits, want 3: %s", len(seen), r.Body)
	}
	if _, ok := seen[private["id"]]; ok {
		t.Fatal("alice's non-public visit is visible to bob")
	}
	if strings.Contains(r.Body.String(), "ないしょ") || strings.Contains(r.Body.String(), "自宅のそば") {
		t.Fatal("text of the non-public visit is visible to bob")
	}
	v := seen[plain["id"]]
	if lat, ok := v["latitude"]; !ok || lat != nil {
		t.Fatalf("latitude of a visit without exact_public = %v (present: %v), want null", lat, ok)
	}
	if lon, ok := v["longitude"]; !ok || lon != nil {
		t.Fatalf("longitude of a visit without exact_public = %v, want null", lon)
	}
	if v["is_mine"] != false || v["place_name"] != "白駒の池" || v["note"] != "苔がきれい" || obj(t, v, "member")["id"] != aliceID {
		t.Fatalf("public visit = %v", v)
	}
	if v := seen[exact["id"]]; v["latitude"] != sharedLat || v["longitude"] != sharedLon {
		t.Fatalf("exact_public visit = %v", v)
	}

	// No endpoint hands the secret position to Bob or to an anonymous
	// visitor: feed, lists, rankings, spots, profile.
	post := alice.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "森", "spot_id": spotID})
	wantStatus(t, post, http.StatusOK)
	postID := obj(t, post.json(t), "post")["id"].(string)
	wantStatus(t, bob.postJSON("/api/posts/"+postID+"/bookmark", nil), http.StatusOK)
	for _, viewer := range []*client{bob, a.client(t)} {
		for _, path := range []string{
			"/api/gps-visits", "/api/gps-visits/public", "/api/posts", "/api/posts/" + postID, "/api/posts/mine",
			"/api/posts/bookmarked", "/api/rankings?type=spots", "/api/rankings?type=posts", "/api/rankings?type=spots&limit=20",
			"/api/spots", "/api/spots/" + spotID, "/api/ai-spots", "/api/auth/me",
		} {
			body := viewer.get(path).Body.String()
			if containsCoordinate(body, secretLat, secretLon) {
				t.Errorf("GET %s leaks the secret coordinates: %s", path, body)
			}
			if strings.Contains(body, "ないしょ") {
				t.Errorf("GET %s leaks the private note", path)
			}
		}
	}
	// The ranking counts the public visits at the spot (1) and nothing of
	// the private one: 1 visit + 1 save*3.
	if items := list(t, bob.get("/api/rankings?type=spots").json(t), "items"); len(items) != 1 || items[0]["score"] != 4.0 {
		t.Fatalf("spot ranking = %v", items)
	}

	// Bob cannot delete or re-publish Alice's visits.
	wantStatus(t, bob.del("/api/gps-visits/"+plain["id"].(string)), http.StatusForbidden)
	wantStatus(t, bob.del("/api/gps-visits/"+private["id"].(string)), http.StatusNotFound)
	patch := map[string]any{"is_public": true, "exact_public": true}
	wantStatus(t, bob.sendJSON(http.MethodPatch, "/api/gps-visits/"+plain["id"].(string)+"/visibility", patch), http.StatusForbidden)
	r = bob.sendJSON(http.MethodPatch, "/api/gps-visits/"+private["id"].(string)+"/visibility", patch)
	wantStatus(t, r, http.StatusNotFound)
	if containsCoordinate(r.Body.String(), secretLat, secretLon) {
		t.Fatal("a refused update returned coordinates")
	}
	if containsCoordinate(bob.get("/api/gps-visits/public").Body.String(), secretLat, secretLon) {
		t.Fatal("bob's refused updates exposed the coordinates")
	}
	if got := list(t, alice.get("/api/gps-visits").json(t), "visits"); len(got) != 4 {
		t.Fatalf("alice has %d visits after bob's attempts", len(got))
	}

	// Alice hides the exact one and opts in on the first.
	r = alice.sendJSON(http.MethodPatch, "/api/gps-visits/"+exact["id"].(string)+"/visibility", map[string]any{"is_public": false})
	wantStatus(t, r, http.StatusOK)
	if v := obj(t, r.json(t), "visit"); v["is_public"] != false || v["exact_public"] != true || v["id"] != exact["id"] {
		t.Fatalf("visibility update = %v", v)
	}
	if containsCoordinate(bob.get("/api/gps-visits/public").Body.String(), sharedLat, sharedLon) {
		t.Fatal("a visit made non-public still shows its coordinates")
	}
	wantStatus(t, alice.sendJSON(http.MethodPatch, "/api/gps-visits/"+plain["id"].(string)+"/visibility", patch), http.StatusOK)
	if !containsCoordinate(bob.get("/api/gps-visits/public").Body.String(), secretLat) {
		t.Fatal("the position is not shown after its owner published it")
	}

	wantStatus(t, alice.del("/api/gps-visits/"+private["id"].(string)), http.StatusOK)
	wantStatus(t, alice.del("/api/gps-visits/"+private["id"].(string)), http.StatusNotFound)
	if got := list(t, alice.get("/api/gps-visits").json(t), "visits"); len(got) != 3 {
		t.Fatalf("alice has %d visits after deleting one", len(got))
	}

	for name, body := range map[string]map[string]any{
		"no date":        {"place_name": "x"},
		"no place":       {"visited_date": "2026-10-08"},
		"latitude only":  {"place_name": "x", "visited_date": "2026-10-08", "latitude": 35.0},
		"bad latitude":   {"place_name": "x", "visited_date": "2026-10-08", "latitude": 95.0, "longitude": 139.0},
		"bad date":       {"place_name": "x", "visited_date": "2026-13-40"},
		"unknown spot":   {"spot_id": "nope", "visited_date": "2026-10-08"},
		"unknown fields": {"place_name": "x", "visited_date": "2026-10-08", "member_id": aliceID},
	} {
		if r := bob.postJSON("/api/gps-visits", body); r.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s = %d, want 422; body: %s", name, r.Code, r.Body)
		}
	}
}

func TestProfileUpdateIntegration(t *testing.T) {
	a, store := newCommunityApp(t)
	alice, bob := a.client(t), a.client(t)
	aliceID, _ := alice.signup("Alice")
	bobID, _ := bob.signup("Bob")

	// Signup with an avatar now stores a sanitised image.
	carol := a.client(t)
	wantStatus(t, carol.verifyInvite(inviteCode), http.StatusOK)
	r := carol.postSignup(map[string]string{"nickname": "Carol"}, &formFile{contentType: "image/jpeg", data: testutil.JPEGWithGPSExif(t, 900, 600)})
	wantStatus(t, r, http.StatusOK)
	carolAvatar := obj(t, r.json(t), "member")["avatar_url"].(string)
	if stored := store.get(carolAvatar); stored == nil || testutil.HasExif(stored) {
		t.Fatalf("signup avatar missing or still has EXIF (%s)", carolAvatar)
	}

	// Nickname only.
	r = alice.multipart(http.MethodPatch, "/api/members/me", map[string]string{"nickname": "  旅するAlice  "})
	wantStatus(t, r, http.StatusOK)
	if m := obj(t, r.json(t), "member"); m["id"] != aliceID || m["nickname"] != "旅するAlice" || m["preferred_language"] != "ja" || m["avatar_url"] != "" {
		t.Fatalf("after nickname update = %v", m)
	}
	// Language only.
	r = alice.multipart(http.MethodPatch, "/api/members/me", map[string]string{"language": "ko"})
	wantStatus(t, r, http.StatusOK)
	if m := obj(t, r.json(t), "member"); m["nickname"] != "旅するAlice" || m["preferred_language"] != "ko" {
		t.Fatalf("after language update = %v", m)
	}
	// Avatar, then a replacement: the first object is removed.
	r = alice.multipart(http.MethodPatch, "/api/members/me", nil, filePart{"avatar", "image/jpeg", testutil.JPEGWithGPSExif(t, 1200, 800)})
	wantStatus(t, r, http.StatusOK)
	first := obj(t, r.json(t), "member")["avatar_url"].(string)
	if stored := store.get(first); stored == nil || testutil.HasExif(stored) {
		t.Fatal("avatar missing or still has EXIF")
	}
	r = alice.multipart(http.MethodPatch, "/api/members/me", map[string]string{"nickname": "Alice2"}, filePart{"avatar", "image/png", testutil.PlainPNG(t, 64, 64)})
	wantStatus(t, r, http.StatusOK)
	second := obj(t, r.json(t), "member")["avatar_url"].(string)
	if second == first || store.get(second) == nil || store.get(first) != nil {
		t.Fatalf("avatar replacement: first %q still stored = %v", first, store.get(first) != nil)
	}

	// Persisted, and shown next to her content.
	if m := obj(t, alice.get("/api/auth/me").json(t), "member"); m["nickname"] != "Alice2" || m["preferred_language"] != "ko" || m["avatar_url"] != second {
		t.Fatalf("me = %v", m)
	}
	wantStatus(t, alice.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "hi"}), http.StatusOK)
	if m := obj(t, list(t, bob.get("/api/posts").json(t), "posts")[0], "member"); m["nickname"] != "Alice2" || m["avatar_url"] != second {
		t.Fatalf("post author = %v", m)
	}

	// Rejected input changes nothing and stores nothing.
	objects := store.count()
	for name, r := range map[string]response{
		"long nickname": alice.multipart(http.MethodPatch, "/api/members/me", map[string]string{"nickname": strings.Repeat("あ", 31)}),
		"language":      alice.multipart(http.MethodPatch, "/api/members/me", map[string]string{"language": "fr"}),
		"not an image":  alice.multipart(http.MethodPatch, "/api/members/me", map[string]string{"nickname": "Changed"}, filePart{"avatar", "image/jpeg", []byte("#!/bin/sh")}),
	} {
		if r.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s = %d, want 422; body: %s", name, r.Code, r.Body)
		}
	}
	if m := obj(t, alice.get("/api/auth/me").json(t), "member"); m["nickname"] != "Alice2" || store.count() != objects {
		t.Fatalf("a rejected update changed the profile: %v", m)
	}

	// There is no way to address another member: extra fields are ignored
	// and Bob is untouched.
	wantStatus(t, alice.multipart(http.MethodPatch, "/api/members/me", map[string]string{"nickname": "Hacked", "id": bobID, "member_id": bobID}), http.StatusOK)
	if m := obj(t, bob.get("/api/auth/me").json(t), "member"); m["id"] != bobID || m["nickname"] != "Bob" {
		t.Fatalf("bob after alice's update = %v", m)
	}
	for _, path := range []string{"/api/members/" + bobID, "/api/members"} {
		if r := alice.multipart(http.MethodPatch, path, map[string]string{"nickname": "Hacked"}); r.Code != http.StatusNotFound {
			t.Errorf("PATCH %s = %d, want 404", path, r.Code)
		}
	}
	// An empty update is a no-op.
	wantStatus(t, bob.multipart(http.MethodPatch, "/api/members/me", nil), http.StatusOK)
}

func TestTranslateIntegration(t *testing.T) {
	const upstreamText = "MYMEMORY WARNING: QUOTA"
	fail := atomic.Bool{}
	mymemory := newStubService(t, func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() {
			w.Write([]byte(`{"responseData":{"translatedText":"` + upstreamText + `"},"responseStatus":429}`))
			return
		}
		w.Write([]byte(`{"responseData":{"translatedText":"A quiet forest"},"responseStatus":200}`))
	})
	a := newAppWith(t, &routes.Deps{
		Config:   &config.Config{TranslationProvider: "mymemory"},
		External: services.ExternalEndpoints{MyMemoryBaseURL: mymemory.URL},
	})
	c := a.client(t)
	c.signup("Mika")
	body := map[string]string{"text": "静かな森", "source_lang": "ja", "target_lang": "en"}

	r := c.postJSON("/api/translate", body)
	wantStatus(t, r, http.StatusOK)
	if got := r.json(t); got["translated_text"] != "A quiet forest" || got["provider"] != "mymemory" || got["cached"] != false || got["source_lang"] != "ja" {
		t.Fatalf("first = %s", r.Body)
	}
	// The same request again is served from the cache without a call out.
	r = c.postJSON("/api/translate", body)
	wantStatus(t, r, http.StatusOK)
	if got := r.json(t); got["translated_text"] != "A quiet forest" || got["provider"] != "mymemory" || got["cached"] != true {
		t.Fatalf("second = %s", r.Body)
	}
	if mymemory.calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want 1", mymemory.calls.Load())
	}

	// Same source and target: the text itself, no call.
	r = c.postJSON("/api/translate", map[string]string{"text": "Hello", "source_lang": "en", "target_lang": "en"})
	wantStatus(t, r, http.StatusOK)
	if got := r.json(t); got["translated_text"] != "Hello" || got["provider"] != "none" || got["cached"] != false {
		t.Fatalf("same language = %s", r.Body)
	}
	// "auto" detects Japanese and finds the cached entry.
	r = c.postJSON("/api/translate", map[string]string{"text": "静かな森", "source_lang": "auto", "target_lang": "en"})
	wantStatus(t, r, http.StatusOK)
	if got := r.json(t); got["cached"] != true || got["source_lang"] != "ja" {
		t.Fatalf("auto = %s", r.Body)
	}
	if mymemory.calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want 1", mymemory.calls.Load())
	}

	// Provider error: 502 with a fixed message.
	fail.Store(true)
	r = c.postJSON("/api/translate", map[string]string{"text": "別の文章", "source_lang": "ja", "target_lang": "en"})
	wantDetail(t, r, http.StatusBadGateway, "translation failed")
	for _, leak := range []string{upstreamText, "MYMEMORY", mymemory.URL, "127.0.0.1", "別の文章"} {
		if strings.Contains(r.Body.String(), leak) {
			t.Fatalf("502 body leaks %q: %s", leak, r.Body)
		}
	}
	// The failure was not cached.
	fail.Store(false)
	r = c.postJSON("/api/translate", map[string]string{"text": "別の文章", "source_lang": "ja", "target_lang": "en"})
	wantStatus(t, r, http.StatusOK)
	if r.json(t)["cached"] != false {
		t.Fatalf("after a failure = %s", r.Body)
	}

	for name, body := range map[string]map[string]string{
		"empty text":  {"text": "", "source_lang": "ja", "target_lang": "en"},
		"long text":   {"text": strings.Repeat("あ", 2001), "source_lang": "ja", "target_lang": "en"},
		"bad source":  {"text": "x", "source_lang": "fr", "target_lang": "en"},
		"bad target":  {"text": "x", "source_lang": "ja", "target_lang": "auto"},
		"no language": {"text": "x"},
	} {
		if r := c.postJSON("/api/translate", body); r.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s = %d, want 422", name, r.Code)
		}
	}
}

func TestTranslateUnavailableIntegration(t *testing.T) {
	openrouter := newStubService(t, func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte(`{"model":"vendor/m","choices":[{"message":{"content":"A quiet forest"}}]}`))
	})
	body := map[string]string{"text": "静かな森", "source_lang": "ja", "target_lang": "en"}

	// No provider, an unknown one, and openrouter without a key: 503.
	for _, cfg := range []config.Config{
		{TranslationProvider: ""},
		{TranslationProvider: "deepl"},
		{TranslationProvider: "openrouter", OpenRouterAPIKey: ""},
	} {
		a := newAppWith(t, &routes.Deps{Config: &cfg, External: services.ExternalEndpoints{OpenRouterBaseURL: openrouter.URL}})
		c := a.client(t)
		c.signup("Mika")
		wantDetail(t, c.postJSON("/api/translate", body), http.StatusServiceUnavailable, "translation service unavailable")
		// The rest of the API is unaffected.
		wantStatus(t, c.get("/api/posts"), http.StatusOK)
	}
	if openrouter.calls.Load() != 0 {
		t.Fatalf("an unconfigured provider was called %d times", openrouter.calls.Load())
	}

	// TRANSLATION_PROVIDER=openrouter with a key translates through the model.
	a := newAppWith(t, &routes.Deps{
		Config:   &config.Config{TranslationProvider: "openrouter", OpenRouterAPIKey: "sk-or-test", OpenRouterModel: "vendor/m"},
		External: services.ExternalEndpoints{OpenRouterBaseURL: openrouter.URL},
	})
	c := a.client(t)
	c.signup("Mika")
	r := c.postJSON("/api/translate", body)
	wantStatus(t, r, http.StatusOK)
	if got := r.json(t); got["translated_text"] != "A quiet forest" || got["provider"] != "openrouter" || got["cached"] != false {
		t.Fatalf("openrouter translation = %s", r.Body)
	}
	if got := c.postJSON("/api/translate", body).json(t); got["cached"] != true || got["provider"] != "openrouter" {
		t.Fatalf("second = %v", got)
	}
	if openrouter.calls.Load() != 1 {
		t.Fatalf("provider calls = %d, want 1", openrouter.calls.Load())
	}
}

func TestAIPlanIntegration(t *testing.T) {
	const apiKey = "sk-or-test-SECRETKEY123"
	const upstreamText = "No auth credentials found"
	fail := atomic.Bool{}
	var lastBody atomic.Value
	openrouter := newStubService(t, func(w http.ResponseWriter, r *http.Request) {
		b, _ := io.ReadAll(r.Body)
		lastBody.Store(string(b))
		if fail.Load() {
			http.Error(w, `{"error":{"message":"`+upstreamText+`"}}`, http.StatusUnauthorized)
			return
		}
		w.Write([]byte(`{"model":"vendor/model-used","choices":[{"message":{"content":"まず龍岡城五稜郭へ。営業時間は公式情報で確認してください。"}}]}`))
	})
	request := map[string]any{"message": "半日でおすすめは？", "language": "ja", "purpose": "scenic", "weather": "sunny", "duration": "half", "transport": "car"}

	// Key unset: 503 at once, nothing leaves the server.
	a := newAppWith(t, &routes.Deps{
		Config:   &config.Config{OpenRouterAPIKey: "", OpenRouterModel: "vendor/model"},
		External: services.ExternalEndpoints{OpenRouterBaseURL: openrouter.URL},
	})
	c := a.client(t)
	c.signup("Mika")
	wantDetail(t, c.postJSON("/api/ai/plan", request), http.StatusServiceUnavailable, "AI service not configured")
	if openrouter.calls.Load() != 0 {
		t.Fatalf("%d outbound calls with no API key", openrouter.calls.Load())
	}
	wantStatus(t, c.get("/api/posts"), http.StatusOK)

	// Key set.
	a = newAppWith(t, &routes.Deps{
		Config:   &config.Config{OpenRouterAPIKey: apiKey, OpenRouterModel: "vendor/model"},
		External: services.ExternalEndpoints{OpenRouterBaseURL: openrouter.URL},
	})
	c = a.client(t)
	c.signup("Mika")
	r := c.postJSON("/api/ai/plan", request)
	wantStatus(t, r, http.StatusOK)
	if got := r.json(t); got["model"] != "vendor/model-used" || !strings.Contains(got["answer"].(string), "龍岡城五稜郭") {
		t.Fatalf("answer = %s", r.Body)
	}
	if strings.Contains(r.Body.String(), apiKey) {
		t.Fatal("API key in the response")
	}
	var sent struct {
		Model     string `json:"model"`
		MaxTokens int    `json:"max_tokens"`
		Messages  []struct{ Role, Content string }
	}
	if err := json.Unmarshal([]byte(lastBody.Load().(string)), &sent); err != nil {
		t.Fatal(err)
	}
	if sent.Model != "vendor/model" || sent.MaxTokens <= 0 || len(sent.Messages) != 2 ||
		strings.Contains(sent.Messages[0].Content, "半日でおすすめ") || !strings.Contains(sent.Messages[1].Content, "半日でおすすめは？") {
		t.Fatalf("request to the provider = %+v", sent)
	}

	// Validation happens before any call. (A new member each time: every
	// request, valid or not, counts towards the 10 per minute.)
	c = a.client(t)
	c.signup("Validator")
	calls := openrouter.calls.Load()
	for name, body := range map[string]map[string]any{
		"no message":    {"language": "ja"},
		"empty message": {"message": "", "language": "ja"},
		"3001 chars":    {"message": strings.Repeat("あ", 3001), "language": "ja"},
		"no language":   {"message": "hi"},
		"bad language":  {"message": "hi", "language": "fr"},
		"bad purpose":   {"message": "hi", "language": "ja", "purpose": "shopping"},
		"bad weather":   {"message": "hi", "language": "ja", "weather": "snowy"},
		"bad duration":  {"message": "hi", "language": "ja", "duration": "week"},
		"bad transport": {"message": "hi", "language": "ja", "transport": "plane"},
		"long genre":    {"message": "hi", "language": "ja", "genre": strings.Repeat("g", 31)},
	} {
		if r := c.postJSON("/api/ai/plan", body); r.Code != http.StatusUnprocessableEntity {
			t.Errorf("%s = %d, want 422", name, r.Code)
		}
	}
	if openrouter.calls.Load() != calls {
		t.Fatal("an invalid request reached the provider")
	}
	c = a.client(t)
	c.signup("Mika2")
	wantStatus(t, c.postJSON("/api/ai/plan", map[string]any{"message": strings.Repeat("あ", 3000), "language": "ja"}), http.StatusOK)

	// Provider error: 502, fixed message, nothing of the upstream in it.
	fail.Store(true)
	r = c.postJSON("/api/ai/plan", request)
	wantDetail(t, r, http.StatusBadGateway, "AI service temporarily unavailable")
	for _, leak := range []string{upstreamText, apiKey, openrouter.URL, "127.0.0.1", "openrouter"} {
		if strings.Contains(r.Body.String(), leak) {
			t.Fatalf("502 body leaks %q: %s", leak, r.Body)
		}
	}
}

func TestWeatherIntegration(t *testing.T) {
	fail := atomic.Bool{}
	var lastQuery atomic.Value
	openMeteo := newStubService(t, func(w http.ResponseWriter, r *http.Request) {
		lastQuery.Store(r.URL.RawQuery)
		if fail.Load() {
			http.Error(w, `{"error":true,"reason":"upstream detail"}`, http.StatusInternalServerError)
			return
		}
		w.Write([]byte(`{"current":{"temperature_2m":13.4,"weather_code":2}}`))
	})
	a := newAppWith(t, &routes.Deps{External: services.ExternalEndpoints{OpenMeteoBaseURL: openMeteo.URL}})
	c := a.client(t)
	c.signup("Mika")

	// Without a position: Saku City.
	r := c.get("/api/weather")
	wantStatus(t, r, http.StatusOK)
	if got := r.json(t); got["weather"] != "晴れ時々くもり" || got["temperature"] != 13.0 || got["icon"] != "🌤️" || got["code"] != 2.0 || got["source"] != "open-meteo" {
		t.Fatalf("weather = %s", r.Body)
	}
	if q := lastQuery.Load().(string); !strings.Contains(q, "latitude=36.25") || !strings.Contains(q, "longitude=138.48") {
		t.Fatalf("default position query = %s", q)
	}
	wantStatus(t, c.get("/api/weather?latitude=35.6812&longitude=139.7671"), http.StatusOK)
	if q := lastQuery.Load().(string); !strings.Contains(q, "latitude=35.68") || strings.Contains(q, "35.6812") {
		t.Fatalf("the exact position was sent upstream: %s", q)
	}

	for _, query := range []string{"?latitude=91&longitude=0", "?latitude=0&longitude=181", "?latitude=abc&longitude=1"} {
		if r := c.get("/api/weather" + query); r.Code != http.StatusUnprocessableEntity {
			t.Errorf("GET /api/weather%s = %d, want 422", query, r.Code)
		}
	}

	fail.Store(true)
	r = c.get("/api/weather?latitude=34.5&longitude=135.5")
	wantDetail(t, r, http.StatusBadGateway, "weather service temporarily unavailable")
	for _, leak := range []string{"upstream detail", openMeteo.URL, "127.0.0.1", "34.5", "open-meteo"} {
		if strings.Contains(r.Body.String(), leak) {
			t.Fatalf("502 body leaks %q: %s", leak, r.Body)
		}
	}
	// Posting still works with the weather service down.
	wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "天気なしで投稿"}), http.StatusOK)
}

// The AI, translation and post limits are counted per member.
func TestMemberRateLimitsIntegration(t *testing.T) {
	a := newAppWith(t, &routes.Deps{Config: &config.Config{TranslationProvider: "mymemory"}})
	alice, bob := a.client(t), a.client(t)
	alice.signup("Alice")
	bob.signup("Bob")

	// Same-language requests are answered without calling out.
	same := map[string]string{"text": "Hello", "source_lang": "en", "target_lang": "en"}
	for i := 1; i <= 30; i++ {
		wantStatus(t, alice.postJSON("/api/translate", same), http.StatusOK)
	}
	wantDetail(t, alice.postJSON("/api/translate", same), http.StatusTooManyRequests, "rate limit exceeded")
	wantStatus(t, bob.postJSON("/api/translate", same), http.StatusOK)

	// AI: 10 per minute. With no key each allowed call is a 503.
	plan := map[string]string{"message": "hi", "language": "ja"}
	for i := 1; i <= 10; i++ {
		wantStatus(t, alice.postJSON("/api/ai/plan", plan), http.StatusServiceUnavailable)
	}
	wantStatus(t, alice.postJSON("/api/ai/plan", plan), http.StatusTooManyRequests)
	wantStatus(t, bob.postJSON("/api/ai/plan", plan), http.StatusServiceUnavailable)

	// Posts: 20 per day. Reading the feed is not limited.
	for i := 1; i <= 20; i++ {
		wantStatus(t, alice.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": fmt.Sprint("post ", i)}), http.StatusOK)
	}
	wantStatus(t, alice.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "one too many"}), http.StatusTooManyRequests)
	wantStatus(t, bob.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "bob's first"}), http.StatusOK)
	r := alice.get("/api/posts?limit=50")
	wantStatus(t, r, http.StatusOK)
	if r.json(t)["total"] != 21.0 {
		t.Fatalf("total = %v, want 21", r.json(t)["total"])
	}
}

// Every response carries a request id, and the log never contains secrets,
// post text or coordinates.
func TestRequestIDAndLogHygieneIntegration(t *testing.T) {
	const (
		apiKey     = "sk-or-test-SECRETKEY123"
		caption    = "ログに出てはいけない本文"
		note       = "ログに出てはいけないメモ"
		searchText = "ひみつの検索語"
		question   = "ログに出てはいけない相談"
		toTransl   = "ログに出てはいけない翻訳文"
	)
	failing := newStubService(t, func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream body that must not be logged", http.StatusInternalServerError)
	})
	var logs bytes.Buffer
	a := newAppWith(t, &routes.Deps{
		Config: &config.Config{TranslationProvider: "mymemory", OpenRouterAPIKey: apiKey, OpenRouterModel: "vendor/m"},
		External: services.ExternalEndpoints{
			OpenRouterBaseURL: failing.URL, MyMemoryBaseURL: failing.URL, OpenMeteoBaseURL: failing.URL,
		},
		Logger: logging.New(&logs, slog.LevelDebug),
		Images: services.NewImageService(newMemStore()),
	})
	// Handlers log failures through the default logger.
	previous := slog.Default()
	slog.SetDefault(logging.New(&logs, slog.LevelDebug))
	t.Cleanup(func() { slog.SetDefault(previous) })

	c := a.client(t)

	// Request ids: present, different each time, and not taken from the client.
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-Request-Id", "forged-by-client")
	w := httptest.NewRecorder()
	a.handler.ServeHTTP(w, req)
	first := w.Header().Get("X-Request-Id")
	second := c.get("/api/spots").Header().Get("X-Request-Id")
	if first == "" || second == "" || first == second || first == "forged-by-client" {
		t.Fatalf("request ids = %q, %q", first, second)
	}
	if id := c.get("/api/posts").Header().Get("X-Request-Id"); id == "" {
		t.Fatal("no request id on a 401")
	}

	wantStatus(t, c.verifyInvite("WRONG-INVITE-1"), http.StatusUnauthorized)
	memberID, transferCode := c.signup("Mika")
	sessionToken := c.token
	wantStatus(t, a.client(t).postJSON("/api/auth/transfer", map[string]string{"transfer_code": "AAAA-BBBB-CCCC"}), http.StatusUnauthorized)

	wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": caption, "tags": "ひみつタグ"},
		filePart{"media", "image/jpeg", testutil.JPEGWithGPSExif(t, 64, 32)}), http.StatusOK)
	wantStatus(t, c.postJSON("/api/gps-visits", map[string]any{"place_name": "ひみつの場所", "visited_date": "2026-10-08",
		"note": note, "latitude": secretLat, "longitude": secretLon}), http.StatusOK)
	wantStatus(t, c.get("/api/posts?q="+searchText), http.StatusOK)
	wantStatus(t, c.get(fmt.Sprintf("/api/weather?latitude=%v&longitude=%v", secretLat, secretLon)), http.StatusBadGateway)
	wantStatus(t, c.postJSON("/api/translate", map[string]string{"text": toTransl, "source_lang": "ja", "target_lang": "en"}), http.StatusBadGateway)
	wantStatus(t, c.postJSON("/api/ai/plan", map[string]string{"message": question, "language": "ja"}), http.StatusBadGateway)
	wantStatus(t, c.get("/api/no-such-route/"+transferCode), http.StatusNotFound)

	out := logs.String()
	for name, secret := range map[string]string{
		"invite code":         inviteCode,
		"wrong invite code":   "WRONG-INVITE-1",
		"transfer code":       transferCode,
		"wrong transfer code": "AAAA-BBBB-CCCC",
		"session token":       sessionToken,
		"API key":             apiKey,
		"caption":             caption,
		"tag":                 "ひみつタグ",
		"visit note":          note,
		"place name":          "ひみつの場所",
		"search text":         searchText,
		"AI question":         question,
		"text to translate":   toTransl,
		"upstream body":       "upstream body that must not be logged",
		"upstream URL":        failing.URL,
	} {
		if secret == "" {
			t.Fatalf("%s is empty: the check would pass vacuously", name)
		}
		if strings.Contains(out, secret) {
			t.Errorf("the log contains the %s", name)
		}
	}
	if containsCoordinate(out, secretLat, secretLon) {
		t.Error("the log contains coordinates")
	}

	// What it does contain: one JSON line per request with the route
	// template, status, request id and member.
	var sawPost, sawUpstream, sawUnmatched bool
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("log line is not JSON: %s", line)
		}
		if entry["request_id"] == nil || entry["request_id"] == "" {
			t.Fatalf("log line without a request id: %s", line)
		}
		if entry["msg"] == "request" && entry["route"] == "/api/posts" && entry["method"] == "POST" {
			sawPost = entry["status"] == 200.0 && entry["member_id"] == memberID && entry["duration_ms"] != nil
		}
		if entry["msg"] == "handler failed" && entry["op"] == "ai plan" {
			sawUpstream = strings.Contains(fmt.Sprint(entry["error"]), "status 500") && entry["member_id"] == memberID
		}
		if entry["msg"] == "request" && entry["route"] == "unmatched" {
			sawUnmatched = true
		}
	}
	if !sawPost || !sawUpstream || !sawUnmatched {
		t.Fatalf("expected log lines missing (post %v, upstream failure %v, unmatched %v):\n%s", sawPost, sawUpstream, sawUnmatched, out)
	}
}

// The whole upload path against the real S3 API (MinIO): the photo is
// fetchable at the URL the API returns, carries no EXIF, and is removed
// from the bucket when its post is deleted.
func TestPostImageOnS3Integration(t *testing.T) {
	s3 := testutil.RequireS3(t)
	bucket := fmt.Sprintf("sakunoie-test-http-%d", time.Now().UnixNano())
	store, err := storage.NewS3Store(storage.S3Config{
		Endpoint: s3.Endpoint, Bucket: bucket, AccessKey: s3.AccessKey, SecretKey: s3.SecretKey,
		PublicURL: s3.Endpoint + "/" + bucket,
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := store.RemoveBucket(context.Background()); err != nil {
			t.Logf("test bucket %s was not removed: %v", bucket, err)
		}
	})
	fetch := func(url string) (int, []byte) {
		t.Helper()
		resp, err := http.Get(url)
		if err != nil {
			t.Fatal(err)
		}
		defer resp.Body.Close()
		body, _ := io.ReadAll(resp.Body)
		return resp.StatusCode, body
	}

	// The bucket does not exist yet: the first upload creates it.
	a := newApp(t, services.NewImageService(store))
	c := a.client(t)
	c.signup("Mika")

	r := c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "S3 へ"},
		filePart{"media", "image/jpeg", testutil.JPEGWithGPSExif(t, 1600, 800)})
	wantStatus(t, r, http.StatusOK)
	postID := obj(t, r.json(t), "post")["id"].(string)

	// The URL comes back when the post is read again, and anyone can fetch it.
	r = c.get("/api/posts/" + postID)
	wantStatus(t, r, http.StatusOK)
	url := list(t, obj(t, r.json(t), "post"), "media")[0]["url"].(string)
	if !strings.HasPrefix(url, s3.Endpoint+"/"+bucket+"/posts/") || !strings.HasSuffix(url, ".jpg") {
		t.Fatalf("media url = %q", url)
	}
	status, body := fetch(url)
	if status != http.StatusOK {
		t.Fatalf("GET %s = %d", url, status)
	}
	if testutil.HasExif(body) {
		t.Fatal("the object in the bucket still carries EXIF")
	}
	img, format, err := image.Decode(bytes.NewReader(body))
	if err != nil || format != "jpeg" {
		t.Fatalf("stored object: format %q, err %v", format, err)
	}
	// 1600x800 with Orientation=6 is an 800x1600 picture: under the 1400
	// width cap, so only rotated.
	if b := img.Bounds(); b.Dx() != 800 || b.Dy() != 1600 {
		t.Fatalf("stored size = %dx%d, want 800x1600", b.Dx(), b.Dy())
	}

	// A file that is rejected never reaches the bucket.
	wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "x"},
		filePart{"media", "image/jpeg", []byte("#!/bin/sh")}), http.StatusUnprocessableEntity)

	wantStatus(t, c.del("/api/posts/"+postID), http.StatusOK)
	if status, _ := fetch(url); status != http.StatusNotFound {
		t.Fatalf("GET after the post was deleted = %d, want 404", status)
	}
}

// A footprint is private unless the member says otherwise: a request that
// leaves out is_public must not publish the visit.
func TestGpsVisitPrivateByDefaultIntegration(t *testing.T) {
	a, _ := newCommunityApp(t)
	alice, bob := a.client(t), a.client(t)
	alice.signup("Alice")
	bob.signup("Bob")

	r := alice.postJSON("/api/gps-visits", map[string]any{"place_name": "言い忘れた場所", "visited_date": "2026-10-08",
		"note": "公開するつもりのないメモ", "latitude": secretLat, "longitude": secretLon})
	wantStatus(t, r, http.StatusOK)
	visit := obj(t, r.json(t), "visit")
	if visit["is_public"] != false || visit["exact_public"] != false {
		t.Fatalf("visit created without is_public = %v, want is_public false, exact_public false", visit)
	}
	// exact_public alone does not publish it either.
	r = alice.postJSON("/api/gps-visits", map[string]any{"place_name": "座標だけ公開", "visited_date": "2026-10-08",
		"exact_public": true, "latitude": secretLat, "longitude": secretLon})
	wantStatus(t, r, http.StatusOK)
	if v := obj(t, r.json(t), "visit"); v["is_public"] != false {
		t.Fatalf("visit with only exact_public = %v", v)
	}

	// Stored that way, not just reported that way.
	var stored models.GpsVisit
	if err := a.db.Gorm.First(&stored, "id = ?", visit["id"]).Error; err != nil {
		t.Fatal(err)
	}
	if stored.IsPublic || stored.ExactPublic {
		t.Fatalf("stored row: is_public %v, exact_public %v", stored.IsPublic, stored.ExactPublic)
	}

	// Bob sees neither, anywhere.
	r = bob.get("/api/gps-visits/public")
	wantStatus(t, r, http.StatusOK)
	if got := list(t, r.json(t), "visits"); len(got) != 0 {
		t.Fatalf("bob's public list = %s", r.Body)
	}
	for _, path := range []string{"/api/gps-visits/public", "/api/gps-visits", "/api/rankings?type=spots"} {
		body := bob.get(path).Body.String()
		if strings.Contains(body, "言い忘れた場所") || strings.Contains(body, "公開するつもりのないメモ") || containsCoordinate(body, secretLat, secretLon) {
			t.Errorf("GET %s shows a visit that was never made public: %s", path, body)
		}
	}
	// Alice still sees both of hers, and can publish one deliberately.
	if got := list(t, alice.get("/api/gps-visits").json(t), "visits"); len(got) != 2 {
		t.Fatalf("alice's visits = %d, want 2", len(got))
	}
	wantStatus(t, alice.sendJSON(http.MethodPatch, "/api/gps-visits/"+visit["id"].(string)+"/visibility", map[string]any{"is_public": true}), http.StatusOK)
	got := list(t, bob.get("/api/gps-visits/public").json(t), "visits")
	if len(got) != 1 || got[0]["id"] != visit["id"] || got[0]["latitude"] != nil {
		t.Fatalf("after publishing, bob sees %v", got)
	}
}
