package testutil

import (
	"bytes"
	"sync"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
)

// Clock is a settable services.Clock for tests.
type Clock struct {
	mu sync.Mutex
	t  time.Time
}

func NewClock(t time.Time) *Clock { return &Clock{t: t.UTC().Truncate(time.Microsecond)} }

func (c *Clock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *Clock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = t.UTC()
}

func (c *Clock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.t = c.t.Add(d)
}

// Test-only keys (fixed synthetic bytes, never used outside tests).
var (
	TestPassportKey = bytes.Repeat([]byte{0x5a}, 32)
	TestHMACKey     = bytes.Repeat([]byte{0x3c}, 32)
)

func PassportCipher() *utils.PassportCipher {
	c, err := utils.NewPassportCipher(map[byte][]byte{1: TestPassportKey}, 1)
	if err != nil {
		panic(err)
	}
	return c
}

func strp(s string) *string { return &s }

// Representative returns a synthetic Japan-address representative.
func Representative() models.GuestInput {
	return models.GuestInput{
		IsRepresentative: true,
		FullName:         "Test Guest A",
		NameKana:         strp("テスト ゲスト"),
		Address:          "1-2-3 Test Town, Saku, Nagano",
		Phone:            strp("000-0000-0001"),
		Email:            strp("test-guest-a@example.test"),
		HasJapanAddress:  true,
	}
}

// ForeignGuest returns a synthetic non-Japan-address guest with a passport.
func ForeignGuest(name, passport string) models.GuestInput {
	return models.GuestInput{
		FullName:        name,
		Address:         "1 Example Street, Testville",
		HasJapanAddress: false,
		Nationality:     strp("Testland"),
		PassportNumber:  strp(passport),
	}
}

// Companion returns a synthetic Japan-address companion.
func Companion(name string) models.CompanionInput {
	return models.CompanionInput{FullName: name, Address: "1-2-3 Test Town, Saku, Nagano", HasJapanAddress: true}
}

// CreateBody returns a valid create request for facility code with the
// given planned count and guests (defaults: representative only).
func CreateBody(facility string, checkIn, checkOut string, planned int, guests ...models.GuestInput) *models.CreateGuestRegistrationBody {
	if len(guests) == 0 {
		guests = []models.GuestInput{Representative()}
	}
	return &models.CreateGuestRegistrationBody{
		FacilityCode:      facility,
		CheckInDate:       checkIn,
		CheckOutDate:      checkOut,
		PlannedGuestCount: planned,
		BookingSource:     strp("airbnb"),
		BookingReference:  strp("TEST-REF-0001"),
		TermsVersion:      "2026-11-01",
		PrivacyVersion:    "2026-11-01",
		TermsAccepted:     true,
		PrivacyAccepted:   true,
		Guests:            guests,
	}
}
