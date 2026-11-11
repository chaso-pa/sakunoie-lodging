package models

import "time"

// GuestInput is one guest in a create request.
type GuestInput struct {
	IsRepresentative bool    `json:"is_representative" doc:"Exactly one guest must be the representative"`
	FullName         string  `json:"full_name" minLength:"1" maxLength:"255" doc:"Full name"`
	NameKana         *string `json:"name_kana,omitempty" maxLength:"255" doc:"Name in katakana (optional)"`
	Address          string  `json:"address" minLength:"1" maxLength:"2000" doc:"Address"`
	Phone            *string `json:"phone,omitempty" maxLength:"50" doc:"Phone (optional)"`
	Email            *string `json:"email,omitempty" maxLength:"255" doc:"Email (optional)"`
	HasJapanAddress  bool    `json:"has_japan_address" doc:"Whether the guest has an address in Japan"`
	Nationality      *string `json:"nationality,omitempty" maxLength:"100" doc:"Required when has_japan_address is false"`
	PassportNumber   *string `json:"passport_number,omitempty" pattern:"^[A-Za-z0-9 ]{6,20}$" doc:"Required when has_japan_address is false. Write-only; stored encrypted, only the last 4 characters are ever returned"`
}

// CreateGuestRegistrationBody is the create DTO. The server decides
// timestamps, capacity snapshot, facility ID and token expiry.
type CreateGuestRegistrationBody struct {
	FacilityCode         string       `json:"facility_code" pattern:"^[a-z0-9-]{1,64}$" doc:"Facility code"`
	CheckInDate          string       `json:"check_in_date" format:"date" doc:"Local check-in date (YYYY-MM-DD)"`
	CheckOutDate         string       `json:"check_out_date" format:"date" doc:"Local check-out date (YYYY-MM-DD), after check-in"`
	EstimatedArrivalTime *string      `json:"estimated_arrival_time,omitempty" pattern:"^([01][0-9]|2[0-3]):[0-5][0-9]$" doc:"Local arrival time HH:MM (optional)"`
	PlannedGuestCount    int          `json:"planned_guest_count" minimum:"1" maximum:"50" doc:"Planned number of guests; at most the facility max_guests"`
	BookingSource        *string      `json:"booking_source,omitempty" maxLength:"50" doc:"e.g. airbnb (optional)"`
	BookingReference     *string      `json:"booking_reference,omitempty" maxLength:"255" doc:"Booking reference (optional)"`
	TermsVersion         string       `json:"terms_version" minLength:"1" maxLength:"64" doc:"Must equal the current terms_version"`
	PrivacyVersion       string       `json:"privacy_version" minLength:"1" maxLength:"64" doc:"Must equal the current privacy_version"`
	TermsAccepted        bool         `json:"terms_accepted" doc:"Must be true"`
	PrivacyAccepted      bool         `json:"privacy_accepted" doc:"Must be true"`
	Guests               []GuestInput `json:"guests" minItems:"1" maxItems:"20" nullable:"false" doc:"1..planned_guest_count guests, exactly one representative"`
}

type CreateGuestRegistrationInput struct {
	IdempotencyKey string `header:"Idempotency-Key" required:"true" minLength:"16" maxLength:"128" pattern:"^[A-Za-z0-9_.:-]+$" doc:"Client-generated key (e.g. a UUID), reused on retries of the same submission"`
	Body           CreateGuestRegistrationBody
}

// EditLinkUnavailable is returned in `code` when a replay cannot mint a token.
const EditLinkUnavailable = "edit_link_unavailable"

type CreateGuestRegistrationResult struct {
	PublicID           string     `json:"public_id" doc:"Opaque registration ID"`
	FacilityCode       string     `json:"facility_code"`
	CheckInDate        string     `json:"check_in_date" format:"date"`
	CheckOutDate       string     `json:"check_out_date" format:"date"`
	PlannedGuestCount  int        `json:"planned_guest_count"`
	RegisteredCount    int        `json:"registered_count" doc:"Guests registered so far"`
	RemainingCount     int        `json:"remaining_count" doc:"planned_guest_count - registered_count"`
	IsComplete         bool       `json:"is_complete" doc:"registered_count == planned_guest_count"`
	EditToken          *string    `json:"edit_token" doc:"One-time secret for the manage URL fragment (#token=...). Null when code is edit_link_unavailable. Never stored by the server"`
	EditTokenExpiresAt *time.Time `json:"edit_token_expires_at" doc:"UTC instant; end of the checkout day in the facility time zone"`
	Code               string     `json:"code,omitempty" enum:"edit_link_unavailable" doc:"Set on an idempotent replay that cannot issue another edit link (3 links already active, or the host reissued the link). The guest should contact the host"`
}

type CreateGuestRegistrationOutput struct {
	Status       int
	CacheControl string `header:"Cache-Control"`
	Body         CreateGuestRegistrationResult
}
