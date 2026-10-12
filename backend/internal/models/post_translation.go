// Prisma model: PostTranslation → @@map("post_translations")
package models

import (
	"time"

	"gorm.io/gorm"
)

// PostTranslation caches a translation. It is unique per
// (source_text_hash, source_lang, target_lang); SourceTextHash is SHA-256 hex.
type PostTranslation struct {
	ID             string    `gorm:"column:id;primaryKey"`
	SourceTextHash string    `gorm:"column:source_text_hash"`
	SourceLang     string    `gorm:"column:source_lang"`
	TargetLang     string    `gorm:"column:target_lang"`
	TranslatedText string    `gorm:"column:translated_text"`
	Provider       string    `gorm:"column:provider"`
	CreatedAt      time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (PostTranslation) TableName() string { return "post_translations" }

func (t *PostTranslation) BeforeCreate(*gorm.DB) error {
	ensureID(&t.ID)
	return nil
}
