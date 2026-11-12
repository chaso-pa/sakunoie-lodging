package services

import (
	"encoding/json"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"gorm.io/gorm"
)

// writeAudit records an event. changed_fields_json holds field NAMES only,
// never values (plan P2(c); deviation from §3.6 recorded in CLAUDE.md §13).
func writeAudit(tx *gorm.DB, clock Clock, facilityID uint64, regID *uint64, actor string, adminID *uint64, action string, fields []string) error {
	var meta any
	if fields != nil {
		meta = fields
	}
	return writeAuditMeta(tx, clock, facilityID, regID, actor, adminID, action, meta)
}

// writeAuditMeta records an event with non-PII metadata in
// changed_fields_json (e.g. CSV export filters and row count). Callers must
// never pass personal data or exported content.
func writeAuditMeta(tx *gorm.DB, clock Clock, facilityID uint64, regID *uint64, actor string, adminID *uint64, action string, meta any) error {
	var js *string
	if meta != nil {
		b, err := json.Marshal(meta)
		if err != nil {
			return err
		}
		s := string(b)
		js = &s
	}
	return tx.Create(&entities.AuditLog{
		FacilityID:        facilityID,
		RegistrationID:    regID,
		ActorType:         actor,
		ActorAdminUserID:  adminID,
		Action:            action,
		ChangedFieldsJSON: js,
		CreatedAt:         clock.Now(),
	}).Error
}
