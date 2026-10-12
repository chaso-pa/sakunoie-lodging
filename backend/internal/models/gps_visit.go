// Prisma model: GpsVisit → @@map("gps_visits")
package models

import (
	"time"

	"gorm.io/gorm"
)

// GpsVisit holds a member's footprint. Latitude and Longitude are private
// unless ExactPublic is true: never return them to other members otherwise.
type GpsVisit struct {
	ID          string    `gorm:"column:id;primaryKey"`
	MemberID    string    `gorm:"column:member_id"`
	SpotID      *string   `gorm:"column:spot_id"`
	PlaceName   string    `gorm:"column:place_name"`
	Emoji       string    `gorm:"column:emoji"`
	Area        string    `gorm:"column:area"`
	VisitedDate time.Time `gorm:"column:visited_date;type:date"`
	Weather     *string   `gorm:"column:weather"`
	Temperature *float64  `gorm:"column:temperature"`
	Note        *string   `gorm:"column:note"`
	IsPublic    bool      `gorm:"column:is_public"`
	ExactPublic bool      `gorm:"column:exact_public"`
	Latitude    *float64  `gorm:"column:latitude"`
	Longitude   *float64  `gorm:"column:longitude"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`

	Member Member `gorm:"foreignKey:MemberID;references:ID"`
}

func (GpsVisit) TableName() string { return "gps_visits" }

func (v *GpsVisit) BeforeCreate(*gorm.DB) error {
	ensureID(&v.ID)
	return nil
}
