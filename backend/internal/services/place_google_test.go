package services

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"testing"
)

// The stub answers follow the response formats published in Google's
// documentation and, for Geocoding, the order the real service answered in
// on 2026-10-14: businesses at the point first, then addresses.

const (
	googleTestKey = "AIza-test-SERVER-KEY-0123456789"

	googleSearchAnswer = `{"places":[
		{"id":"ChIJ_test_SHIRAKOMA-1","displayName":{"text":"Shirakoma Pond","languageCode":"en"},
		 "formattedAddress":"Hata, Sakuho, Minamisaku District, Nagano 384-0701, Japan",
		 "location":{"latitude":36.0511,"longitude":138.3597},"primaryType":"tourist_attraction",
		 "googleMapsUri":"https://maps.google.com/?cid=1234567890"},
		{"id":"ChIJ_test_CAFE-2","displayName":{"text":"Cosmos Café"},"formattedAddress":"Saku, Nagano",
		 "location":{"latitude":36.27801,"longitude":138.46468},"primaryType":"cafe",
		 "googleMapsUri":"https://evil.example/?cid=1"},
		{"id":"bad id","displayName":{"text":"Bad id"},"location":{"latitude":1,"longitude":1}},
		{"id":"ChIJ_no_name","location":{"latitude":1,"longitude":1}},
		{"id":"ChIJ_no_position","displayName":{"text":"No position"}}
	]}`

	googleDetailsAnswer = `{"id":"ChIJ_test_SHIRAKOMA-1","displayName":{"text":"白駒の池","languageCode":"ja"},
		"formattedAddress":"日本、〒384-0701 長野県南佐久郡佐久穂町畑","location":{"latitude":36.0511,"longitude":138.3597},
		"primaryType":"tourist_attraction","googleMapsUri":"https://maps.google.com/?cid=1234567890"}`

	googleGeocodeAnswer = `{"results":[
		{"placeId":"ChIJ_test_UNION-5","formattedAddress":"1-2 Sakudairaekihigashi, Saku, Nagano 385-0028, Japan",
		 "location":{"latitude":36.27802,"longitude":138.46469},"types":["establishment","point_of_interest"]},
		{"placeId":"GhIJ_test_PLUS","formattedAddress":"6FXG+HQ Saku, Nagano, Japan","types":["plus_code"]},
		{"placeId":"ChIJ_test_ROUTE-6","formattedAddress":"Route 9, Saku, Nagano, Japan","types":["route"]},
		{"place":"//places.googleapis.com/places/ChIJ_test_ADDRESS-3","placeId":"ChIJ_test_ADDRESS-3",
		 "location":{"latitude":36.278012,"longitude":138.464683},"granularity":"ROOFTOP",
		 "formattedAddress":"1 Sakudairaekihigashi, Saku, Nagano 385-0028, Japan","types":["street_address"]},
		{"placeId":"ChIJ_test_CITY-4","formattedAddress":"Saku, Nagano, Japan","types":["locality"]}
	]}`
)

func newGoogleForTest(t *testing.T, srv *placeStub) *GooglePlaceProvider {
	t.Helper()
	p, err := NewGooglePlaceProvider(GooglePlaceOptions{APIKey: googleTestKey, PlacesBaseURL: srv.URL, GeocodingBaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	return p
}

// assertGoogleAuth checks that the key travelled in the header and nowhere
// in the URL.
func assertGoogleAuth(t *testing.T, req placeRequest) {
	t.Helper()
	if got := req.Header.Get("X-Goog-Api-Key"); got != googleTestKey {
		t.Fatalf("X-Goog-Api-Key = %q", got)
	}
	if strings.Contains(req.RawQuery, googleTestKey) || strings.Contains(req.Path, googleTestKey) || req.Query.Get("key") != "" {
		t.Fatalf("the API key is in the URL: %s?%s", req.Path, req.RawQuery)
	}
}

func TestGoogleNeedsAKey(t *testing.T) {
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte(`{}`)) })
	for _, key := range []string{"", "   "} {
		if _, err := NewGooglePlaceProvider(GooglePlaceOptions{APIKey: key, PlacesBaseURL: srv.URL}); !errors.Is(err, ErrGooglePlacesKeyMissing) {
			t.Fatalf("key %q: err = %v", key, err)
		}
	}

	// Selected without a key: no provider is active, and nothing is called.
	providers, err := NewPlaceProviders(PlaceProviderConfig{
		Provider: "google",
		Google:   GooglePlaceOptions{PlacesBaseURL: srv.URL, GeocodingBaseURL: srv.URL},
		OSM:      OSMPlaceOptions{NominatimBaseURL: srv.URL, PhotonBaseURL: srv.URL},
	})
	if !errors.Is(err, ErrPlaceNotConfigured) || providers.active != nil || len(providers.bySource) != 0 {
		t.Fatalf("providers = %+v, err = %v", providers, err)
	}
	if srv.calls() != 0 {
		t.Fatalf("%d calls were made without a key", srv.calls())
	}
}

func TestPlaceProviderSelection(t *testing.T) {
	cases := []struct {
		provider, key string
		wantActive    string
		wantSources   int
		wantErr       bool
	}{
		{"", "", "osm", 1, false},
		{"osm", "", "osm", 1, false},
		{" OSM ", "", "osm", 1, false},
		{"osm", googleTestKey, "osm", 2, false}, // stored Google spots stay readable
		{"google", googleTestKey, "google", 1, false},
		{"Google", googleTestKey, "google", 1, false},
		{"google", "", "", 0, true},
		{"mapbox", "", "", 0, true},
		{"none", googleTestKey, "", 1, true},
	}
	for _, c := range cases {
		providers, err := NewPlaceProviders(PlaceProviderConfig{Provider: c.provider, Google: GooglePlaceOptions{APIKey: c.key}})
		active := ""
		if providers.active != nil {
			active = providers.active.Name()
		}
		if active != c.wantActive || len(providers.bySource) != c.wantSources || (err != nil) != c.wantErr {
			t.Errorf("provider %q, key set %v: active %q, sources %d, err %v", c.provider, c.key != "", active, len(providers.bySource), err)
		}
		if c.wantErr && !errors.Is(err, ErrPlaceNotConfigured) {
			t.Errorf("provider %q: err = %v, want ErrPlaceNotConfigured", c.provider, err)
		}
	}
	// A malformed base URL disables the OSM provider instead of calling
	// somewhere unintended.
	if providers, err := NewPlaceProviders(PlaceProviderConfig{OSM: OSMPlaceOptions{NominatimBaseURL: "nominatim.example.test"}}); !errors.Is(err, ErrPlaceNotConfigured) || providers.active != nil {
		t.Fatalf("active = %v, err = %v", providers.active, err)
	}
}

func TestGoogleSearch(t *testing.T) {
	answer := googleSearchAnswer
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte(answer)) })
	p := newGoogleForTest(t, srv)
	ctx := context.Background()

	places, err := p.Search(ctx, PlaceSearchQuery{Text: " Shirakoma  Pond ", Language: "en", Bias: &PlacePoint{36.278061, 138.464689}, Limit: 8})
	if err != nil {
		t.Fatal(err)
	}
	if len(places) != 2 {
		t.Fatalf("places = %+v", places)
	}
	want := Place{
		Source: "google", ExternalID: "ChIJ_test_SHIRAKOMA-1", Name: "Shirakoma Pond",
		Address:     "Hata, Sakuho, Minamisaku District, Nagano 384-0701, Japan",
		CategoryKey: PlaceCategorySight, Latitude: 36.0511, Longitude: 138.3597,
		MapURL: "https://maps.google.com/?cid=1234567890",
	}
	if places[0] != want {
		t.Fatalf("place = %+v\nwant    %+v", places[0], want)
	}
	// A map link that is not Google's is replaced by one built from the id.
	if places[1].MapURL != "https://www.google.com/maps/place/?q=place_id:ChIJ_test_CAFE-2" || places[1].CategoryKey != PlaceCategoryCafe {
		t.Fatalf("second place = %+v", places[1])
	}

	req := srv.last()
	if req.Method != http.MethodPost || req.Path != "/v1/places:searchText" || req.RawQuery != "" {
		t.Fatalf("request = %s %s?%s", req.Method, req.Path, req.RawQuery)
	}
	assertGoogleAuth(t, req)
	// Exactly these fields: all Pro tier or below, nothing from Enterprise,
	// no photos.
	if got := req.Header.Get("X-Goog-FieldMask"); got != "places.id,places.displayName,places.formattedAddress,places.location,places.primaryType,places.googleMapsUri" {
		t.Fatalf("field mask = %q", got)
	}
	var body map[string]any
	if err := json.Unmarshal([]byte(req.Body), &body); err != nil {
		t.Fatalf("body is not JSON: %s", req.Body)
	}
	circle := body["locationBias"].(map[string]any)["circle"].(map[string]any)
	centre := circle["center"].(map[string]any)
	if body["textQuery"] != "Shirakoma Pond" || body["languageCode"] != "en" || body["regionCode"] != "JP" || body["pageSize"] != 8.0 ||
		circle["radius"] != 50000.0 || centre["latitude"] != 36.28 || centre["longitude"] != 138.46 || len(body) != 5 {
		t.Fatalf("body = %s", req.Body)
	}
	if strings.Contains(req.Body, "36.278061") || strings.Contains(req.Body, "138.464689") {
		t.Fatalf("the exact bias position was sent: %s", req.Body)
	}

	// Google's results are never cached: the same search calls again.
	if _, err := p.Search(ctx, PlaceSearchQuery{Text: "Shirakoma Pond", Language: "en", Bias: &PlacePoint{36.278061, 138.464689}, Limit: 8}); err != nil {
		t.Fatal(err)
	}
	if srv.calls() != 2 {
		t.Fatalf("calls = %d, want 2 (no cache)", srv.calls())
	}
	// Chinese is sent as zh-CN; no results is an empty list.
	answer = `{}`
	if places, err := p.Search(ctx, PlaceSearchQuery{Text: "x", Language: "zh", Limit: 50}); err != nil || len(places) != 0 {
		t.Fatalf("places = %v, err = %v", places, err)
	}
	if !strings.Contains(srv.last().Body, `"languageCode":"zh-CN"`) || !strings.Contains(srv.last().Body, `"pageSize":20`) {
		t.Fatalf("body = %s", srv.last().Body)
	}
}

func TestGoogleNearby(t *testing.T) {
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte(googleSearchAnswer)) })
	p := newGoogleForTest(t, srv)
	ctx := context.Background()
	here := PlacePoint{36.278061234, 138.464689876}

	places, err := p.Nearby(ctx, PlaceNearbyQuery{Center: here, RadiusM: 800, Category: PlaceCategoryAll, Language: "ja", Limit: 15})
	if err != nil {
		t.Fatal(err)
	}
	// Nearest first: the café is at the centre, the pond 27km away.
	if len(places) != 2 || places[0].ExternalID != "ChIJ_test_CAFE-2" {
		t.Fatalf("places = %+v", places)
	}
	req := srv.last()
	if req.Method != http.MethodPost || req.Path != "/v1/places:searchNearby" {
		t.Fatalf("request = %s %s", req.Method, req.Path)
	}
	assertGoogleAuth(t, req)
	if got := req.Header.Get("X-Goog-FieldMask"); got != googleSearchFields {
		t.Fatalf("field mask = %q", got)
	}
	var body map[string]any
	json.Unmarshal([]byte(req.Body), &body)
	circle := body["locationRestriction"].(map[string]any)["circle"].(map[string]any)
	centre := circle["center"].(map[string]any)
	// The member's position leaves the server rounded to 3 decimals.
	if centre["latitude"] != 36.278 || centre["longitude"] != 138.465 || circle["radius"] != 800.0 ||
		body["maxResultCount"] != 15.0 || body["rankPreference"] != "POPULARITY" || body["languageCode"] != "ja" || body["regionCode"] != "JP" {
		t.Fatalf("body = %s", req.Body)
	}
	if _, filtered := body["includedTypes"]; filtered {
		t.Fatalf("all categories sent a type filter: %s", req.Body)
	}
	for _, exact := range []string{"36.2780", "138.4646"} {
		if strings.Contains(req.Body, exact) {
			t.Fatalf("a finer position (%s) was sent: %s", exact, req.Body)
		}
	}

	if _, err := p.Nearby(ctx, PlaceNearbyQuery{Center: here, RadiusM: 300, Category: PlaceCategorySight, Language: "ko", Limit: 99}); err != nil {
		t.Fatal(err)
	}
	json.Unmarshal([]byte(srv.last().Body), &body)
	types, _ := json.Marshal(body["includedTypes"])
	if string(types) != `["tourist_attraction","museum"]` || body["rankPreference"] != "DISTANCE" || body["maxResultCount"] != 20.0 || body["languageCode"] != "ko" {
		t.Fatalf("body = %s", srv.last().Body)
	}
}

func TestGoogleReverse(t *testing.T) {
	answer := googleGeocodeAnswer
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte(answer)) })
	p := newGoogleForTest(t, srv)
	ctx := context.Background()

	place, err := p.Reverse(ctx, PlacePoint{36.2780149, 138.4646851}, "en")
	if err != nil {
		t.Fatal(err)
	}
	want := Place{
		Source: "google", ExternalID: "ChIJ_test_ADDRESS-3", Name: "1 Sakudairaekihigashi, Saku, Nagano 385-0028, Japan",
		CategoryKey: PlaceCategoryOther, Latitude: 36.278012, Longitude: 138.464683,
		MapURL: "https://www.google.com/maps/place/?q=place_id:ChIJ_test_ADDRESS-3",
	}
	if place == nil || *place != want {
		t.Fatalf("place = %+v\nwant    %+v", place, want)
	}
	req := srv.last()
	if req.Method != http.MethodGet || req.Path != "/v4/geocode/location" || req.Query.Get("location.latitude") != "36.27801" ||
		req.Query.Get("location.longitude") != "138.46469" || req.Query.Get("languageCode") != "en" || req.Query.Get("regionCode") != "JP" {
		t.Fatalf("request = %s %s?%s", req.Method, req.Path, req.RawQuery)
	}
	assertGoogleAuth(t, req)

	answer = `{"results":[]}`
	if place, err := p.Reverse(ctx, PlacePoint{30, 150}, "ja"); err != nil || place != nil {
		t.Fatalf("place = %+v, err = %v", place, err)
	}

	// Without a street address, an area or a road is better than nothing.
	answer = `{"results":[
		{"placeId":"ChIJ_test_SHOP-7","formattedAddress":"Saku","types":["establishment","store"]},
		{"placeId":"ChIJ_test_ROUTE-6","formattedAddress":"Route 9, Saku","types":["route"]},
		{"placeId":"ChIJ_test_CITY-4","formattedAddress":"Saku, Nagano","types":["locality"]}]}`
	if place, err := p.Reverse(ctx, PlacePoint{36.2, 138.4}, "ja"); err != nil || place == nil || place.ExternalID != "ChIJ_test_ROUTE-6" {
		t.Fatalf("place = %+v, err = %v", place, err)
	}

	// A point with only businesses and plus codes names nothing.
	answer = `{"results":[
		{"placeId":"ChIJ_test_SHOP-7","formattedAddress":"Saku","types":["establishment","point_of_interest"]},
		{"placeId":"GhIJ_test_PLUS","formattedAddress":"6FXG+HQ Saku","types":["plus_code"]}]}`
	if place, err := p.Reverse(ctx, PlacePoint{36.2, 138.4}, "ja"); err != nil || place != nil {
		t.Fatalf("place = %+v, err = %v", place, err)
	}
}

func TestGoogleResolve(t *testing.T) {
	status, answer := http.StatusOK, googleDetailsAnswer
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) {
		w.WriteHeader(status)
		w.Write([]byte(answer))
	})
	p := newGoogleForTest(t, srv)
	ctx := context.Background()

	place, err := p.Resolve(ctx, "ChIJ_test_SHIRAKOMA-1", "ja")
	if err != nil {
		t.Fatal(err)
	}
	want := Place{
		Source: "google", ExternalID: "ChIJ_test_SHIRAKOMA-1", Name: "白駒の池", Address: "日本、〒384-0701 長野県南佐久郡佐久穂町畑",
		CategoryKey: PlaceCategorySight, Latitude: 36.0511, Longitude: 138.3597, MapURL: "https://maps.google.com/?cid=1234567890",
	}
	if *place != want {
		t.Fatalf("place = %+v\nwant    %+v", *place, want)
	}
	req := srv.last()
	if req.Method != http.MethodGet || req.Path != "/v1/places/ChIJ_test_SHIRAKOMA-1" || req.Query.Get("languageCode") != "ja" || req.Query.Get("regionCode") != "JP" {
		t.Fatalf("request = %s %s?%s", req.Method, req.Path, req.RawQuery)
	}
	assertGoogleAuth(t, req)
	if got := req.Header.Get("X-Goog-FieldMask"); got != "id,displayName,formattedAddress,location,primaryType,googleMapsUri" {
		t.Fatalf("field mask = %q", got)
	}
	// No cache: reading the same spot again asks Google again.
	if _, err := p.Resolve(ctx, "ChIJ_test_SHIRAKOMA-1", "ja"); err != nil || srv.calls() != 2 {
		t.Fatalf("err = %v, calls = %d (want 2)", err, srv.calls())
	}

	// An id Google does not know, or cannot parse.
	for _, code := range []int{http.StatusNotFound, http.StatusBadRequest} {
		status, answer = code, `{"error":{"code":404,"message":"Place ID is no longer valid","status":"NOT_FOUND"}}`
		if _, err := p.Resolve(ctx, "ChIJ_gone", "ja"); !errors.Is(err, ErrPlaceNotFound) {
			t.Fatalf("status %d: err = %v", code, err)
		}
	}

	// Ids of the wrong shape never leave the server: the id is a path
	// segment of the request URL.
	calls := srv.calls()
	for _, id := range []string{"", "a b", "../../v1/places:searchText", "ChIJ/..", "ChIJ?fields=*", "ChIJ#x", "ChIJ%2F", "場所", strings.Repeat("a", 513)} {
		if p.ValidExternalID(id) {
			t.Errorf("ValidExternalID(%q) = true", id)
		}
		if _, err := p.Resolve(ctx, id, "ja"); !errors.Is(err, ErrPlaceNotFound) {
			t.Errorf("Resolve(%q): err = %v", id, err)
		}
	}
	if srv.calls() != calls {
		t.Fatal("an invalid id was sent upstream")
	}
}

func TestGoogleFailures(t *testing.T) {
	const upstream = "API key not valid. Please pass a valid API key."
	status := http.StatusForbidden
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) {
		w.WriteHeader(status)
		w.Write([]byte(`{"error":{"message":"` + upstream + `","status":"PERMISSION_DENIED"}}`))
	})
	p := newGoogleForTest(t, srv)
	ctx := context.Background()
	here := PlacePoint{35.123456, 139.654321}

	for _, status = range []int{http.StatusForbidden, http.StatusTooManyRequests, http.StatusInternalServerError} {
		_, errSearch := p.Search(ctx, PlaceSearchQuery{Text: "ひみつの検索語", Language: "ja", Bias: &here, Limit: 5})
		_, errNearby := p.Nearby(ctx, PlaceNearbyQuery{Center: here, RadiusM: 500, Language: "ja", Limit: 5})
		_, errReverse := p.Reverse(ctx, here, "ja")
		_, errResolve := p.Resolve(ctx, "ChIJ_secret_place_id", "ja")
		for op, err := range map[string]error{"search": errSearch, "nearby": errNearby, "reverse": errReverse, "resolve": errResolve} {
			if !errors.Is(err, ErrPlaceUnavailable) || !errors.Is(err, ErrUpstream) {
				t.Errorf("status %d, %s: err = %v", status, op, err)
				continue
			}
			assertNoLeak(t, err, googleTestKey, upstream, "ひみつの検索語", "35.12", "139.65", "ChIJ_secret_place_id", "127.0.0.1", "http://")
		}
	}

	// Unreachable.
	srv.Close()
	_, err := p.Search(ctx, PlaceSearchQuery{Text: "ひみつの検索語", Language: "ja", Limit: 5})
	if !errors.Is(err, ErrPlaceUnavailable) {
		t.Fatalf("err = %v", err)
	}
	assertNoLeak(t, err, googleTestKey, "ひみつの検索語", "127.0.0.1", "http://")
}

func TestGoogleCategory(t *testing.T) {
	for primaryType, want := range map[string]string{
		"cafe": PlaceCategoryCafe, "ramen_restaurant": PlaceCategoryFood, "restaurant": PlaceCategoryFood,
		"tourist_attraction": PlaceCategorySight, "museum": PlaceCategorySight, "park": PlaceCategoryNature,
		"hotel": PlaceCategoryLodging, "lodging": PlaceCategoryLodging, "train_station": PlaceCategoryTransport,
		"convenience_store": PlaceCategoryShop, "gift_shop": PlaceCategoryShop, "spa": PlaceCategoryOnsen,
		"historical_landmark": PlaceCategoryHistory, "": PlaceCategoryOther, "street_address": PlaceCategoryOther,
	} {
		if got := googleCategory(primaryType); got != want {
			t.Errorf("googleCategory(%q) = %q, want %q", primaryType, got, want)
		}
	}
	for language, want := range map[string]string{"ja": "ja", "en": "en", "ko": "ko", "zh": "zh-CN", "": "ja"} {
		if got := googleLanguage(language); got != want {
			t.Errorf("googleLanguage(%q) = %q, want %q", language, got, want)
		}
	}
}
