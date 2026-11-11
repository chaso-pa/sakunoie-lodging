package entities

import "time"

type APIIdempotencyKey struct {
	ID             uint64  `gorm:"primaryKey"`
	KeyHash        []byte  // SHA-256(Idempotency-Key)
	RequestHash    []byte  // HMAC-SHA256(IDEMPOTENCY_HMAC_KEY, canonical DTO JSON)
	ResponseJSON   *string `gorm:"column:response_json"` // {"registration_public_id": "..."} only
	ResponseStatus *uint16
	CreatedAt      time.Time
	ExpiresAt      time.Time
}

func (APIIdempotencyKey) TableName() string { return "api_idempotency_keys" }
