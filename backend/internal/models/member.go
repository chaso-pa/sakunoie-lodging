// Prisma model: Member → @@map("members")
package models

import (
	"time"

	"gorm.io/gorm"
)

type Member struct {
	ID                string    `gorm:"column:id;primaryKey"`
	Nickname          string    `gorm:"column:nickname"`
	AvatarURL         *string   `gorm:"column:avatar_url"`
	PreferredLanguage string    `gorm:"column:preferred_language"`
	TransferCodeHash  string    `gorm:"column:transfer_code_hash"`
	SessionVersion    int       `gorm:"column:session_version"`
	CreatedAt         time.Time `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt         time.Time `gorm:"column:updated_at;autoUpdateTime"`
}

func (Member) TableName() string { return "members" }

func (m *Member) BeforeCreate(*gorm.DB) error {
	ensureID(&m.ID)
	return nil
}
