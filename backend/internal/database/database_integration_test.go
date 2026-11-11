package database_test

import (
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
)

func TestIntegrationSessionIsUTC(t *testing.T) {
	db := testutil.RequireDB(t)
	var tz string
	if err := db.Raw("SELECT @@session.time_zone").Scan(&tz).Error; err != nil {
		t.Fatal(err)
	}
	if tz != "+00:00" {
		t.Fatalf("session time_zone = %q", tz)
	}

	tokyo, _ := time.LoadLocation("Asia/Tokyo")
	instant := time.Date(2026, 12, 1, 9, 30, 0, 123456000, tokyo) // 00:30:00.123456 UTC
	f := entities.Facility{Code: "tz-roundtrip", Name: "TZ", Timezone: "Asia/Tokyo", MaxGuests: 1, IsActive: true, CreatedAt: instant, UpdatedAt: instant}
	if err := db.Create(&f).Error; err != nil {
		t.Fatal(err)
	}
	var stored string
	db.Raw("SELECT DATE_FORMAT(created_at, '%Y-%m-%d %H:%i:%s.%f') FROM facilities WHERE id = ?", f.ID).Scan(&stored)
	if stored != "2026-12-01 00:30:00.123456" {
		t.Fatalf("stored %q, want the UTC wall clock", stored)
	}
	var got entities.Facility
	db.Take(&got, f.ID)
	if !got.CreatedAt.Equal(instant) || got.CreatedAt.Location() != time.UTC {
		t.Fatalf("round trip %s (%s)", got.CreatedAt, got.CreatedAt.Location())
	}
}
