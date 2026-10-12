// Prisma model: AiSpot → @@map("ai_spots")
package models

import "gorm.io/gorm"

// AiSpot is static reference data for the AI travel assistant. Purposes,
// Genres and Weathers each hold a JSON array encoded as a string.
type AiSpot struct {
	ID          string `gorm:"column:id;primaryKey"`
	Name        string `gorm:"column:name"`
	Purposes    string `gorm:"column:purposes"`
	Area        string `gorm:"column:area"`
	Kind        string `gorm:"column:kind"`
	DriveMin    int    `gorm:"column:drive_min"`
	TransitMin  int    `gorm:"column:transit_min"`
	StayMin     int    `gorm:"column:stay_min"`
	Genres      string `gorm:"column:genres"`
	Weathers    string `gorm:"column:weathers"`
	Description string `gorm:"column:description"`
	MapQuery    string `gorm:"column:map_query"`
	SearchURL   string `gorm:"column:search_url"`
}

func (AiSpot) TableName() string { return "ai_spots" }

func (s *AiSpot) BeforeCreate(*gorm.DB) error {
	ensureID(&s.ID)
	return nil
}
