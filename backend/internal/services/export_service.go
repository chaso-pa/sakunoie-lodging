package services

import (
	"bytes"
	"context"
	"encoding/csv"
	"strconv"
	"strings"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"gorm.io/gorm"
)

// utf8BOM lets Excel (Japanese operations) open the file as UTF-8 (Q7).
const utf8BOM = "\xEF\xBB\xBF"

// ExportColumns is the CSV header: one row per guest. Passport numbers are
// exported as the last 4 characters only.
var ExportColumns = []string{
	"facility_code", "check_in_date", "check_out_date", "registration_public_id",
	"booking_source", "booking_reference", "planned_guest_count", "registered_count",
	"identity_verified_at", "guest_index", "is_representative", "full_name", "name_kana",
	"address", "phone", "email", "has_japan_address", "nationality", "passport_last4",
}

// ExportFilter selects one facility and an optional inclusive check-in range.
type ExportFilter struct {
	FacilityID uint64
	From, To   string
}

// ExportResult is the CSV file and its suggested name.
type ExportResult struct {
	Filename string
	CSV      []byte
}

// Export builds the CSV for an authorized facility (403 otherwise) and
// writes the audit row `registrations.exported` with the filters and counts,
// never the content. The file is built in memory, so the audit row is
// committed before any byte is sent; a failed audit write means no file.
func (s *AdminRegistrationService) Export(ctx context.Context, scope AdminScope, f ExportFilter) (*ExportResult, error) {
	if err := checkScope(scope); err != nil {
		return nil, err
	}
	if !scope.Has(f.FacilityID) {
		return nil, ErrForbidden
	}
	from, to, err := dateRange(f.From, f.To)
	if err != nil {
		return nil, err
	}
	var out *ExportResult
	err = s.tx(ctx, func(tx *gorm.DB) error {
		var fac entities.Facility
		if err := tx.Where("id = ?", f.FacilityID).Take(&fac).Error; err != nil {
			return err
		}
		var regs []entities.GuestRegistration
		if err := scopedRegistrations(tx, []uint64{fac.ID}, from, to, "").
			Order("guest_registrations.check_in_date ASC, guest_registrations.id ASC").
			Find(&regs).Error; err != nil {
			return err
		}
		byReg := map[uint64][]entities.Guest{}
		if len(regs) > 0 {
			ids := make([]uint64, len(regs))
			for i := range regs {
				ids[i] = regs[i].ID
			}
			var guests []entities.Guest
			if err := tx.Where("registration_id IN ?", ids).
				Order("registration_id, is_representative DESC, id ASC").Find(&guests).Error; err != nil {
				return err
			}
			for _, g := range guests {
				byReg[g.RegistrationID] = append(byReg[g.RegistrationID], g)
			}
		}

		var buf bytes.Buffer
		buf.WriteString(utf8BOM)
		w := csv.NewWriter(&buf)
		w.UseCRLF = true
		if err := w.Write(ExportColumns); err != nil {
			return err
		}
		rows := 0
		for i := range regs {
			r := &regs[i]
			guests := byReg[r.ID]
			verified := ""
			if r.IdentityVerifiedAt != nil {
				verified = r.IdentityVerifiedAt.UTC().Format(time.RFC3339)
			}
			for gi, g := range guests {
				rec := []string{
					fac.Code,
					utils.CivilDateFromDB(r.CheckInDate).String(),
					utils.CivilDateFromDB(r.CheckOutDate).String(),
					r.PublicID,
					deref(r.BookingSource),
					deref(r.BookingReference),
					strconv.Itoa(int(r.PlannedGuestCount)),
					strconv.Itoa(len(guests)),
					verified,
					strconv.Itoa(gi + 1),
					strconv.FormatBool(g.IsRepresentative),
					g.FullName,
					deref(g.NameKana),
					g.Address,
					deref(g.Phone),
					deref(g.Email),
					strconv.FormatBool(g.HasJapanAddress),
					deref(g.Nationality),
					deref(g.PassportNumberLast4),
				}
				for j := range rec {
					rec[j] = utils.CSVSafe(rec[j])
				}
				if err := w.Write(rec); err != nil {
					return err
				}
				rows++
			}
		}
		w.Flush()
		if err := w.Error(); err != nil {
			return err
		}

		meta := map[string]any{
			"facility_id":        fac.ID,
			"from":               f.From,
			"to":                 f.To,
			"registration_count": len(regs),
			"row_count":          rows,
			"format":             "csv_utf8_bom",
		}
		if err := writeAuditMeta(tx, s.clock, fac.ID, nil, entities.ActorAdmin, &scope.AdminUserID, AuditRegistrationsExpo, meta); err != nil {
			return err
		}
		out = &ExportResult{Filename: exportFilename(fac.Code, f.From, f.To), CSV: buf.Bytes()}
		return nil
	})
	return out, err
}

func exportFilename(code, from, to string) string {
	if from == "" {
		from = "start"
	}
	if to == "" {
		to = "end"
	}
	// Dates are validated; the code is reduced to [a-z0-9-] so the name is
	// always safe inside the quoted Content-Disposition filename.
	safe := strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return '_'
	}, code)
	return "registrations-" + safe + "-" + from + "-" + to + ".csv"
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}
