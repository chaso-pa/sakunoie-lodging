package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"
)

// placeRequest is one request a place stub received.
type placeRequest struct {
	Method   string
	Path     string
	RawQuery string
	Query    url.Values
	Header   http.Header
	Body     string
}

type placeStub struct {
	*httptest.Server
	mu       sync.Mutex
	requests []placeRequest
}

// newPlaceStub starts a server standing in for a place provider. It records
// every request, including the headers and the raw query.
func newPlaceStub(t *testing.T, handler func(w http.ResponseWriter, r placeRequest)) *placeStub {
	t.Helper()
	s := &placeStub{}
	s.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		req := placeRequest{Method: r.Method, Path: r.URL.Path, RawQuery: r.URL.RawQuery, Query: r.URL.Query(), Header: r.Header.Clone(), Body: string(body)}
		s.mu.Lock()
		s.requests = append(s.requests, req)
		s.mu.Unlock()
		handler(w, req)
	}))
	t.Cleanup(s.Close)
	return s
}

func (s *placeStub) calls() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.requests)
}

func (s *placeStub) last() placeRequest {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.requests[len(s.requests)-1]
}

// Answers recorded from the public services on 2026-10-13, trimmed.
const (
	nominatimSearchEN = `[{"place_id":1,"osm_type":"node","osm_id":1451328164,"lat":"36.3430426","lon":"138.6370993",
		"category":"railway","type":"station","name":"Karuizawa Station",
		"display_name":"Karuizawa Station, National Highway Route 18, 軽井沢東, Karuizawa, Kitasaku County, Nagano Prefecture, 389-0104, Japan",
		"address":{"railway":"Karuizawa Station","road":"National Highway Route 18","town":"Karuizawa","county":"Kitasaku County","province":"Nagano Prefecture","ISO3166-2-lvl4":"JP-20","postcode":"389-0104","country":"Japan","country_code":"jp"},
		"namedetails":{"name":"軽井沢駅","name:en":"Karuizawa Station","name:ko":"가루이자와역"}},
		{"place_id":2,"osm_type":"way","osm_id":0,"lat":"36.1","lon":"138.1","name":"no id","display_name":"no id"},
		{"place_id":3,"osm_type":"node","osm_id":5,"lat":"north","lon":"138.1","name":"bad position","display_name":"bad position"}]`

	nominatimLookupJA = `[{"place_id":4,"osm_type":"node","osm_id":6353535276,"lat":"36.2780100","lon":"138.4646828",
		"category":"amenity","type":"cafe","name":"Cosmos Café",
		"display_name":"Cosmos Café, 国道141号, 佐久平駅東, 佐久市, 長野県, 385-0028, 日本",
		"address":{"amenity":"Cosmos Café","road":"国道141号","quarter":"佐久平駅東","city":"佐久市","province":"長野県","ISO3166-2-lvl4":"JP-20","postcode":"385-0028","country":"日本","country_code":"jp"},
		"namedetails":{"name":"Cosmos Café","name:ja":"コスモスカフェ","name:ko":"코스모스 카페"}}]`

	nominatimReverseUnnamed = `{"place_id":5,"osm_type":"node","osm_id":6353535274,"lat":"36.2779230","lon":"138.4644771",
		"category":"amenity","type":"toilets","name":"",
		"display_name":"1, 佐久平駅東, 佐久市, 長野県, 385-0028, 日本",
		"address":{"house_number":"1","quarter":"佐久平駅東","city":"佐久市","province":"長野県","ISO3166-2-lvl4":"JP-20","postcode":"385-0028","country":"日本","country_code":"jp"}}`

	photonNearby = `{"type":"FeatureCollection","features":[
		{"geometry":{"coordinates":[138.4630285,36.2780359],"type":"Point"},"properties":{"osm_type":"W","osm_id":1307020430,"osm_key":"tourism","osm_value":"hotel","name":"Toyoko Inn","street":"1","housenumber":"1","district":"Sakudairaeki-kita","city":"Saku","state":"Nagano Prefecture"}},
		{"geometry":{"coordinates":[138.4646828,36.27801],"type":"Point"},"properties":{"osm_type":"N","osm_id":6353535276,"osm_key":"amenity","osm_value":"cafe","name":"Cosmos Café","street":"National Route 141","district":"Sakudairaeki-higashi","city":"Saku","state":"Nagano Prefecture"}},
		{"geometry":{"coordinates":[138.4645618,36.2785936],"type":"Point"},"properties":{"osm_type":"W","osm_id":1307021040,"osm_key":"amenity","osm_value":"parking","name":"Station parking"}},
		{"geometry":{"coordinates":[138.463192,36.278649],"type":"Point"},"properties":{"osm_type":"W","osm_id":1307021620,"osm_key":"shop","osm_value":"florist"}},
		{"geometry":{"coordinates":[138.4646828,36.27801],"type":"Point"},"properties":{"osm_type":"N","osm_id":6353535276,"osm_key":"amenity","osm_value":"cafe","name":"Cosmos Café (duplicate)"}},
		{"geometry":{"coordinates":[138.5,36.3],"type":"Point"},"properties":{"osm_type":"N","osm_id":77,"osm_key":"amenity","osm_value":"cafe","name":"Far away café"}},
		{"geometry":{"coordinates":[138.4644532,36.2768289],"type":"Point"},"properties":{"osm_type":"X","osm_id":9,"osm_key":"shop","osm_value":"confectionery","name":"Bad type"}}
	]}`
)

func newOSMForTest(t *testing.T, nominatim, photon *placeStub) *OSMPlaceProvider {
	t.Helper()
	opts := OSMPlaceOptions{UserAgent: "tabinote-test/1.0 (ops@example.test)"}
	if nominatim != nil {
		opts.NominatimBaseURL = nominatim.URL
	}
	if photon != nil {
		opts.PhotonBaseURL = photon.URL
	}
	p, err := NewOSMPlaceProvider(opts)
	if err != nil {
		t.Fatal(err)
	}
	return p
}

func TestOSMSearch(t *testing.T) {
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte(nominatimSearchEN)) })
	p := newOSMForTest(t, srv, nil)
	ctx := context.Background()

	places, err := p.Search(ctx, PlaceSearchQuery{Text: "  Karuizawa   Station ", Language: "en", Bias: &PlacePoint{36.278061, 138.464689}, Limit: 5})
	if err != nil {
		t.Fatal(err)
	}
	// The rows without an id or a position are dropped.
	if len(places) != 1 {
		t.Fatalf("places = %+v", places)
	}
	want := Place{
		Source: "osm", ExternalID: "N1451328164", Name: "Karuizawa Station", OriginalName: "軽井沢駅",
		Address:     "National Highway Route 18, 軽井沢東, Karuizawa, Kitasaku County, Nagano Prefecture",
		CategoryKey: PlaceCategoryTransport, Latitude: 36.3430426, Longitude: 138.6370993,
		MapURL: "https://www.openstreetmap.org/node/1451328164",
	}
	if places[0] != want {
		t.Fatalf("place = %+v\nwant    %+v", places[0], want)
	}

	req := srv.last()
	if req.Method != http.MethodGet || req.Path != "/search" {
		t.Fatalf("request = %s %s", req.Method, req.Path)
	}
	for key, value := range map[string]string{
		"q": "Karuizawa Station", "format": "jsonv2", "addressdetails": "1", "namedetails": "1",
		"accept-language": "en", "countrycodes": "jp", "limit": "10",
		// A box of half a degree around the position rounded to 2 decimals.
		"viewbox": "137.96,36.78,138.96,35.78",
	} {
		if got := req.Query.Get(key); got != value {
			t.Errorf("query %s = %q, want %q", key, got, value)
		}
	}
	if strings.Contains(req.RawQuery, "36.278061") || strings.Contains(req.RawQuery, "138.464689") {
		t.Fatalf("the exact bias position was sent upstream: %s", req.RawQuery)
	}
	// Nominatim's policy: a User-Agent that identifies the application.
	if ua := req.Header.Get("User-Agent"); ua != "tabinote-test/1.0 (ops@example.test)" {
		t.Fatalf("User-Agent = %q", ua)
	}

	// The same search again, in any letter case, is answered from the cache.
	if _, err := p.Search(ctx, PlaceSearchQuery{Text: "karuizawa station", Language: "en", Bias: &PlacePoint{36.2812, 138.4601}, Limit: 5}); err != nil {
		t.Fatal(err)
	}
	if srv.calls() != 1 {
		t.Fatalf("upstream calls = %d, want 1 (cached)", srv.calls())
	}
	// Another language is another question.
	if _, err := p.Search(ctx, PlaceSearchQuery{Text: "karuizawa station", Language: "ko"}); err != nil {
		t.Fatal(err)
	}
	if srv.calls() != 2 || srv.last().Query.Get("accept-language") != "ko,en" {
		t.Fatalf("calls = %d, accept-language = %q", srv.calls(), srv.last().Query.Get("accept-language"))
	}
	// Without a bias the box is centred on Saku.
	if got := srv.last().Query.Get("viewbox"); got != "137.98,36.75,138.98,35.75" {
		t.Fatalf("default viewbox = %q", got)
	}
}

func TestOSMSearchDefaultUserAgentIsNotGos(t *testing.T) {
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte(`[]`)) })
	p, err := NewOSMPlaceProvider(OSMPlaceOptions{NominatimBaseURL: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	places, err := p.Search(context.Background(), PlaceSearchQuery{Text: "385-0028", Language: "ja"})
	if err != nil || len(places) != 0 {
		t.Fatalf("places = %v, err = %v", places, err)
	}
	ua := srv.last().Header.Get("User-Agent")
	if ua != DefaultPlaceUserAgent || strings.HasPrefix(ua, "Go-http-client") {
		t.Fatalf("User-Agent = %q", ua)
	}
	if srv.last().Query.Get("q") != "385-0028" || srv.last().Query.Get("accept-language") != "ja" {
		t.Fatalf("query = %s", srv.last().RawQuery)
	}
}

func TestOSMReverse(t *testing.T) {
	answer := nominatimReverseUnnamed
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte(answer)) })
	p := newOSMForTest(t, srv, nil)
	ctx := context.Background()

	place, err := p.Reverse(ctx, PlacePoint{36.2780149, 138.4646851}, "ja")
	if err != nil {
		t.Fatal(err)
	}
	// An unnamed object is named after its nearest address parts, and a
	// Japanese address reads largest part first.
	if place == nil || place.ExternalID != "N6353535274" || place.Name != "佐久平駅東1" || place.Address != "長野県佐久市佐久平駅東1" ||
		place.CategoryKey != PlaceCategoryOther || place.OriginalName != "" {
		t.Fatalf("place = %+v", place)
	}
	req := srv.last()
	if req.Path != "/reverse" || req.Query.Get("lat") != "36.27801" || req.Query.Get("lon") != "138.46469" ||
		req.Query.Get("zoom") != "18" || req.Query.Get("format") != "jsonv2" || req.Query.Get("accept-language") != "ja" {
		t.Fatalf("request = %s?%s", req.Path, req.RawQuery)
	}

	// Nominatim answers 200 with an error member where it knows nothing.
	answer = `{"error":"Unable to geocode"}`
	place, err = p.Reverse(ctx, PlacePoint{30, 150}, "ja")
	if err != nil || place != nil {
		t.Fatalf("place = %+v, err = %v", place, err)
	}
}

func TestOSMResolve(t *testing.T) {
	answer := nominatimLookupJA
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte(answer)) })
	p := newOSMForTest(t, srv, nil)
	ctx := context.Background()

	place, err := p.Resolve(ctx, "N6353535276", "ko")
	if err != nil {
		t.Fatal(err)
	}
	want := Place{
		Source: "osm", ExternalID: "N6353535276", Name: "코스모스 카페", OriginalName: "Cosmos Café",
		Address: "長野県佐久市佐久平駅東国道141号", CategoryKey: PlaceCategoryCafe,
		Latitude: 36.27801, Longitude: 138.4646828, MapURL: "https://www.openstreetmap.org/node/6353535276", Area: "東信",
	}
	if *place != want {
		t.Fatalf("place = %+v\nwant    %+v", *place, want)
	}
	// Always asked in Japanese, whatever the member's language.
	req := srv.last()
	if req.Path != "/lookup" || req.Query.Get("osm_ids") != "N6353535276" || req.Query.Get("accept-language") != "ja" || req.Query.Get("namedetails") != "1" {
		t.Fatalf("request = %s?%s", req.Path, req.RawQuery)
	}

	// No tag for the language: the local name, and no separate original.
	place, err = p.Resolve(ctx, "N6353535276", "en")
	if err != nil || place.Name != "Cosmos Café" || place.OriginalName != "" {
		t.Fatalf("place = %+v, err = %v", place, err)
	}
	if place, _ = p.Resolve(ctx, "N6353535276", "ja"); place == nil || place.Name != "コスモスカフェ" || place.OriginalName != "Cosmos Café" {
		t.Fatalf("ja place = %+v", place)
	}
	calls := srv.calls()
	if _, err := p.Resolve(ctx, "N6353535276", "ja"); err != nil || srv.calls() != calls {
		t.Fatalf("a repeated lookup was not cached (err %v, calls %d → %d)", err, calls, srv.calls())
	}

	// An id Nominatim does not know.
	answer = `[]`
	if _, err := p.Resolve(ctx, "W42", "ja"); !errors.Is(err, ErrPlaceNotFound) {
		t.Fatalf("unknown id: err = %v", err)
	}
	// An answer about some other object is not taken for the one asked for.
	answer = nominatimLookupJA
	if _, err := p.Resolve(ctx, "R6353535276", "ja"); !errors.Is(err, ErrPlaceNotFound) {
		t.Fatalf("mismatched id: err = %v", err)
	}

	// Ids of the wrong shape never leave the server.
	calls = srv.calls()
	for _, id := range []string{"", "N", "6353535276", "n123", "N12a", "N1 OR 1=1", "N1&osm_ids=W2", "../N1", "X123", "N" + strings.Repeat("9", 20)} {
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

func TestOSMResolveAreaOnlyInNagano(t *testing.T) {
	// 高山村 exists in Nagano (北信) and in Gunma.
	row := func(iso string) string {
		return `[{"osm_type":"way","osm_id":7,"lat":"36.6","lon":"138.4","category":"tourism","type":"attraction","name":"滝",
			"display_name":"滝, 高山村, 県, 日本","address":{"village":"高山村","ISO3166-2-lvl4":"` + iso + `","country":"日本"}}]`
	}
	for iso, want := range map[string]string{"JP-20": "北信", "JP-10": ""} {
		srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte(row(iso))) })
		place, err := newOSMForTest(t, srv, nil).Resolve(context.Background(), "W7", "ja")
		if err != nil || place.Area != want {
			t.Errorf("%s: area = %q, err = %v; want %q", iso, place.Area, err, want)
		}
	}
}

func TestOSMNearby(t *testing.T) {
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte(photonNearby)) })
	p := newOSMForTest(t, nil, srv)
	ctx := context.Background()

	places, err := p.Nearby(ctx, PlaceNearbyQuery{Center: PlacePoint{36.278061234, 138.464689876}, RadiusM: 500, Category: PlaceCategoryAll, Language: "en", Limit: 10})
	if err != nil {
		t.Fatal(err)
	}
	// Kept: the café and the hotel, nearest first. Dropped: the car park
	// (noise), the unnamed florist, the duplicate, the café 4km away and
	// the row with an unknown object type.
	if len(places) != 2 || places[0].Name != "Cosmos Café" || places[1].Name != "Toyoko Inn" {
		t.Fatalf("places = %+v", places)
	}
	cafe := Place{
		Source: "osm", ExternalID: "N6353535276", Name: "Cosmos Café",
		Address: "National Route 141, Sakudairaeki-higashi, Saku, Nagano Prefecture", CategoryKey: PlaceCategoryCafe,
		Latitude: 36.27801, Longitude: 138.4646828, MapURL: "https://www.openstreetmap.org/node/6353535276",
	}
	if places[0] != cafe {
		t.Fatalf("café = %+v\nwant   %+v", places[0], cafe)
	}
	// A block number is not printed as a street.
	if places[1].Address != "Sakudairaeki-kita, Saku, Nagano Prefecture" || places[1].CategoryKey != PlaceCategoryLodging || places[1].ExternalID != "W1307020430" {
		t.Fatalf("hotel = %+v", places[1])
	}

	req := srv.last()
	if req.Path != "/reverse" || req.Query.Get("radius") != "0.50" || req.Query.Get("lang") != "en" || req.Query.Get("limit") != "50" {
		t.Fatalf("request = %s?%s", req.Path, req.RawQuery)
	}
	// The member's position leaves the server rounded to 3 decimals.
	if req.Query.Get("lat") != "36.278" || req.Query.Get("lon") != "138.465" {
		t.Fatalf("position sent = %s, %s", req.Query.Get("lat"), req.Query.Get("lon"))
	}
	for _, exact := range []string{"36.2780", "36.27806", "138.4646", "138.46468"} {
		if strings.Contains(req.RawQuery, exact) {
			t.Fatalf("a finer position (%s) was sent upstream: %s", exact, req.RawQuery)
		}
	}
	if got := strings.Join(req.Query["osm_tag"], ","); got != "amenity,tourism,shop,historic,leisure,natural" {
		t.Fatalf("osm_tag = %q", got)
	}
	if ua := req.Header.Get("User-Agent"); ua != "tabinote-test/1.0 (ops@example.test)" {
		t.Fatalf("User-Agent = %q", ua)
	}

	// A position that rounds to the same point shares the cache entry.
	if _, err := p.Nearby(ctx, PlaceNearbyQuery{Center: PlacePoint{36.27841, 138.46452}, RadiusM: 500, Category: PlaceCategoryAll, Language: "ko"}); err != nil {
		t.Fatal(err)
	}
	if srv.calls() != 1 {
		t.Fatalf("upstream calls = %d, want 1", srv.calls())
	}

	// A category narrows the tags asked for and what is kept; Japanese and
	// Chinese readers get the local names.
	places, err = p.Nearby(ctx, PlaceNearbyQuery{Center: PlacePoint{36.278, 138.465}, RadiusM: 500, Category: PlaceCategoryCafe, Language: "zh"})
	if err != nil || len(places) != 1 || places[0].ExternalID != "N6353535276" {
		t.Fatalf("cafés = %+v, err = %v", places, err)
	}
	req = srv.last()
	if got := strings.Join(req.Query["osm_tag"], ","); got != "amenity:cafe" || req.Query.Get("lang") != "default" {
		t.Fatalf("osm_tag = %q, lang = %q", got, req.Query.Get("lang"))
	}
}

func TestOSMFailures(t *testing.T) {
	const (
		secretQuery = "ひみつの検索語"
		upstream    = "upstream detail that must not leak"
	)
	ctx := context.Background()
	here := PlacePoint{35.123456, 139.654321}
	leaks := []string{secretQuery, url.QueryEscape(secretQuery), upstream, "35.12", "139.65", "127.0.0.1", "http://"}

	// each calls all four operations and checks every error.
	each := func(t *testing.T, p *OSMPlaceProvider) {
		t.Helper()
		_, errSearch := p.Search(ctx, PlaceSearchQuery{Text: secretQuery, Language: "ja", Bias: &here})
		_, errNearby := p.Nearby(ctx, PlaceNearbyQuery{Center: here, RadiusM: 500, Language: "ja"})
		_, errReverse := p.Reverse(ctx, here, "ja")
		_, errResolve := p.Resolve(ctx, "N1", "ja")
		for op, err := range map[string]error{"search": errSearch, "nearby": errNearby, "reverse": errReverse, "resolve": errResolve} {
			if !errors.Is(err, ErrPlaceUnavailable) || !errors.Is(err, ErrUpstream) {
				t.Errorf("%s: err = %v, want ErrPlaceUnavailable wrapping ErrUpstream", op, err)
				continue
			}
			assertNoLeak(t, err, leaks...)
		}
	}

	t.Run("status 500", func(t *testing.T) {
		srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { http.Error(w, upstream, http.StatusInternalServerError) })
		each(t, newOSMForTest(t, srv, srv))
	})
	t.Run("status 429", func(t *testing.T) {
		srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { http.Error(w, upstream, http.StatusTooManyRequests) })
		p := newOSMForTest(t, srv, srv)
		each(t, p)
		// A failure is not cached: the next call asks again.
		calls := srv.calls()
		p.Search(ctx, PlaceSearchQuery{Text: secretQuery, Language: "ja", Bias: &here})
		if srv.calls() != calls+1 {
			t.Fatal("a failed search was cached")
		}
	})
	t.Run("not JSON", func(t *testing.T) {
		srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte("<html>" + upstream)) })
		each(t, newOSMForTest(t, srv, srv))
	})
	t.Run("dead port", func(t *testing.T) {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		dead := "http://" + l.Addr().String()
		l.Close()
		p, err := NewOSMPlaceProvider(OSMPlaceOptions{NominatimBaseURL: dead, PhotonBaseURL: dead})
		if err != nil {
			t.Fatal(err)
		}
		each(t, p)
	})
	t.Run("timeout", func(t *testing.T) {
		release := make(chan struct{})
		srv := newPlaceStub(t, func(http.ResponseWriter, placeRequest) { <-release })
		defer close(release)
		p := newOSMForTest(t, srv, srv)
		p.http.Timeout = 50 * time.Millisecond
		start := time.Now()
		each(t, p)
		if elapsed := time.Since(start); elapsed > 2*time.Second {
			t.Fatalf("four timed out calls took %v", elapsed)
		}
	})
	t.Run("redirect is not followed", func(t *testing.T) {
		elsewhere := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte(`[]`)) })
		srv := newPlaceStub(t, func(w http.ResponseWriter, r placeRequest) {
			w.Header().Set("Location", elsewhere.URL+r.Path+"?"+r.RawQuery)
			w.WriteHeader(http.StatusFound)
		})
		each(t, newOSMForTest(t, srv, srv))
		if elsewhere.calls() != 0 {
			t.Fatalf("%d requests followed a redirect to another host", elsewhere.calls())
		}
	})
}

// The production timeout is short enough that a hung upstream cannot hold
// a request for long.
func TestPlaceTimeoutsAreShort(t *testing.T) {
	p := newOSMForTest(t, nil, nil)
	if p.http.Timeout != 5*time.Second || googlePlaceTimeout != 5*time.Second {
		t.Fatalf("timeouts = %v, %v", p.http.Timeout, googlePlaceTimeout)
	}
}

func TestPlaceBaseURL(t *testing.T) {
	for _, bad := range []string{"ftp://example.test", "example.test", "//example.test", "http://", "https://user:pw@example.test", "https://example.test?x=1", "https://example.test/#f", "javascript:alert(1)"} {
		if _, err := NewOSMPlaceProvider(OSMPlaceOptions{NominatimBaseURL: bad}); err == nil {
			t.Errorf("NominatimBaseURL %q was accepted", bad)
		}
		if _, err := NewOSMPlaceProvider(OSMPlaceOptions{PhotonBaseURL: bad}); err == nil {
			t.Errorf("PhotonBaseURL %q was accepted", bad)
		}
		if _, err := NewGooglePlaceProvider(GooglePlaceOptions{APIKey: "k", PlacesBaseURL: bad}); err == nil {
			t.Errorf("GooglePlacesBaseURL %q was accepted", bad)
		}
	}
	p, err := NewOSMPlaceProvider(OSMPlaceOptions{NominatimBaseURL: " https://nominatim.example.test/api/ ", PhotonBaseURL: "http://photon.internal:2322"})
	if err != nil || p.nominatimURL != "https://nominatim.example.test/api" || p.photonURL != "http://photon.internal:2322" {
		t.Fatalf("provider = %+v, err = %v", p, err)
	}
	if p, _ := NewOSMPlaceProvider(OSMPlaceOptions{}); p.nominatimURL != DefaultNominatimBaseURL || p.photonURL != DefaultPhotonBaseURL {
		t.Fatalf("defaults = %s, %s", p.nominatimURL, p.photonURL)
	}
}

// The text a member types can only ever be a query value: whatever it
// contains, the request goes to the configured host and path.
func TestOSMSearchTextCannotChangeTheRequest(t *testing.T) {
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) { w.Write([]byte(`[]`)) })
	p := newOSMForTest(t, srv, nil)
	for _, text := range []string{
		"http://evil.test/", "//evil.test/x", "x&countrycodes=us&format=xml", "x#frag", "../../lookup?osm_ids=N1",
		"x\r\nHost: evil.test", "@evil.test", "x?y=1",
	} {
		if _, err := p.Search(context.Background(), PlaceSearchQuery{Text: text, Language: "ja"}); err != nil {
			t.Fatalf("%q: %v", text, err)
		}
		req := srv.last()
		if req.Path != "/search" || req.Query.Get("q") != normalizeQuery(text) || req.Query.Get("countrycodes") != "jp" ||
			req.Query.Get("format") != "jsonv2" || len(req.Query["q"]) != 1 {
			t.Fatalf("%q changed the request: %s?%s", text, req.Path, req.RawQuery)
		}
		if host := req.Header.Get("Host"); host != "" {
			t.Fatalf("%q set a Host header: %q", text, host)
		}
	}
}

func TestUpstreamThrottle(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	var slept []time.Duration
	th := newUpstreamThrottle(1100*time.Millisecond, 3*time.Second)
	th.now = func() time.Time { return now }
	th.sleep = func(_ context.Context, d time.Duration) error {
		slept = append(slept, d)
		return nil
	}
	ctx := context.Background()

	// Four calls at the same instant: the first goes at once, the next
	// three wait 1.1s, 2.2s and 3.3s... but 3.3s is over the 3s a call may
	// queue, so the fourth is refused without waiting.
	for i := 0; i < 3; i++ {
		if err := th.wait(ctx); err != nil {
			t.Fatalf("call %d: %v", i+1, err)
		}
	}
	if err := th.wait(ctx); !errors.Is(err, ErrPlaceBusy) {
		t.Fatalf("fourth call: err = %v, want ErrPlaceBusy", err)
	}
	if fmt.Sprint(slept) != "[1.1s 2.2s]" {
		t.Fatalf("waits = %v", slept)
	}
	// The refused call took no turn: half a second later the next call
	// waits for the turn after the third one.
	now = now.Add(500 * time.Millisecond)
	if err := th.wait(ctx); err != nil || slept[len(slept)-1] != 2800*time.Millisecond {
		t.Fatalf("err = %v, waits = %v", err, slept)
	}
	// After a quiet spell there is no wait.
	now = now.Add(time.Minute)
	slept = nil
	if err := th.wait(ctx); err != nil || len(slept) != 0 {
		t.Fatalf("err = %v, waits = %v", err, slept)
	}

	// A request that ends while queued stops waiting.
	th.sleep = sleepContext
	cancelled, cancel := context.WithCancel(ctx)
	cancel()
	if err := th.wait(cancelled); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled wait: err = %v", err)
	}

	// No interval, no throttle.
	if err := newUpstreamThrottle(0, time.Second).wait(ctx); err != nil {
		t.Fatal(err)
	}
}

// Calls to the real upstream are spaced even when members search at the
// same moment, and a member who would wait too long is told it is busy.
func TestOSMCallsAreSpaced(t *testing.T) {
	var mu sync.Mutex
	var arrivals []time.Time
	srv := newPlaceStub(t, func(w http.ResponseWriter, _ placeRequest) {
		mu.Lock()
		arrivals = append(arrivals, time.Now())
		mu.Unlock()
		w.Write([]byte(`[]`))
	})
	p, err := NewOSMPlaceProvider(OSMPlaceOptions{NominatimBaseURL: srv.URL, MinInterval: 120 * time.Millisecond})
	if err != nil {
		t.Fatal(err)
	}
	p.nominatimThrottle.maxWait = 300 * time.Millisecond

	var wg sync.WaitGroup
	results := make([]error, 6)
	for i := range results {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, results[i] = p.Search(context.Background(), PlaceSearchQuery{Text: fmt.Sprint("query ", i), Language: "ja"})
		}()
	}
	wg.Wait()

	busy := 0
	for _, err := range results {
		switch {
		case err == nil:
		case errors.Is(err, ErrPlaceBusy):
			busy++
		default:
			t.Fatalf("unexpected error: %v", err)
		}
	}
	// Turns at 0, 120, 240ms fit in the 300ms a call may queue; the other
	// three are refused.
	if len(arrivals) != 3 || busy != 3 {
		t.Fatalf("upstream calls = %d, busy = %d; want 3 and 3", len(arrivals), busy)
	}
	for i := 1; i < len(arrivals); i++ {
		if gap := arrivals[i].Sub(arrivals[i-1]); gap < 100*time.Millisecond {
			t.Fatalf("calls %d and %d were %v apart", i, i+1, gap)
		}
	}
}

func TestPlaceCache(t *testing.T) {
	now := time.Unix(1_800_000_000, 0)
	c := newPlaceCache(time.Minute, 3)
	c.now = func() time.Time { return now }

	c.put("a", []Place{{Name: "A"}})
	got, ok := c.get("a")
	if !ok || len(got) != 1 || got[0].Name != "A" {
		t.Fatalf("get = %v, %v", got, ok)
	}
	// The caller's copy is its own.
	got[0].Name = "changed"
	if again, _ := c.get("a"); again[0].Name != "A" {
		t.Fatal("the cached slice was modified through a returned copy")
	}
	// An empty answer is cached too. It goes in a moment later than "a", so
	// that "a" is unambiguously the entry closest to expiry below.
	now = now.Add(time.Second)
	c.put("empty", nil)
	if got, ok := c.get("empty"); !ok || len(got) != 0 {
		t.Fatalf("empty = %v, %v", got, ok)
	}

	// The cap: the entry closest to expiry goes first.
	now = now.Add(10 * time.Second)
	c.put("b", nil)
	c.put("c", nil)
	if c.len() != 3 {
		t.Fatalf("len = %d, want 3", c.len())
	}
	if _, ok := c.get("a"); ok {
		t.Fatal("the oldest entry survived the cap")
	}
	// Expiry.
	now = now.Add(55 * time.Second)
	if _, ok := c.get("empty"); ok {
		t.Fatal("an expired entry was returned")
	}
	if _, ok := c.get("b"); !ok {
		t.Fatal("a live entry was lost")
	}
	c.put("d", nil)
	if c.len() != 3 {
		t.Fatalf("len after expiry sweep = %d, want 3 (b, c, d)", c.len())
	}

	// Many distinct keys never grow it past the cap.
	for i := 0; i < 1000; i++ {
		c.put(fmt.Sprint("k", i), nil)
	}
	if c.len() != 3 {
		t.Fatalf("len = %d after 1000 puts", c.len())
	}
	if osmCacheMaxEntries != 500 || osmCacheTTL != 15*time.Minute {
		t.Fatalf("production cache = %d entries, %v", osmCacheMaxEntries, osmCacheTTL)
	}
}

func TestOSMCategory(t *testing.T) {
	cases := []struct {
		key, value, want string
		noise            bool
	}{
		{"amenity", "restaurant", PlaceCategoryFood, false},
		{"amenity", "cafe", PlaceCategoryCafe, false},
		{"amenity", "public_bath", PlaceCategoryOnsen, false},
		{"natural", "hot_spring", PlaceCategoryOnsen, false},
		{"amenity", "place_of_worship", PlaceCategoryHistory, false},
		{"historic", "castle", PlaceCategoryHistory, false},
		{"tourism", "hotel", PlaceCategoryLodging, false},
		{"tourism", "viewpoint", PlaceCategorySight, false},
		{"tourism", "information", PlaceCategoryOther, true},
		{"shop", "bakery", PlaceCategoryShop, false},
		{"leisure", "park", PlaceCategoryNature, false},
		{"natural", "peak", PlaceCategoryNature, false},
		{"natural", "tree", PlaceCategoryNature, true},
		{"railway", "station", PlaceCategoryTransport, false},
		{"highway", "bus_stop", PlaceCategoryTransport, false},
		{"amenity", "parking", PlaceCategoryOther, true},
		{"amenity", "toilets", PlaceCategoryOther, true},
		{"place", "city", PlaceCategoryOther, false},
		{"", "", PlaceCategoryOther, false},
	}
	for _, c := range cases {
		got, noise := osmCategory(c.key, c.value)
		if got != c.want || noise != c.noise {
			t.Errorf("osmCategory(%q, %q) = %q, %v; want %q, %v", c.key, c.value, got, noise, c.want, c.noise)
		}
	}
	// Every category has a label and an emoji that fit the spots table.
	for key := range placeCategories {
		label, emoji := PlaceCategory(key)
		if label == "" || emoji == "" || len([]rune(label)) > 30 || len([]rune(emoji)) > 10 {
			t.Errorf("category %q: label %q, emoji %q", key, label, emoji)
		}
		if placeCategoryKeyOfLabel(label) != key {
			t.Errorf("label %q does not map back to %q", label, key)
		}
		if label == lodgingCategory {
			t.Errorf("category %q uses the label reserved for the lodging itself", key)
		}
	}
	if label, emoji := PlaceCategory("nonsense"); label != "その他" || emoji != "📍" {
		t.Fatalf("unknown category = %q %q", label, emoji)
	}
}

func TestNaganoArea(t *testing.T) {
	if len(naganoAreas) != 77 {
		t.Fatalf("municipalities = %d, want 77", len(naganoAreas))
	}
	counts := map[string]int{}
	for _, area := range naganoAreas {
		counts[area]++
		if !postAreas[area] {
			t.Fatalf("area %q is not one a post can be filed under", area)
		}
	}
	if counts["東信"] != 15 || counts["北信"] != 15 || counts["中信"] != 19 || counts["南信"] != 28 {
		t.Fatalf("counts = %v", counts)
	}
	for name, want := range map[string]string{
		"佐久市": "東信", "軽井沢町": "東信", "上田市": "東信", "長野市": "北信", "野沢温泉村": "北信",
		"松本市": "中信", "白馬村": "中信", "松川村": "中信", "松川町": "南信", "諏訪市": "南信", "飯田市": "南信",
		"高崎市": "", "": "", "Saku": "",
	} {
		if got := NaganoArea(name); got != want {
			t.Errorf("NaganoArea(%q) = %q, want %q", name, got, want)
		}
	}
}

func TestCoordinateHelpers(t *testing.T) {
	p := PlacePoint{36.278061234, 138.464689876}
	if got := p.rounded(nearbyPositionDecimals); got != (PlacePoint{36.278, 138.465}) {
		t.Fatalf("nearby rounding = %+v", got)
	}
	if got := p.rounded(biasPositionDecimals); got != (PlacePoint{36.28, 138.46}) {
		t.Fatalf("bias rounding = %+v", got)
	}
	// Rounding to 3 decimals moves a point by at most about 70m.
	if d := distanceMeters(p, p.rounded(nearbyPositionDecimals)); d > 75 {
		t.Fatalf("rounding moved the point %vm", d)
	}
	// Sakudaira station to Karuizawa station is about 17km.
	if d := distanceMeters(PlacePoint{36.2780, 138.4647}, PlacePoint{36.3430, 138.6371}); d < 16500 || d > 17500 {
		t.Fatalf("distance = %v", d)
	}
	for _, bad := range [][2]float64{{91, 0}, {-91, 0}, {0, 181}, {0, -181}} {
		if validCoordinates(bad[0], bad[1]) {
			t.Errorf("validCoordinates(%v) = true", bad)
		}
	}
}
