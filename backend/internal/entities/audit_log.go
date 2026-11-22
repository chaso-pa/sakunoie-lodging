package entities

import "time"

const (
	ActorGuestToken = "guest_token"
	ActorAdmin      = "admin"
	ActorSystem     = "system"
)

type AuditLog struct {
	ID                uint64 `gorm:"primaryKey"`
	FacilityID        uint64
	RegistrationID    *uint64
	ActorType         string
	ActorAdminUserID  *uint64
	Action            string
	ChangedFieldsJSON *string `gorm:"column:changed_fields_json"` // JSON array of field names only
	CreatedAt         time.Time
}

func (AuditLog) TableName() string { return "audit_logs" }
