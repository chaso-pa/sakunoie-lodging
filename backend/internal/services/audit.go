package services

import (
	"encoding/json"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"gorm.io/gorm"
)

// writeAudit records an event. changed_fields_json holds field NAMES only,
// never values (plan P2(c); deviation from §3.6 recorded in CLAUDE.md §13).
func writeAudit(tx *gorm.DB, clock Clock, facilityID uint64, regID *uint64, actor string, adminID *uint64, action string, fields []string) error {
	var js *string
	if fields != nil {
		b, err := json.Marshal(fields)
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
