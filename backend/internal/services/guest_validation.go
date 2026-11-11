package services

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"github.com/google/uuid"
)

var passportRe = regexp.MustCompile(`^[A-Z0-9]{6,20}$`)

// guestFields is the common shape of a guest in create and add-companion
// requests.
type guestFields struct {
	IsRepresentative bool
	FullName         string
	NameKana         *string
	Address          string
	Phone            *string
	Email            *string
	HasJapanAddress  bool
	Nationality      *string
	PassportNumber   *string
}

// trimmedOrNil returns nil for nil or blank strings.
func trimmedOrNil(s *string) *string {
	if s == nil {
		return nil
	}
	t := strings.TrimSpace(*s)
	if t == "" {
		return nil
	}
	return &t
}

// validateGuest checks the conditional legal fields. has_japan_address=false
// requires nationality and a passport number (contract §3.3, VERIFY with the
// facility license). A passport number sent for a guest with a Japanese
// address is not stored (data minimization).
func validateGuest(loc string, g *guestFields) []FieldError {
	var errs []FieldError
	g.FullName = strings.TrimSpace(g.FullName)
	g.Address = strings.TrimSpace(g.Address)
	g.NameKana = trimmedOrNil(g.NameKana)
	g.Phone = trimmedOrNil(g.Phone)
	g.Email = trimmedOrNil(g.Email)
	g.Nationality = trimmedOrNil(g.Nationality)
	if g.FullName == "" {
		errs = append(errs, FieldError{loc + ".full_name", "is required"})
	}
	if g.Address == "" {
		errs = append(errs, FieldError{loc + ".address", "is required"})
	}
	if g.HasJapanAddress {
		g.PassportNumber = nil
		return errs
	}
	if g.Nationality == nil {
		errs = append(errs, FieldError{loc + ".nationality", "is required when has_japan_address is false"})
	}
	if g.PassportNumber == nil || strings.TrimSpace(*g.PassportNumber) == "" {
		errs = append(errs, FieldError{loc + ".passport_number", "is required when has_japan_address is false"})
		return errs
	}
	n := utils.NormalizePassport(*g.PassportNumber)
	if !passportRe.MatchString(n) {
		errs = append(errs, FieldError{loc + ".passport_number", "must be 6-20 letters or digits"})
		return errs
	}
	g.PassportNumber = &n
	return errs
}

// buildGuest turns validated fields into a row, encrypting the passport
// number with the guest public_id as AAD.
func buildGuest(cipher *utils.PassportCipher, regID uint64, g *guestFields, clock Clock) (*entities.Guest, error) {
	row := &entities.Guest{
		PublicID:         uuid.NewString(),
		RegistrationID:   regID,
		IsRepresentative: g.IsRepresentative,
		FullName:         g.FullName,
		NameKana:         g.NameKana,
		Address:          g.Address,
		Phone:            g.Phone,
		Email:            g.Email,
		HasJapanAddress:  g.HasJapanAddress,
		Nationality:      g.Nationality,
		CreatedAt:        clock.Now(),
		UpdatedAt:        clock.Now(),
	}
	if g.PassportNumber != nil {
		ct, err := cipher.Encrypt(*g.PassportNumber, []byte(row.PublicID))
		if err != nil {
			return nil, fmt.Errorf("encrypt passport: %w", err)
		}
		last4 := utils.PassportLast4(*g.PassportNumber)
		row.PassportNumberCiphertext = ct
		row.PassportNumberLast4 = &last4
	}
	return row, nil
}
