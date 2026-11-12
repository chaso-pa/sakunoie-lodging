package services

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"gorm.io/gorm"
)

//go:embed policy_versions.json
var policyVersionsJSON []byte

// PolicyVersions are the terms/privacy versions a guest must accept.
type PolicyVersions struct {
	TermsVersion   string `json:"terms_version"`
	PrivacyVersion string `json:"privacy_version"`
}

// CurrentPolicyVersions returns the embedded versions.
func CurrentPolicyVersions() PolicyVersions {
	var p PolicyVersions
	if err := json.Unmarshal(policyVersionsJSON, &p); err != nil || p.TermsVersion == "" || p.PrivacyVersion == "" {
		panic(fmt.Sprintf("services: invalid embedded policy_versions.json: %v", err))
	}
	return p
}

type FacilityService struct {
	db     *gorm.DB
	policy PolicyVersions
}

func NewFacilityService(db *gorm.DB) *FacilityService {
	return &FacilityService{db: db, policy: CurrentPolicyVersions()}
}

// GetPublic returns public facility data for an active facility. The
// internal numeric ID is never exposed.
func (s *FacilityService) GetPublic(ctx context.Context, code string) (*models.FacilityPublic, error) {
	f, err := loadActiveFacility(s.db.WithContext(ctx), code)
	if err != nil {
		if errors.Is(err, ErrFacilityUnavailable) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &models.FacilityPublic{
		Code:           f.Code,
		Name:           f.Name,
		MaxGuests:      int(f.MaxGuests),
		Timezone:       f.Timezone,
		TermsVersion:   s.policy.TermsVersion,
		PrivacyVersion: s.policy.PrivacyVersion,
	}, nil
}

func loadActiveFacility(db *gorm.DB, code string) (*entities.Facility, error) {
	var f entities.Facility
	err := db.Where("code = ? AND is_active = ?", code, true).Take(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrFacilityUnavailable
	}
	if err != nil {
		return nil, err
	}
	return &f, nil
}

// notDeleted is the explicit soft-delete scope (DeletedAt is a plain
// pointer, so GORM adds no implicit scope).
func notDeleted(db *gorm.DB) *gorm.DB {
	return db.Where("guest_registrations.deleted_at IS NULL")
}
