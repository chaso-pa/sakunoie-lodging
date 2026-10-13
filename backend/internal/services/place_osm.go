package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
)

// Default base URLs of the OpenStreetMap-based services.
const (
	DefaultNominatimBaseURL = "https://nominatim.openstreetmap.org"
	DefaultPhotonBaseURL    = "https://photon.komoot.io"
	// DefaultPlaceUserAgent identifies this application to the upstream
	// services, as Nominatim's usage policy requires. A deployment should
	// replace it (PLACE_USER_AGENT) with one that names a contact.
	DefaultPlaceUserAgent = "sakunoie-lodging-backend/1.0 (TABINOTE place search)"
)

const (
	osmPlaceTimeout = 5 * time.Second
	// osmThrottleMaxWait is how long a call may queue for its turn. Longer
	// than this and the member is told the search is busy.
	osmThrottleMaxWait  = 3 * time.Second
	osmCacheTTL         = 15 * time.Minute
	osmCacheMaxEntries  = 500
	nominatimMaxResults = 10
	// photonMaxResults is asked for on every nearby search: unnamed objects
	// and street furniture are dropped afterwards, so more than the page
	// the member sees is needed.
	photonMaxResults = 50
	// osmBiasHalfSpan is half the side, in degrees, of the box a keyword
	// search prefers results in (about 50km around the bias point).
	osmBiasHalfSpan = 0.5
)

// osmExternalID is an OSM object: N, W or R (node, way, relation) and its
// number, which is also what Nominatim's lookup endpoint takes.
var osmExternalID = regexp.MustCompile(`^[NWR][0-9]{1,19}$`)

// OSMPlaceProvider finds places in OpenStreetMap data through two public
// services:
//
//   - Nominatim for keyword search, reverse lookup and resolving one
//     object. Its policy allows searches a user triggers, at no more than
//     one request per second for the whole application, with a User-Agent
//     naming the application and with results cached; it forbids
//     autocomplete, so the client must search on submit only.
//   - Photon (komoot) for nearby search. Its terms ask for fair use and
//     promise throttling of extensive use.
//
// Every call goes through a per-service throttle and a shared short-lived
// cache, so repeated queries cost the services nothing. Overpass is not
// used: its public instances are not meant to back an application.
type OSMPlaceProvider struct {
	nominatimURL string
	photonURL    string
	userAgent    string
	http         *http.Client

	nominatimThrottle *upstreamThrottle
	photonThrottle    *upstreamThrottle
	cache             *placeCache
}

// OSMPlaceOptions configures the provider. Zero values mean the public
// services and the default User-Agent. MinInterval is the spacing between
// calls to each service (config.DefaultPlaceMinInterval in a real server);
// zero or less disables it, for a self-hosted instance and in tests.
type OSMPlaceOptions struct {
	NominatimBaseURL string
	PhotonBaseURL    string
	UserAgent        string
	MinInterval      time.Duration
}

func NewOSMPlaceProvider(opts OSMPlaceOptions) (*OSMPlaceProvider, error) {
	nominatim, err := placeBaseURL(opts.NominatimBaseURL, DefaultNominatimBaseURL)
	if err != nil {
		return nil, err
	}
	photon, err := placeBaseURL(opts.PhotonBaseURL, DefaultPhotonBaseURL)
	if err != nil {
		return nil, err
	}
	return &OSMPlaceProvider{
		nominatimURL:      nominatim,
		photonURL:         photon,
		userAgent:         orDefault(strings.TrimSpace(opts.UserAgent), DefaultPlaceUserAgent),
		http:              newPlaceHTTPClient(osmPlaceTimeout),
		nominatimThrottle: newUpstreamThrottle(opts.MinInterval, osmThrottleMaxWait),
		photonThrottle:    newUpstreamThrottle(opts.MinInterval, osmThrottleMaxWait),
		cache:             newPlaceCache(osmCacheTTL, osmCacheMaxEntries),
	}, nil
}

func (p *OSMPlaceProvider) Name() string { return PlaceProviderOSM }

func (p *OSMPlaceProvider) Attribution() models.PlaceAttribution {
	return models.PlaceAttribution{Text: "© OpenStreetMap contributors", URL: "https://www.openstreetmap.org/copyright"}
}

func (p *OSMPlaceProvider) ValidExternalID(id string) bool { return osmExternalID.MatchString(id) }

type nominatimPlace struct {
	OSMType     string            `json:"osm_type"`
	OSMID       int64             `json:"osm_id"`
	Lat         string            `json:"lat"`
	Lon         string            `json:"lon"`
	Category    string            `json:"category"`
	Type        string            `json:"type"`
	Name        string            `json:"name"`
	DisplayName string            `json:"display_name"`
	Address     map[string]string `json:"address"`
	NameDetails map[string]string `json:"namedetails"`
	Error       json.RawMessage   `json:"error"`
}

// Search asks Nominatim for q.Text within Japan.
func (p *OSMPlaceProvider) Search(ctx context.Context, q PlaceSearchQuery) ([]Place, error) {
	text := normalizeQuery(q.Text)
	bias := PlacePoint{SakuLatitude, SakuLongitude}
	if q.Bias != nil {
		bias = *q.Bias
	}
	bias = bias.rounded(biasPositionDecimals)
	key := strings.Join([]string{"search", q.Language, formatCoordinate(bias.Latitude, biasPositionDecimals), formatCoordinate(bias.Longitude, biasPositionDecimals), strings.ToLower(text)}, "|")

	places, ok := p.cache.get(key)
	if !ok {
		query := p.nominatimQuery(nominatimLanguages(q.Language))
		query.Set("q", text)
		query.Set("limit", strconv.Itoa(nominatimMaxResults))
		query.Set("countrycodes", "jp")
		// left, top, right, bottom. Without bounded=1 the box is a
		// preference, not a restriction.
		query.Set("viewbox", strings.Join([]string{
			formatCoordinate(bias.Longitude-osmBiasHalfSpan, biasPositionDecimals),
			formatCoordinate(bias.Latitude+osmBiasHalfSpan, biasPositionDecimals),
			formatCoordinate(bias.Longitude+osmBiasHalfSpan, biasPositionDecimals),
			formatCoordinate(bias.Latitude-osmBiasHalfSpan, biasPositionDecimals),
		}, ","))
		var rows []nominatimPlace
		if err := p.getJSON(ctx, p.nominatimThrottle, "nominatim", p.nominatimURL+"/search", query, &rows); err != nil {
			return nil, err
		}
		places = make([]Place, 0, len(rows))
		for i := range rows {
			if place, ok := rows[i].place(q.Language, false); ok {
				places = append(places, place)
			}
		}
		p.cache.put(key, places)
	}
	if q.Limit > 0 && len(places) > q.Limit {
		places = places[:q.Limit]
	}
	return places, nil
}

// Reverse asks Nominatim what stands at the point.
func (p *OSMPlaceProvider) Reverse(ctx context.Context, point PlacePoint, language string) (*Place, error) {
	point = point.rounded(reversePositionDecimals)
	lat := formatCoordinate(point.Latitude, reversePositionDecimals)
	lon := formatCoordinate(point.Longitude, reversePositionDecimals)
	key := strings.Join([]string{"reverse", language, lat, lon}, "|")

	places, ok := p.cache.get(key)
	if !ok {
		query := p.nominatimQuery(nominatimLanguages(language))
		query.Set("lat", lat)
		query.Set("lon", lon)
		query.Set("zoom", "18")
		var row nominatimPlace
		if err := p.getJSON(ctx, p.nominatimThrottle, "nominatim", p.nominatimURL+"/reverse", query, &row); err != nil {
			return nil, err
		}
		// Nominatim answers 200 {"error": "Unable to geocode"} for a point
		// it knows nothing about (open sea, for one).
		places = []Place{}
		if len(row.Error) == 0 {
			if place, ok := row.place(language, false); ok {
				places = append(places, place)
			}
		}
		p.cache.put(key, places)
	}
	if len(places) == 0 {
		return nil, nil
	}
	return &places[0], nil
}

// Resolve looks one object up.
//
// It always asks in Japanese, whatever the member's language: the address
// stored on the spot is then the local one, the same for every member, and
// the municipality can be matched to its Nagano region. The name in the
// member's language comes from the object's name tags in the same answer.
func (p *OSMPlaceProvider) Resolve(ctx context.Context, externalID, language string) (*Place, error) {
	if !p.ValidExternalID(externalID) {
		return nil, ErrPlaceNotFound
	}
	key := strings.Join([]string{"resolve", language, externalID}, "|")
	places, ok := p.cache.get(key)
	if !ok {
		query := p.nominatimQuery("ja")
		query.Set("osm_ids", externalID)
		var rows []nominatimPlace
		if err := p.getJSON(ctx, p.nominatimThrottle, "nominatim", p.nominatimURL+"/lookup", query, &rows); err != nil {
			return nil, err
		}
		places = []Place{}
		for i := range rows {
			if place, ok := rows[i].place(language, true); ok && place.ExternalID == externalID {
				places = append(places, place)
				break
			}
		}
		p.cache.put(key, places)
	}
	if len(places) == 0 {
		return nil, ErrPlaceNotFound
	}
	return &places[0], nil
}

type photonResponse struct {
	Features []struct {
		Geometry struct {
			Coordinates []float64 `json:"coordinates"` // longitude, latitude
		} `json:"geometry"`
		Properties struct {
			OSMType     string `json:"osm_type"`
			OSMID       int64  `json:"osm_id"`
			OSMKey      string `json:"osm_key"`
			OSMValue    string `json:"osm_value"`
			Name        string `json:"name"`
			HouseNumber string `json:"housenumber"`
			Street      string `json:"street"`
			Locality    string `json:"locality"`
			District    string `json:"district"`
			City        string `json:"city"`
			State       string `json:"state"`
		} `json:"properties"`
	} `json:"features"`
}

// Nearby asks Photon for the named places around the centre, which leaves
// the server rounded to nearbyPositionDecimals.
func (p *OSMPlaceProvider) Nearby(ctx context.Context, q PlaceNearbyQuery) ([]Place, error) {
	centre := q.Center.rounded(nearbyPositionDecimals)
	lat := formatCoordinate(centre.Latitude, nearbyPositionDecimals)
	lon := formatCoordinate(centre.Longitude, nearbyPositionDecimals)
	category := orDefault(q.Category, PlaceCategoryAll)
	photonLanguage := photonLanguage(q.Language)
	key := strings.Join([]string{"nearby", photonLanguage, lat, lon, strconv.Itoa(q.RadiusM), category}, "|")

	places, ok := p.cache.get(key)
	if !ok {
		query := url.Values{}
		query.Set("lat", lat)
		query.Set("lon", lon)
		query.Set("radius", strconv.FormatFloat(float64(q.RadiusM)/1000, 'f', 2, 64)) // km
		query.Set("limit", strconv.Itoa(photonMaxResults))
		query.Set("lang", photonLanguage)
		for _, tag := range photonTags[category] {
			query.Add("osm_tag", tag)
		}
		var parsed photonResponse
		if err := p.getJSON(ctx, p.photonThrottle, "photon", p.photonURL+"/reverse", query, &parsed); err != nil {
			return nil, err
		}

		places = []Place{}
		seen := map[string]bool{}
		for _, f := range parsed.Features {
			props := f.Properties
			name := strings.TrimSpace(props.Name)
			id := photonExternalID(props.OSMType, props.OSMID)
			if name == "" || id == "" || seen[id] || len(f.Geometry.Coordinates) != 2 {
				continue
			}
			categoryKey, noise := osmCategory(props.OSMKey, props.OSMValue)
			if noise || !categoryMatches(category, categoryKey) {
				continue
			}
			position := PlacePoint{f.Geometry.Coordinates[1], f.Geometry.Coordinates[0]}
			// Photon measures the radius from the object's centre too, but
			// a few metres of slack would let an object in and out as the
			// rounded centre moves; filter on our own distance.
			if !validCoordinates(position.Latitude, position.Longitude) || distanceMeters(centre, position) > float64(q.RadiusM) {
				continue
			}
			seen[id] = true
			places = append(places, Place{
				Source:      PlaceProviderOSM,
				ExternalID:  id,
				Name:        name,
				Address:     photonAddress(props.Street, props.HouseNumber, orDefault(props.District, props.Locality), props.City, props.State),
				CategoryKey: categoryKey,
				Latitude:    position.Latitude,
				Longitude:   position.Longitude,
				MapURL:      osmMapURL(id),
			})
		}
		sort.SliceStable(places, func(i, j int) bool {
			return distanceMeters(centre, PlacePoint{places[i].Latitude, places[i].Longitude}) <
				distanceMeters(centre, PlacePoint{places[j].Latitude, places[j].Longitude})
		})
		p.cache.put(key, places)
	}
	if q.Limit > 0 && len(places) > q.Limit {
		places = places[:q.Limit]
	}
	return places, nil
}

func (p *OSMPlaceProvider) nominatimQuery(languages string) url.Values {
	query := url.Values{}
	query.Set("format", "jsonv2")
	query.Set("addressdetails", "1")
	query.Set("namedetails", "1")
	query.Set("accept-language", languages)
	return query
}

// getJSON waits for the service's turn, sends the request and decodes a
// 200 answer into out. Its errors never contain the URL, which carries the
// search text or a position.
func (p *OSMPlaceProvider) getJSON(ctx context.Context, throttle *upstreamThrottle, service, endpoint string, query url.Values, out any) error {
	if err := throttle.wait(ctx); err != nil {
		if err == ErrPlaceBusy {
			return err
		}
		return placeUnavailable(upstreamError(service, err))
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"?"+query.Encode(), nil)
	if err != nil {
		return placeUnavailable(upstreamFormatError(service, "cannot build request"))
	}
	req.Header.Set("User-Agent", p.userAgent)
	req.Header.Set("Accept", "application/json")
	status, body, err := readUpstream(p.http, service, req)
	if err != nil {
		return placeUnavailable(err)
	}
	if status != http.StatusOK {
		return placeUnavailable(upstreamStatusError(service, status))
	}
	if err := json.Unmarshal(body, out); err != nil {
		return placeUnavailable(upstreamFormatError(service, "response is not valid JSON"))
	}
	return nil
}

// place converts a Nominatim row. resolved is true for the Japanese lookup
// behind Resolve: the name is then picked from the name tags, and the
// region is derived from the address.
func (n *nominatimPlace) place(language string, resolved bool) (Place, bool) {
	id := nominatimExternalID(n.OSMType, n.OSMID)
	lat, errLat := strconv.ParseFloat(n.Lat, 64)
	lon, errLon := strconv.ParseFloat(n.Lon, 64)
	if id == "" || errLat != nil || errLon != nil || !validCoordinates(lat, lon) {
		return Place{}, false
	}

	parts := splitDisplayName(n.DisplayName)
	name := strings.TrimSpace(n.Name)
	if name != "" && len(parts) > 0 && parts[0] == name {
		parts = parts[1:]
	}
	local := strings.TrimSpace(n.NameDetails["name"])
	if resolved {
		for _, key := range nameTagKeys(language) {
			if v := strings.TrimSpace(n.NameDetails[key]); v != "" {
				name = v
				break
			}
		}
	}
	// What is left is the address, smallest part first, ending with the
	// postcode and the country. Those two are dropped.
	address := parts
	if country := n.Address["country"]; country != "" && len(address) > 0 && address[len(address)-1] == country {
		address = address[:len(address)-1]
	}
	if postcode := n.Address["postcode"]; postcode != "" && len(address) > 0 && address[len(address)-1] == postcode {
		address = address[:len(address)-1]
	}
	japanese := resolved || language == "ja"
	if name == "" {
		// An object without a name (a house, a stretch of road): its
		// nearest address parts stand in for one.
		nearest := address[:min(2, len(address))]
		if len(nearest) == 0 {
			return Place{}, false
		}
		if japanese {
			name = nearest[len(nearest)-1]
			if len(nearest) == 2 {
				name += nearest[0]
			}
		} else {
			name = strings.Join(nearest, ", ")
		}
	}

	place := Place{
		Source:     PlaceProviderOSM,
		ExternalID: id,
		Name:       name,
		Latitude:   lat,
		Longitude:  lon,
		MapURL:     osmMapURL(id),
	}
	if local != "" && local != name {
		place.OriginalName = local
	}
	if japanese {
		// Japanese addresses read largest part first, without separators.
		for i := len(address) - 1; i >= 0; i-- {
			place.Address += address[i]
		}
	} else {
		place.Address = strings.Join(address, ", ")
	}
	place.CategoryKey, _ = osmCategory(n.Category, n.Type)
	if resolved && n.Address["ISO3166-2-lvl4"] == "JP-20" {
		for _, key := range []string{"city", "town", "village", "municipality"} {
			if area := NaganoArea(n.Address[key]); area != "" {
				place.Area = area
				break
			}
		}
	}
	return place, true
}

func splitDisplayName(displayName string) []string {
	var parts []string
	for _, part := range strings.Split(displayName, ",") {
		if part = strings.TrimSpace(part); part != "" {
			parts = append(parts, part)
		}
	}
	return parts
}

func nominatimExternalID(osmType string, id int64) string {
	if id <= 0 {
		return ""
	}
	switch osmType {
	case "node":
		return "N" + strconv.FormatInt(id, 10)
	case "way":
		return "W" + strconv.FormatInt(id, 10)
	case "relation":
		return "R" + strconv.FormatInt(id, 10)
	}
	return ""
}

func photonExternalID(osmType string, id int64) string {
	if id <= 0 || (osmType != "N" && osmType != "W" && osmType != "R") {
		return ""
	}
	return osmType + strconv.FormatInt(id, 10)
}

// osmMapURL is the object's page on openstreetmap.org. id is already known
// to match osmExternalID.
func osmMapURL(id string) string {
	kind := map[byte]string{'N': "node", 'W': "way", 'R': "relation"}[id[0]]
	return "https://www.openstreetmap.org/" + kind + "/" + id[1:]
}

// photonAddress joins the address parts Photon gives, smallest first.
//
// In Japan most addresses are block numbers rather than street names, and
// Photon then reports the block as "street" and the lot as "housenumber".
// Printed in the western order they read as noise ("2 2"), so a street
// without a letter in it is left out and the address starts at the
// neighbourhood.
func photonAddress(street, houseNumber string, parts ...string) string {
	var address []string
	street = strings.TrimSpace(street)
	if strings.IndexFunc(street, unicode.IsLetter) >= 0 {
		address = append(address, strings.TrimSpace(street+" "+strings.TrimSpace(houseNumber)))
	}
	for _, part := range parts {
		if part = strings.TrimSpace(part); part != "" {
			address = append(address, part)
		}
	}
	return strings.Join(address, ", ")
}

// nominatimLanguages is the accept-language list for a member's language.
// Nominatim falls back to the local name by itself when none matches.
func nominatimLanguages(language string) string {
	switch language {
	case "en":
		return "en"
	case "ko":
		return "ko,en"
	case "zh":
		return "zh,zh-Hans,zh-Hant"
	}
	return "ja"
}

// nameTagKeys lists the OSM name tags to prefer for a language.
func nameTagKeys(language string) []string {
	switch language {
	case "en":
		return []string{"name:en"}
	case "ko":
		return []string{"name:ko", "name:en"}
	case "zh":
		return []string{"name:zh", "name:zh-Hans", "name:zh-Hant"}
	}
	return []string{"name:ja", "name"}
}

// photonLanguage maps a member's language to one the public Photon
// instance has (default, en, de, fr). "default" is the local name, which a
// Chinese reader can read as it is written; Korean falls back to English.
func photonLanguage(language string) string {
	if language == "en" || language == "ko" {
		return "en"
	}
	return "default"
}

// photonTags are the osm_tag filters sent for each category. A bare key
// matches every value of that key; several filters are alternatives.
var photonTags = map[string][]string{
	PlaceCategoryAll:  {"amenity", "tourism", "shop", "historic", "leisure", "natural"},
	PlaceCategoryFood: {"amenity:restaurant", "amenity:fast_food", "amenity:pub", "amenity:bar", "amenity:food_court", "amenity:ice_cream", "amenity:biergarten"},
	PlaceCategoryCafe: {"amenity:cafe"},
	PlaceCategoryShop: {"shop"},
	PlaceCategorySight: {"tourism:attraction", "tourism:museum", "tourism:gallery", "tourism:viewpoint", "tourism:theme_park",
		"tourism:zoo", "tourism:aquarium", "tourism:artwork", "historic", "amenity:place_of_worship"},
	PlaceCategoryNature:    {"leisure:park", "leisure:garden", "leisure:nature_reserve", "natural"},
	PlaceCategoryOnsen:     {"amenity:public_bath", "natural:hot_spring"},
	PlaceCategoryLodging:   {"tourism:hotel", "tourism:guest_house", "tourism:hostel", "tourism:motel", "tourism:chalet", "tourism:alpine_hut", "tourism:camp_site"},
	PlaceCategoryTransport: {"railway:station", "railway:halt", "amenity:bus_station", "highway:bus_stop"},
}

// categoryMatches reports whether a place of category got belongs in a
// nearby search for want. Sights include historic places.
func categoryMatches(want, got string) bool {
	switch want {
	case PlaceCategoryAll, "":
		return true
	case PlaceCategorySight:
		return got == PlaceCategorySight || got == PlaceCategoryHistory
	}
	return want == got
}

// osmNoise are objects nobody posts about. They are dropped from nearby
// results; a keyword search can still find them.
var osmNoise = map[string]bool{
	"amenity:parking": true, "amenity:parking_space": true, "amenity:parking_entrance": true,
	"amenity:bicycle_parking": true, "amenity:motorcycle_parking": true, "amenity:toilets": true,
	"amenity:bench": true, "amenity:vending_machine": true, "amenity:waste_basket": true,
	"amenity:waste_disposal": true, "amenity:recycling": true, "amenity:atm": true,
	"amenity:post_box": true, "amenity:telephone": true, "amenity:drinking_water": true,
	"amenity:shelter": true, "amenity:fuel": true, "amenity:charging_station": true,
	"amenity:car_wash": true, "amenity:school": true, "amenity:kindergarten": true,
	"amenity:childcare": true, "amenity:clinic": true, "amenity:doctors": true,
	"amenity:dentist": true, "amenity:hospital": true, "amenity:social_facility": true,
	"amenity:nursing_home": true, "amenity:grave_yard": true, "amenity:fire_station": true,
	"tourism:information": true, "leisure:pitch": true, "leisure:playground": true,
	"natural:tree": true, "natural:tree_row": true, "natural:shrub": true,
}

// osmCategory maps an OSM tag to a category key, and says whether the
// object is noise for a nearby search.
func osmCategory(key, value string) (category string, noise bool) {
	noise = osmNoise[key+":"+value]
	switch key {
	case "amenity":
		switch value {
		case "restaurant", "fast_food", "food_court", "pub", "bar", "biergarten", "ice_cream":
			return PlaceCategoryFood, noise
		case "cafe":
			return PlaceCategoryCafe, noise
		case "public_bath":
			return PlaceCategoryOnsen, noise
		case "marketplace":
			return PlaceCategoryShop, noise
		case "bus_station", "ferry_terminal", "taxi":
			return PlaceCategoryTransport, noise
		case "place_of_worship":
			return PlaceCategoryHistory, noise
		case "theatre", "cinema", "arts_centre", "library", "planetarium":
			return PlaceCategorySight, noise
		}
	case "tourism":
		switch value {
		case "hotel", "guest_house", "hostel", "motel", "chalet", "alpine_hut", "camp_site", "caravan_site", "apartment":
			return PlaceCategoryLodging, noise
		case "picnic_site":
			return PlaceCategoryNature, noise
		case "information":
			return PlaceCategoryOther, noise
		}
		return PlaceCategorySight, noise
	case "shop":
		return PlaceCategoryShop, noise
	case "historic":
		return PlaceCategoryHistory, noise
	case "leisure":
		switch value {
		case "park", "garden", "nature_reserve":
			return PlaceCategoryNature, noise
		}
	case "natural":
		if value == "hot_spring" {
			return PlaceCategoryOnsen, noise
		}
		return PlaceCategoryNature, noise
	case "railway":
		switch value {
		case "station", "halt", "tram_stop":
			return PlaceCategoryTransport, noise
		}
	case "highway":
		if value == "bus_stop" {
			return PlaceCategoryTransport, noise
		}
	case "public_transport", "aeroway":
		return PlaceCategoryTransport, noise
	}
	return PlaceCategoryOther, noise
}

// normalizeQuery trims the text and collapses runs of white space.
func normalizeQuery(text string) string {
	return strings.Join(strings.Fields(text), " ")
}

func formatCoordinate(v float64, decimals int) string {
	return strconv.FormatFloat(roundCoordinate(v, decimals), 'f', decimals, 64)
}
