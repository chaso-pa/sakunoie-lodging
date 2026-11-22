package models

import "time"

// Invitation statuses, derived from used_at, revoked_at and expires_at.
const (
	InvitationActive  = "active"
	InvitationUsed    = "used"
	InvitationExpired = "expired"
	InvitationRevoked = "revoked"
)

// --- Admin -----------------------------------------------------------------

type CreateAdminInvitationBody struct {
	FacilityID       uint64  `json:"facility_id" minimum:"1" doc:"Authorized facility (403 otherwise), from GET /admin/facilities"`
	CheckInDate      string  `json:"check_in_date" format:"date" doc:"Facility-local check-in date of the booking (YYYY-MM-DD)"`
	CheckOutDate     string  `json:"check_out_date" format:"date" doc:"Facility-local check-out date (YYYY-MM-DD); after check_in_date and not in the past. The invitation expires at the end of this day in the facility time zone"`
	BookingReference *string `json:"booking_reference,omitempty" maxLength:"255" doc:"Booking code for the host (optional)"`
	Note             *string `json:"note,omitempty" maxLength:"255" doc:"Host-only note, e.g. the guest's name (optional). Never shown to the guest"`
}

type CreateAdminInvitationInput struct {
	Body CreateAdminInvitationBody
}

type AdminInvitation struct {
	PublicID             string           `json:"public_id" format:"uuid"`
	Facility             AdminFacilityRef `json:"facility"`
	CheckInDate          string           `json:"check_in_date" format:"date"`
	CheckOutDate         string           `json:"check_out_date" format:"date"`
	BookingReference     *string          `json:"booking_reference"`
	Note                 *string          `json:"note"`
	Status               string           `json:"status" enum:"active,used,expired,revoked" doc:"Derived: used (a registration was created), revoked, expired (end of the check-out day passed) or active"`
	ExpiresAt            time.Time        `json:"expires_at" doc:"UTC instant; end of the check-out day in the facility time zone"`
	UsedAt               *time.Time       `json:"used_at"`
	RevokedAt            *time.Time       `json:"revoked_at"`
	RegistrationPublicID *string          `json:"registration_public_id" doc:"The registration created with this invitation, once used"`
	CreatedAt            time.Time        `json:"created_at"`
}

type CreateAdminInvitationResult struct {
	Invitation        AdminInvitation `json:"invitation"`
	InvitationToken   string          `json:"invitation_token" doc:"One-time secret for the registration URL fragment (#invite=...). Shown once; never stored by the server and cannot be retrieved later"`
	InvitationURLPath string          `json:"invitation_url_path" doc:"Locale-less registration path of the facility. The frontend builds <web origin>/<locale><path>#invite=<invitation_token>; the token never goes into a query string"`
}

type CreateAdminInvitationOutput struct {
	Body CreateAdminInvitationResult
}

type ListAdminInvitationsInput struct {
	FacilityID uint64 `query:"facility_id" doc:"Restrict to one authorized facility (403 if not authorized). Omitted: all authorized facilities"`
	Status     string `query:"status" enum:"active,used,expired,revoked" doc:"Filter by derived status (optional)"`
	Page       int    `query:"page" default:"1" minimum:"1" maximum:"100000" doc:"1-based page number"`
}

type AdminInvitationList struct {
	Items    []AdminInvitation `json:"items" nullable:"false"`
	Page     int               `json:"page"`
	PageSize int               `json:"page_size"`
	Total    int               `json:"total"`
}

type ListAdminInvitationsOutput struct {
	Body AdminInvitationList
}

type RevokeAdminInvitationInput struct {
	ID string `path:"public_id" format:"uuid" doc:"Invitation public ID"`
}

type RevokeAdminInvitationOutput struct {
	Body AdminInvitation
}

// --- Public ----------------------------------------------------------------

type GetCurrentInvitationInput struct{}

// CurrentInvitation is the minimum the wizard needs to prefill the stay. The
// host's note and booking reference are never returned.
type CurrentInvitation struct {
	FacilityCode string    `json:"facility_code"`
	CheckInDate  string    `json:"check_in_date" format:"date" doc:"Set by the host; the registration must use it"`
	CheckOutDate string    `json:"check_out_date" format:"date" doc:"Set by the host; the registration must use it"`
	ExpiresAt    time.Time `json:"expires_at" doc:"UTC instant; end of the check-out day in the facility time zone"`
}

type GetCurrentInvitationOutput struct {
	CacheControl string `header:"Cache-Control"`
	Body         CurrentInvitation
}
