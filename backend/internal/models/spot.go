// Prisma model: Spot → @@map("spots")
package models

import (
	"time"

	"gorm.io/gorm"
)

type Spot struct {
	ID          string    `gorm:"column:id;primaryKey"`
	Name        string    `gorm:"column:name"`
	Emoji       string    `gorm:"column:emoji"`
	Area        string    `gorm:"column:area"`
	Category    string    `gorm:"column:category"`
	Description string    `gorm:"column:description"`
	IsSample    bool      `gorm:"column:is_sample"`
	CreatedAt   time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Spot) TableName() string { return "spots" }

func (s *Spot) BeforeCreate(*gorm.DB) error {
	ensureID(&s.ID)
	return nil
}
