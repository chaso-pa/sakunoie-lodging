package services

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
)

// Place provider names, as set in PLACE_PROVIDER. They double as the source
// of the spots registered from their results.
const (
	PlaceProviderOSM    = models.SpotSourceOSM
	PlaceProviderGoogle = models.SpotSourceGoogle
)

var (
	// ErrPlaceNotConfigured means no usable provider is selected: the name
	// is unknown, or it is google and no API key is set.
	ErrPlaceNotConfigured = errors.New("place search is not configured")
	// ErrPlaceUnavailable means the provider could not be reached or
	// answered with something unusable.
	ErrPlaceUnavailable = errors.New("place search temporarily unavailable")
	// ErrPlaceBusy means too many calls are already waiting for their turn
	// with the provider.
	ErrPlaceBusy = errors.New("place search is busy")
	// ErrPlaceNotFound means the provider knows no place with that id.
	ErrPlaceNotFound = errors.New("place not found")
)

// Place is one place as a provider describes it. Nothing in it is stored
// until a member registers it, and for Google only the id ever is.
type Place struct {
	Source       string // PlaceProviderOSM or PlaceProviderGoogle
	ExternalID   string
	Name         string // in the requested language where the provider has it
	OriginalName string // local-language name, "" when the same as Name
	Address      string // address or locality, "" when unknown
	CategoryKey  string
	Latitude     float64
	Longitude    float64
	MapURL       string
	// Area is the Nagano region (東信, 北信, 中信, 南信) the place lies in,
	// or "". Only Resolve of the OSM provider fills it.
	Area string
	// PhotoURL is reserved. No provider fills it yet: OSM has no photos
	// and Google's cost one billed request each.
	PhotoURL string
}

// PlaceSearchQuery is a keyword search. Bias, when set, prefers results
// near that point without excluding the rest.
type PlaceSearchQuery struct {
	Text     string
	Language string
	Bias     *PlacePoint
	Limit    int
}

// PlaceNearbyQuery asks for places around Center. The provider coarsens
// Center before sending it anywhere.
type PlaceNearbyQuery struct {
	Center   PlacePoint
	RadiusM  int
	Category string // a category key, or PlaceCategoryAll
	Language string
	Limit    int
}

type PlacePoint struct {
	Latitude  float64
	Longitude float64
}

// PlaceProvider is a place search backend. Every method returns
// ErrPlaceUnavailable (wrapping an error that is safe to log) when the
// upstream fails, and ErrPlaceBusy when it would have to wait too long.
type PlaceProvider interface {
	// Name is the provider's name, which is also the source of its places.
	Name() string
	// Attribution is the credit to show next to its results.
	Attribution() models.PlaceAttribution
	// Search finds shops, sights, addresses and postcodes by text.
	Search(ctx context.Context, q PlaceSearchQuery) ([]Place, error)
	// Nearby lists places around a position, nearest first.
	Nearby(ctx context.Context, q PlaceNearbyQuery) ([]Place, error)
	// Reverse names what is at a position. It returns nil, nil when the
	// provider knows nothing there.
	Reverse(ctx context.Context, point PlacePoint, language string) (*Place, error)
	// Resolve returns the place with this external id, or ErrPlaceNotFound.
	Resolve(ctx context.Context, externalID, language string) (*Place, error)
	// ValidExternalID reports whether id has the shape of this provider's
	// ids. Ids that fail are never sent upstream.
	ValidExternalID(id string) bool
}

// Category keys. They are the provider-neutral vocabulary of the API; the
// label and emoji stored on a spot come from placeCategories.
const (
	PlaceCategoryAll       = "all"
	PlaceCategoryFood      = "food"
	PlaceCategoryCafe      = "cafe"
	PlaceCategoryShop      = "shop"
	PlaceCategorySight     = "sight"
	PlaceCategoryHistory   = "history"
	PlaceCategoryNature    = "nature"
	PlaceCategoryOnsen     = "onsen"
	PlaceCategoryLodging   = "lodging"
	PlaceCategoryTransport = "transport"
	PlaceCategoryOther     = "other"
)

type placeCategory struct {
	Label string
	Emoji string
}

// placeCategories gives each key the Japanese label and emoji a spot is
// stored with. 自然 and 歴史 are the labels the seeded spots already use.
// Lodging is 宿泊施設, not 宿泊: that label belongs to the lodging itself
// and is what keeps it out of the spot ranking (lodgingCategory).
var placeCategories = map[string]placeCategory{
	PlaceCategoryFood:      {"グルメ", "🍽️"},
	PlaceCategoryCafe:      {"カフェ", "☕"},
	PlaceCategoryShop:      {"ショップ", "🛍️"},
	PlaceCategorySight:     {"観光", "📸"},
	PlaceCategoryHistory:   {"歴史", "🏯"},
	PlaceCategoryNature:    {"自然", "🌲"},
	PlaceCategoryOnsen:     {"温泉", "♨️"},
	PlaceCategoryLodging:   {"宿泊施設", "🏨"},
	PlaceCategoryTransport: {"交通", "🚉"},
	PlaceCategoryOther:     {"その他", "📍"},
}

// defaultSpotEmoji is what a spot without a known category shows.
const defaultSpotEmoji = "📍"

// PlaceCategory returns the label and emoji of a category key. Unknown
// keys read as "other".
func PlaceCategory(key string) (label, emoji string) {
	c, ok := placeCategories[key]
	if !ok {
		c = placeCategories[PlaceCategoryOther]
	}
	return c.Label, c.Emoji
}

func validPlaceCategory(key string) bool {
	_, ok := placeCategories[key]
	return ok
}

// placeCategoryKeyOfLabel is the reverse of PlaceCategory for the labels
// stored on spots. Labels it does not know (a curated spot's own) are "other".
func placeCategoryKeyOfLabel(label string) string {
	for key, c := range placeCategories {
		if c.Label == label {
			return key
		}
	}
	return PlaceCategoryOther
}

// Precision, in decimal places, of a position sent to a provider.
//
//   - Nearby search: 3 decimals, about 110m north-south and 90m east-west
//     at Saku's latitude. The member's own position is what goes out here,
//     so it is coarsened as far as a list of places within a few hundred
//     metres stays meaningful.
//   - Search bias: 2 decimals, about 1km. It only tilts the ranking.
//   - Reverse lookup: 5 decimals, about 1m. The point is one the member
//     picked on a map in order to name what stands there; a coarser point
//     names the neighbouring building instead.
const (
	nearbyPositionDecimals  = 3
	biasPositionDecimals    = 2
	reversePositionDecimals = 5
)

func roundCoordinate(v float64, decimals int) float64 {
	scale := math.Pow(10, float64(decimals))
	return math.Round(v*scale) / scale
}

func (p PlacePoint) rounded(decimals int) PlacePoint {
	return PlacePoint{roundCoordinate(p.Latitude, decimals), roundCoordinate(p.Longitude, decimals)}
}

func validCoordinates(latitude, longitude float64) bool {
	return !math.IsNaN(latitude) && !math.IsNaN(longitude) &&
		latitude >= -90 && latitude <= 90 && longitude >= -180 && longitude <= 180
}

// distanceMeters is the great-circle distance between two points.
func distanceMeters(a, b PlacePoint) float64 {
	const earthRadius = 6371000.0
	rad := func(deg float64) float64 { return deg * math.Pi / 180 }
	dLat := rad(b.Latitude - a.Latitude)
	dLon := rad(b.Longitude - a.Longitude)
	h := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rad(a.Latitude))*math.Cos(rad(b.Latitude))*math.Sin(dLon/2)*math.Sin(dLon/2)
	return 2 * earthRadius * math.Asin(math.Min(1, math.Sqrt(h)))
}

// placeBaseURL checks a configured base URL and returns it without a
// trailing slash. Requests are built as this value plus a fixed path and an
// encoded query, so user input can never choose the host.
func placeBaseURL(value, fallback string) (string, error) {
	raw := strings.TrimRight(orDefault(strings.TrimSpace(value), fallback), "/")
	u, err := url.Parse(raw)
	if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" || u.RawQuery != "" || u.Fragment != "" || u.User != nil {
		return "", errors.New("base URL must be an http(s) URL without query, fragment or credentials")
	}
	return raw, nil
}

// newPlaceHTTPClient returns a client that never follows redirects: a
// provider answering with a redirect must not be able to send the query,
// the position or the API key to another host.
func newPlaceHTTPClient(timeout time.Duration) *http.Client {
	return &http.Client{
		Timeout: timeout,
		CheckRedirect: func(*http.Request, []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

func placeUnavailable(err error) error {
	return fmt.Errorf("%w: %w", ErrPlaceUnavailable, err)
}

// truncateRunes cuts s to at most n characters.
func truncateRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) <= n {
		return s
	}
	return string(runes[:n])
}
