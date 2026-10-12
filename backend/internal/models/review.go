// Prisma model: Review → @@map("reviews")
package models

import (
	"time"

	"gorm.io/gorm"
)

type Review struct {
	ID        string    `gorm:"column:id;primaryKey"`
	MemberID  string    `gorm:"column:member_id"`
	SpotID    string    `gorm:"column:spot_id"`
	Stars     int       `gorm:"column:stars"`
	Text      string    `gorm:"column:text"`
	IsSample  bool      `gorm:"column:is_sample"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`

	Member Member `gorm:"foreignKey:MemberID;references:ID"`
}

func (Review) TableName() string { return "reviews" }

func (r *Review) BeforeCreate(*gorm.DB) error {
	ensureID(&r.ID)
	return nil
}
