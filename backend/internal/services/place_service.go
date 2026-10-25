package services

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

const (
	defaultPlaceSearchLimit = 8
	maxPlaceSearchLimit     = 10
	defaultPlaceNearbyLimit = 15
	maxPlaceNearbyLimit     = 20
	defaultNearbyRadiusM    = 800
	minNearbyRadiusM        = 100
	maxNearbyRadiusM        = 2000
	maxPlaceQueryLength     = 100
	// localSpotMatches is how many already registered spots a keyword
	// search puts ahead of the provider's results.
	localSpotMatches = 5

	maxSpotNameLength         = 100
	maxSpotOriginalNameLength = 200
	maxManualAddressLength    = 200
	maxSpotAddressLength      = 300
	maxSpotMapURLLength       = 500

	// placeResolveTimeout bounds the whole of one response's lookups of
	// Google spots, and placeResolveWorkers how many run at once.
	placeResolveTimeout = 4 * time.Second
	placeResolveWorkers = 4
)

// PlaceProviderConfig selects and configures the place provider.
type PlaceProviderConfig struct {
	// Provider is PLACE_PROVIDER: "osm" (also when empty) or "google".
	// Anything else leaves place search unconfigured.
	Provider string
	OSM      OSMPlaceOptions
	Google   GooglePlaceOptions
}

// PlaceProviders is the provider searches go to, and the providers that
// can resolve the spots already stored, by source. A Google spot stays
// readable after the search provider is switched back to OSM as long as
// the key is still configured.
type PlaceProviders struct {
	active   PlaceProvider
	bySource map[string]PlaceProvider
}

// NewPlaceProviders builds the providers the configuration allows. Google
// is only built with a key, and is never called without one.
//
// The returned error says why no search provider is active; the providers
// are usable either way, and every search then answers
// ErrPlaceNotConfigured.
func NewPlaceProviders(cfg PlaceProviderConfig) (*PlaceProviders, error) {
	p := &PlaceProviders{bySource: map[string]PlaceProvider{}}
	if google, err := NewGooglePlaceProvider(cfg.Google); err == nil {
		p.bySource[PlaceProviderGoogle] = google
	} else if !errors.Is(err, ErrGooglePlacesKeyMissing) {
		slog.Warn("google place provider is disabled", "error", err)
	}

	name := strings.ToLower(strings.TrimSpace(cfg.Provider))
	switch name {
	case "", PlaceProviderOSM:
		osm, err := NewOSMPlaceProvider(cfg.OSM)
		if err != nil {
			return p, fmt.Errorf("%w: osm: %w", ErrPlaceNotConfigured, err)
		}
		p.bySource[PlaceProviderOSM] = osm
		p.active = osm
	case PlaceProviderGoogle:
		google, ok := p.bySource[PlaceProviderGoogle]
		if !ok {
			return p, fmt.Errorf("%w: google needs GOOGLE_MAPS_API_KEY", ErrPlaceNotConfigured)
		}
		p.active = google
	default:
		return p, fmt.Errorf("%w: unknown provider", ErrPlaceNotConfigured)
	}
	return p, nil
}

// NewPlaceProvidersOf wires given providers directly: active answers
// searches, and each one resolves the spots of its own source.
func NewPlaceProvidersOf(active PlaceProvider, others ...PlaceProvider) *PlaceProviders {
	p := &PlaceProviders{active: active, bySource: map[string]PlaceProvider{}}
	for _, provider := range append([]PlaceProvider{active}, others...) {
		if provider != nil {
			p.bySource[provider.Name()] = provider
		}
	}
	return p
}

// PlaceService is place search, the registration of spots from its results
// or from a member's own entry, and the reading of stored spots.
type PlaceService struct {
	db        *gorm.DB
	providers *PlaceProviders
}

func NewPlaceService(db *gorm.DB, providers *PlaceProviders) *PlaceService {
	if providers == nil {
		providers = &PlaceProviders{bySource: map[string]PlaceProvider{}}
	}
	return &PlaceService{db: db, providers: providers}
}

// PlaceResults is the answer to a search.
type PlaceResults struct {
	Results     []models.PlaceResult
	Provider    string
	Attribution models.PlaceAttribution
	// NeedsGoogleMap is set by Reverse when it did not ask Google because
	// the point is not on a Google map.
	NeedsGoogleMap bool
}

func (s *PlaceService) provider() (PlaceProvider, error) {
	if s.providers.active == nil {
		return nil, ErrPlaceNotConfigured
	}
	return s.providers.active, nil
}

// language returns the requested language, or the member's own when none
// was requested.
func (s *PlaceService) language(ctx context.Context, memberID, requested string) (string, error) {
	if requested != "" {
		language, err := ValidateLanguage(requested)
		if err != nil {
			return "", invalid(err.Error())
		}
		return language, nil
	}
	var member models.Member
	if err := s.db.WithContext(ctx).Select("preferred_language").Where("id = ?", memberID).First(&member).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return defaultLanguage, nil
		}
		return "", fmt.Errorf("get member language: %w", err)
	}
	if !supportedLanguages[member.PreferredLanguage] {
		return defaultLanguage, nil
	}
	return member.PreferredLanguage, nil
}

// PlaceSearchParams is a keyword search as the handler read it.
type PlaceSearchParams struct {
	Query     string
	Language  string
	Latitude  *float64
	Longitude *float64
	Limit     int
}

// Search finds places by text: first the spots already registered whose
// name or address matches, then the provider's results.
func (s *PlaceService) Search(ctx context.Context, memberID string, p PlaceSearchParams) (*PlaceResults, error) {
	text := normalizeQuery(p.Query)
	if text == "" || utf8.RuneCountInString(text) > maxPlaceQueryLength {
		return nil, invalid("query must be 1 to 100 characters")
	}
	if (p.Latitude == nil) != (p.Longitude == nil) {
		return nil, invalid("latitude and longitude must be given together")
	}
	query := PlaceSearchQuery{Text: text, Limit: maxPlaceSearchLimit}
	if p.Latitude != nil {
		if !validCoordinates(*p.Latitude, *p.Longitude) {
			return nil, invalid("latitude or longitude is out of range")
		}
		query.Bias = &PlacePoint{*p.Latitude, *p.Longitude}
	}
	limit := p.Limit
	if limit <= 0 {
		limit = defaultPlaceSearchLimit
	}
	limit = min(limit, maxPlaceSearchLimit)

	provider, err := s.provider()
	if err != nil {
		return nil, err
	}
	if query.Language, err = s.language(ctx, memberID, p.Language); err != nil {
		return nil, err
	}
	places, err := provider.Search(ctx, query)
	if err != nil {
		return nil, err
	}
	results, err := s.results(ctx, places)
	if err != nil {
		return nil, err
	}

	local, err := s.matchingSpots(ctx, memberID, text)
	if err != nil {
		return nil, err
	}
	merged := make([]models.PlaceResult, 0, len(local)+len(results))
	listed := map[string]bool{}
	for i := range local {
		merged = append(merged, spotPlaceResult(&local[i]))
		listed[local[i].ID] = true
	}
	for _, r := range results {
		if r.SpotID != nil && listed[*r.SpotID] {
			continue
		}
		merged = append(merged, r)
	}
	if len(merged) > limit {
		merged = merged[:limit]
	}
	return &PlaceResults{Results: merged, Provider: provider.Name(), Attribution: provider.Attribution()}, nil
}

// PlaceNearbyParams is a nearby search as the handler read it.
type PlaceNearbyParams struct {
	Latitude  float64
	Longitude float64
	RadiusM   int
	Category  string
	Language  string
	Limit     int
}

// Nearby lists places around the member's position, nearest first.
//
// The position is used for this one call and is neither stored nor logged.
// The provider receives it rounded; the distances in the answer are
// measured here, from the position as given.
func (s *PlaceService) Nearby(ctx context.Context, memberID string, p PlaceNearbyParams) (*PlaceResults, error) {
	if !validCoordinates(p.Latitude, p.Longitude) {
		return nil, invalid("latitude or longitude is out of range")
	}
	radius := p.RadiusM
	if radius == 0 {
		radius = defaultNearbyRadiusM
	}
	if radius < minNearbyRadiusM || radius > maxNearbyRadiusM {
		return nil, invalid("radius_m must be between 100 and 2000")
	}
	category := orDefault(p.Category, PlaceCategoryAll)
	if category != PlaceCategoryAll && (!validPlaceCategory(category) || category == PlaceCategoryOther || category == PlaceCategoryHistory) {
		return nil, invalid("category is not one of the supported values")
	}
	limit := p.Limit
	if limit <= 0 {
		limit = defaultPlaceNearbyLimit
	}
	limit = min(limit, maxPlaceNearbyLimit)

	provider, err := s.provider()
	if err != nil {
		return nil, err
	}
	language, err := s.language(ctx, memberID, p.Language)
	if err != nil {
		return nil, err
	}
	here := PlacePoint{p.Latitude, p.Longitude}
	places, err := provider.Nearby(ctx, PlaceNearbyQuery{Center: here, RadiusM: radius, Category: category, Language: language, Limit: maxPlaceNearbyLimit})
	if err != nil {
		return nil, err
	}
	results, err := s.results(ctx, places)
	if err != nil {
		return nil, err
	}
	for i := range results {
		d := int(distanceMeters(here, PlacePoint{places[i].Latitude, places[i].Longitude}) + 0.5)
		results[i].DistanceM = &d
	}
	sort.SliceStable(results, func(i, j int) bool { return *results[i].DistanceM < *results[j].DistanceM })
	if len(results) > limit {
		results = results[:limit]
	}
	return &PlaceResults{Results: results, Provider: provider.Name(), Attribution: provider.Attribution()}, nil
}

// Reverse names what is at a point picked on the map. The result is nil
// when the provider knows nothing there.
//
// onGoogleMap tells whether the point was picked on a Google map. Google's
// terms forbid showing its content on any other map, so with the Google
// provider and another map nothing is asked and the result is nil with
// NeedsGoogleMap set; the member can still register the point by hand.
func (s *PlaceService) Reverse(ctx context.Context, memberID string, latitude, longitude float64, requestedLanguage string, onGoogleMap bool) (*models.PlaceResult, *PlaceResults, error) {
	if !validCoordinates(latitude, longitude) {
		return nil, nil, invalid("latitude or longitude is out of range")
	}
	provider, err := s.provider()
	if err != nil {
		return nil, nil, err
	}
	language, err := s.language(ctx, memberID, requestedLanguage)
	if err != nil {
		return nil, nil, err
	}
	meta := &PlaceResults{Provider: provider.Name(), Attribution: provider.Attribution()}
	if provider.Name() == PlaceProviderGoogle && !onGoogleMap {
		meta.NeedsGoogleMap = true
		return nil, meta, nil
	}
	place, err := provider.Reverse(ctx, PlacePoint{latitude, longitude}, language)
	if err != nil {
		return nil, nil, err
	}
	if place == nil {
		return nil, meta, nil
	}
	results, err := s.results(ctx, []Place{*place})
	if err != nil {
		return nil, nil, err
	}
	return &results[0], meta, nil
}

// results converts provider places, marking the ones already registered
// as spots. The order and length of places are kept.
func (s *PlaceService) results(ctx context.Context, places []Place) ([]models.PlaceResult, error) {
	keys := make([]string, len(places))
	for i, p := range places {
		keys[i] = externalKey(p.Source, p.ExternalID)
	}
	registered := map[string]string{}
	if len(keys) > 0 {
		var rows []models.Spot
		if err := s.db.WithContext(ctx).Select("id, external_key").Where("external_key IN ?", keys).Find(&rows).Error; err != nil {
			return nil, fmt.Errorf("find registered spots: %w", err)
		}
		for _, r := range rows {
			if r.ExternalKey != nil {
				registered[*r.ExternalKey] = r.ID
			}
		}
	}

	out := make([]models.PlaceResult, len(places))
	for i, p := range places {
		label, emoji := PlaceCategory(p.CategoryKey)
		id, lat, lon := p.ExternalID, p.Latitude, p.Longitude
		r := models.PlaceResult{
			Source:      p.Source,
			ExternalID:  &id,
			Name:        p.Name,
			Address:     p.Address,
			Category:    label,
			CategoryKey: orDefault(p.CategoryKey, PlaceCategoryOther),
			Emoji:       emoji,
			Latitude:    &lat,
			Longitude:   &lon,
		}
		if p.OriginalName != "" {
			original := p.OriginalName
			r.OriginalName = &original
		}
		if p.MapURL != "" {
			mapURL := p.MapURL
			r.MapURL = &mapURL
		}
		if spotID, ok := registered[keys[i]]; ok {
			r.SpotID = &spotID
		}
		out[i] = r
	}
	return out, nil
}

// matchingSpots returns the stored spots whose name or address contains
// text: the curated ones, the ones registered from OpenStreetMap, and the
// hand-entered ones, as far as the member may see them (visibleSpots): a
// hand-entered spot that no post uses is only shown to the member who
// entered it.
//
// Google spots are never matched: their names are not stored.
func (s *PlaceService) matchingSpots(ctx context.Context, memberID, text string) ([]models.Spot, error) {
	like := containsPattern(text)
	var spots []models.Spot
	err := s.db.WithContext(ctx).
		Where("source <> ?", models.SpotSourceGoogle).
		Where("(name LIKE ? OR original_name LIKE ? OR address LIKE ?)", like, like, like).
		Scopes(visibleSpots(memberID)).
		Order("created_at, id").Limit(localSpotMatches).Find(&spots).Error
	if err != nil {
		return nil, fmt.Errorf("search spots: %w", err)
	}
	return spots, nil
}

// spotPlaceResult shows a stored (non-Google) spot as a search result.
func spotPlaceResult(s *models.Spot) models.PlaceResult {
	id := s.ID
	key := placeCategoryKeyOfLabel(s.Category)
	r := models.PlaceResult{
		Source:       s.Source,
		ExternalID:   s.ExternalID,
		SpotID:       &id,
		Name:         s.Name,
		OriginalName: s.OriginalName,
		Category:     s.Category,
		CategoryKey:  key,
		Emoji:        s.Emoji,
		Latitude:     s.Latitude,
		Longitude:    s.Longitude,
		MapURL:       s.MapURL,
	}
	if s.Address != nil {
		r.Address = *s.Address
	}
	return r
}

// externalKey is the value of spots.external_key: what makes one external
// place one row.
func externalKey(source, externalID string) string {
	sum := sha256.Sum256([]byte(source + ":" + externalID))
	return hex.EncodeToString(sum[:])
}

// RegisterSpotParams is a "use this place" request. Source decides which
// fields are read: ExternalID and Language for a provider result, the rest
// for a place the member typed in.
type RegisterSpotParams struct {
	Source     string
	ExternalID string
	Language   string
	Name       string
	Address    string
	Latitude   *float64
	Longitude  *float64
	Category   string
	Area       string
}

// Register returns the spot for a chosen place, creating it when needed.
// created is false when the same external place was registered before, by
// anyone: everybody then shares that row.
func (s *PlaceService) Register(ctx context.Context, memberID string, p RegisterSpotParams) (spot *models.SpotResponse, created bool, err error) {
	var row *models.Spot
	var place *Place
	switch p.Source {
	case models.SpotSourceManual:
		row, err = s.registerManual(ctx, memberID, p)
		created = err == nil
	case models.SpotSourceOSM, models.SpotSourceGoogle:
		row, place, created, err = s.registerExternal(ctx, memberID, p)
	default:
		return nil, false, invalid("source must be one of osm, google, manual")
	}
	if err != nil {
		return nil, false, err
	}
	if row.Source == models.SpotSourceGoogle && place != nil {
		// The provider was asked a moment ago; show that answer instead of
		// paying for the same lookup again.
		view := s.storedView(memberID, row)
		showPlace(&view, place)
		return &view, created, nil
	}
	view, err := s.PresentOne(ctx, memberID, row)
	if err != nil {
		return nil, false, err
	}
	return &view, created, nil
}

// registerManual stores a place exactly as the member typed it. Nothing is
// looked up and nothing is filled in: a place entered without coordinates
// has none.
func (s *PlaceService) registerManual(ctx context.Context, memberID string, p RegisterSpotParams) (*models.Spot, error) {
	if p.ExternalID != "" {
		return nil, invalid("external_id is not allowed when source is manual")
	}
	name := normalizeQuery(p.Name)
	if name == "" || utf8.RuneCountInString(name) > maxSpotNameLength {
		return nil, invalid("name must be 1 to 100 characters")
	}
	address := normalizeQuery(p.Address)
	if utf8.RuneCountInString(address) > maxManualAddressLength {
		return nil, invalid("address must be at most 200 characters")
	}
	if (p.Latitude == nil) != (p.Longitude == nil) {
		return nil, invalid("latitude and longitude must be given together")
	}
	if p.Latitude != nil && !validCoordinates(*p.Latitude, *p.Longitude) {
		return nil, invalid("latitude or longitude is out of range")
	}
	category := orDefault(p.Category, PlaceCategoryOther)
	if !validPlaceCategory(category) {
		return nil, invalid("category is not one of the supported values")
	}
	if p.Area != "" && !postAreas[p.Area] {
		return nil, invalid("area must be one of 東信, 北信, 中信, 南信")
	}

	label, emoji := PlaceCategory(category)
	row := models.Spot{
		Name:              name,
		Emoji:             emoji,
		Area:              p.Area,
		Category:          label,
		Source:            models.SpotSourceManual,
		Latitude:          p.Latitude,
		Longitude:         p.Longitude,
		CreatedByMemberID: &memberID,
	}
	if address != "" {
		row.Address = &address
	}
	if err := s.db.WithContext(ctx).Create(&row).Error; err != nil {
		return nil, fmt.Errorf("create manual spot: %w", err)
	}
	return &row, nil
}

// registerExternal stores a place chosen from the provider's results.
//
// The client sends the id only; what is stored comes from the provider's
// own answer for that id, so a member cannot attach a name or an address
// of their choosing to an OpenStreetMap or Google place.
func (s *PlaceService) registerExternal(ctx context.Context, memberID string, p RegisterSpotParams) (*models.Spot, *Place, bool, error) {
	if p.Name != "" || p.Address != "" || p.Latitude != nil || p.Longitude != nil || p.Category != "" || p.Area != "" {
		return nil, nil, false, invalid("name, address, latitude, longitude, category and area are only allowed when source is manual")
	}
	provider, err := s.provider()
	if err != nil {
		return nil, nil, false, err
	}
	if provider.Name() != p.Source {
		return nil, nil, false, invalid("source does not match the place search in use")
	}
	if !provider.ValidExternalID(p.ExternalID) {
		return nil, nil, false, invalid("external_id is not a valid id for this source")
	}

	// Already registered: no need to ask the provider again.
	if existing, err := s.findExternal(ctx, p.Source, p.ExternalID); err != nil || existing != nil {
		return existing, nil, false, err
	}

	language, err := s.language(ctx, memberID, p.Language)
	if err != nil {
		return nil, nil, false, err
	}
	place, err := provider.Resolve(ctx, p.ExternalID, language)
	if err != nil {
		return nil, nil, false, err
	}
	// The provider may answer with a different, canonical id.
	if place.ExternalID != p.ExternalID {
		if !provider.ValidExternalID(place.ExternalID) {
			return nil, nil, false, placeUnavailable(upstreamFormatError(provider.Name(), "unusable place id"))
		}
		if existing, err := s.findExternal(ctx, p.Source, place.ExternalID); err != nil || existing != nil {
			return existing, place, false, err
		}
	}

	row := newExternalSpot(place, memberID)
	if err := s.db.WithContext(ctx).Create(row).Error; err != nil {
		// Another request registered the same place between the check and
		// the insert: the unique key on external_key refused this one.
		var mysqlErr *mysqldriver.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == mysqlDuplicateEntry {
			existing, findErr := s.findExternal(ctx, place.Source, place.ExternalID)
			if findErr != nil {
				return nil, nil, false, findErr
			}
			if existing != nil {
				return existing, place, false, nil
			}
		}
		return nil, nil, false, fmt.Errorf("create spot: %w", err)
	}
	return row, place, true, nil
}

const mysqlDuplicateEntry = 1062

func (s *PlaceService) findExternal(ctx context.Context, source, externalID string) (*models.Spot, error) {
	var row models.Spot
	err := s.db.WithContext(ctx).Where("external_key = ?", externalKey(source, externalID)).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("find spot by external id: %w", err)
	}
	return &row, nil
}

// newExternalSpot builds the row for a provider's place.
//
// For Google it keeps the place ID and nothing else: no name, address,
// coordinates, category or map link from the result reaches the database.
// For OpenStreetMap, whose data may be stored with attribution, it keeps
// the name, address, position and category.
func newExternalSpot(place *Place, memberID string) *models.Spot {
	id := place.ExternalID
	key := externalKey(place.Source, place.ExternalID)
	row := &models.Spot{
		Emoji:             defaultSpotEmoji,
		Source:            place.Source,
		ExternalID:        &id,
		ExternalKey:       &key,
		CreatedByMemberID: &memberID,
	}
	if place.Source != models.SpotSourceOSM {
		return row
	}

	label, emoji := PlaceCategory(place.CategoryKey)
	lat, lon := place.Latitude, place.Longitude
	row.Name = truncateRunes(place.Name, maxSpotNameLength)
	row.Emoji = emoji
	row.Category = label
	row.Area = place.Area
	row.Latitude = &lat
	row.Longitude = &lon
	if place.OriginalName != "" {
		original := truncateRunes(place.OriginalName, maxSpotOriginalNameLength)
		row.OriginalName = &original
	}
	if place.Address != "" {
		address := truncateRunes(place.Address, maxSpotAddressLength)
		row.Address = &address
	}
	if place.MapURL != "" && len(place.MapURL) <= maxSpotMapURLLength {
		mapURL := place.MapURL
		row.MapURL = &mapURL
	}
	return row
}

// Present turns spot rows into what a member is shown, keyed by spot id.
//
// Rows from the seed, OpenStreetMap or a member's own entry are shown as
// stored. A Google row holds a place ID only, so its name, address,
// position and category are fetched from Google here, in the viewer's
// language, on every read and without caching. When that is not possible
// (no key, Google unreachable, the place gone) the spot is still returned,
// with an empty name, details_available false and a map link built from
// the ID, and nothing else in the response is affected.
func (s *PlaceService) Present(ctx context.Context, viewerID string, spots []*models.Spot) (map[string]models.SpotResponse, error) {
	out := make(map[string]models.SpotResponse, len(spots))
	var google []*models.Spot
	for _, spot := range spots {
		if spot == nil {
			continue
		}
		if _, done := out[spot.ID]; done {
			continue
		}
		out[spot.ID] = s.storedView(viewerID, spot)
		if spot.Source == models.SpotSourceGoogle && spot.ExternalID != nil {
			google = append(google, spot)
		}
	}
	provider := s.providers.bySource[PlaceProviderGoogle]
	if len(google) == 0 || provider == nil {
		return out, nil
	}

	language, err := s.language(ctx, viewerID, "")
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, placeResolveTimeout)
	defer cancel()

	resolved := make([]*Place, len(google))
	failures := make([]error, len(google))
	slots := make(chan struct{}, placeResolveWorkers)
	var wg sync.WaitGroup
	for i, spot := range google {
		wg.Add(1)
		go func() {
			defer wg.Done()
			slots <- struct{}{}
			defer func() { <-slots }()
			resolved[i], failures[i] = provider.Resolve(ctx, *spot.ExternalID, language)
		}()
	}
	wg.Wait()

	failed := 0
	var firstFailure error
	for i, spot := range google {
		place := resolved[i]
		if failures[i] != nil || place == nil {
			failed++
			if firstFailure == nil {
				firstFailure = failures[i]
			}
			continue
		}
		view := out[spot.ID]
		showPlace(&view, place)
		out[spot.ID] = view
	}
	if failed > 0 {
		// The error is the provider's sanitised one: a status or a cause,
		// never a place name, an id or a URL.
		slog.WarnContext(ctx, "google spots shown without details", "count", failed, "error", firstFailure)
	}
	return out, nil
}

// storedView is a spot as its row alone describes it. For a Google row
// that is an empty name, details_available false and a map link built
// from the place ID.
func (s *PlaceService) storedView(viewerID string, spot *models.Spot) models.SpotResponse {
	view := models.NewSpotResponse(spot)
	view.IsMine = viewerID != "" && spot.CreatedByMemberID != nil && *spot.CreatedByMemberID == viewerID
	if spot.Source == models.SpotSourceGoogle && spot.ExternalID != nil {
		mapURL := GoogleMapsPlaceURL(*spot.ExternalID)
		view.MapURL = &mapURL
	}
	return view
}

// showPlace fills a Google spot's view with what the provider just said.
func showPlace(view *models.SpotResponse, place *Place) {
	label, emoji := PlaceCategory(place.CategoryKey)
	lat, lon := place.Latitude, place.Longitude
	view.Name = place.Name
	view.Category = label
	view.Emoji = emoji
	view.Latitude = &lat
	view.Longitude = &lon
	view.DetailsAvailable = true
	if place.Address != "" {
		address := place.Address
		view.Address = &address
	}
	if place.MapURL != "" {
		mapURL := place.MapURL
		view.MapURL = &mapURL
	}
}

// PresentOne is Present for a single row.
func (s *PlaceService) PresentOne(ctx context.Context, viewerID string, spot *models.Spot) (models.SpotResponse, error) {
	views, err := s.Present(ctx, viewerID, []*models.Spot{spot})
	if err != nil {
		return models.SpotResponse{}, err
	}
	return views[spot.ID], nil
}
