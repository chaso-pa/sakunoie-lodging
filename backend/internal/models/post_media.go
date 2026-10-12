// Prisma model: PostMedia → @@map("post_media")
package models

import (
	"time"

	"gorm.io/gorm"
)

type PostMedia struct {
	ID        string    `gorm:"column:id;primaryKey"`
	PostID    string    `gorm:"column:post_id"`
	MediaURL  string    `gorm:"column:media_url"`
	MediaType string    `gorm:"column:media_type"`
	SortOrder int       `gorm:"column:sort_order"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (PostMedia) TableName() string { return "post_media" }

func (m *PostMedia) BeforeCreate(*gorm.DB) error {
	ensureID(&m.ID)
	return nil
}
