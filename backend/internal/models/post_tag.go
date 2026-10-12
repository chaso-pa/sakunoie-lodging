// Prisma model: PostTag → @@map("post_tags")
package models

import "gorm.io/gorm"

type PostTag struct {
	ID     string `gorm:"column:id;primaryKey"`
	PostID string `gorm:"column:post_id"`
	Tag    string `gorm:"column:tag"`
}

func (PostTag) TableName() string { return "post_tags" }

func (t *PostTag) BeforeCreate(*gorm.DB) error {
	ensureID(&t.ID)
	return nil
}
