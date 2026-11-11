package models

type GetFacilityPublicInput struct {
	Code string `path:"code" pattern:"^[a-z0-9-]{1,64}$" doc:"Facility code, e.g. saku-no-ie"`
}

type FacilityPublic struct {
	Code           string `json:"code" doc:"Stable facility code"`
	Name           string `json:"name" doc:"Facility name (localized labels live in frontend content)"`
	MaxGuests      int    `json:"max_guests" doc:"Current capacity policy (maximum planned guests)"`
	Timezone       string `json:"timezone" doc:"IANA time zone of the facility"`
	TermsVersion   string `json:"terms_version" doc:"Current terms version the guest must accept"`
	PrivacyVersion string `json:"privacy_version" doc:"Current privacy policy version the guest must accept"`
}

type GetFacilityPublicOutput struct {
	Body FacilityPublic
}
