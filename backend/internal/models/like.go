// Prisma model: Like → @@map("likes")
package models

import (
	"time"

	"gorm.io/gorm"
)

// Like is unique per (member_id, post_id).
type Like struct {
	ID        string    `gorm:"column:id;primaryKey"`
	MemberID  string    `gorm:"column:member_id"`
	PostID    string    `gorm:"column:post_id"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (Like) TableName() string { return "likes" }

func (l *Like) BeforeCreate(*gorm.DB) error {
	ensureID(&l.ID)
	return nil
}
