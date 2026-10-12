package services

import (
	"context"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/database"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
	mysqldriver "github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
)

func TestParseCalendarDate(t *testing.T) {
	for _, ok := range []string{"2026-10-08", "2024-02-29", "1999-12-31"} {
		if d, err := models.ParseCalendarDate(ok); err != nil || d.String() != ok {
			t.Errorf("ParseCalendarDate(%q) = %q, %v", ok, d, err)
		}
	}
	for _, bad := range []string{"", "2026-13-01", "2026-02-30", "2026-10-8", "10/08/2026", "2026-10-08T00:00:00Z", "yesterday"} {
		if _, err := models.ParseCalendarDate(bad); err == nil {
			t.Errorf("ParseCalendarDate(%q) accepted", bad)
		}
	}

	// Scanning takes the stored wall-clock date, whatever zone the driver
	// attached to it.
	for _, loc := range []*time.Location{time.UTC, time.FixedZone("JST", 9*3600), time.FixedZone("PST", -8*3600)} {
		var d models.CalendarDate
		if err := d.Scan(time.Date(2026, 10, 8, 0, 0, 0, 0, loc)); err != nil || d != "2026-10-08" {
			t.Errorf("Scan(time in %s) = %q, %v", loc, d, err)
		}
	}
	var d models.CalendarDate
	if err := d.Scan([]byte("2026-10-08 00:00:00.000")); err != nil || d != "2026-10-08" {
		t.Errorf("Scan(bytes) = %q, %v", d, err)
	}
	if err := d.Scan(42); err == nil {
		t.Error("Scan(int) accepted")
	}
}

// withProcessZone opens a new connection to the test database with the
// process's local zone set to name, as if the server ran there: both Go's
// time.Local and the driver's loc=Local follow it.
func withProcessZone(t *testing.T, dsn, name string, fn func(db *gorm.DB)) {
	t.Helper()
	loc, err := time.LoadLocation(name)
	if err != nil {
		t.Fatal(err)
	}
	previous := time.Local
	time.Local = loc
	defer func() { time.Local = previous }()

	db, err := database.Open(dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if sqlDB, err := db.DB(); err == nil {
			sqlDB.Close()
		}
	}()
	fn(db)
}

// A date a member types is the date stored and the date returned, whichever
// zone the server process runs in, and whichever zone later reads it back.
func TestCalendarDatesIgnoreServerTimeZoneIntegration(t *testing.T) {
	base := testutil.NewDB(t, "services")
	ctx := context.Background()
	alice := addMember(t, base.Gorm, "Alice")
	zones := []string{"UTC", "Asia/Tokyo", "America/Los_Angeles"}
	const typed = "2026-10-08"

	for _, zone := range zones {
		withProcessZone(t, base.DSN, zone, func(db *gorm.DB) {
			visit, err := NewGpsVisitService(db).Create(ctx, alice, CreateGpsVisitParams{PlaceName: zone, VisitedDate: typed})
			if err != nil {
				t.Fatal(err)
			}
			if visit.VisitedDate != typed {
				t.Errorf("%s: visit response visited_date = %q, want %q", zone, visit.VisitedDate, typed)
			}
			post, err := NewPostService(db, newTrackingUploader()).Create(ctx, alice, CreatePostParams{Caption: zone, VisitedAt: typed})
			if err != nil {
				t.Fatal(err)
			}
			if post.VisitedAt == nil || *post.VisitedAt != typed {
				t.Errorf("%s: post response visited_at = %v, want %q", zone, post.VisitedAt, typed)
			}

			// What MySQL holds, as text, with no driver conversion.
			var storedVisit, storedPost string
			db.Raw("SELECT CAST(visited_date AS CHAR) FROM gps_visits WHERE id = ?", visit.ID).Scan(&storedVisit)
			db.Raw("SELECT CAST(visited_at AS CHAR) FROM posts WHERE id = ?", post.ID).Scan(&storedPost)
			if storedVisit != typed {
				t.Errorf("%s: stored visited_date = %q, want %q", zone, storedVisit, typed)
			}
			if storedPost != typed+" 00:00:00.000" {
				t.Errorf("%s: stored visited_at = %q, want midnight of %s", zone, storedPost, typed)
			}
		})
	}

	// Rows written in every zone read back the same in every zone, and the
	// month filter finds them all.
	for _, zone := range zones {
		withProcessZone(t, base.DSN, zone, func(db *gorm.DB) {
			visits, err := NewGpsVisitService(db).ListMine(ctx, alice)
			if err != nil || len(visits) != len(zones) {
				t.Fatalf("%s: %d visits, %v", zone, len(visits), err)
			}
			for _, v := range visits {
				if v.VisitedDate != typed {
					t.Errorf("read in %s, written in %s: visited_date = %q", zone, v.PlaceName, v.VisitedDate)
				}
			}
			posts := NewPostService(db, newTrackingUploader())
			mine, err := posts.Mine(ctx, alice)
			if err != nil || len(mine) != len(zones) {
				t.Fatalf("%s: %d posts, %v", zone, len(mine), err)
			}
			for _, p := range mine {
				if p.VisitedAt == nil || *p.VisitedAt != typed {
					t.Errorf("read in %s, written in %s: visited_at = %v", zone, p.Caption, p.VisitedAt)
				}
			}
			for month, want := range map[string]int{"2026-10": len(zones), "2026-09": 0, "2026-11": 0} {
				if _, total, err := posts.List(ctx, alice, PostFilter{Month: month}); err != nil || int(total) != want {
					t.Errorf("%s: month %s matched %d posts, want %d (%v)", zone, month, total, want, err)
				}
			}
		})
	}

	// The case that used to shift the date: the database connection in one
	// zone (loc=UTC) and the process in another. A time.Time for midnight
	// in Tokyo is 15:00 the day before in UTC, so a DATE column got the 7th.
	utcCfg, err := mysqldriver.ParseDSN(base.DSN)
	if err != nil {
		t.Fatal(err)
	}
	utcCfg.Loc = time.UTC
	for _, zone := range []string{"Asia/Tokyo", "America/Los_Angeles"} {
		withProcessZone(t, utcCfg.FormatDSN(), zone, func(db *gorm.DB) {
			visit, err := NewGpsVisitService(db).Create(ctx, alice, CreateGpsVisitParams{PlaceName: "utc-dsn " + zone, VisitedDate: typed})
			if err != nil {
				t.Fatal(err)
			}
			var stored string
			db.Raw("SELECT CAST(visited_date AS CHAR) FROM gps_visits WHERE id = ?", visit.ID).Scan(&stored)
			if visit.VisitedDate != typed || stored != typed {
				t.Errorf("loc=UTC connection, process in %s: response %q, stored %q, want %q", zone, visit.VisitedDate, stored, typed)
			}
			post, err := NewPostService(db, newTrackingUploader()).Create(ctx, alice, CreatePostParams{Caption: "utc-dsn " + zone, VisitedAt: typed})
			if err != nil {
				t.Fatal(err)
			}
			db.Raw("SELECT CAST(visited_at AS CHAR) FROM posts WHERE id = ?", post.ID).Scan(&stored)
			if post.VisitedAt == nil || *post.VisitedAt != typed || stored != typed+" 00:00:00.000" {
				t.Errorf("loc=UTC connection, process in %s: response %v, stored %q", zone, post.VisitedAt, stored)
			}
			if err := NewPostService(db, newTrackingUploader()).Delete(ctx, alice, post.ID); err != nil {
				t.Fatal(err)
			}
		})
	}

	// The last day of a month stays in that month.
	withProcessZone(t, base.DSN, "America/Los_Angeles", func(db *gorm.DB) {
		posts := NewPostService(db, newTrackingUploader())
		if _, err := posts.Create(ctx, alice, CreatePostParams{Caption: "月末", VisitedAt: "2026-08-31"}); err != nil {
			t.Fatal(err)
		}
		if _, total, _ := posts.List(ctx, alice, PostFilter{Month: "2026-08"}); total != 1 {
			t.Errorf("2026-08-31 is not in month 2026-08 (matched %d)", total)
		}
		if _, total, _ := posts.List(ctx, alice, PostFilter{Month: "2026-09"}); total != 0 {
			t.Errorf("2026-08-31 leaked into month 2026-09")
		}
	})
}

// A post without a visit date is filed under the month it was posted in,
// read in Japan time, in whatever zone the server runs.
func TestMonthFilterUsesJapanTimeForCreatedAtIntegration(t *testing.T) {
	base := testutil.NewDB(t, "services")
	ctx := context.Background()
	alice := addMember(t, base.Gorm, "Alice")
	// 01:00 on 1 October in Japan, which is still 30 September in UTC and
	// in California.
	postedAt := time.Date(2026, 9, 30, 16, 0, 0, 0, time.UTC)

	for _, zone := range []string{"UTC", "Asia/Tokyo", "America/Los_Angeles"} {
		withProcessZone(t, base.DSN, zone, func(db *gorm.DB) {
			posts := NewPostService(db, newTrackingUploader())
			post, err := posts.Create(ctx, alice, CreatePostParams{Caption: zone})
			if err != nil {
				t.Fatal(err)
			}
			if err := db.Model(&models.Post{}).Where("id = ?", post.ID).UpdateColumn("created_at", postedAt).Error; err != nil {
				t.Fatal(err)
			}
			inMonth := func(month string) bool {
				list, _, err := posts.List(ctx, alice, PostFilter{Month: month, Limit: 50})
				if err != nil {
					t.Fatal(err)
				}
				for _, p := range list {
					if p.ID == post.ID {
						return true
					}
				}
				return false
			}
			if !inMonth("2026-10") || inMonth("2026-09") {
				t.Errorf("%s: a post made at 01:00 JST on 1 October: in 2026-10 = %v, in 2026-09 = %v; want true, false",
					zone, inMonth("2026-10"), inMonth("2026-09"))
			}
			// Remove it so the next zone's rows are judged alone.
			if err := posts.Delete(ctx, alice, post.ID); err != nil {
				t.Fatal(err)
			}
		})
	}
	if _, _, err := NewPostService(base.Gorm, newTrackingUploader()).List(ctx, alice, PostFilter{Month: "2026-13"}); err == nil {
		t.Error("month 2026-13 was accepted")
	}
}
