package entities

import "time"

// RegistrationInvitation is a per-booking invitation link issued by the host.
// A registration can only be created with a valid invitation, and each one
// authorizes at most one registration (CLAUDE.md §13, 2026-11-15).
type RegistrationInvitation struct {
	ID               uint64 `gorm:"primaryKey"`
	PublicID         string
	FacilityID       uint64
	TokenHash        []byte    // SHA-256 of the raw token; the raw token is never stored
	CheckInDate      time.Time // DATE (midnight UTC, see utils.CivilDateFromDB)
	CheckOutDate     time.Time // DATE; the expiry is the end of this day in facility time
	BookingReference *string
	Note             *string // for the host only; never returned to guests
	ExpiresAt        time.Time
	UsedAt           *time.Time
	RegistrationID   *uint64
	RevokedAt        *time.Time
	CreatedBy        uint64
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

func (RegistrationInvitation) TableName() string { return "registration_invitations" }
