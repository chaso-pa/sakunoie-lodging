package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
	"gorm.io/gorm"
)

// fakePlaceProvider is a PlaceProvider with canned answers.
type fakePlaceProvider struct {
	name   string
	places map[string]Place // by external id
	// fail, when set, is returned by every call.
	fail error
	// gate, when set, is called inside Resolve before it answers.
	gate func()

	resolves  atomic.Int64
	languages sync.Map // external id → last language asked for
}

func (f *fakePlaceProvider) Name() string { return f.name }

func (f *fakePlaceProvider) Attribution() models.PlaceAttribution {
	return models.PlaceAttribution{Text: "fake", URL: "https://example.test"}
}

func (f *fakePlaceProvider) ValidExternalID(id string) bool {
	return id != "" && !strings.ContainsAny(id, " /?")
}

func (f *fakePlaceProvider) all() []Place {
	var out []Place
	for _, p := range f.places {
		out = append(out, p)
	}
	return out
}

func (f *fakePlaceProvider) Search(context.Context, PlaceSearchQuery) ([]Place, error) {
	return f.all(), f.fail
}

func (f *fakePlaceProvider) Nearby(context.Context, PlaceNearbyQuery) ([]Place, error) {
	return f.all(), f.fail
}

func (f *fakePlaceProvider) Reverse(context.Context, PlacePoint, string) (*Place, error) {
	for _, p := range f.places {
		return &p, f.fail
	}
	return nil, f.fail
}

func (f *fakePlaceProvider) Resolve(_ context.Context, externalID, language string) (*Place, error) {
	f.resolves.Add(1)
	f.languages.Store(externalID, language)
	if f.gate != nil {
		f.gate()
	}
	if f.fail != nil {
		return nil, f.fail
	}
	p, ok := f.places[externalID]
	if !ok {
		return nil, ErrPlaceNotFound
	}
	return &p, nil
}

// What a Google result carries. None of it may reach the database.
const (
	googleID      = "ChIJ_test_STORAGE_RULE-9"
	googleName    = "Google の店名 Sakura Coffee"
	googleAddress = "Google の住所 123 Usuda, Saku"
	googleMapURL  = "https://maps.google.com/?cid=998877665544"
	googleLat     = 36.198765
	googleLon     = 138.487654
)

func googleFake() *fakePlaceProvider {
	return &fakePlaceProvider{name: PlaceProviderGoogle, places: map[string]Place{
		googleID: {Source: PlaceProviderGoogle, ExternalID: googleID, Name: googleName, Address: googleAddress,
			CategoryKey: PlaceCategoryCafe, Latitude: googleLat, Longitude: googleLon, MapURL: googleMapURL},
	}}
}

func osmFake() *fakePlaceProvider {
	return &fakePlaceProvider{name: PlaceProviderOSM, places: map[string]Place{
		"N6353535276": {Source: PlaceProviderOSM, ExternalID: "N6353535276", Name: "Cosmos Café", OriginalName: "コスモスカフェ",
			Address: "長野県佐久市佐久平駅東国道141号", CategoryKey: PlaceCategoryCafe, Latitude: 36.27801, Longitude: 138.4646828,
			MapURL: "https://www.openstreetmap.org/node/6353535276", Area: "東信"},
	}}
}

// databaseDump returns every value of every row of every table, as text.
func databaseDump(t *testing.T, db *gorm.DB) string {
	t.Helper()
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	var tables []string
	rows, err := sqlDB.Query("SELECT table_name FROM information_schema.tables WHERE table_schema = DATABASE() AND table_type = 'BASE TABLE'")
	if err != nil {
		t.Fatal(err)
	}
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatal(err)
		}
		tables = append(tables, name)
	}
	rows.Close()
	if len(tables) < 10 {
		t.Fatalf("only %d tables found: %v", len(tables), tables)
	}

	var dump strings.Builder
	for _, table := range tables {
		rows, err := sqlDB.Query("SELECT * FROM `" + table + "`")
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

// assertNoGoogleContent fails when anything from the Google result other
// than the place ID is found anywhere in the database.
func assertNoGoogleContent(t *testing.T, db *gorm.DB) {
	t.Helper()
	dump := databaseDump(t, db)
	if !strings.Contains(dump, googleID) {
		t.Fatal("the place ID is not in the database: the check below would pass vacuously")
	}
	for what, forbidden := range map[string]string{
		"name": googleName, "part of the name": "Sakura Coffee", "address": googleAddress, "part of the address": "Usuda",
		"map URL": googleMapURL, "map URL id": "998877665544",
		"latitude": "36.198765", "coarse latitude": "36.19", "longitude": "138.487654", "coarse longitude": "138.48",
		"category label": "カフェ", "category emoji": "☕",
	} {
		if strings.Contains(dump, forbidden) {
			t.Errorf("the database contains the Google %s (%q)", what, forbidden)
		}
	}
}

// A spot registered from Google is a place ID and nothing else, whatever
// is done with it afterwards.
func TestGoogleSpotStoresOnlyThePlaceIDIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	google := googleFake()
	places := NewPlaceService(db, NewPlaceProvidersOf(google))
	posts := NewPostService(db, newTrackingUploader()).WithPlaces(places)
	spots := NewSpotService(db).WithPlaces(places)
	alice, bob := addMember(t, db, "Alice"), addMember(t, db, "Bob")
	if err := db.Model(&models.Member{}).Where("id = ?", bob).Update("preferred_language", "en").Error; err != nil {
		t.Fatal(err)
	}

	spot, created, err := places.Register(ctx, alice, RegisterSpotParams{Source: "google", ExternalID: googleID})
	if err != nil || !created {
		t.Fatalf("created = %v, err = %v", created, err)
	}
	// The member is shown the place in full...
	if spot.Name != googleName || spot.Address == nil || *spot.Address != googleAddress || spot.Latitude == nil || *spot.Latitude != googleLat ||
		*spot.Longitude != googleLon || spot.Category != "カフェ" || spot.Emoji != "☕" || *spot.MapURL != googleMapURL ||
		!spot.DetailsAvailable || spot.Source != "google" || *spot.ExternalID != googleID || !spot.IsMine || spot.Area != "" {
		t.Fatalf("spot = %+v", spot)
	}
	// ...from the one lookup registration needed.
	if n := google.resolves.Load(); n != 1 {
		t.Fatalf("registration made %d lookups, want 1", n)
	}

	// The row itself: the id, the source, who registered it, and defaults.
	var row models.Spot
	if err := db.First(&row, "id = ?", spot.ID).Error; err != nil {
		t.Fatal(err)
	}
	want := models.Spot{
		ID: row.ID, Emoji: "📍", Source: "google", ExternalID: row.ExternalID, ExternalKey: row.ExternalKey,
		CreatedByMemberID: &alice, CreatedAt: row.CreatedAt, UpdatedAt: row.UpdatedAt,
	}
	if !reflect.DeepEqual(row, want) || *row.ExternalID != googleID || *row.ExternalKey != externalKey("google", googleID) {
		t.Fatalf("row = %+v", row)
	}
	assertNoGoogleContent(t, db)

	// Use the spot every way a spot can be used, then look again.
	post := mustCreatePost(t, posts, alice, CreatePostParams{Caption: "Google の場所で投稿", SpotID: spot.ID, Area: "東信"})
	if post.Spot == nil || post.Spot.Name != googleName || post.Spot.Source != "google" || !post.Spot.DetailsAvailable ||
		*post.Spot.Latitude != googleLat || *post.Spot.MapURL != googleMapURL || post.Area == nil || *post.Area != "東信" {
		t.Fatalf("post = %+v, spot = %+v", post, post.Spot)
	}
	if _, err := spots.CreateReview(ctx, bob, spot.ID, 5, "よかった"); err != nil {
		t.Fatal(err)
	}
	yes := true
	visits := NewGpsVisitService(db)
	if _, err := visits.Create(ctx, bob, CreateGpsVisitParams{SpotID: spot.ID, PlaceName: "自分で書いた名前", VisitedDate: "2026-10-01", IsPublic: &yes}); err != nil {
		t.Fatal(err)
	}
	// A footprint takes its place name from the spot when none is given.
	// A Google spot has no stored name to take, so one must be typed.
	var validation *ValidationError
	if _, err := visits.Create(ctx, bob, CreateGpsVisitParams{SpotID: spot.ID, VisitedDate: "2026-10-01"}); !errors.As(err, &validation) {
		t.Fatalf("footprint without a place name at a Google spot: err = %v", err)
	}
	posts.ToggleLike(ctx, bob, post.ID)
	posts.ToggleBookmark(ctx, bob, post.ID)

	feed, total, err := posts.List(ctx, bob, PostFilter{Area: "東信"})
	if err != nil || total != 1 || feed[0].Spot.Name != googleName {
		t.Fatalf("feed = %+v, total = %d, err = %v", feed, total, err)
	}
	// Each reader is answered in their own language.
	if lang, _ := google.languages.Load(googleID); lang != "en" {
		t.Fatalf("language asked for bob = %v, want en", lang)
	}
	detail, err := spots.Get(ctx, alice, spot.ID)
	if err != nil || detail.Spot.Name != googleName || len(detail.Posts) != 1 || len(detail.Reviews) != 1 {
		t.Fatalf("detail = %+v, err = %v", detail, err)
	}
	if lang, _ := google.languages.Load(googleID); lang != "ja" {
		t.Fatalf("language asked for alice = %v, want ja", lang)
	}
	ranking, err := posts.SpotRanking(ctx, alice, 5)
	if err != nil || len(ranking) != 1 || ranking[0].Name != googleName || ranking[0].Detail != "カフェ" || ranking[0].Emoji != "☕" || ranking[0].Score != 5 {
		t.Fatalf("spot ranking = %+v, err = %v", ranking, err)
	}
	postRanking, err := posts.PostRanking(ctx, alice, 5)
	if err != nil || len(postRanking) != 1 || postRanking[0].Detail != googleName+" · Alice" || postRanking[0].Area != "東信" {
		t.Fatalf("post ranking = %+v, err = %v", postRanking, err)
	}
	// Registering it again, by another member, changes nothing.
	again, created, err := places.Register(ctx, bob, RegisterSpotParams{Source: "google", ExternalID: googleID})
	if err != nil || created || again.ID != spot.ID || again.IsMine || again.Name != googleName {
		t.Fatalf("again = %+v, created = %v, err = %v", again, created, err)
	}
	// Searching never matches a Google spot by its (unstored) name, and
	// marks the result as registered by its id alone.
	found, err := places.Search(ctx, bob, PlaceSearchParams{Query: "Sakura Coffee"})
	if err != nil || len(found.Results) != 1 || found.Results[0].SpotID == nil || *found.Results[0].SpotID != spot.ID || found.Results[0].Source != "google" {
		t.Fatalf("search = %+v, err = %v", found, err)
	}
	assertNoGoogleContent(t, db)

	// Every read asked Google again: nothing was kept between them.
	if n := google.resolves.Load(); n < 7 {
		t.Fatalf("lookups = %d; reads are being answered from somewhere else", n)
	}

	// Google down: everything still answers, with the spot unnamed.
	google.fail = placeUnavailable(upstreamStatusError("google places", 503))
	feed, _, err = posts.List(ctx, alice, PostFilter{})
	if err != nil || len(feed) != 1 {
		t.Fatalf("feed with Google down: %+v, err = %v", feed, err)
	}
	down := feed[0].Spot
	if down.ID != spot.ID || down.Name != "" || down.DetailsAvailable || down.Address != nil || down.Latitude != nil ||
		down.MapURL == nil || *down.MapURL != "https://www.google.com/maps/place/?q=place_id:"+googleID || down.Emoji != "📍" {
		t.Fatalf("spot with Google down = %+v", down)
	}
	if _, err := posts.SpotRanking(ctx, alice, 5); err != nil {
		t.Fatal(err)
	}
	if _, err := spots.Get(ctx, alice, spot.ID); err != nil {
		t.Fatal(err)
	}
	// Posting at an already registered spot does not need Google either.
	mustCreatePost(t, posts, bob, CreatePostParams{Caption: "Google が落ちていても投稿できる", SpotID: spot.ID})

	// No key at all (the provider was switched back): same degraded view,
	// and no call is attempted.
	before := google.resolves.Load()
	keyless := NewPostService(db, newTrackingUploader()).WithPlaces(NewPlaceService(db, NewPlaceProvidersOf(osmFake())))
	feed, _, err = keyless.List(ctx, alice, PostFilter{})
	if err != nil || len(feed) != 2 || feed[0].Spot.DetailsAvailable || feed[0].Spot.Name != "" {
		t.Fatalf("feed without a Google provider: %+v, err = %v", feed, err)
	}
	if google.resolves.Load() != before {
		t.Fatal("Google was called by a service that has no Google provider")
	}
}

func TestRegisterOSMSpotIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	osm := osmFake()
	places := NewPlaceService(db, NewPlaceProvidersOf(osm))
	alice, bob := addMember(t, db, "Alice"), addMember(t, db, "Bob")

	spot, created, err := places.Register(ctx, alice, RegisterSpotParams{Source: "osm", ExternalID: "N6353535276", Language: "en"})
	if err != nil || !created {
		t.Fatalf("created = %v, err = %v", created, err)
	}
	if spot.Name != "Cosmos Café" || *spot.OriginalName != "コスモスカフェ" || *spot.Address != "長野県佐久市佐久平駅東国道141号" ||
		*spot.Latitude != 36.27801 || *spot.Longitude != 138.4646828 || spot.Area != "東信" || spot.Category != "カフェ" || spot.Emoji != "☕" ||
		*spot.MapURL != "https://www.openstreetmap.org/node/6353535276" || spot.Source != "osm" || *spot.ExternalID != "N6353535276" ||
		!spot.DetailsAvailable || !spot.IsMine || spot.IsSample || spot.CreatedAt.IsZero() {
		t.Fatalf("spot = %+v", spot)
	}
	if lang, _ := osm.languages.Load("N6353535276"); lang != "en" {
		t.Fatalf("language = %v", lang)
	}
	// OpenStreetMap data is stored: the row has it all.
	var row models.Spot
	db.First(&row, "id = ?", spot.ID)
	if row.Name != "Cosmos Café" || row.Address == nil || row.Latitude == nil || *row.CreatedByMemberID != alice {
		t.Fatalf("row = %+v", row)
	}

	// The other member choosing the same result gets the same spot, and
	// the provider is not asked again.
	again, created, err := places.Register(ctx, bob, RegisterSpotParams{Source: "osm", ExternalID: "N6353535276"})
	if err != nil || created || again.ID != spot.ID || again.IsMine {
		t.Fatalf("again = %+v, created = %v, err = %v", again, created, err)
	}
	if n := osm.resolves.Load(); n != 1 {
		t.Fatalf("lookups = %d, want 1", n)
	}
	var count int64
	db.Model(&models.Spot{}).Count(&count)
	if count != 1 {
		t.Fatalf("spots = %d, want 1", count)
	}

	// Errors.
	var validation *ValidationError
	for name, p := range map[string]RegisterSpotParams{
		"no id":               {Source: "osm"},
		"malformed id":        {Source: "osm", ExternalID: "N1 /x"},
		"other provider":      {Source: "google", ExternalID: "ChIJ_x"},
		"unknown source":      {Source: "seed", ExternalID: "N1"},
		"forged name":         {Source: "osm", ExternalID: "N1", Name: "自分で決めた名前"},
		"forged position":     {Source: "osm", ExternalID: "N1", Latitude: ptr(1.0), Longitude: ptr(2.0)},
		"forged address":      {Source: "osm", ExternalID: "N1", Address: "どこか"},
		"unsupported lang":    {Source: "osm", ExternalID: "N1", Language: "fr"},
		"manual with an id":   {Source: "manual", Name: "x", ExternalID: "N1"},
		"manual without name": {Source: "manual", Name: "   "},
	} {
		if _, _, err := places.Register(ctx, alice, p); !errors.As(err, &validation) {
			t.Errorf("%s: err = %v, want a validation error", name, err)
		}
	}
	if _, _, err := places.Register(ctx, alice, RegisterSpotParams{Source: "osm", ExternalID: "N404"}); !errors.Is(err, ErrPlaceNotFound) {
		t.Fatalf("unknown place: err = %v", err)
	}
	osm.fail = placeUnavailable(upstreamStatusError("nominatim", 500))
	if _, _, err := places.Register(ctx, alice, RegisterSpotParams{Source: "osm", ExternalID: "N7"}); !errors.Is(err, ErrPlaceUnavailable) {
		t.Fatalf("provider down: err = %v", err)
	}
	// With the provider down, what is already registered is still usable,
	// and so is a hand-entered place.
	if again, _, err := places.Register(ctx, bob, RegisterSpotParams{Source: "osm", ExternalID: "N6353535276"}); err != nil || again.ID != spot.ID {
		t.Fatalf("registered spot with the provider down: %+v, %v", again, err)
	}
	if _, _, err := places.Register(ctx, bob, RegisterSpotParams{Source: "manual", Name: "手入力の場所"}); err != nil {
		t.Fatal(err)
	}
	db.Model(&models.Spot{}).Count(&count)
	if count != 2 {
		t.Fatalf("spots = %d, want 2 (nothing left behind by the failures)", count)
	}

	// Not configured.
	unconfigured := NewPlaceService(db, nil)
	if _, _, err := unconfigured.Register(ctx, alice, RegisterSpotParams{Source: "osm", ExternalID: "N1"}); !errors.Is(err, ErrPlaceNotConfigured) {
		t.Fatalf("err = %v", err)
	}
	if _, err := unconfigured.Search(ctx, alice, PlaceSearchParams{Query: "x"}); !errors.Is(err, ErrPlaceNotConfigured) {
		t.Fatalf("err = %v", err)
	}
	if _, err := unconfigured.Nearby(ctx, alice, PlaceNearbyParams{Latitude: 36, Longitude: 138}); !errors.Is(err, ErrPlaceNotConfigured) {
		t.Fatalf("err = %v", err)
	}
	if _, _, err := unconfigured.Reverse(ctx, alice, 36, 138, ""); !errors.Is(err, ErrPlaceNotConfigured) {
		t.Fatalf("err = %v", err)
	}
	if _, _, err := unconfigured.Register(ctx, alice, RegisterSpotParams{Source: "manual", Name: "設定なしでも登録できる"}); err != nil {
		t.Fatal(err)
	}
}

func ptr[T any](v T) *T { return &v }

// Many members choosing the same search result at the same moment end up
// with one row. Every request is held inside the provider until all of
// them have passed the "already registered?" check, so the inserts really
// do race and the unique key is what decides.
func TestRegisterSameExternalPlaceConcurrentlyIntegration(t *testing.T) {
	for _, source := range []string{PlaceProviderOSM, PlaceProviderGoogle} {
		t.Run(source, func(t *testing.T) {
			db := testutil.NewDB(t, "services").Gorm
			ctx := context.Background()
			const members = 12

			provider, id := osmFake(), "N6353535276"
			if source == PlaceProviderGoogle {
				provider, id = googleFake(), googleID
			}
			var arrived sync.WaitGroup
			arrived.Add(members)
			release := make(chan struct{})
			go func() {
				arrived.Wait()
				close(release)
			}()
			var once [members]sync.Once
			var next atomic.Int64
			provider.gate = func() {
				i := next.Add(1) - 1
				if i < members {
					once[i].Do(arrived.Done)
				}
				<-release
			}
			places := NewPlaceService(db, NewPlaceProvidersOf(provider))

			ids := make([]string, members)
			for i := range ids {
				ids[i] = addMember(t, db, fmt.Sprint("Member", i))
			}
			type outcome struct {
				spotID  string
				created bool
				err     error
			}
			results := make([]outcome, members)
			var wg sync.WaitGroup
			for i := range results {
				wg.Add(1)
				go func() {
					defer wg.Done()
					spot, created, err := places.Register(ctx, ids[i], RegisterSpotParams{Source: source, ExternalID: id})
					results[i] = outcome{created: created, err: err}
					if spot != nil {
						results[i].spotID = spot.ID
					}
				}()
			}
			wg.Wait()

			createdCount := 0
			for i, r := range results {
				if r.err != nil {
					t.Fatalf("member %d: %v", i, r.err)
				}
				if r.spotID != results[0].spotID || r.spotID == "" {
					t.Fatalf("member %d got spot %q, member 0 got %q", i, r.spotID, results[0].spotID)
				}
				if r.created {
					createdCount++
				}
			}
			if createdCount != 1 {
				t.Fatalf("%d requests say they created the spot, want exactly 1", createdCount)
			}
			var rows int64
			db.Model(&models.Spot{}).Count(&rows)
			if rows != 1 {
				t.Fatalf("spots = %d, want 1", rows)
			}
			if n := provider.resolves.Load(); n != members {
				t.Fatalf("lookups = %d: the requests did not all race past the first check", n)
			}
		})
	}
}

// The unique key is exact: ids that differ only in letter case are
// different places (Google's ids are case sensitive, and the database's
// default collation is not).
func TestExternalKeyIsCaseSensitiveIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	google := googleFake()
	google.places["ChIJabc"] = Place{Source: "google", ExternalID: "ChIJabc", Name: "lower", Latitude: 1, Longitude: 1}
	google.places["ChIJABC"] = Place{Source: "google", ExternalID: "ChIJABC", Name: "upper", Latitude: 2, Longitude: 2}
	places := NewPlaceService(db, NewPlaceProvidersOf(google))
	alice := addMember(t, db, "Alice")

	lower, createdLower, err := places.Register(ctx, alice, RegisterSpotParams{Source: "google", ExternalID: "ChIJabc"})
	if err != nil {
		t.Fatal(err)
	}
	upper, createdUpper, err := places.Register(ctx, alice, RegisterSpotParams{Source: "google", ExternalID: "ChIJABC"})
	if err != nil {
		t.Fatal(err)
	}
	if !createdLower || !createdUpper || lower.ID == upper.ID || lower.Name != "lower" || upper.Name != "upper" {
		t.Fatalf("lower = %+v (%v), upper = %+v (%v)", lower, createdLower, upper, createdUpper)
	}
	// The same id under another source is another place too.
	if externalKey("osm", "N1") == externalKey("google", "N1") {
		t.Fatal("external keys of different sources collide")
	}
	// And the database refuses a second row with the same key outright.
	key := externalKey("google", "ChIJabc")
	dup := models.Spot{Source: "google", Emoji: "📍", ExternalID: ptr("ChIJabc"), ExternalKey: &key}
	if err := db.Create(&dup).Error; err == nil {
		t.Fatal("a second row with the same external key was inserted")
	}
}

func TestRegisterManualSpotIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	osm := osmFake()
	places := NewPlaceService(db, NewPlaceProvidersOf(osm))
	alice, bob := addMember(t, db, "Alice"), addMember(t, db, "Bob")

	// With everything.
	full, created, err := places.Register(ctx, alice, RegisterSpotParams{
		Source: "manual", Name: "  山の上の  パン屋 ", Address: " 佐久市のどこか ", Latitude: ptr(36.25), Longitude: ptr(138.5),
		Category: PlaceCategoryFood, Area: "東信",
	})
	if err != nil || !created {
		t.Fatalf("created = %v, err = %v", created, err)
	}
	if full.Name != "山の上の パン屋" || *full.Address != "佐久市のどこか" || *full.Latitude != 36.25 || *full.Longitude != 138.5 ||
		full.Category != "グルメ" || full.Emoji != "🍽️" || full.Area != "東信" || full.Source != "manual" || full.ExternalID != nil ||
		full.MapURL != nil || full.OriginalName != nil || !full.IsMine || !full.DetailsAvailable || full.IsSample {
		t.Fatalf("spot = %+v", full)
	}

	// Name only: no address, no position, and none made up.
	bare, _, err := places.Register(ctx, alice, RegisterSpotParams{Source: "manual", Name: "名前だけの場所"})
	if err != nil {
		t.Fatal(err)
	}
	if bare.Latitude != nil || bare.Longitude != nil || bare.Address != nil || bare.Area != "" || bare.Category != "その他" || bare.Emoji != "📍" || bare.MapURL != nil {
		t.Fatalf("spot = %+v", bare)
	}
	var row models.Spot
	db.First(&row, "id = ?", bare.ID)
	if row.Latitude != nil || row.Longitude != nil || row.Address != nil || row.ExternalID != nil || row.ExternalKey != nil ||
		row.MapURL != nil || *row.CreatedByMemberID != alice || row.Source != "manual" {
		t.Fatalf("row = %+v", row)
	}
	// Nothing was looked up for either.
	if osm.resolves.Load() != 0 {
		t.Fatal("a manual registration called the provider")
	}
	// The same name twice is two places: there is nothing to match on.
	twin, created, err := places.Register(ctx, bob, RegisterSpotParams{Source: "manual", Name: "名前だけの場所"})
	if err != nil || !created || twin.ID == bare.ID || twin.IsMine != true {
		t.Fatalf("twin = %+v, created = %v, err = %v", twin, created, err)
	}

	var validation *ValidationError
	for name, p := range map[string]RegisterSpotParams{
		"empty name":         {Name: ""},
		"long name":          {Name: strings.Repeat("あ", 101)},
		"long address":       {Name: "x", Address: strings.Repeat("あ", 201)},
		"latitude only":      {Name: "x", Latitude: ptr(36.0)},
		"longitude only":     {Name: "x", Longitude: ptr(138.0)},
		"latitude too big":   {Name: "x", Latitude: ptr(90.1), Longitude: ptr(138.0)},
		"latitude too small": {Name: "x", Latitude: ptr(-90.1), Longitude: ptr(138.0)},
		"longitude too big":  {Name: "x", Latitude: ptr(36.0), Longitude: ptr(180.1)},
		"unknown category":   {Name: "x", Category: "宿泊"},
		"unknown area":       {Name: "x", Area: "関東"},
	} {
		p.Source = "manual"
		if _, _, err := places.Register(ctx, alice, p); !errors.As(err, &validation) {
			t.Errorf("%s: err = %v, want a validation error", name, err)
		}
	}
	// The limits themselves are accepted.
	if _, _, err := places.Register(ctx, alice, RegisterSpotParams{Source: "manual", Name: strings.Repeat("あ", 100), Address: strings.Repeat("い", 200),
		Latitude: ptr(-90.0), Longitude: ptr(180.0)}); err != nil {
		t.Fatal(err)
	}
}

// Search lists registered spots ahead of the provider's results. A
// hand-entered spot reaches other members only once a post uses it.
func TestSearchIncludesRegisteredSpotsIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	osm := osmFake()
	places := NewPlaceService(db, NewPlaceProvidersOf(osm))
	posts := NewPostService(db, newTrackingUploader()).WithPlaces(places)
	alice, bob := addMember(t, db, "Alice"), addMember(t, db, "Bob")
	seeded := addSpot(t, db, "Cosmos 公園", "東信", "自然")

	manual, _, err := places.Register(ctx, alice, RegisterSpotParams{Source: "manual", Name: "Cosmos 手入力", Address: "佐久市", Latitude: ptr(36.2), Longitude: ptr(138.4)})
	if err != nil {
		t.Fatal(err)
	}
	names := func(viewer string) []string {
		found, err := places.Search(ctx, viewer, PlaceSearchParams{Query: "cosmos"})
		if err != nil {
			t.Fatal(err)
		}
		if found.Provider != "osm" || found.Attribution.Text != "fake" {
			t.Fatalf("meta = %+v", found)
		}
		var out []string
		for _, r := range found.Results {
			registered := "-"
			if r.SpotID != nil {
				registered = "spot"
			}
			out = append(out, r.Source+":"+r.Name+":"+registered)
		}
		return out
	}

	// Alice sees her own entry; Bob does not, yet.
	if got, want := names(alice), []string{"seed:Cosmos 公園:spot", "manual:Cosmos 手入力:spot", "osm:Cosmos Café:-"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("alice sees %v, want %v", got, want)
	}
	if got, want := names(bob), []string{"seed:Cosmos 公園:spot", "osm:Cosmos Café:-"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("bob sees %v, want %v", got, want)
	}
	// Once a post uses it, it is a place of the community.
	mustCreatePost(t, posts, alice, CreatePostParams{Caption: "手入力の場所で投稿", SpotID: manual.ID})
	if got, want := names(bob), []string{"seed:Cosmos 公園:spot", "manual:Cosmos 手入力:spot", "osm:Cosmos Café:-"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("bob sees %v, want %v", got, want)
	}

	// A registered provider result is listed once, as the spot.
	registered, _, err := places.Register(ctx, bob, RegisterSpotParams{Source: "osm", ExternalID: "N6353535276"})
	if err != nil {
		t.Fatal(err)
	}
	found, err := places.Search(ctx, alice, PlaceSearchParams{Query: "cosmos"})
	if err != nil || len(found.Results) != 3 {
		t.Fatalf("results = %+v, err = %v", found, err)
	}
	last := found.Results[2]
	if last.Source != "osm" || last.SpotID == nil || *last.SpotID != registered.ID || *last.ExternalID != "N6353535276" ||
		last.Category != "カフェ" || last.CategoryKey != "cafe" || last.Emoji != "☕" || *last.Latitude != 36.27801 || last.DistanceM != nil || last.PhotoURL != nil {
		t.Fatalf("registered result = %+v", last)
	}
	first := found.Results[0]
	if *first.SpotID != seeded || first.ExternalID != nil || first.CategoryKey != "nature" || first.Latitude != nil || first.Address != "" {
		t.Fatalf("seeded result = %+v", first)
	}

	// LIKE wildcards in the text are literal.
	for _, text := range []string{"%", "_", "100%"} {
		found, err := places.Search(ctx, alice, PlaceSearchParams{Query: text})
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range found.Results {
			if r.Source != "osm" {
				t.Fatalf("query %q matched the stored spot %q", text, r.Name)
			}
		}
	}

	// Validation.
	var validation *ValidationError
	for name, p := range map[string]PlaceSearchParams{
		"blank":          {Query: "   "},
		"too long":       {Query: strings.Repeat("あ", 101)},
		"latitude only":  {Query: "x", Latitude: ptr(36.0)},
		"out of range":   {Query: "x", Latitude: ptr(91.0), Longitude: ptr(0.0)},
		"unknown locale": {Query: "x", Language: "fr"},
	} {
		if _, err := places.Search(ctx, alice, p); !errors.As(err, &validation) {
			t.Errorf("%s: err = %v, want a validation error", name, err)
		}
	}
	for name, p := range map[string]PlaceNearbyParams{
		"out of range":     {Latitude: 91, Longitude: 0},
		"radius too small": {Latitude: 36, Longitude: 138, RadiusM: 99},
		"radius too large": {Latitude: 36, Longitude: 138, RadiusM: 2001},
		"unknown category": {Latitude: 36, Longitude: 138, Category: "bars"},
	} {
		if _, err := places.Nearby(ctx, alice, p); !errors.As(err, &validation) {
			t.Errorf("nearby %s: err = %v, want a validation error", name, err)
		}
	}
	if _, _, err := places.Reverse(ctx, alice, 0, 181, ""); !errors.As(err, &validation) {
		t.Fatalf("reverse out of range: err = %v", err)
	}

	// Nearby measures from the position as given.
	near, err := places.Nearby(ctx, alice, PlaceNearbyParams{Latitude: 36.27801, Longitude: 138.4646828})
	if err != nil || len(near.Results) != 1 || near.Results[0].DistanceM == nil || *near.Results[0].DistanceM != 0 {
		t.Fatalf("nearby = %+v, err = %v", near, err)
	}
	result, meta, err := places.Reverse(ctx, alice, 36.27801, 138.4646828, "en")
	if err != nil || result == nil || result.Name != "Cosmos Café" || meta.Provider != "osm" || *result.SpotID != registered.ID {
		t.Fatalf("reverse = %+v, %+v, err = %v", result, meta, err)
	}
	osm.places = nil
	if result, meta, err := places.Reverse(ctx, alice, 30, 150, ""); err != nil || result != nil || meta == nil {
		t.Fatalf("reverse of nowhere = %+v, %+v, %v", result, meta, err)
	}
}

// As member-registered spots pile up, the public list stays the curated
// one and the ranking stays a ranking of places people posted about.
func TestSpotListAndRankingWithRegisteredSpotsIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	osm := osmFake()
	for i := 0; i < 30; i++ {
		id := fmt.Sprint("N", 1000+i)
		osm.places[id] = Place{Source: "osm", ExternalID: id, Name: fmt.Sprintf("あ 登録だけの場所 %02d", i), CategoryKey: PlaceCategoryShop, Latitude: 36, Longitude: 138}
	}
	places := NewPlaceService(db, NewPlaceProvidersOf(osm))
	posts := NewPostService(db, newTrackingUploader()).WithPlaces(places)
	spots := NewSpotService(db).WithPlaces(places)
	alice, bob := addMember(t, db, "Alice"), addMember(t, db, "Bob")
	pond := addSpot(t, db, "白駒の池", "東信", "自然")
	castle := addSpot(t, db, "松本城", "中信", "歴史")

	for id := range osm.places {
		if _, _, err := places.Register(ctx, alice, RegisterSpotParams{Source: "osm", ExternalID: id}); err != nil {
			t.Fatal(err)
		}
	}
	manual, _, err := places.Register(ctx, bob, RegisterSpotParams{Source: "manual", Name: "誰かの家かもしれない場所", Address: "佐久市某所 1-2-3", Latitude: ptr(36.2), Longitude: ptr(138.4)})
	if err != nil {
		t.Fatal(err)
	}

	// The public list: the two curated spots, nothing a member registered.
	list, err := spots.List(ctx, "")
	if err != nil || len(list) != 2 {
		t.Fatalf("public list has %d spots, want 2; err = %v", len(list), err)
	}
	for _, item := range list {
		if item.Source != "seed" || item.ExternalID != nil || item.Address != nil || item.Latitude != nil || item.IsMine {
			t.Fatalf("public list item = %+v", item)
		}
	}
	if east, _ := spots.List(ctx, "東信"); len(east) != 1 || east[0].ID != pond {
		t.Fatalf("東信 = %+v", east)
	}

	// The ranking before anyone posted: curated spots only.
	ranking, err := posts.SpotRanking(ctx, alice, 20)
	if err != nil || len(ranking) != 2 {
		t.Fatalf("ranking = %+v, err = %v", ranking, err)
	}

	// A post at the café and one at the hand-entered place bring both in.
	cafe, _, _ := places.Register(ctx, bob, RegisterSpotParams{Source: "osm", ExternalID: "N6353535276"})
	cafePost := mustCreatePost(t, posts, bob, CreatePostParams{Caption: "カフェ", SpotID: cafe.ID})
	mustCreatePost(t, posts, bob, CreatePostParams{Caption: "手入力", SpotID: manual.ID})
	posts.ToggleLike(ctx, alice, cafePost.ID)
	ranking, err = posts.SpotRanking(ctx, alice, 20)
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, item := range ranking {
		got = append(got, fmt.Sprint(item.Rank, ":", item.Name, ":", item.Score))
	}
	// By score; among equals the curated spots first, then by name.
	want := []string{"1:Cosmos Café:1", "2:松本城:0", "3:白駒の池:0", "4:誰かの家かもしれない場所:0"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("ranking = %v\nwant      %v", got, want)
	}
	if ranking[0].Detail != "東信 · カフェ" || ranking[0].Area != "東信" || ranking[0].PostCount != 1 || ranking[0].LikeCount != 1 || *ranking[0].SpotID != cafe.ID {
		t.Fatalf("first = %+v", ranking[0])
	}
	// A spot without a region shows its category alone.
	if ranking[3].Detail != "その他" || ranking[3].Area != "" {
		t.Fatalf("manual spot = %+v", ranking[3])
	}
	_ = castle

	// A member-registered spot's page is reachable with its details.
	detail, err := spots.Get(ctx, alice, manual.ID)
	if err != nil || detail.Spot.Name != "誰かの家かもしれない場所" || *detail.Spot.Address != "佐久市某所 1-2-3" || detail.Spot.IsMine || len(detail.Posts) != 1 {
		t.Fatalf("detail = %+v, err = %v", detail, err)
	}
	if mine, _ := spots.Get(ctx, bob, manual.ID); !mine.Spot.IsMine {
		t.Fatal("is_mine is false for the member who registered the spot")
	}
}

// The region of a post: the spot's own when it has one, otherwise the one
// chosen in the form.
func TestPostAreaWithRegionlessSpotIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	places := NewPlaceService(db, NewPlaceProvidersOf(osmFake()))
	posts := NewPostService(db, newTrackingUploader()).WithPlaces(places)
	alice := addMember(t, db, "Alice")
	pond := addSpot(t, db, "白駒の池", "東信", "自然")
	tokyo, _, err := places.Register(ctx, alice, RegisterSpotParams{Source: "manual", Name: "東京のどこか"})
	if err != nil {
		t.Fatal(err)
	}

	withSpotArea := mustCreatePost(t, posts, alice, CreatePostParams{Caption: "a", SpotID: pond, Area: "南信"})
	fromForm := mustCreatePost(t, posts, alice, CreatePostParams{Caption: "b", SpotID: tokyo.ID, Area: "北信"})
	noArea := mustCreatePost(t, posts, alice, CreatePostParams{Caption: "c", SpotID: tokyo.ID})
	if *withSpotArea.Area != "東信" || *fromForm.Area != "北信" || noArea.Area != nil {
		t.Fatalf("areas = %v, %v, %v", *withSpotArea.Area, *fromForm.Area, noArea.Area)
	}
	if fromForm.Spot.Area != "" || fromForm.Spot.Source != "manual" || fromForm.Spot.Name != "東京のどこか" || fromForm.Spot.Latitude != nil {
		t.Fatalf("spot = %+v", fromForm.Spot)
	}
	var validation *ValidationError
	if _, err := posts.Create(ctx, alice, CreatePostParams{Caption: "d", SpotID: tokyo.ID, Area: "関東"}); !errors.As(err, &validation) {
		t.Fatalf("invalid area with a regionless spot: err = %v", err)
	}

	for area, want := range map[string]int64{"東信": 1, "北信": 1, "南信": 0, "": 3} {
		_, total, err := posts.List(ctx, alice, PostFilter{Area: area})
		if err != nil || total != want {
			t.Errorf("area %q: total = %d, err = %v; want %d", area, total, err, want)
		}
	}
	// The text search matches the region of such a post as well.
	if _, total, _ := posts.List(ctx, alice, PostFilter{Query: "北信"}); total != 1 {
		t.Fatalf("search by region found %d posts, want 1", total)
	}
}

// One rule decides who may see or use a spot, on every path that takes a
// spot id: a hand-entered spot belongs to its creator alone until a post
// points at it. To anyone else it does not exist.
func TestUnpostedManualSpotIsHiddenIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	places := NewPlaceService(db, NewPlaceProvidersOf(osmFake()))
	posts := NewPostService(db, newTrackingUploader()).WithPlaces(places)
	spots := NewSpotService(db).WithPlaces(places)
	visits := NewGpsVisitService(db)
	alice, bob := addMember(t, db, "Alice"), addMember(t, db, "Bob")
	yes := true

	secret, _, err := places.Register(ctx, alice, RegisterSpotParams{Source: "manual", Name: "まだ共有していない場所", Address: "佐久市某所 1-2-3",
		Latitude: ptr(36.2), Longitude: ptr(138.4)})
	if err != nil {
		t.Fatal(err)
	}
	const missing = "00000000-0000-4000-8000-000000000000"

	// inSearch reports whether a search by the member lists the spot (the
	// fake provider adds its own result to every search).
	inSearch := func(member string) bool {
		found, err := places.Search(ctx, member, PlaceSearchParams{Query: "共有していない"})
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range found.Results {
			if r.SpotID != nil && *r.SpotID == secret.ID {
				return true
			}
		}
		return false
	}
	// hidden checks that the spot is, to the member, exactly like an id
	// that was never issued.
	var validation *ValidationError
	hidden := func(t *testing.T, member string) {
		t.Helper()
		for _, id := range []string{secret.ID, missing} {
			if _, err := spots.Get(ctx, member, id); !errors.Is(err, ErrSpotNotFound) {
				t.Errorf("Get(%s): err = %v, want ErrSpotNotFound", id, err)
			}
			if _, err := spots.CreateReview(ctx, member, id, 5, "口コミ"); !errors.Is(err, ErrSpotNotFound) {
				t.Errorf("CreateReview(%s): err = %v, want ErrSpotNotFound", id, err)
			}
			_, postErr := posts.Create(ctx, member, CreatePostParams{Caption: "他人の場所で投稿", SpotID: id})
			_, visitErr := visits.Create(ctx, member, CreateGpsVisitParams{SpotID: id, PlaceName: "足あと", VisitedDate: "2026-10-01", IsPublic: &yes})
			for what, err := range map[string]error{"post": postErr, "footprint": visitErr} {
				if !errors.As(err, &validation) || validation.Message != "spot_id does not match any spot" {
					t.Errorf("%s with spot %s: err = %v, want the unknown-spot error", what, id, err)
				}
			}
		}
		if inSearch(member) {
			t.Errorf("search shows the spot to another member")
		}
	}
	rows := func(table string) int64 {
		var n int64
		db.Table(table).Count(&n)
		return n
	}

	t.Run("before any post", func(t *testing.T) {
		hidden(t, bob)
		if rows("posts") != 0 || rows("reviews") != 0 || rows("gps_visits") != 0 {
			t.Fatalf("a refused request left rows: posts %d, reviews %d, footprints %d", rows("posts"), rows("reviews"), rows("gps_visits"))
		}
		// The creator sees and uses it.
		detail, err := spots.Get(ctx, alice, secret.ID)
		if err != nil || detail.Spot.Name != "まだ共有していない場所" || !detail.Spot.IsMine {
			t.Fatalf("creator's view = %+v, err = %v", detail, err)
		}
		if !inSearch(alice) {
			t.Fatal("the creator's search does not list their own spot")
		}
		if _, err := spots.CreateReview(ctx, alice, secret.ID, 4, "自分用のメモ"); err != nil {
			t.Fatal(err)
		}
		// Even the creator's public footprint does not rank it: the
		// ranking is the same list for everyone.
		if _, err := visits.Create(ctx, alice, CreateGpsVisitParams{SpotID: secret.ID, VisitedDate: "2026-10-01", IsPublic: &yes}); err != nil {
			t.Fatal(err)
		}
		for _, viewer := range []string{alice, bob} {
			if ranking, err := posts.SpotRanking(ctx, viewer, 20); err != nil || len(ranking) != 0 {
				t.Fatalf("ranking = %+v, err = %v", ranking, err)
			}
		}
	})

	var post *models.PostResponse
	t.Run("after the creator posts", func(t *testing.T) {
		post = mustCreatePost(t, posts, alice, CreatePostParams{Caption: "共有します", SpotID: secret.ID})
		detail, err := spots.Get(ctx, bob, secret.ID)
		if err != nil || detail.Spot.Name != "まだ共有していない場所" || *detail.Spot.Address != "佐久市某所 1-2-3" || detail.Spot.IsMine {
			t.Fatalf("bob's view = %+v, err = %v", detail, err)
		}
		if !inSearch(bob) {
			t.Fatal("search does not list the spot after it was posted")
		}
		if _, err := spots.CreateReview(ctx, bob, secret.ID, 5, "行ってみたい"); err != nil {
			t.Fatal(err)
		}
		if _, err := visits.Create(ctx, bob, CreateGpsVisitParams{SpotID: secret.ID, VisitedDate: "2026-10-02"}); err != nil {
			t.Fatal(err)
		}
		if ranking, _ := posts.SpotRanking(ctx, bob, 20); len(ranking) != 1 || ranking[0].Name != "まだ共有していない場所" {
			t.Fatalf("ranking = %+v", ranking)
		}
	})

	t.Run("after the only post is deleted", func(t *testing.T) {
		if err := posts.Delete(ctx, alice, post.ID); err != nil {
			t.Fatal(err)
		}
		before := [3]int64{rows("posts"), rows("reviews"), rows("gps_visits")}
		hidden(t, bob)
		if after := [3]int64{rows("posts"), rows("reviews"), rows("gps_visits")}; after != before {
			t.Fatalf("rows changed: %v → %v", before, after)
		}
		if ranking, _ := posts.SpotRanking(ctx, bob, 20); len(ranking) != 0 {
			t.Fatalf("ranking = %+v", ranking)
		}
		if _, err := spots.Get(ctx, alice, secret.ID); err != nil {
			t.Fatalf("the creator lost their own spot: %v", err)
		}
		// Another member's post would keep it visible: any post counts.
		mustCreatePost(t, posts, alice, CreatePostParams{Caption: "もう一度共有", SpotID: secret.ID})
		bobPost := mustCreatePost(t, posts, bob, CreatePostParams{Caption: "ボブも投稿", SpotID: secret.ID})
		if bobPost.Spot == nil || bobPost.Spot.ID != secret.ID {
			t.Fatalf("bob's post = %+v", bobPost)
		}
	})

	// Seed and OpenStreetMap spots are visible to everyone from the start.
	t.Run("other sources are unaffected", func(t *testing.T) {
		seeded := addSpot(t, db, "白駒の池", "東信", "自然")
		osm, _, err := places.Register(ctx, alice, RegisterSpotParams{Source: "osm", ExternalID: "N6353535276"})
		if err != nil {
			t.Fatal(err)
		}
		for _, id := range []string{seeded, osm.ID} {
			if _, err := spots.Get(ctx, bob, id); err != nil {
				t.Errorf("Get(%s): %v", id, err)
			}
			if _, err := spots.CreateReview(ctx, bob, id, 5, "口コミ"); err != nil {
				t.Errorf("CreateReview(%s): %v", id, err)
			}
			if _, err := visits.Create(ctx, bob, CreateGpsVisitParams{SpotID: id, VisitedDate: "2026-10-03"}); err != nil {
				t.Errorf("footprint at %s: %v", id, err)
			}
			mustCreatePost(t, posts, bob, CreatePostParams{Caption: "投稿", SpotID: id})
		}
	})
}
