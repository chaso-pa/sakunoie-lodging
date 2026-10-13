package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
)

// Default base URLs of the Google Maps Platform services.
const (
	DefaultGooglePlacesBaseURL    = "https://places.googleapis.com"
	DefaultGoogleGeocodingBaseURL = "https://geocode.googleapis.com"
)

const (
	googlePlaceTimeout = 5 * time.Second
	googleRegionCode   = "JP"
	// googleBiasRadiusM is the radius of the circle a text search prefers
	// results in. 50km is the largest the API accepts.
	googleBiasRadiusM = 50000.0

	// Field masks. A request is billed at the highest tier among the
	// fields it names, and displayName, primaryType and googleMapsUri are
	// all Pro fields: a list of places is of no use without names, so
	// every call below is a Pro call and nothing from the Enterprise tier
	// (ratings, opening hours, phone numbers) is ever asked for. Photos are
	// not requested either: each one shown would be a further billed call.
	googlePlaceFields  = "id,displayName,formattedAddress,location,primaryType,googleMapsUri"
	googleSearchFields = "places.id,places.displayName,places.formattedAddress,places.location,places.primaryType,places.googleMapsUri"
)

// googlePlaceID is the shape of a place ID. The ID becomes a path segment
// of the Place Details URL, so nothing else is let through.
var googlePlaceID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,512}$`)

// ErrGooglePlacesKeyMissing is returned when no server-side key is set.
var ErrGooglePlacesKeyMissing = errors.New("google places: API key is not configured")

// GooglePlaceProvider finds places through Places API (New) and names map
// positions through Geocoding API v4.
//
// Nothing it returns may be kept: Google's terms allow storing a place ID
// and nothing else from a result. This type therefore has no cache, and
// PlaceService stores only the ID of a spot registered from it.
//
// It was written against the published request and response formats and
// is tested against stub servers only; it has not been run against the
// real services.
type GooglePlaceProvider struct {
	placesURL    string
	geocodingURL string
	apiKey       string
	http         *http.Client
}

type GooglePlaceOptions struct {
	APIKey           string
	PlacesBaseURL    string
	GeocodingBaseURL string
}

func NewGooglePlaceProvider(opts GooglePlaceOptions) (*GooglePlaceProvider, error) {
	key := strings.TrimSpace(opts.APIKey)
	if key == "" {
		return nil, ErrGooglePlacesKeyMissing
	}
	places, err := placeBaseURL(opts.PlacesBaseURL, DefaultGooglePlacesBaseURL)
	if err != nil {
		return nil, err
	}
	geocoding, err := placeBaseURL(opts.GeocodingBaseURL, DefaultGoogleGeocodingBaseURL)
	if err != nil {
		return nil, err
	}
	return &GooglePlaceProvider{
		placesURL:    places,
		geocodingURL: geocoding,
		apiKey:       key,
		http:         newPlaceHTTPClient(googlePlaceTimeout),
	}, nil
}

func (p *GooglePlaceProvider) Name() string { return PlaceProviderGoogle }

func (p *GooglePlaceProvider) Attribution() models.PlaceAttribution {
	return models.PlaceAttribution{Text: "Powered by Google", URL: "https://www.google.com/maps"}
}

func (p *GooglePlaceProvider) ValidExternalID(id string) bool { return googlePlaceID.MatchString(id) }

type googleLatLng struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type googlePlace struct {
	ID          string `json:"id"`
	DisplayName *struct {
		Text string `json:"text"`
	} `json:"displayName"`
	FormattedAddress string        `json:"formattedAddress"`
	Location         *googleLatLng `json:"location"`
	PrimaryType      string        `json:"primaryType"`
	GoogleMapsURI    string        `json:"googleMapsUri"`
}

type googleCircle struct {
	Center googleLatLng `json:"center"`
	Radius float64      `json:"radius"`
}

type googleArea struct {
	Circle googleCircle `json:"circle"`
}

type googleSearchResponse struct {
	Places []googlePlace `json:"places"`
}

// Search calls Text Search (New).
func (p *GooglePlaceProvider) Search(ctx context.Context, q PlaceSearchQuery) ([]Place, error) {
	bias := PlacePoint{SakuLatitude, SakuLongitude}
	if q.Bias != nil {
		bias = *q.Bias
	}
	bias = bias.rounded(biasPositionDecimals)
	body := struct {
		TextQuery    string     `json:"textQuery"`
		LanguageCode string     `json:"languageCode"`
		RegionCode   string     `json:"regionCode"`
		PageSize     int        `json:"pageSize"`
		LocationBias googleArea `json:"locationBias"`
	}{
		TextQuery:    normalizeQuery(q.Text),
		LanguageCode: googleLanguage(q.Language),
		RegionCode:   googleRegionCode,
		PageSize:     clamp(q.Limit, 1, 20),
		LocationBias: googleArea{googleCircle{googleLatLng{bias.Latitude, bias.Longitude}, googleBiasRadiusM}},
	}
	var parsed googleSearchResponse
	if _, err := p.call(ctx, http.MethodPost, p.placesURL+"/v1/places:searchText", nil, googleSearchFields, body, &parsed); err != nil {
		return nil, err
	}
	return googlePlaces(parsed.Places), nil
}

// Nearby calls Nearby Search (New) around the centre, which leaves the
// server rounded to nearbyPositionDecimals.
func (p *GooglePlaceProvider) Nearby(ctx context.Context, q PlaceNearbyQuery) ([]Place, error) {
	centre := q.Center.rounded(nearbyPositionDecimals)
	body := struct {
		LocationRestriction googleArea `json:"locationRestriction"`
		MaxResultCount      int        `json:"maxResultCount"`
		RankPreference      string     `json:"rankPreference"`
		LanguageCode        string     `json:"languageCode"`
		RegionCode          string     `json:"regionCode"`
		IncludedTypes       []string   `json:"includedTypes,omitempty"`
	}{
		LocationRestriction: googleArea{googleCircle{googleLatLng{centre.Latitude, centre.Longitude}, float64(q.RadiusM)}},
		MaxResultCount:      clamp(q.Limit, 1, 20),
		// Without a type filter, the nearest objects are mostly of no
		// interest; let Google pick the prominent ones in the circle.
		RankPreference: "POPULARITY",
		LanguageCode:   googleLanguage(q.Language),
		RegionCode:     googleRegionCode,
		IncludedTypes:  googleNearbyTypes[q.Category],
	}
	if len(body.IncludedTypes) > 0 {
		body.RankPreference = "DISTANCE"
	}
	var parsed googleSearchResponse
	if _, err := p.call(ctx, http.MethodPost, p.placesURL+"/v1/places:searchNearby", nil, googleSearchFields, body, &parsed); err != nil {
		return nil, err
	}
	places := googlePlaces(parsed.Places)
	sort.SliceStable(places, func(i, j int) bool {
		return distanceMeters(centre, PlacePoint{places[i].Latitude, places[i].Longitude}) <
			distanceMeters(centre, PlacePoint{places[j].Latitude, places[j].Longitude})
	})
	return places, nil
}

// Reverse calls Geocoding API v4 for the point. What comes back is an
// address with a place ID rather than a shop or a sight, which is what
// "this spot on the map" means.
func (p *GooglePlaceProvider) Reverse(ctx context.Context, point PlacePoint, language string) (*Place, error) {
	point = point.rounded(reversePositionDecimals)
	query := url.Values{}
	query.Set("location.latitude", formatCoordinate(point.Latitude, reversePositionDecimals))
	query.Set("location.longitude", formatCoordinate(point.Longitude, reversePositionDecimals))
	query.Set("languageCode", googleLanguage(language))
	query.Set("regionCode", googleRegionCode)
	var parsed struct {
		Results []struct {
			PlaceID          string        `json:"placeId"`
			FormattedAddress string        `json:"formattedAddress"`
			Location         *googleLatLng `json:"location"`
		} `json:"results"`
	}
	if _, err := p.call(ctx, http.MethodGet, p.geocodingURL+"/v4/geocode/location", query, "", nil, &parsed); err != nil {
		return nil, err
	}
	for _, r := range parsed.Results {
		if !p.ValidExternalID(r.PlaceID) || strings.TrimSpace(r.FormattedAddress) == "" {
			continue
		}
		place := &Place{
			Source:      PlaceProviderGoogle,
			ExternalID:  r.PlaceID,
			Name:        r.FormattedAddress,
			CategoryKey: PlaceCategoryOther,
			Latitude:    point.Latitude,
			Longitude:   point.Longitude,
			MapURL:      GoogleMapsPlaceURL(r.PlaceID),
		}
		if r.Location != nil && validCoordinates(r.Location.Latitude, r.Location.Longitude) {
			place.Latitude, place.Longitude = r.Location.Latitude, r.Location.Longitude
		}
		return place, nil
	}
	return nil, nil
}

// Resolve calls Place Details (New).
func (p *GooglePlaceProvider) Resolve(ctx context.Context, externalID, language string) (*Place, error) {
	if !p.ValidExternalID(externalID) {
		return nil, ErrPlaceNotFound
	}
	query := url.Values{}
	query.Set("languageCode", googleLanguage(language))
	query.Set("regionCode", googleRegionCode)
	var parsed googlePlace
	status, err := p.call(ctx, http.MethodGet, p.placesURL+"/v1/places/"+url.PathEscape(externalID), query, googlePlaceFields, nil, &parsed)
	if err != nil {
		// 400 is an ID Google cannot parse, 404 one it does not know.
		if status == http.StatusBadRequest || status == http.StatusNotFound {
			return nil, ErrPlaceNotFound
		}
		return nil, err
	}
	places := googlePlaces([]googlePlace{parsed})
	if len(places) == 0 {
		return nil, ErrPlaceNotFound
	}
	return &places[0], nil
}

// call sends one request and decodes a 200 answer into out. The key
// travels in a header, never in the URL. The returned status is 0 when no
// answer arrived. Errors carry neither the URL nor any upstream text.
func (p *GooglePlaceProvider) call(ctx context.Context, method, endpoint string, query url.Values, fieldMask string, body, out any) (int, error) {
	const service = "google places"
	var reader *bytes.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return 0, placeUnavailable(upstreamFormatError(service, "cannot build request"))
		}
		reader = bytes.NewReader(payload)
	} else {
		reader = bytes.NewReader(nil)
	}
	if len(query) > 0 {
		endpoint += "?" + query.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return 0, placeUnavailable(upstreamFormatError(service, "cannot build request"))
	}
	req.Header.Set("X-Goog-Api-Key", p.apiKey)
	if fieldMask != "" {
		req.Header.Set("X-Goog-FieldMask", fieldMask)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	status, respBody, err := readUpstream(p.http, service, req)
	if err != nil {
		return 0, placeUnavailable(err)
	}
	if status != http.StatusOK {
		return status, placeUnavailable(upstreamStatusError(service, status))
	}
	if err := json.Unmarshal(respBody, out); err != nil {
		return status, placeUnavailable(upstreamFormatError(service, "response is not valid JSON"))
	}
	return status, nil
}

func googlePlaces(rows []googlePlace) []Place {
	places := make([]Place, 0, len(rows))
	for _, r := range rows {
		if !googlePlaceID.MatchString(r.ID) || r.DisplayName == nil || strings.TrimSpace(r.DisplayName.Text) == "" ||
			r.Location == nil || !validCoordinates(r.Location.Latitude, r.Location.Longitude) {
			continue
		}
		place := Place{
			Source:      PlaceProviderGoogle,
			ExternalID:  r.ID,
			Name:        strings.TrimSpace(r.DisplayName.Text),
			Address:     strings.TrimSpace(r.FormattedAddress),
			CategoryKey: googleCategory(r.PrimaryType),
			Latitude:    r.Location.Latitude,
			Longitude:   r.Location.Longitude,
			MapURL:      GoogleMapsPlaceURL(r.ID),
		}
		// Only a link to Google Maps itself is passed on.
		if u, err := url.Parse(r.GoogleMapsURI); err == nil && u.Scheme == "https" && isGoogleMapsHost(u.Hostname()) {
			place.MapURL = r.GoogleMapsURI
		}
		places = append(places, place)
	}
	return places
}

func isGoogleMapsHost(host string) bool {
	return host == "maps.google.com" || host == "www.google.com" || host == "google.com" || host == "maps.app.goo.gl"
}

// GoogleMapsPlaceURL links to a place on Google Maps from its ID alone, so
// a stored Google spot has a map link even when its details cannot be
// fetched. id must be a valid place ID.
func GoogleMapsPlaceURL(id string) string {
	return "https://www.google.com/maps/place/?q=place_id:" + url.QueryEscape(id)
}

// googleLanguage maps a member's language to a Google language code.
func googleLanguage(language string) string {
	switch language {
	case "en", "ko":
		return language
	case "zh":
		return "zh-CN"
	}
	return "ja"
}

// googleNearbyTypes are the place types (Table A of the Places API) asked
// for in each category. "all" sends no filter.
var googleNearbyTypes = map[string][]string{
	PlaceCategoryFood:      {"restaurant"},
	PlaceCategoryCafe:      {"cafe"},
	PlaceCategoryShop:      {"store"},
	PlaceCategorySight:     {"tourist_attraction", "museum"},
	PlaceCategoryNature:    {"park", "national_park"},
	PlaceCategoryOnsen:     {"spa"},
	PlaceCategoryLodging:   {"lodging"},
	PlaceCategoryTransport: {"train_station", "bus_station"},
}

// googleCategory maps a primary type to a category key.
func googleCategory(primaryType string) string {
	t := primaryType
	switch {
	case t == "":
		return PlaceCategoryOther
	case t == "cafe", t == "coffee_shop", t == "tea_house", t == "bakery":
		return PlaceCategoryCafe
	case strings.HasSuffix(t, "restaurant"), t == "bar", t == "pub", t == "food_court", t == "meal_takeaway", t == "ice_cream_shop":
		return PlaceCategoryFood
	case t == "spa", t == "sauna", t == "public_bath", t == "onsen":
		return PlaceCategoryOnsen
	case t == "lodging", strings.HasSuffix(t, "hotel"), t == "hostel", t == "guest_house", t == "motel", t == "inn",
		t == "japanese_inn", t == "bed_and_breakfast", t == "campground", t == "cottage":
		return PlaceCategoryLodging
	case strings.HasSuffix(t, "station"), t == "airport", t == "bus_stop", t == "ferry_terminal":
		return PlaceCategoryTransport
	case t == "historical_landmark", t == "historical_place", t == "cultural_landmark", t == "monument",
		t == "place_of_worship", t == "shinto_shrine", t == "buddhist_temple", t == "church", t == "castle":
		return PlaceCategoryHistory
	case t == "park", t == "national_park", t == "state_park", t == "garden", t == "botanical_garden", t == "hiking_area",
		t == "beach", t == "wildlife_park", t == "nature_preserve":
		return PlaceCategoryNature
	case t == "tourist_attraction", t == "museum", t == "art_gallery", t == "zoo", t == "aquarium", t == "amusement_park",
		t == "observation_deck", t == "visitor_center":
		return PlaceCategorySight
	case strings.HasSuffix(t, "store"), strings.HasSuffix(t, "shop"), t == "shopping_mall", t == "supermarket", t == "market":
		return PlaceCategoryShop
	}
	return PlaceCategoryOther
}

func clamp(v, low, high int) int {
	return max(low, min(v, high))
}
