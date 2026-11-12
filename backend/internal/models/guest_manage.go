package models

import "time"

// Manage endpoints authenticate with the edit token as
// `Authorization: Bearer <token>` (security scheme editToken).

type ManageFacility struct {
	Code      string `json:"code"`
	Name      string `json:"name"`
	Timezone  string `json:"timezone"`
	MaxGuests int    `json:"max_guests"`
}

// ManageGuest is a masked guest view. Full PII is never returned.
type ManageGuest struct {
	PublicID         string  `json:"public_id"`
	IsRepresentative bool    `json:"is_representative"`
	FullNameMasked   string  `json:"full_name_masked" doc:"e.g. T*** G***"`
	NameKanaMasked   *string `json:"name_kana_masked"`
	PhoneMasked      *string `json:"phone_masked"`
	EmailMasked      *string `json:"email_masked"`
	HasJapanAddress  bool    `json:"has_japan_address"`
	PassportLast4    *string `json:"passport_last4" doc:"Last 4 characters only"`
	Editable         bool    `json:"editable" doc:"Whether the edit token may correct this guest (companions only)"`
}

type ManageSummary struct {
	PublicID             string         `json:"public_id"`
	Facility             ManageFacility `json:"facility"`
	CheckInDate          string         `json:"check_in_date" format:"date"`
	CheckOutDate         string         `json:"check_out_date" format:"date"`
	EstimatedArrivalTime *string        `json:"estimated_arrival_time" doc:"HH:MM"`
	PlannedGuestCount    int            `json:"planned_guest_count"`
	RegisteredCount      int            `json:"registered_count"`
	RemainingCount       int            `json:"remaining_count"`
	IsComplete           bool           `json:"is_complete"`
	BookingSource        *string        `json:"booking_source"`
	BookingReference     *string        `json:"booking_reference"`
	EditExpiresAt        time.Time      `json:"edit_expires_at" doc:"UTC instant after which the edit token returns 410"`
	Version              string         `json:"version" doc:"Same value as the ETag header; send it as If-Match on PATCH"`
	Guests               []ManageGuest  `json:"guests" nullable:"false"`
}

type ManageOutput struct {
	ETag         string `header:"ETag"`
	CacheControl string `header:"Cache-Control"`
	Body         ManageSummary
}

type GetManageInput struct{}

// PatchManageRegistrationBody: only the allowlisted low-risk fields are
// editable. The remaining declared fields exist so that sending them is
// answered with 403 field_not_editable instead of a generic 422.
type PatchManageRegistrationBody struct {
	EstimatedArrivalTime *string `json:"estimated_arrival_time,omitempty" pattern:"^(([01][0-9]|2[0-3]):[0-5][0-9])?$" doc:"HH:MM; empty string clears it"`
	BookingSource        *string `json:"booking_source,omitempty" maxLength:"50" doc:"Empty string clears it"`
	BookingReference     *string `json:"booking_reference,omitempty" maxLength:"255" doc:"Empty string clears it"`
	PlannedGuestCount    *int    `json:"planned_guest_count,omitempty" minimum:"1" maximum:"50" doc:"At least the registered count and at most the current facility max_guests"`

	CheckInDate  *string `json:"check_in_date,omitempty" doc:"Not editable with an edit token (403 field_not_editable); ask the host"`
	CheckOutDate *string `json:"check_out_date,omitempty" doc:"Not editable with an edit token (403 field_not_editable); ask the host"`
	FacilityCode *string `json:"facility_code,omitempty" doc:"Not editable with an edit token (403 field_not_editable)"`
}

type PatchManageRegistrationInput struct {
	IfMatch string `header:"If-Match" required:"true" doc:"ETag from the last GET (required; 412 when stale)"`
	Body    PatchManageRegistrationBody
}

// CompanionInput is a companion added through the edit token.
type CompanionInput struct {
	FullName        string  `json:"full_name" minLength:"1" maxLength:"255"`
	NameKana        *string `json:"name_kana,omitempty" maxLength:"255"`
	Address         string  `json:"address" minLength:"1" maxLength:"2000"`
	Phone           *string `json:"phone,omitempty" maxLength:"50"`
	Email           *string `json:"email,omitempty" maxLength:"255"`
	HasJapanAddress bool    `json:"has_japan_address"`
	Nationality     *string `json:"nationality,omitempty" maxLength:"100" doc:"Required when has_japan_address is false"`
	PassportNumber  *string `json:"passport_number,omitempty" pattern:"^[A-Za-z0-9 ]{6,20}$" doc:"Required when has_japan_address is false; write-only"`
}

type AddCompanionsInput struct {
	Body struct {
		Guests []CompanionInput `json:"guests" minItems:"1" maxItems:"20" nullable:"false" doc:"Companions to append; registered + new must not exceed planned_guest_count"`
	}
}

// PatchManageGuestBody: companions only. Representative edits and identity
// fields are host-only (403 field_not_editable).
type PatchManageGuestBody struct {
	FullName *string `json:"full_name,omitempty" minLength:"1" maxLength:"255"`
	NameKana *string `json:"name_kana,omitempty" maxLength:"255" doc:"Empty string clears it"`
	Address  *string `json:"address,omitempty" minLength:"1" maxLength:"2000"`
	Phone    *string `json:"phone,omitempty" maxLength:"50" doc:"Empty string clears it"`
	Email    *string `json:"email,omitempty" maxLength:"255" doc:"Empty string clears it"`

	IsRepresentative *bool   `json:"is_representative,omitempty" doc:"Not editable with an edit token (403 field_not_editable)"`
	HasJapanAddress  *bool   `json:"has_japan_address,omitempty" doc:"Not editable with an edit token (403 field_not_editable)"`
	Nationality      *string `json:"nationality,omitempty" doc:"Not editable with an edit token (403 field_not_editable)"`
	PassportNumber   *string `json:"passport_number,omitempty" doc:"Not editable with an edit token (403 field_not_editable)"`
}

type PatchManageGuestInput struct {
	GuestPublicID string `path:"guest_public_id" format:"uuid" doc:"Guest public ID from the manage summary"`
	IfMatch       string `header:"If-Match" required:"true" doc:"ETag from the last GET (required; 412 when stale)"`
	Body          PatchManageGuestBody
}
