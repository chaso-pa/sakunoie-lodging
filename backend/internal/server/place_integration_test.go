package server_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/config"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/logging"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/middlewares"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/routes"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/services"
)

// osmStub stands in for Nominatim and Photon. It answers by path, and
// records the raw query of every request so tests can check what left the
// server.
type osmStub struct {
	*stubService
	mu      sync.Mutex
	queries []string
	fail    atomic.Bool
}

const (
	stubCafeID = "N6353535276"

	stubSearchAnswer = `[{"osm_type":"node","osm_id":6353535276,"lat":"36.2780100","lon":"138.4646828","category":"amenity","type":"cafe",
		"name":"Cosmos Café","display_name":"Cosmos Café, National Route 141, Sakudairaeki-higashi, Saku, Nagano Prefecture, 385-0028, Japan",
		"address":{"city":"Saku","province":"Nagano Prefecture","ISO3166-2-lvl4":"JP-20","postcode":"385-0028","country":"Japan"},
		"namedetails":{"name":"コスモスカフェ","name:en":"Cosmos Café"}},
		{"osm_type":"way","osm_id":1307020430,"lat":"36.2780359","lon":"138.4630285","category":"tourism","type":"hotel",
		"name":"Toyoko Inn","display_name":"Toyoko Inn, Sakudairaeki-kita, Saku, Nagano Prefecture, 385-0029, Japan",
		"address":{"city":"Saku","postcode":"385-0029","country":"Japan"},"namedetails":{"name":"東横イン"}}]`

	stubLookupAnswer = `[{"osm_type":"node","osm_id":6353535276,"lat":"36.2780100","lon":"138.4646828","category":"amenity","type":"cafe",
		"name":"コスモスカフェ","display_name":"コスモスカフェ, 国道141号, 佐久平駅東, 佐久市, 長野県, 385-0028, 日本",
		"address":{"road":"国道141号","quarter":"佐久平駅東","city":"佐久市","province":"長野県","ISO3166-2-lvl4":"JP-20","postcode":"385-0028","country":"日本"},
		"namedetails":{"name":"コスモスカフェ","name:en":"Cosmos Café"}}]`

	stubReverseAnswer = `{"osm_type":"way","osm_id":1307020430,"lat":"36.2780359","lon":"138.4630285","category":"tourism","type":"hotel",
		"name":"東横イン","display_name":"東横イン, 佐久平駅北, 佐久市, 長野県, 385-0029, 日本",
		"address":{"city":"佐久市","province":"長野県","postcode":"385-0029","country":"日本"},"namedetails":{"name":"東横イン"}}`

	stubNearbyAnswer = `{"features":[
		{"geometry":{"coordinates":[138.4630285,36.2780359]},"properties":{"osm_type":"W","osm_id":1307020430,"osm_key":"tourism","osm_value":"hotel","name":"Toyoko Inn","district":"Sakudairaeki-kita","city":"Saku"}},
		{"geometry":{"coordinates":[138.4646828,36.27801]},"properties":{"osm_type":"N","osm_id":6353535276,"osm_key":"amenity","osm_value":"cafe","name":"Cosmos Café","street":"National Route 141","city":"Saku"}},
		{"geometry":{"coordinates":[138.4645618,36.2785936]},"properties":{"osm_type":"W","osm_id":1307021040,"osm_key":"amenity","osm_value":"parking","name":"Parking"}}]}`
)

func newOSMStub(t *testing.T) *osmStub {
	t.Helper()
	s := &osmStub{}
	s.stubService = newStubService(t, func(w http.ResponseWriter, r *http.Request) {
		s.mu.Lock()
		s.queries = append(s.queries, r.URL.Path+"?"+r.URL.RawQuery)
		s.mu.Unlock()
		if s.fail.Load() {
			http.Error(w, "upstream body that must not leak", http.StatusInternalServerError)
			return
		}
		switch {
		case r.URL.Path == "/search":
			w.Write([]byte(stubSearchAnswer))
		case r.URL.Path == "/lookup" && r.URL.Query().Get("osm_ids") == stubCafeID:
			w.Write([]byte(stubLookupAnswer))
		case r.URL.Path == "/lookup":
			w.Write([]byte(`[]`))
		case r.URL.Path == "/reverse" && r.URL.Query().Get("radius") != "": // Photon
			w.Write([]byte(stubNearbyAnswer))
		case r.URL.Path == "/reverse": // Nominatim
			w.Write([]byte(stubReverseAnswer))
		default:
			http.NotFound(w, r)
		}
	})
	return s
}

func (s *osmStub) sent() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return strings.Join(s.queries, "\n")
}

func newPlaceApp(t *testing.T, osm *osmStub, logs io.Writer) *app {
	t.Helper()
	deps := &routes.Deps{External: services.ExternalEndpoints{NominatimBaseURL: osm.URL, PhotonBaseURL: osm.URL}}
	if logs != nil {
		deps.Logger = logging.New(logs, slog.LevelDebug)
	}
	return newAppWith(t, deps)
}

func (c *client) setLanguage(language string) {
	c.t.Helper()
	wantStatus(c.t, c.multipart(http.MethodPatch, "/api/members/me", map[string]string{"language": language}), http.StatusOK)
}

// The whole place picker flow, with two members.
func TestPlaceFlowIntegration(t *testing.T) {
	osm := newOSMStub(t)
	a := newPlaceApp(t, osm, nil)
	alice, bob := a.client(t), a.client(t)
	aliceID, _ := alice.signup("Alice")
	bob.signup("Bob")
	alice.setLanguage("en")

	// Keyword search, in the member's own language by default.
	r := alice.postJSON("/api/places/search", map[string]any{"query": "Cosmos"})
	wantStatus(t, r, http.StatusOK)
	body := r.json(t)
	if body["provider"] != "osm" || obj(t, body, "attribution")["text"] != "© OpenStreetMap contributors" ||
		obj(t, body, "attribution")["url"] != "https://www.openstreetmap.org/copyright" {
		t.Fatalf("search meta = %s", r.Body)
	}
	results := list(t, body, "results")
	if len(results) != 2 {
		t.Fatalf("results = %s", r.Body)
	}
	cafe := results[0]
	for key, want := range map[string]any{
		"source": "osm", "external_id": stubCafeID, "spot_id": nil, "name": "Cosmos Café", "original_name": "コスモスカフェ",
		"address": "National Route 141, Sakudairaeki-higashi, Saku, Nagano Prefecture", "category": "カフェ", "category_key": "cafe", "emoji": "☕",
		"latitude": 36.27801, "longitude": 138.4646828, "map_url": "https://www.openstreetmap.org/node/6353535276",
		"distance_m": nil, "photo_url": nil,
	} {
		if got, present := cafe[key]; !present || got != want {
			t.Errorf("result.%s = %v (present %v), want %v", key, got, present, want)
		}
	}
	if !strings.Contains(osm.sent(), "accept-language=en") {
		t.Fatalf("the member's language was not used: %s", osm.sent())
	}
	// An explicit language wins; a limit trims.
	r = alice.postJSON("/api/places/search", map[string]any{"query": "Cosmos", "language": "ko", "limit": 1})
	wantStatus(t, r, http.StatusOK)
	if got := list(t, r.json(t), "results"); len(got) != 1 || !strings.Contains(osm.sent(), "accept-language=ko%2Cen") {
		t.Fatalf("results = %v, sent = %s", got, osm.sent())
	}

	// Nearby, nearest first, with distances from where the member stands.
	r = bob.postJSON("/api/places/nearby", map[string]any{"latitude": 36.27806, "longitude": 138.46468})
	wantStatus(t, r, http.StatusOK)
	near := list(t, r.json(t), "results")
	if len(near) != 2 || near[0]["name"] != "Cosmos Café" || near[0]["distance_m"] != 6.0 || near[1]["name"] != "Toyoko Inn" ||
		near[1]["distance_m"] != 148.0 || near[1]["category_key"] != "lodging" || near[1]["external_id"] != "W1307020430" {
		t.Fatalf("nearby = %s", r.Body)
	}
	if !strings.Contains(osm.sent(), "lat=36.278&") || !strings.Contains(osm.sent(), "lon=138.465&") || !strings.Contains(osm.sent(), "radius=0.80") {
		t.Fatalf("nearby request = %s", osm.sent())
	}

	// Reverse lookup of a point picked on the map.
	r = bob.postJSON("/api/places/reverse", map[string]any{"latitude": 36.27803, "longitude": 138.46303})
	wantStatus(t, r, http.StatusOK)
	hotel := obj(t, r.json(t), "result")
	if hotel["name"] != "東横イン" || hotel["external_id"] != "W1307020430" || hotel["address"] != "長野県佐久市佐久平駅北" || r.json(t)["provider"] != "osm" {
		t.Fatalf("reverse = %s", r.Body)
	}

	// "Use this place": Alice registers the café.
	r = alice.postJSON("/api/spots", map[string]any{"source": "osm", "external_id": stubCafeID})
	wantStatus(t, r, http.StatusOK)
	if r.json(t)["created"] != true {
		t.Fatalf("created = %v", r.json(t)["created"])
	}
	spot := obj(t, r.json(t), "spot")
	spotID := spot["id"].(string)
	for key, want := range map[string]any{
		"name": "Cosmos Café", "original_name": "コスモスカフェ", "emoji": "☕", "area": "東信", "category": "カフェ", "description": "",
		"is_sample": false, "source": "osm", "external_id": stubCafeID, "address": "長野県佐久市佐久平駅東国道141号",
		"latitude": 36.27801, "longitude": 138.4646828, "map_url": "https://www.openstreetmap.org/node/6353535276",
		"details_available": true, "is_mine": true,
	} {
		if got, present := spot[key]; !present || got != want {
			t.Errorf("spot.%s = %v (present %v), want %v", key, got, present, want)
		}
	}
	if spot["created_at"] == nil || spot["created_at"] == "" {
		t.Errorf("spot.created_at = %v", spot["created_at"])
	}
	// Bob chooses the same result: the same spot, not a second one.
	r = bob.postJSON("/api/spots", map[string]any{"source": "osm", "external_id": stubCafeID})
	wantStatus(t, r, http.StatusOK)
	if again := obj(t, r.json(t), "spot"); r.json(t)["created"] != false || again["id"] != spotID || again["is_mine"] != false {
		t.Fatalf("second registration = %s", r.Body)
	}
	var rows int
	if err := a.db.Sessions.QueryRow("SELECT COUNT(*) FROM spots").Scan(&rows); err != nil || rows != 1 {
		t.Fatalf("spots = %d, err = %v; want 1", rows, err)
	}
	var registrant sql.NullString
	a.db.Sessions.QueryRow("SELECT created_by_member_id FROM spots WHERE id = ?", spotID).Scan(&registrant)
	if registrant.String != aliceID {
		t.Fatalf("created_by_member_id = %q, want Alice", registrant.String)
	}
	// From now on the search result says it is a spot already.
	r = bob.postJSON("/api/places/search", map[string]any{"query": "Cosmos"})
	if got := list(t, r.json(t), "results"); got[0]["spot_id"] != spotID || got[0]["source"] != "osm" || len(got) != 2 {
		t.Fatalf("search after registration = %s", r.Body)
	}

	// Hand-entered places, with and without a position.
	r = bob.postJSON("/api/spots", map[string]any{"source": "manual", "name": "山のパン屋", "address": "佐久市のどこか",
		"latitude": 36.25, "longitude": 138.5, "category": "food", "area": "東信"})
	wantStatus(t, r, http.StatusOK)
	bakery := obj(t, r.json(t), "spot")
	if r.json(t)["created"] != true || bakery["source"] != "manual" || bakery["name"] != "山のパン屋" || bakery["address"] != "佐久市のどこか" ||
		bakery["latitude"] != 36.25 || bakery["longitude"] != 138.5 || bakery["category"] != "グルメ" || bakery["emoji"] != "🍽️" ||
		bakery["area"] != "東信" || bakery["external_id"] != nil || bakery["map_url"] != nil || bakery["is_mine"] != true {
		t.Fatalf("manual spot = %s", r.Body)
	}
	r = bob.postJSON("/api/spots", map[string]any{"source": "manual", "name": "名前だけの場所"})
	wantStatus(t, r, http.StatusOK)
	bare := obj(t, r.json(t), "spot")
	// No position was given, so there is none: not Saku's, not a guess.
	if lat, present := bare["latitude"]; !present || lat != nil || bare["longitude"] != nil || bare["address"] != nil || bare["area"] != "" || bare["category"] != "その他" {
		t.Fatalf("spot without a position = %s", r.Body)
	}
	var lat, lon sql.NullFloat64
	a.db.Sessions.QueryRow("SELECT latitude, longitude FROM spots WHERE id = ?", bare["id"]).Scan(&lat, &lon)
	if lat.Valid || lon.Valid {
		t.Fatalf("stored position = %v, %v; want NULL", lat, lon)
	}

	// A post at the new spot shows the spot in the feed.
	r = alice.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "駅前のカフェ", "spot_id": spotID})
	wantStatus(t, r, http.StatusOK)
	postSpot := obj(t, r.json(t), "post", "spot")
	for key, want := range map[string]any{
		"id": spotID, "name": "Cosmos Café", "emoji": "☕", "area": "東信", "source": "osm", "address": "長野県佐久市佐久平駅東国道141号",
		"latitude": 36.27801, "longitude": 138.4646828, "map_url": "https://www.openstreetmap.org/node/6353535276", "details_available": true,
	} {
		if got, present := postSpot[key]; !present || got != want {
			t.Errorf("post.spot.%s = %v (present %v), want %v", key, got, present, want)
		}
	}
	// A spot without a region takes the region chosen in the form.
	r = bob.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "名前だけ", "spot_id": bare["id"].(string), "area": "北信"})
	wantStatus(t, r, http.StatusOK)
	if post := obj(t, r.json(t), "post"); post["area"] != "北信" || obj(t, post, "spot")["area"] != "" {
		t.Fatalf("post at a regionless spot = %s", r.Body)
	}
	// And a post with no place at all works as before.
	r = bob.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "場所なし"})
	wantStatus(t, r, http.StatusOK)
	if post := obj(t, r.json(t), "post"); post["spot"] != nil {
		t.Fatalf("post without a place = %s", r.Body)
	}

	r = bob.get("/api/posts?area=東信")
	wantStatus(t, r, http.StatusOK)
	feed := list(t, r.json(t), "posts")
	if len(feed) != 1 || obj(t, feed[0], "spot")["name"] != "Cosmos Café" || obj(t, feed[0], "spot")["latitude"] != 36.27801 || feed[0]["area"] != "東信" {
		t.Fatalf("feed = %s", r.Body)
	}
	if r = bob.get("/api/posts?q=Cosmos"); r.json(t)["total"] != 1.0 {
		t.Fatalf("feed search by spot name = %s", r.Body)
	}

	// The spot's page, and the ranking.
	r = bob.get("/api/spots/" + spotID)
	wantStatus(t, r, http.StatusOK)
	if page := r.json(t); obj(t, page, "spot")["address"] != "長野県佐久市佐久平駅東国道141号" || obj(t, page, "spot")["is_mine"] != false || len(list(t, page, "posts")) != 1 {
		t.Fatalf("spot page = %s", r.Body)
	}
	wantStatus(t, bob.postJSON("/api/spots/"+spotID+"/reviews", map[string]any{"stars": 4, "text": "静かでよかった"}), http.StatusOK)
	r = alice.get("/api/rankings?type=spots&limit=10")
	wantStatus(t, r, http.StatusOK)
	ranked := list(t, r.json(t), "items")
	if len(ranked) != 2 || ranked[0]["name"] != "Cosmos Café" || ranked[0]["detail"] != "東信 · カフェ" || ranked[1]["name"] != "名前だけの場所" || ranked[1]["detail"] != "その他" {
		t.Fatalf("ranking = %s", r.Body)
	}

	// The public spot list shows none of this, to anyone.
	a.addSpot(t, "白駒の池", "東信", "自然", true)
	for _, c := range []*client{a.client(t), alice} {
		r = c.get("/api/spots")
		wantStatus(t, r, http.StatusOK)
		public := list(t, r.json(t), "spots")
		if len(public) != 1 || public[0]["name"] != "白駒の池" || public[0]["source"] != "seed" || public[0]["address"] != nil || public[0]["external_id"] != nil {
			t.Fatalf("public spot list = %s", r.Body)
		}
		for _, memberContent := range []string{"Cosmos", "山のパン屋", "名前だけの場所", "佐久市のどこか", "36.25", stubCafeID} {
			if strings.Contains(r.Body.String(), memberContent) {
				t.Errorf("the public spot list contains %q", memberContent)
			}
		}
	}
}

func TestPlaceValidationIntegration(t *testing.T) {
	osm := newOSMStub(t)
	a := newPlaceApp(t, osm, nil)
	c := a.client(t)
	c.signup("Mika")

	bad := []struct {
		path string
		body map[string]any
	}{
		{"/api/places/search", map[string]any{}},
		{"/api/places/search", map[string]any{"query": ""}},
		{"/api/places/search", map[string]any{"query": "   "}},
		{"/api/places/search", map[string]any{"query": strings.Repeat("あ", 101)}},
		{"/api/places/search", map[string]any{"query": "x", "language": "fr"}},
		{"/api/places/search", map[string]any{"query": "x", "latitude": 36.0}},
		{"/api/places/search", map[string]any{"query": "x", "latitude": 91, "longitude": 0}},
		{"/api/places/search", map[string]any{"query": "x", "limit": 11}},
		{"/api/places/nearby", map[string]any{}},
		{"/api/places/nearby", map[string]any{"latitude": 36.0}},
		{"/api/places/nearby", map[string]any{"latitude": -90.5, "longitude": 138}},
		{"/api/places/nearby", map[string]any{"latitude": 36, "longitude": 180.5}},
		{"/api/places/nearby", map[string]any{"latitude": "36", "longitude": 138}},
		{"/api/places/nearby", map[string]any{"latitude": 36, "longitude": 138, "radius_m": 50}},
		{"/api/places/nearby", map[string]any{"latitude": 36, "longitude": 138, "radius_m": 5000}},
		{"/api/places/nearby", map[string]any{"latitude": 36, "longitude": 138, "category": "history"}},
		{"/api/places/nearby", map[string]any{"latitude": 36, "longitude": 138, "limit": 21}},
		{"/api/places/reverse", map[string]any{"latitude": 36}},
		{"/api/places/reverse", map[string]any{"latitude": 95, "longitude": 138}},
		{"/api/spots", map[string]any{}},
		{"/api/spots", map[string]any{"source": "seed", "name": "x"}},
		{"/api/spots", map[string]any{"source": "osm"}},
		{"/api/spots", map[string]any{"source": "osm", "external_id": "N1; DROP TABLE spots"}},
		{"/api/spots", map[string]any{"source": "osm", "external_id": "../N1"}},
		{"/api/spots", map[string]any{"source": "osm", "external_id": stubCafeID, "name": "自分でつけた名前"}},
		{"/api/spots", map[string]any{"source": "osm", "external_id": stubCafeID, "latitude": 1, "longitude": 2}},
		{"/api/spots", map[string]any{"source": "google", "external_id": "ChIJ_x"}}, // not the provider in use
		{"/api/spots", map[string]any{"source": "manual"}},
		{"/api/spots", map[string]any{"source": "manual", "name": " "}},
		{"/api/spots", map[string]any{"source": "manual", "name": strings.Repeat("あ", 101)}},
		{"/api/spots", map[string]any{"source": "manual", "name": "x", "address": strings.Repeat("あ", 201)}},
		{"/api/spots", map[string]any{"source": "manual", "name": "x", "latitude": 36.0}},
		{"/api/spots", map[string]any{"source": "manual", "name": "x", "latitude": 90.5, "longitude": 138}},
		{"/api/spots", map[string]any{"source": "manual", "name": "x", "latitude": 36, "longitude": -180.5}},
		{"/api/spots", map[string]any{"source": "manual", "name": "x", "category": "宿泊"}},
		{"/api/spots", map[string]any{"source": "manual", "name": "x", "area": "関東"}},
		{"/api/spots", map[string]any{"source": "manual", "name": "x", "external_id": "N1"}},
	}
	for _, c2 := range bad {
		if r := c.postJSON(c2.path, c2.body); r.Code != http.StatusUnprocessableEntity {
			t.Errorf("POST %s %v = %d, want 422; body: %s", c2.path, c2.body, r.Code, r.Body)
		}
	}
	// None of the rejected requests reached the provider or the table.
	if osm.calls.Load() != 0 {
		t.Fatalf("%d upstream calls were made for invalid requests: %s", osm.calls.Load(), osm.sent())
	}
	var rows int
	a.db.Sessions.QueryRow("SELECT COUNT(*) FROM spots").Scan(&rows)
	if rows != 0 {
		t.Fatalf("spots = %d after invalid requests", rows)
	}

	// An id of the right shape that the provider does not know.
	wantDetail(t, c.postJSON("/api/spots", map[string]any{"source": "osm", "external_id": "N404"}), http.StatusNotFound, "place not found")
	// The boundaries are accepted.
	wantStatus(t, c.postJSON("/api/spots", map[string]any{"source": "manual", "name": strings.Repeat("あ", 100), "address": strings.Repeat("い", 200),
		"latitude": 90, "longitude": -180}), http.StatusOK)
	wantStatus(t, c.postJSON("/api/places/nearby", map[string]any{"latitude": 36.278, "longitude": 138.465, "radius_m": 2000, "category": "cafe", "limit": 20}), http.StatusOK)
	// Nothing known at the point: a null result, not an error.
	a2 := newAppWith(t, &routes.Deps{External: services.ExternalEndpoints{
		NominatimBaseURL: newStubService(t, func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{"error":"Unable to geocode"}`)) }).URL,
	}})
	c = a2.client(t)
	c.signup("Mika")
	r := c.postJSON("/api/places/reverse", map[string]any{"latitude": 30, "longitude": 150})
	wantStatus(t, r, http.StatusOK)
	if result, present := r.json(t)["result"]; !present || result != nil || r.json(t)["provider"] != "osm" {
		t.Fatalf("reverse of nowhere = %s", r.Body)
	}
}

// The provider being down, or missing, affects the place endpoints only.
func TestPlaceProviderDownIntegration(t *testing.T) {
	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	dead := "http://" + l.Addr().String()
	l.Close()

	osm := newOSMStub(t)
	a := newPlaceApp(t, osm, nil)
	c := a.client(t)
	c.signup("Mika")
	// Register one place while the provider is up.
	r := c.postJSON("/api/spots", map[string]any{"source": "osm", "external_id": stubCafeID})
	wantStatus(t, r, http.StatusOK)
	spotID := obj(t, r.json(t), "spot")["id"].(string)

	check := func(t *testing.T, c *client, leaks ...string) {
		t.Helper()
		requests := map[string]map[string]any{
			"/api/places/search":  {"query": "ひみつの検索語", "latitude": 35.123456, "longitude": 139.654321},
			"/api/places/nearby":  {"latitude": 35.123456, "longitude": 139.654321},
			"/api/places/reverse": {"latitude": 35.123456, "longitude": 139.654321},
			"/api/spots":          {"source": "osm", "external_id": "W987654321"},
		}
		for path, body := range requests {
			r := c.postJSON(path, body)
			wantDetail(t, r, http.StatusBadGateway, "place search temporarily unavailable")
			for _, leak := range append(leaks, "upstream body", "127.0.0.1", "nominatim", "photon", "ひみつ", "35.12", "139.65", "W987654321", "connection") {
				if strings.Contains(r.Body.String(), leak) {
					t.Errorf("POST %s: the 502 body leaks %q: %s", path, leak, r.Body)
				}
			}
		}
		// Everything that does not need the provider still works.
		wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "場所なしで投稿"}), http.StatusOK)
		wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "登録済みの場所で投稿", "spot_id": spotID}), http.StatusOK)
		wantStatus(t, c.postJSON("/api/spots", map[string]any{"source": "manual", "name": "検索が落ちていても登録できる場所"}), http.StatusOK)
		r := c.postJSON("/api/spots", map[string]any{"source": "osm", "external_id": stubCafeID})
		wantStatus(t, r, http.StatusOK)
		if obj(t, r.json(t), "spot")["id"] != spotID {
			t.Fatalf("already registered spot with the provider down = %s", r.Body)
		}
		for _, path := range []string{"/api/posts", "/api/spots", "/api/spots/" + spotID, "/api/rankings", "/api/auth/me"} {
			wantStatus(t, c.get(path), http.StatusOK)
		}
	}

	t.Run("upstream answers 500", func(t *testing.T) {
		osm.fail.Store(true)
		defer osm.fail.Store(false)
		check(t, c, osm.URL)
	})

	t.Run("upstream unreachable", func(t *testing.T) {
		a := newAppWith(t, &routes.Deps{External: services.ExternalEndpoints{NominatimBaseURL: dead, PhotonBaseURL: dead}})
		c := a.client(t)
		c.signup("Mika")
		for path, body := range map[string]map[string]any{
			"/api/places/search":  {"query": "ひみつの検索語"},
			"/api/places/nearby":  {"latitude": 35.123456, "longitude": 139.654321},
			"/api/places/reverse": {"latitude": 35.123456, "longitude": 139.654321},
			"/api/spots":          {"source": "osm", "external_id": stubCafeID},
		} {
			r := c.postJSON(path, body)
			wantDetail(t, r, http.StatusBadGateway, "place search temporarily unavailable")
			for _, leak := range []string{dead, "127.0.0.1", "refused", "dial", "35.12"} {
				if strings.Contains(r.Body.String(), leak) {
					t.Errorf("POST %s: the 502 body leaks %q: %s", path, leak, r.Body)
				}
			}
		}
		wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "場所なしで投稿"}), http.StatusOK)
		wantStatus(t, c.postJSON("/api/spots", map[string]any{"source": "manual", "name": "手入力"}), http.StatusOK)
	})
}

// PLACE_PROVIDER=google without a key, or an unknown provider: a clear 503
// and no call anywhere.
func TestPlaceProviderNotConfiguredIntegration(t *testing.T) {
	for _, provider := range []string{"google", "mapbox"} {
		t.Run(provider, func(t *testing.T) {
			upstream := newStubService(t, func(w http.ResponseWriter, _ *http.Request) { w.Write([]byte(`{}`)) })
			a := newAppWith(t, &routes.Deps{
				Config: &config.Config{PlaceProvider: provider},
				External: services.ExternalEndpoints{
					NominatimBaseURL: upstream.URL, PhotonBaseURL: upstream.URL,
					GooglePlacesBaseURL: upstream.URL, GoogleGeocodingBaseURL: upstream.URL,
				},
			})
			c := a.client(t)
			c.signup("Mika")

			for path, body := range map[string]map[string]any{
				"/api/places/search":  {"query": "Cosmos"},
				"/api/places/nearby":  {"latitude": 36.278, "longitude": 138.465},
				"/api/places/reverse": {"latitude": 36.278, "longitude": 138.465},
				"/api/spots":          {"source": "google", "external_id": "ChIJ_test"},
			} {
				wantDetail(t, c.postJSON(path, body), http.StatusServiceUnavailable, "place search is not configured")
			}
			wantDetail(t, c.postJSON("/api/spots", map[string]any{"source": "osm", "external_id": stubCafeID}), http.StatusServiceUnavailable, "place search is not configured")
			if upstream.calls.Load() != 0 {
				t.Fatalf("%d upstream calls were made without a usable provider", upstream.calls.Load())
			}

			// The rest of the app is unaffected.
			r := c.postJSON("/api/spots", map[string]any{"source": "manual", "name": "手入力の場所"})
			wantStatus(t, r, http.StatusOK)
			spotID := obj(t, r.json(t), "spot")["id"].(string)
			wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "手入力の場所で投稿", "spot_id": spotID}), http.StatusOK)
			wantStatus(t, c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "場所なし"}), http.StatusOK)
			wantStatus(t, c.get("/api/posts"), http.StatusOK)
		})
	}
}

// What a Google result carries in the stub below.
const (
	gPlaceID = "ChIJ_http_STORAGE_RULE-7"
	gName    = "Google の店名 Hoshino Bakery"
	gAddress = "Google の住所 456 Nakagomi, Saku"
	gMapURL  = "https://maps.google.com/?cid=112233445566"
	gKey     = "AIza-http-test-SERVER-KEY"
)

func newGoogleStub(t *testing.T, fail *atomic.Bool) *stubService {
	t.Helper()
	place := fmt.Sprintf(`{"id":%q,"displayName":{"text":%q},"formattedAddress":%q,"location":{"latitude":36.212345,"longitude":138.498765},"primaryType":"bakery","googleMapsUri":%q}`,
		gPlaceID, gName, gAddress, gMapURL)
	return newStubService(t, func(w http.ResponseWriter, r *http.Request) {
		if fail.Load() || r.Header.Get("X-Goog-Api-Key") != gKey {
			http.Error(w, `{"error":{"message":"upstream body that must not leak"}}`, http.StatusForbidden)
			return
		}
		switch {
		case strings.HasSuffix(r.URL.Path, ":searchText"), strings.HasSuffix(r.URL.Path, ":searchNearby"):
			w.Write([]byte(`{"places":[` + place + `]}`))
		case r.URL.Path == "/v1/places/"+gPlaceID:
			w.Write([]byte(place))
		case strings.HasPrefix(r.URL.Path, "/v1/places/"):
			http.Error(w, `{"error":{"status":"NOT_FOUND"}}`, http.StatusNotFound)
		case r.URL.Path == "/v4/geocode/location":
			w.Write([]byte(fmt.Sprintf(`{"results":[{"placeId":%q,"formattedAddress":%q,"location":{"latitude":36.212345,"longitude":138.498765}}]}`, gPlaceID, gAddress)))
		default:
			http.NotFound(w, r)
		}
	})
}

// logWithoutTimes drops the timestamp of every log line. A timestamp such
// as 14:20:35.123 would otherwise now and then look like a coordinate.
func logWithoutTimes(t *testing.T, out string) string {
	t.Helper()
	var kept strings.Builder
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		var entry map[string]any
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatalf("log line is not JSON: %s", line)
		}
		delete(entry, "time")
		b, _ := json.Marshal(entry)
		kept.Write(b)
		kept.WriteByte('\n')
	}
	return kept.String()
}

// tableDump returns every value in the database as text.
func tableDump(t *testing.T, db *sql.DB) string {
	t.Helper()
	rows, err := db.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'")
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	for rows.Next() {
		var name string
		rows.Scan(&name)
		tables = append(tables, name)
	}
	rows.Close()
	var dump strings.Builder
	for _, table := range tables {
		rows, err := db.Query("SELECT * FROM `" + table + "`")
		if err != nil {
			t.Fatal(err)
		}
		columns, _ := rows.Columns()
		for rows.Next() {
			values := make([]sql.RawBytes, len(columns))
			targets := make([]any, len(columns))
			for i := range values {
				targets[i] = &values[i]
			}
			if err := rows.Scan(targets...); err != nil {
				t.Fatal(err)
			}
			for i, v := range values {
				fmt.Fprintf(&dump, "%s.%s=%s\n", table, columns[i], v)
			}
		}
		rows.Close()
	}
	return dump.String()
}

// With Google as the provider, over HTTP: members see the places in full,
// the database holds the place ID only, and the log holds none of it.
func TestGooglePlaceStorageRuleIntegration(t *testing.T) {
	var fail atomic.Bool
	google := newGoogleStub(t, &fail)
	var logs bytes.Buffer
	a := newAppWith(t, &routes.Deps{
		Config:   &config.Config{PlaceProvider: "google", GoogleMapsAPIKey: gKey},
		External: services.ExternalEndpoints{GooglePlacesBaseURL: google.URL, GoogleGeocodingBaseURL: google.URL},
		Logger:   logging.New(&logs, slog.LevelDebug),
	})
	previous := slog.Default()
	slog.SetDefault(logging.New(&logs, slog.LevelDebug))
	t.Cleanup(func() { slog.SetDefault(previous) })
	alice, bob := a.client(t), a.client(t)
	alice.signup("Alice")
	bob.signup("Bob")

	r := alice.postJSON("/api/places/search", map[string]any{"query": "Hoshino"})
	wantStatus(t, r, http.StatusOK)
	found := list(t, r.json(t), "results")
	if len(found) != 1 || found[0]["name"] != gName || found[0]["source"] != "google" || found[0]["external_id"] != gPlaceID ||
		found[0]["address"] != gAddress || found[0]["category_key"] != "cafe" || found[0]["map_url"] != gMapURL || found[0]["photo_url"] != nil ||
		r.json(t)["provider"] != "google" || obj(t, r.json(t), "attribution")["text"] != "Powered by Google" {
		t.Fatalf("search = %s", r.Body)
	}
	wantStatus(t, alice.postJSON("/api/places/nearby", map[string]any{"latitude": 36.2123, "longitude": 138.4987}), http.StatusOK)
	// A point picked on another map is not looked up: Google's content may only be shown on a Google map.
	before := google.calls.Load()
	r = alice.postJSON("/api/places/reverse", map[string]any{"latitude": 36.212345, "longitude": 138.498765})
	wantStatus(t, r, http.StatusOK)
	if r.json(t)["result"] != nil || r.json(t)["needs_google_map"] != true || r.json(t)["provider"] != "google" || google.calls.Load() != before {
		t.Fatalf("reverse off a Google map = %s (Google calls %d → %d)", r.Body, before, google.calls.Load())
	}
	r = alice.postJSON("/api/places/reverse", map[string]any{"latitude": 36.212345, "longitude": 138.498765, "google_map": true})
	wantStatus(t, r, http.StatusOK)
	if obj(t, r.json(t), "result")["external_id"] != gPlaceID || r.json(t)["needs_google_map"] != false {
		t.Fatalf("reverse = %s", r.Body)
	}

	r = alice.postJSON("/api/spots", map[string]any{"source": "google", "external_id": gPlaceID})
	wantStatus(t, r, http.StatusOK)
	spot := obj(t, r.json(t), "spot")
	spotID := spot["id"].(string)
	if r.json(t)["created"] != true || spot["name"] != gName || spot["address"] != gAddress || spot["latitude"] != 36.212345 ||
		spot["source"] != "google" || spot["external_id"] != gPlaceID || spot["details_available"] != true || spot["area"] != "" || spot["map_url"] != gMapURL {
		t.Fatalf("spot = %s", r.Body)
	}
	r = bob.postJSON("/api/spots", map[string]any{"source": "google", "external_id": gPlaceID})
	wantStatus(t, r, http.StatusOK)
	if r.json(t)["created"] != false || obj(t, r.json(t), "spot")["id"] != spotID {
		t.Fatalf("second registration = %s", r.Body)
	}
	wantDetail(t, bob.postJSON("/api/spots", map[string]any{"source": "google", "external_id": "ChIJ_unknown"}), http.StatusNotFound, "place not found")

	r = alice.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "パン屋さん", "spot_id": spotID, "area": "東信"})
	wantStatus(t, r, http.StatusOK)
	postID := obj(t, r.json(t), "post")["id"].(string)
	wantStatus(t, bob.postJSON("/api/posts/"+postID+"/like", nil), http.StatusOK)
	wantStatus(t, bob.postJSON("/api/spots/"+spotID+"/reviews", map[string]any{"stars": 5, "text": "おいしい"}), http.StatusOK)
	wantStatus(t, bob.postJSON("/api/gps-visits", map[string]any{"spot_id": spotID, "place_name": "自分でつけた名前", "visited_date": "2026-10-01"}), http.StatusOK)

	r = bob.get("/api/posts")
	wantStatus(t, r, http.StatusOK)
	feedSpot := obj(t, list(t, r.json(t), "posts")[0], "spot")
	if feedSpot["name"] != gName || feedSpot["address"] != gAddress || feedSpot["latitude"] != 36.212345 || feedSpot["details_available"] != true ||
		feedSpot["source"] != "google" || feedSpot["emoji"] != "☕" || list(t, r.json(t), "posts")[0]["area"] != "東信" {
		t.Fatalf("feed spot = %v", feedSpot)
	}
	r = bob.get("/api/spots/" + spotID)
	wantStatus(t, r, http.StatusOK)
	if obj(t, r.json(t), "spot")["name"] != gName {
		t.Fatalf("spot page = %s", r.Body)
	}
	r = bob.get("/api/rankings?type=spots")
	wantStatus(t, r, http.StatusOK)
	if items := list(t, r.json(t), "items"); len(items) != 1 || items[0]["name"] != gName {
		t.Fatalf("ranking = %s", r.Body)
	}
	wantStatus(t, bob.get("/api/rankings?type=posts"), http.StatusOK)
	// The public list never shows it, and never calls Google.
	calls := google.calls.Load()
	r = a.client(t).get("/api/spots")
	wantStatus(t, r, http.StatusOK)
	if len(list(t, r.json(t), "spots")) != 0 || google.calls.Load() != calls {
		t.Fatalf("public list = %s (Google calls %d → %d)", r.Body, calls, google.calls.Load())
	}

	// The database: the place ID, and nothing else of Google's.
	forbidden := map[string]string{
		"name": gName, "part of the name": "Hoshino", "address": gAddress, "part of the address": "Nakagomi",
		"map URL": gMapURL, "map URL id": "112233445566", "latitude": "36.212345", "coarse latitude": "36.21",
		"longitude": "138.498765", "coarse longitude": "138.49", "category": "カフェ", "emoji": "☕",
	}
	dump := tableDump(t, a.db.Sessions)
	if !strings.Contains(dump, "spots.external_id="+gPlaceID) || !strings.Contains(dump, "spots.source=google") {
		t.Fatalf("the Google spot row is missing from the dump:\n%s", dump)
	}
	for what, value := range forbidden {
		if strings.Contains(dump, value) {
			t.Errorf("the database contains the Google %s (%q)", what, value)
		}
	}

	// Google down: the feed, the spot page and the ranking still answer.
	fail.Store(true)
	r = bob.get("/api/posts")
	wantStatus(t, r, http.StatusOK)
	down := obj(t, list(t, r.json(t), "posts")[0], "spot")
	if down["id"] != spotID || down["name"] != "" || down["details_available"] != false || down["address"] != nil || down["latitude"] != nil ||
		down["map_url"] != "https://www.google.com/maps/place/?q=place_id:"+gPlaceID || down["emoji"] != "📍" {
		t.Fatalf("feed spot with Google down = %v", down)
	}
	wantStatus(t, bob.get("/api/spots/"+spotID), http.StatusOK)
	wantStatus(t, bob.get("/api/rankings?type=spots"), http.StatusOK)
	wantStatus(t, bob.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "Google が落ちていても", "spot_id": spotID}), http.StatusOK)
	wantDetail(t, bob.postJSON("/api/places/search", map[string]any{"query": "Hoshino"}), http.StatusBadGateway, "place search temporarily unavailable")

	// The log: no Google content, no key, no upstream text, in either the
	// working or the failing half.
	out := logWithoutTimes(t, logs.String())
	if !strings.Contains(out, "google spots shown without details") {
		t.Fatalf("the degraded read was not logged:\n%s", out)
	}
	forbidden["API key"] = gKey
	forbidden["place ID"] = gPlaceID
	forbidden["upstream body"] = "upstream body that must not leak"
	forbidden["upstream URL"] = google.URL
	forbidden["search text"] = "Hoshino"
	for what, value := range forbidden {
		if strings.Contains(out, value) {
			t.Errorf("the log contains the %s (%q)", what, value)
		}
	}
}

// Positions and search text never reach the log, whether the provider
// answers or fails.
func TestPlaceLogHygieneIntegration(t *testing.T) {
	const (
		searchText = "ひみつの検索語 Himitsu"
		manualName = "ログに出てはいけない場所の名前"
		manualAddr = "ログに出てはいけない住所 9-9-9"
		// Distinct from every fixture, at each precision that is used.
		hereLat, hereLon = 35.123456, 139.654321
		pickLat, pickLon = 34.987654, 137.456789
	)
	osm := newOSMStub(t)
	var logs bytes.Buffer
	a := newPlaceApp(t, osm, &logs)
	previous := slog.Default()
	slog.SetDefault(logging.New(&logs, slog.LevelDebug))
	t.Cleanup(func() { slog.SetDefault(previous) })
	c := a.client(t)
	memberID, _ := c.signup("Mika")

	run := func(want int) {
		t.Helper()
		wantStatus(t, c.postJSON("/api/places/search", map[string]any{"query": searchText, "latitude": hereLat, "longitude": hereLon}), want)
		wantStatus(t, c.postJSON("/api/places/nearby", map[string]any{"latitude": hereLat, "longitude": hereLon, "radius_m": 2000}), want)
		wantStatus(t, c.postJSON("/api/places/reverse", map[string]any{"latitude": pickLat, "longitude": pickLon}), want)
	}
	run(http.StatusOK)
	wantStatus(t, c.postJSON("/api/spots", map[string]any{"source": "osm", "external_id": stubCafeID}), http.StatusOK)
	wantStatus(t, c.postJSON("/api/spots", map[string]any{"source": "manual", "name": manualName, "address": manualAddr, "latitude": pickLat, "longitude": pickLon}), http.StatusOK)
	// A rejected request is not logged with its content either.
	wantStatus(t, c.postJSON("/api/places/nearby", map[string]any{"latitude": hereLat, "longitude": 500.5}), http.StatusUnprocessableEntity)
	osm.fail.Store(true)
	// Different text and points, so the answers are not served from the cache.
	wantStatus(t, c.postJSON("/api/places/search", map[string]any{"query": searchText + " 2", "latitude": hereLat, "longitude": hereLon}), http.StatusBadGateway)
	wantStatus(t, c.postJSON("/api/places/nearby", map[string]any{"latitude": hereLat, "longitude": hereLon, "radius_m": 1500}), http.StatusBadGateway)
	wantStatus(t, c.postJSON("/api/places/reverse", map[string]any{"latitude": pickLat + 0.0001, "longitude": pickLon}), http.StatusBadGateway)
	wantStatus(t, c.postJSON("/api/spots", map[string]any{"source": "osm", "external_id": "W424242"}), http.StatusBadGateway)

	// The requests did carry the positions upstream (rounded), so the
	// check below is not vacuous.
	if sent := osm.sent(); !strings.Contains(sent, "lat=35.123&") || !strings.Contains(sent, "lat=34.98765&") {
		t.Fatalf("upstream requests = %s", sent)
	}

	out := logWithoutTimes(t, logs.String())
	for what, secret := range map[string]string{
		"search text": "ひみつの検索語", "search text (latin)": "Himitsu", "manual name": manualName, "manual address": manualAddr,
		"upstream body": "upstream body that must not leak", "upstream URL": osm.URL, "place name": "Cosmos", "external id": "W424242",
		"latitude": "35.123", "coarse latitude": "35.12", "longitude": "139.654", "coarse longitude": "139.65",
		"picked latitude": "34.987", "picked longitude": "137.456", "invalid longitude": "500.5",
	} {
		if strings.Contains(out, secret) {
			t.Errorf("the log contains the %s (%q)", what, secret)
		}
	}
	// What it does contain: the route, the status and the member, and the
	// sanitised cause of each failure.
	for _, want := range []string{
		`"route":"/api/places/search"`, `"route":"/api/places/nearby"`, `"route":"/api/places/reverse"`, `"route":"/api/spots"`,
		`"status":200`, `"status":502`, `"status":422`, `"op":"search places"`, `"op":"nearby places"`, `"op":"reverse place"`, `"op":"register spot"`,
		"nominatim: upstream request failed: status 500", "photon: upstream request failed: status 500", `"member_id":"` + memberID + `"`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("the log lacks %s", want)
		}
	}
}

// The place limits are counted per member.
func TestPlaceRateLimitsIntegration(t *testing.T) {
	osm := newOSMStub(t)
	a := newPlaceApp(t, osm, nil)
	alice, bob := a.client(t), a.client(t)
	alice.signup("Alice")
	bob.signup("Bob")

	limits := []struct {
		path  string
		body  map[string]any
		limit int
	}{
		{"/api/places/search", map[string]any{"query": "Cosmos"}, middlewares.PlaceSearchLimitPerMinute},
		{"/api/places/nearby", map[string]any{"latitude": 36.278, "longitude": 138.465}, middlewares.PlaceNearbyLimitPerMinute},
		{"/api/places/reverse", map[string]any{"latitude": 36.278, "longitude": 138.463}, middlewares.PlaceReverseLimitPerMinute},
		{"/api/spots", map[string]any{"source": "manual", "name": "手入力の場所"}, middlewares.SpotRegisterLimitPerDay},
	}
	if limits[0].limit != 20 || limits[1].limit != 10 || limits[2].limit != 20 || limits[3].limit != 40 {
		t.Fatalf("limits = %+v", limits)
	}
	for _, l := range limits {
		for i := 1; i <= l.limit; i++ {
			if r := alice.postJSON(l.path, l.body); r.Code != http.StatusOK {
				t.Fatalf("POST %s #%d = %d, want 200; body: %s", l.path, i, r.Code, r.Body)
			}
		}
		limited := alice.postJSON(l.path, l.body)
		wantDetail(t, limited, http.StatusTooManyRequests, "rate limit exceeded")
		// Retry-After says how long to hold off: within the window.
		window := 60
		if l.path == "/api/spots" {
			window = 24 * 60 * 60
		}
		if wait, err := strconv.Atoi(limited.Header().Get("Retry-After")); err != nil || wait < 1 || wait > window {
			t.Fatalf("POST %s: Retry-After = %q, want 1..%d", l.path, limited.Header().Get("Retry-After"), window)
		}
		// An invalid request counts too: the limit is on attempts.
		wantStatus(t, alice.postJSON(l.path, map[string]any{}), http.StatusTooManyRequests)
		// Another member has their own allowance.
		wantStatus(t, bob.postJSON(l.path, l.body), http.StatusOK)
	}
	// Repeating one question cost the upstream one call each (the cache),
	// however many times the members asked.
	if n := osm.calls.Load(); n != 3 {
		t.Fatalf("upstream calls = %d, want 3: %s", n, osm.sent())
	}
	// The limits are separate from each other and from posting.
	wantStatus(t, alice.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "まだ投稿できる"}), http.StatusOK)
	var rows int
	a.db.Sessions.QueryRow("SELECT COUNT(*) FROM spots").Scan(&rows)
	if rows != 41 {
		t.Fatalf("spots = %d, want 41 (40 by Alice, 1 by Bob)", rows)
	}
}

// Two members registering the same result at once, through the API.
func TestRegisterSpotConcurrentlyIntegration(t *testing.T) {
	osm := newOSMStub(t)
	a := newPlaceApp(t, osm, nil)
	const members = 8
	clients := make([]*client, members)
	for i := range clients {
		clients[i] = a.client(t)
		clients[i].signup(fmt.Sprint("Member", i))
	}

	type outcome struct {
		code    int
		id      string
		created bool
	}
	results := make([]outcome, members)
	start := make(chan struct{})
	var wg sync.WaitGroup
	for i, c := range clients {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			r := c.postJSON("/api/spots", map[string]any{"source": "osm", "external_id": stubCafeID})
			results[i].code = r.Code
			if r.Code == http.StatusOK {
				body := r.json(t)
				results[i].id = body["spot"].(map[string]any)["id"].(string)
				results[i].created = body["created"].(bool)
			}
		}()
	}
	close(start)
	wg.Wait()

	created := 0
	for i, r := range results {
		if r.code != http.StatusOK || r.id == "" || r.id != results[0].id {
			t.Fatalf("member %d: %+v, member 0: %+v", i, r, results[0])
		}
		if r.created {
			created++
		}
	}
	var rows int
	a.db.Sessions.QueryRow("SELECT COUNT(*) FROM spots").Scan(&rows)
	if rows != 1 || created != 1 {
		t.Fatalf("spots = %d, created = %d; want 1 and 1", rows, created)
	}
}

// A full upstream queue answers 503 with Retry-After, which is how a client
// tells it from the 503 of a provider that is not configured.
func TestPlaceBusyIntegration(t *testing.T) {
	release := make(chan struct{})
	slow := newStubService(t, func(w http.ResponseWriter, _ *http.Request) {
		<-release
		w.Write([]byte(`[]`))
	})
	defer close(release)
	// One call every 10 seconds: the first search takes the turn, and the
	// second would wait far longer than a call may queue.
	a := newAppWith(t, &routes.Deps{
		Config:   &config.Config{PlaceMinInterval: 10 * time.Second},
		External: services.ExternalEndpoints{NominatimBaseURL: slow.URL, PhotonBaseURL: slow.URL},
	})
	c := a.client(t)
	c.signup("Mika")

	first := make(chan response, 1)
	go func() { first <- c.postJSON("/api/places/search", map[string]any{"query": "first"}) }()
	for slow.calls.Load() == 0 {
		time.Sleep(5 * time.Millisecond)
	}
	r := c.postJSON("/api/places/search", map[string]any{"query": "second"})
	wantDetail(t, r, http.StatusServiceUnavailable, "place search is busy, try again shortly")
	if r.Header().Get("Retry-After") != "3" {
		t.Fatalf("Retry-After = %q, want 3", r.Header().Get("Retry-After"))
	}
	if slow.calls.Load() != 1 {
		t.Fatalf("the refused search was sent upstream (%d calls)", slow.calls.Load())
	}
	release <- struct{}{}
	wantStatus(t, <-first, http.StatusOK)

	// Not configured: 503 without Retry-After.
	b := newAppWith(t, &routes.Deps{Config: &config.Config{PlaceProvider: "google"}})
	c = b.client(t)
	c.signup("Mika")
	r = c.postJSON("/api/places/search", map[string]any{"query": "x"})
	wantDetail(t, r, http.StatusServiceUnavailable, "place search is not configured")
	if got := r.Header().Get("Retry-After"); got != "" {
		t.Fatalf("Retry-After on a not-configured 503 = %q", got)
	}
}

// Over HTTP: another member's hand-entered spot answers exactly like an
// unknown id until a post uses it, and again once that post is gone.
func TestUnpostedManualSpotIsHiddenIntegration(t *testing.T) {
	osm := newOSMStub(t)
	a := newPlaceApp(t, osm, nil)
	alice, bob := a.client(t), a.client(t)
	alice.signup("Alice")
	bob.signup("Bob")
	const (
		name    = "まだ共有していない場所"
		address = "佐久市某所 1-2-3"
		missing = "00000000-0000-4000-8000-000000000000"
	)
	r := alice.postJSON("/api/spots", map[string]any{"source": "manual", "name": name, "address": address, "latitude": 36.2345, "longitude": 138.4567})
	wantStatus(t, r, http.StatusOK)
	secret := obj(t, r.json(t), "spot")["id"].(string)

	// attempts returns status and body of every way Bob can address a spot.
	attempts := func(c *client, id string) map[string]response {
		return map[string]response{
			"detail":    c.get("/api/spots/" + id),
			"review":    c.postJSON("/api/spots/"+id+"/reviews", map[string]any{"stars": 5, "text": "口コミ"}),
			"post":      c.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "他人の場所で投稿", "spot_id": id}),
			"footprint": c.postJSON("/api/gps-visits", map[string]any{"spot_id": id, "place_name": "足あと", "visited_date": "2026-10-01", "is_public": true}),
		}
	}
	hiddenFromBob := func(t *testing.T) {
		t.Helper()
		got, unknown := attempts(bob, secret), attempts(bob, missing)
		for path, want := range map[string]int{"detail": 404, "review": 404, "post": 422, "footprint": 422} {
			r := got[path]
			if r.Code != want {
				t.Errorf("%s: status = %d, want %d; body: %s", path, r.Code, want, r.Body)
			}
			// Indistinguishable from an id that does not exist.
			if r.Code != unknown[path].Code || r.json(t)["detail"] != unknown[path].json(t)["detail"] {
				t.Errorf("%s: hidden spot answered %d %v, unknown id %d %v", path, r.Code, r.json(t)["detail"], unknown[path].Code, unknown[path].json(t)["detail"])
			}
			for _, leak := range []string{name, address, "36.2345", "138.4567"} {
				if strings.Contains(r.Body.String(), leak) {
					t.Errorf("%s: the answer contains %q", path, leak)
				}
			}
		}
		// Nor through any list Bob can read.
		for _, path := range []string{"/api/spots", "/api/posts", "/api/rankings?type=spots&limit=20", "/api/rankings?type=posts&limit=20", "/api/gps-visits/public"} {
			r := bob.get(path)
			wantStatus(t, r, http.StatusOK)
			for _, leak := range []string{name, address, "36.2345"} {
				if strings.Contains(r.Body.String(), leak) {
					t.Errorf("GET %s shows %q to another member", path, leak)
				}
			}
		}
		r := bob.postJSON("/api/places/search", map[string]any{"query": "共有していない"})
		wantStatus(t, r, http.StatusOK)
		if strings.Contains(r.Body.String(), name) {
			t.Errorf("search shows the spot to another member: %s", r.Body)
		}
	}

	hiddenFromBob(t)
	var posts, reviews int
	a.db.Sessions.QueryRow("SELECT (SELECT COUNT(*) FROM posts), (SELECT COUNT(*) FROM reviews)").Scan(&posts, &reviews)
	if posts != 0 || reviews != 0 {
		t.Fatalf("refused requests left %d posts and %d reviews", posts, reviews)
	}
	// Alice, the creator, succeeds on each path. The post comes last.
	wantStatus(t, alice.get("/api/spots/"+secret), http.StatusOK)
	wantStatus(t, alice.postJSON("/api/spots/"+secret+"/reviews", map[string]any{"stars": 4, "text": "自分用"}), http.StatusOK)
	wantStatus(t, alice.postJSON("/api/gps-visits", map[string]any{"spot_id": secret, "visited_date": "2026-10-01"}), http.StatusOK)
	hiddenFromBob(t)

	r = alice.multipart(http.MethodPost, "/api/posts", map[string]string{"caption": "共有します", "spot_id": secret})
	wantStatus(t, r, http.StatusOK)
	postID := obj(t, r.json(t), "post")["id"].(string)

	// Now it is a place of the community.
	r = bob.get("/api/spots/" + secret)
	wantStatus(t, r, http.StatusOK)
	if spot := obj(t, r.json(t), "spot"); spot["name"] != name || spot["address"] != address || spot["is_mine"] != false {
		t.Fatalf("spot page for bob = %s", r.Body)
	}
	for path, r := range attempts(bob, secret) {
		if r.Code != http.StatusOK {
			t.Errorf("%s after the post: status = %d; body: %s", path, r.Code, r.Body)
		}
	}
	// Bob's own post now keeps it visible; remove it to test the last one.
	r = bob.get("/api/posts/mine")
	for _, p := range list(t, r.json(t), "posts") {
		wantStatus(t, bob.del("/api/posts/"+p["id"].(string)), http.StatusOK)
	}
	wantStatus(t, bob.get("/api/spots/"+secret), http.StatusOK)

	// Alice deletes the only post left: the spot is hers alone again.
	wantStatus(t, alice.del("/api/posts/"+postID), http.StatusOK)
	hiddenFromBob(t)
	wantStatus(t, alice.get("/api/spots/"+secret), http.StatusOK)
}
