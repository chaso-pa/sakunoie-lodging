// Prisma model: Post → @@map("posts")
package models

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	ID          string     `gorm:"column:id;primaryKey"`
	MemberID    string     `gorm:"column:member_id"`
	SpotID      *string    `gorm:"column:spot_id"`
	Caption     string     `gorm:"column:caption"`
	Area        *string    `gorm:"column:area"`
	Language    string     `gorm:"column:language"`
	Weather     *string    `gorm:"column:weather"`
	Temperature *float64   `gorm:"column:temperature"`
	VisitedAt   *time.Time `gorm:"column:visited_at"`
	IsSample    bool       `gorm:"column:is_sample"`
	CreatedAt   time.Time  `gorm:"column:created_at;autoCreateTime"`
	UpdatedAt   time.Time  `gorm:"column:updated_at;autoUpdateTime"`

	Member Member      `gorm:"foreignKey:MemberID;references:ID"`
	Spot   *Spot       `gorm:"foreignKey:SpotID;references:ID"`
	Media  []PostMedia `gorm:"foreignKey:PostID;references:ID"`
	Tags   []PostTag   `gorm:"foreignKey:PostID;references:ID"`
}

func (Post) TableName() string { return "posts" }

func (p *Post) BeforeCreate(*gorm.DB) error {
	ensureID(&p.ID)
	return nil
}
