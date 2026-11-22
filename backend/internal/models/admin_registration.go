package models

import "time"

// Admin endpoints authenticate with a Firebase ID token as
// `Authorization: Bearer <token>` (security scheme adminBearer). Every
// response carries Cache-Control: no-store.

type AdminFacility struct {
	ID        uint64 `json:"id" doc:"Facility ID; use it as facility_id in list and export filters. Not an authorization credential"`
	Code      string `json:"code"`
	Name      string `json:"name"`
	Timezone  string `json:"timezone" doc:"IANA time zone"`
	MaxGuests int    `json:"max_guests"`
	IsActive  bool   `json:"is_active"`
}

type ListAdminFacilitiesInput struct{}

type ListAdminFacilitiesOutput struct {
	Body struct {
		Facilities []AdminFacility `json:"facilities" nullable:"false" doc:"Facilities the admin is authorized for (owner role)"`
	}
}

// AdminFacilityRef is the facility of a ledger row.
type AdminFacilityRef struct {
	ID   uint64 `json:"id"`
	Code string `json:"code"`
	Name string `json:"name"`
}

type ListAdminRegistrationsInput struct {
	FacilityID uint64 `query:"facility_id" doc:"Restrict to one authorized facility (403 if not authorized). Omitted: all authorized facilities"`
	From       string `query:"from" format:"date" doc:"Earliest check-in date (inclusive, facility-local YYYY-MM-DD)"`
	To         string `query:"to" format:"date" doc:"Latest check-in date (inclusive, facility-local YYYY-MM-DD)"`
	Q          string `query:"q" maxLength:"100" doc:"Search guest names, kana and booking reference (substring)"`
	Page       int    `query:"page" default:"1" minimum:"1" maximum:"100000" doc:"1-based page number"`
}

type AdminRegistrationRow struct {
	PublicID           string           `json:"public_id"`
	Facility           AdminFacilityRef `json:"facility"`
	CheckInDate        string           `json:"check_in_date" format:"date"`
	CheckOutDate       string           `json:"check_out_date" format:"date"`
	BookingSource      *string          `json:"booking_source"`
	BookingReference   *string          `json:"booking_reference"`
	RepresentativeName string           `json:"representative_name"`
	PlannedGuestCount  int              `json:"planned_guest_count"`
	RegisteredCount    int              `json:"registered_count"`
	RemainingCount     int              `json:"remaining_count"`
	IsComplete         bool             `json:"is_complete"`
	IdentityVerified   bool             `json:"identity_verified"`
	IdentityVerifiedAt *time.Time       `json:"identity_verified_at"`
	SubmittedAt        time.Time        `json:"submitted_at"`
}

// AdminLedgerMetrics summarize every registration matching the filter, not
// only the current page.
type AdminLedgerMetrics struct {
	Registrations int `json:"registrations"`
	Guests        int `json:"guests" doc:"Registered guests"`
	Incomplete    int `json:"incomplete" doc:"Registrations with fewer guests than planned"`
	Verified      int `json:"verified" doc:"Registrations with identity verified by the host"`
}

type AdminRegistrationList struct {
	Items    []AdminRegistrationRow `json:"items" nullable:"false"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"page_size"`
	Total    int                    `json:"total"`
	Metrics  AdminLedgerMetrics     `json:"metrics"`
}

type ListAdminRegistrationsOutput struct {
	Body AdminRegistrationList
}

// AdminGuest is a guest in the admin detail. Address, phone and email are
// masked unless the request sets reveal=true (which is audited). Passport
// numbers are never returned (last 4 only).
type AdminGuest struct {
	PublicID         string  `json:"public_id"`
	IsRepresentative bool    `json:"is_representative"`
	FullName         string  `json:"full_name"`
	NameKana         *string `json:"name_kana"`
	Address          string  `json:"address" doc:"Masked unless reveal=true"`
	Phone            *string `json:"phone" doc:"Masked unless reveal=true"`
	Email            *string `json:"email" doc:"Masked unless reveal=true"`
	HasJapanAddress  bool    `json:"has_japan_address"`
	Nationality      *string `json:"nationality"`
	PassportLast4    *string `json:"passport_last4" doc:"Last 4 characters only"`
}

type AdminRegistrationDetail struct {
	PublicID             string        `json:"public_id"`
	Facility             AdminFacility `json:"facility"`
	CheckInDate          string        `json:"check_in_date" format:"date"`
	CheckOutDate         string        `json:"check_out_date" format:"date"`
	EstimatedArrivalTime *string       `json:"estimated_arrival_time" doc:"HH:MM"`
	PlannedGuestCount    int           `json:"planned_guest_count"`
	CapacitySnapshot     int           `json:"capacity_snapshot" doc:"Facility max_guests at submission"`
	RegisteredCount      int           `json:"registered_count"`
	RemainingCount       int           `json:"remaining_count"`
	IsComplete           bool          `json:"is_complete"`
	BookingSource        *string       `json:"booking_source"`
	BookingReference     *string       `json:"booking_reference"`
	TermsVersion         string        `json:"terms_version"`
	PrivacyVersion       string        `json:"privacy_version"`
	SubmittedAt          time.Time     `json:"submitted_at"`
	IdentityVerified     bool          `json:"identity_verified"`
	IdentityVerifiedAt   *time.Time    `json:"identity_verified_at"`
	ActiveEditLinks      int           `json:"active_edit_links" doc:"Unrevoked, unexpired edit links"`
	EditLinkExpiresAt    *time.Time    `json:"edit_link_expires_at" doc:"Latest expiry among active edit links"`
	Revealed             bool          `json:"revealed" doc:"Whether address/phone/email are unmasked"`
	Version              string        `json:"version" doc:"Same value as the ETag header; send it as If-Match on PATCH"`
	Guests               []AdminGuest  `json:"guests" nullable:"false"`
}

type AdminRegistrationOutput struct {
	ETag string `header:"ETag"`
	Body AdminRegistrationDetail
}

type GetAdminRegistrationInput struct {
	ID     string `path:"id" format:"uuid" doc:"Registration public ID"`
	Reveal bool   `query:"reveal" doc:"Return address, phone and email unmasked. Writes an audit row"`
}

// AdminPatchGuest corrects one guest. Omitted fields are unchanged.
type AdminPatchGuest struct {
	PublicID         string  `json:"public_id" format:"uuid" doc:"Guest public ID"`
	IsRepresentative *bool   `json:"is_representative,omitempty" doc:"Only true is accepted: makes this guest the representative and unsets the previous one"`
	FullName         *string `json:"full_name,omitempty" minLength:"1" maxLength:"255"`
	NameKana         *string `json:"name_kana,omitempty" maxLength:"255" doc:"Empty string clears it"`
	Address          *string `json:"address,omitempty" minLength:"1" maxLength:"2000"`
	Phone            *string `json:"phone,omitempty" maxLength:"50" doc:"Empty string clears it"`
	Email            *string `json:"email,omitempty" maxLength:"255" doc:"Empty string clears it"`
	HasJapanAddress  *bool   `json:"has_japan_address,omitempty" doc:"true discards any stored passport number"`
	Nationality      *string `json:"nationality,omitempty" maxLength:"100" doc:"Required when has_japan_address is false"`
	PassportNumber   *string `json:"passport_number,omitempty" pattern:"^[A-Za-z0-9 ]{6,20}$" doc:"Write-only; re-encrypted. Required when has_japan_address is false and none is stored"`
}

// AdminPatchRegistrationBody is a host correction. Omitted fields are
// unchanged. Changed field names (never values) are audited.
type AdminPatchRegistrationBody struct {
	CheckInDate          *string           `json:"check_in_date,omitempty" format:"date"`
	CheckOutDate         *string           `json:"check_out_date,omitempty" format:"date"`
	EstimatedArrivalTime *string           `json:"estimated_arrival_time,omitempty" pattern:"^(([01][0-9]|2[0-3]):[0-5][0-9])?$" doc:"HH:MM; empty string clears it"`
	PlannedGuestCount    *int              `json:"planned_guest_count,omitempty" minimum:"1" maximum:"50" doc:"At least the registered count; above the current facility max_guests only with capacity_override"`
	CapacityOverride     bool              `json:"capacity_override,omitempty" doc:"Allow planned_guest_count above the current facility max_guests"`
	BookingSource        *string           `json:"booking_source,omitempty" maxLength:"50" doc:"Empty string clears it"`
	BookingReference     *string           `json:"booking_reference,omitempty" maxLength:"255" doc:"Empty string clears it"`
	Guests               []AdminPatchGuest `json:"guests,omitempty" maxItems:"20" doc:"Guest corrections"`
}

type PatchAdminRegistrationInput struct {
	ID      string `path:"id" format:"uuid" doc:"Registration public ID"`
	IfMatch string `header:"If-Match" required:"true" doc:"ETag from the last GET (required; 412 when stale)"`
	Body    AdminPatchRegistrationBody
}

type AdminRegistrationActionInput struct {
	ID string `path:"id" format:"uuid" doc:"Registration public ID"`
}

type ReissueEditLinkResult struct {
	EditToken          string    `json:"edit_token" doc:"New one-time secret for the manage URL fragment (#token=...). Shown once; never stored by the server. All earlier links are revoked"`
	EditTokenExpiresAt time.Time `json:"edit_token_expires_at" doc:"UTC instant; end of the current checkout day in the facility time zone"`
	Expired            bool      `json:"expired" doc:"True when the checkout day has already ended, so the new link is not usable"`
}

type ReissueEditLinkOutput struct {
	Body ReissueEditLinkResult
}

type ExportAdminRegistrationsInput struct {
	FacilityID uint64 `query:"facility_id" required:"true" minimum:"1" doc:"Authorized facility (403 otherwise)"`
	From       string `query:"from" format:"date" doc:"Earliest check-in date (inclusive)"`
	To         string `query:"to" format:"date" doc:"Latest check-in date (inclusive)"`
}

type ExportAdminRegistrationsOutput struct {
	ContentType        string `header:"Content-Type"`
	ContentDisposition string `header:"Content-Disposition"`
	Body               []byte
}
