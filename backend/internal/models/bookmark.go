// Prisma model: Bookmark → @@map("bookmarks")
package models

import (
	"time"

	"gorm.io/gorm"
)

// Bookmark is unique per (member_id, post_id).
type Bookmark struct {
	ID        string    `gorm:"column:id;primaryKey"`
	MemberID  string    `gorm:"column:member_id"`
	PostID    string    `gorm:"column:post_id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (Bookmark) TableName() string { return "bookmarks" }

func (b *Bookmark) BeforeCreate(*gorm.DB) error {
	ensureID(&b.ID)
	return nil
}
