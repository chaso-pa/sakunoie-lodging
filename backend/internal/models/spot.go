// Prisma model: Spot → @@map("spots")
package models

import (
	"time"

	"gorm.io/gorm"
)

// Where a spot row came from.
const (
	SpotSourceSeed   = "seed"   // curated by the operator (cmd/seed)
	SpotSourceOSM    = "osm"    // chosen from an OpenStreetMap search result
	SpotSourceGoogle = "google" // chosen from a Google Places result
	SpotSourceManual = "manual" // typed in by a member
)

// Spot is a place a post, review or footprint can point at.
//
// A row with Source "google" holds the place ID and nothing else from
// Google: Name, Category, Area and Description stay empty, Emoji is the
// default pin, and OriginalName, Address, Latitude, Longitude and MapURL
// stay NULL. Google's terms allow storing the place ID only; everything
// shown for such a spot is fetched when it is read (services.PlaceService).
type Spot struct {
	ID                string    `gorm:"column:id;primaryKey"`
	Name              string    `gorm:"column:name"`
	Emoji             string    `gorm:"column:emoji"`
	Area              string    `gorm:"column:area"`
	Category          string    `gorm:"column:category"`
	Description       string    `gorm:"column:description"`
	IsSample          bool      `gorm:"column:is_sample"`
	Source            string    `gorm:"column:source"`
	ExternalID        *string   `gorm:"column:external_id"`
	ExternalKey       *string   `gorm:"column:external_key"`
	OriginalName      *string   `gorm:"column:original_name"`
	Address           *string   `gorm:"column:address"`
	Latitude          *float64  `gorm:"column:latitude"`
	Longitude         *float64  `gorm:"column:longitude"`
	MapURL            *string   `gorm:"column:map_url"`
	CreatedByMemberID *string   `gorm:"column:created_by_member_id"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Spot) TableName() string { return "spots" }

func (s *Spot) BeforeCreate(*gorm.DB) error {
	ensureID(&s.ID)
	// The column default only applies when the column is left out of the
	// INSERT, and GORM always writes it.
	if s.Source == "" {
		s.Source = SpotSourceSeed
	}
	return nil
}
