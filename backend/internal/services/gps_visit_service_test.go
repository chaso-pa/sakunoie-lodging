package services

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
)

func TestVisitResponseCoordinates(t *testing.T) {
	lat, lon := 35.123456, 139.654321
	cases := []struct {
		name                  string
		isPublic, exactPublic bool
		viewer                string
		wantCoordinates       bool
	}{
		{"owner, private", false, false, "owner", true},
		{"owner, public", true, false, "owner", true},
		{"owner, exact", true, true, "owner", true},
		{"other, public only", true, false, "other", false},
		{"other, exact public", true, true, "other", true},
		// A private visit must never be passed in for another member; if it
		// is, the coordinates still stay behind.
		{"other, private", false, false, "other", false},
		{"other, private but exact flag set", false, true, "other", false},
		{"no viewer", true, false, "", false},
	}
	for _, c := range cases {
		v := &models.GpsVisit{MemberID: "owner", IsPublic: c.isPublic, ExactPublic: c.exactPublic, Latitude: &lat, Longitude: &lon}
		resp := visitResponse(v, c.viewer)
		got := resp.Latitude != nil || resp.Longitude != nil
		if got != c.wantCoordinates {
			t.Errorf("%s: coordinates present = %v, want %v", c.name, got, c.wantCoordinates)
		}
		if (resp.Latitude == nil) != (resp.Longitude == nil) {
			t.Errorf("%s: only one coordinate present", c.name)
		}
		raw, _ := json.Marshal(resp)
		if !c.wantCoordinates && (strings.Contains(string(raw), "35.12") || strings.Contains(string(raw), "139.65")) {
			t.Errorf("%s: coordinates in the JSON: %s", c.name, raw)
		}
	}
}

func TestGpsVisitPrivacyIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	s := NewGpsVisitService(db)
	alice, bob := addMember(t, db, "Alice"), addMember(t, db, "Bob")
	pond := addSpot(t, db, "白駒の池", "東信", "自然")
	lat, lon := 35.123456, 139.654321
	yes, no := true, false

	create := func(p CreateGpsVisitParams) *models.GpsVisitResponse {
		t.Helper()
		p.VisitedDate = "2026-10-08"
		p.Latitude, p.Longitude = &lat, &lon
		v, err := s.Create(ctx, alice, p)
		if err != nil {
			t.Fatal(err)
		}
		return v
	}
	// Nothing said about visibility: private, exact position not public.
	omitted := create(CreateGpsVisitParams{PlaceName: "言い忘れた場所"})
	if omitted.IsPublic || omitted.ExactPublic {
		t.Fatalf("defaults = public %v, exact %v; want false, false", omitted.IsPublic, omitted.ExactPublic)
	}
	if v := mustPublicByID(t, s, bob, omitted.ID); v != nil {
		t.Fatalf("a visit created without is_public is in another member's public list: %+v", v)
	}
	if err := s.Delete(ctx, alice, omitted.ID); err != nil {
		t.Fatal(err)
	}
	// Public, exact position not public.
	byDefault := create(CreateGpsVisitParams{SpotID: pond, IsPublic: &yes})
	if !byDefault.IsPublic || byDefault.ExactPublic {
		t.Fatalf("public visit = public %v, exact %v; want true, false", byDefault.IsPublic, byDefault.ExactPublic)
	}
	if byDefault.PlaceName != "白駒の池" || byDefault.Emoji != "🌲" || byDefault.Area != "東信" {
		t.Fatalf("spot fields not filled in: %+v", byDefault)
	}
	private := create(CreateGpsVisitParams{PlaceName: "自宅の近く", IsPublic: &no, Note: "ひみつ"})
	exact := create(CreateGpsVisitParams{PlaceName: "展望台", IsPublic: &yes, ExactPublic: &yes})
	if private.Emoji != "📍" || private.Area != "未設定" {
		t.Fatalf("defaults for a free-text place: %+v", private)
	}
	// The creator gets their coordinates back whatever the settings.
	for _, v := range []*models.GpsVisitResponse{byDefault, private, exact} {
		if v.Latitude == nil || *v.Latitude != lat || v.Longitude == nil || *v.Longitude != lon || !v.IsMine {
			t.Fatalf("owner's own visit lost its coordinates: %+v", v)
		}
	}

	// Alice sees all three of hers, with coordinates.
	mine, err := s.ListMine(ctx, alice)
	if err != nil || len(mine) != 3 {
		t.Fatalf("alice's visits = %d, %v", len(mine), err)
	}
	for _, v := range mine {
		if v.Latitude == nil || v.Longitude == nil {
			t.Fatalf("alice's own visit %s has no coordinates", v.PlaceName)
		}
	}
	// Bob has none of his own.
	if bobs, _ := s.ListMine(ctx, bob); len(bobs) != 0 {
		t.Fatalf("bob's own list has %d visits", len(bobs))
	}

	// Bob's view of the public list.
	public, err := s.ListPublic(ctx, bob)
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]models.GpsVisitResponse{}
	for _, v := range public {
		seen[v.ID] = v
	}
	if _, ok := seen[private.ID]; ok || len(public) != 2 {
		t.Fatalf("the private visit is in bob's public list (%d visits)", len(public))
	}
	if v := seen[byDefault.ID]; v.Latitude != nil || v.Longitude != nil || v.IsMine || v.Member.Nickname != "Alice" {
		t.Fatalf("public visit without exact_public shows coordinates to bob: %+v", v)
	}
	if v := seen[exact.ID]; v.Latitude == nil || *v.Latitude != lat || v.Longitude == nil || *v.Longitude != lon {
		t.Fatalf("exact_public visit hides coordinates: %+v", v)
	}
	raw, _ := json.Marshal(public)
	if strings.Contains(string(raw), "ひみつ") || strings.Contains(string(raw), "自宅の近く") {
		t.Fatal("the private visit's text is in bob's list")
	}
	// Alice's view of the same list includes her own coordinates.
	for _, v := range mustPublic(t, s, alice) {
		if v.Latitude == nil || !v.IsMine {
			t.Fatalf("alice's own public visit lost coordinates: %+v", v)
		}
	}

	// Bob cannot change or delete Alice's visits. The private one looks
	// missing to him; the public ones are refused.
	if err := s.Delete(ctx, bob, private.ID); !errors.Is(err, ErrVisitNotFound) {
		t.Fatalf("bob deleting a private visit: %v", err)
	}
	if err := s.Delete(ctx, bob, exact.ID); !errors.Is(err, ErrForbidden) {
		t.Fatalf("bob deleting a public visit: %v", err)
	}
	if _, err := s.SetVisibility(ctx, bob, byDefault.ID, true, &yes); !errors.Is(err, ErrForbidden) {
		t.Fatalf("bob changing visibility: %v", err)
	}
	if _, err := s.SetVisibility(ctx, bob, private.ID, true, &yes); !errors.Is(err, ErrVisitNotFound) {
		t.Fatalf("bob publishing a private visit: %v", err)
	}
	if v := mustPublicByID(t, s, bob, byDefault.ID); v == nil || v.Latitude != nil || v.ExactPublic {
		t.Fatalf("bob's refused change took effect: %+v", v)
	}
	if got, _ := s.ListMine(ctx, alice); len(got) != 3 {
		t.Fatalf("alice has %d visits after bob's attempts", len(got))
	}

	// Alice makes the exact one private: it disappears for bob.
	updated, err := s.SetVisibility(ctx, alice, exact.ID, false, nil)
	if err != nil || updated.IsPublic || !updated.ExactPublic || updated.Latitude == nil {
		t.Fatalf("visibility update = %+v, %v", updated, err)
	}
	if v := mustPublicByID(t, s, bob, exact.ID); v != nil {
		t.Fatalf("a visit made private is still listed: %+v", v)
	}
	// And publishes the exact position of the first: bob now sees it.
	if _, err := s.SetVisibility(ctx, alice, byDefault.ID, true, &yes); err != nil {
		t.Fatal(err)
	}
	if v := mustPublicByID(t, s, bob, byDefault.ID); v == nil || v.Latitude == nil {
		t.Fatalf("exact position not shown after opting in: %+v", v)
	}

	if err := s.Delete(ctx, alice, private.ID); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete(ctx, alice, private.ID); !errors.Is(err, ErrVisitNotFound) {
		t.Fatalf("second delete: %v", err)
	}
}

func mustPublic(t *testing.T, s *GpsVisitService, viewer string) []models.GpsVisitResponse {
	t.Helper()
	visits, err := s.ListPublic(context.Background(), viewer)
	if err != nil {
		t.Fatal(err)
	}
	return visits
}

func mustPublicByID(t *testing.T, s *GpsVisitService, viewer, id string) *models.GpsVisitResponse {
	t.Helper()
	for _, v := range mustPublic(t, s, viewer) {
		if v.ID == id {
			return &v
		}
	}
	return nil
}

func TestCreateGpsVisitValidationIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	ctx := context.Background()
	s := NewGpsVisitService(db)
	alice := addMember(t, db, "Alice")
	lat, lon, far, cold := 35.0, 139.0, 200.0, -100.0

	// Without a position (permission denied in the browser) a visit is
	// still recorded.
	v, err := s.Create(ctx, alice, CreateGpsVisitParams{PlaceName: "カフェ", VisitedDate: "2026-10-08", Weather: "晴れ"})
	if err != nil || v.Latitude != nil || v.Longitude != nil || v.VisitedDate != "2026-10-08" || *v.Weather != "晴れ" {
		t.Fatalf("visit without position = %+v, %v", v, err)
	}

	var validation *ValidationError
	for name, p := range map[string]CreateGpsVisitParams{
		"no place":       {VisitedDate: "2026-10-08"},
		"bad date":       {PlaceName: "x", VisitedDate: "yesterday"},
		"latitude only":  {PlaceName: "x", VisitedDate: "2026-10-08", Latitude: &lat},
		"longitude only": {PlaceName: "x", VisitedDate: "2026-10-08", Longitude: &lon},
		"out of range":   {PlaceName: "x", VisitedDate: "2026-10-08", Latitude: &far, Longitude: &lon},
		"temperature":    {PlaceName: "x", VisitedDate: "2026-10-08", Temperature: &cold},
		"unknown spot":   {SpotID: "no-such-spot", VisitedDate: "2026-10-08"},
		"long note":      {PlaceName: "x", VisitedDate: "2026-10-08", Note: strings.Repeat("あ", 2001)},
	} {
		if _, err := s.Create(ctx, alice, p); !errors.As(err, &validation) {
			t.Errorf("%s: err = %v, want a ValidationError", name, err)
		}
	}
}

// The column defaults agree with the rule: a row inserted without the
// visibility columns is private and its exact position hidden.
func TestGpsVisitColumnDefaultsIntegration(t *testing.T) {
	db := testutil.NewDB(t, "services").Gorm
	alice, bob := addMember(t, db, "Alice"), addMember(t, db, "Bob")

	err := db.Exec(`INSERT INTO gps_visits (id, member_id, place_name, emoji, area, visited_date, latitude, longitude)
		VALUES ('raw-insert', ?, '直接入れた行', '📍', '未設定', '2026-10-08', 35.123456, 139.654321)`, alice).Error
	if err != nil {
		t.Fatal(err)
	}
	var row models.GpsVisit
	if err := db.First(&row, "id = ?", "raw-insert").Error; err != nil {
		t.Fatal(err)
	}
	if row.IsPublic || row.ExactPublic {
		t.Fatalf("column defaults: is_public %v, exact_public %v; want false, false", row.IsPublic, row.ExactPublic)
	}
	if v := mustPublicByID(t, NewGpsVisitService(db), bob, "raw-insert"); v != nil {
		t.Fatalf("a row inserted without is_public is in another member's public list: %+v", v)
	}
}
