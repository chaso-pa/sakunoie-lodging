package entities

import "time"

type RegistrationAccessToken struct {
	ID             uint64 `gorm:"primaryKey"`
	RegistrationID uint64
	TokenHash      []byte // SHA-256 of the raw token; the raw token is never stored
	ExpiresAt      time.Time
	RevokedAt      *time.Time
	LastUsedAt     *time.Time
	CreatedAt      time.Time
}

func (RegistrationAccessToken) TableName() string { return "registration_access_tokens" }
