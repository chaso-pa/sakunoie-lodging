package entities

import "time"

type GuestRegistration struct {
	ID                   uint64 `gorm:"primaryKey"`
	PublicID             string
	FacilityID           uint64
	CheckInDate          time.Time // DATE, midnight UTC
	CheckOutDate         time.Time // DATE, midnight UTC
	EstimatedArrivalTime *string   `gorm:"type:time"` // "HH:MM:SS"
	PlannedGuestCount    uint32
	CapacitySnapshot     uint32
	BookingSource        *string
	BookingReference     *string
	TermsVersion         string
	PrivacyVersion       string
	TermsAcceptedAt      time.Time
	PrivacyAcceptedAt    time.Time
	SubmittedAt          time.Time
	IdentityVerifiedAt   *time.Time
	IdentityVerifiedBy   *uint64
	CreatedAt            time.Time
	UpdatedAt            time.Time
	DeletedAt            *time.Time
}

func (GuestRegistration) TableName() string { return "guest_registrations" }
