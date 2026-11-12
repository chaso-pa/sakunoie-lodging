package services

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"github.com/go-sql-driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// GuestManageService implements the token-scoped manage endpoints
// (contract §4.4, §4.6, §4.7). TokenService.Resolve locks the registration
// row (then the token); every mutation re-counts guests under that lock and
// bumps updated_at (the ETag).
type GuestManageService struct {
	db     *gorm.DB
	clock  Clock
	tokens *TokenService
	cipher *utils.PassportCipher
}

func NewGuestManageService(db *gorm.DB, clock Clock, tokens *TokenService, cipher *utils.PassportCipher) *GuestManageService {
	return &GuestManageService{db: db, clock: clock, tokens: tokens, cipher: cipher}
}

// ETag is the opaque version of a registration (its updated_at).
func ETag(reg *entities.GuestRegistration) string {
	return "r" + strconv.FormatInt(reg.UpdatedAt.UnixMicro(), 36)
}

// ifMatch reports whether an If-Match header value is exactly etag, either
// as the quoted strong ETag or as the bare `version` from the body. "*",
// weak (W/) tags and lists are rejected: a write must name the version it
// was based on.
func ifMatch(header, etag string) bool {
	v := strings.TrimSpace(header)
	return v == `"`+etag+`"` || v == etag
}

// manageTx runs fn in a READ COMMITTED transaction, so reads after the
// registration lock see every committed row.
func (s *GuestManageService) manageTx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return readCommittedTx(ctx, s.db, fn)
}

// readCommittedTx runs fn in a READ COMMITTED transaction and retries it
// once after a deadlock (MariaDB 1213), which rolls the whole transaction
// back. Lock order is registration row first, then its tokens/guests; the
// retry is a backstop for orderings this does not cover (e.g. gap locks).
// fn must not keep state across attempts.
func readCommittedTx(ctx context.Context, db *gorm.DB, fn func(tx *gorm.DB) error) error {
	opts := &sql.TxOptions{Isolation: sql.LevelReadCommitted}
	err := db.WithContext(ctx).Transaction(fn, opts)
	if isDeadlock(err) && ctx.Err() == nil {
		err = db.WithContext(ctx).Transaction(fn, opts)
	}
	return err
}

func isDeadlock(err error) bool {
	var me *mysql.MySQLError
	return errors.As(err, &me) && me.Number == 1213
}

// lockRegistration re-reads the registration FOR UPDATE.
func lockRegistration(tx *gorm.DB, id uint64) (*entities.GuestRegistration, error) {
	var reg entities.GuestRegistration
	err := tx.Scopes(notDeleted).Clauses(clause.Locking{Strength: "UPDATE"}).Where("id = ?", id).Take(&reg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrTokenInvalid
	}
	return &reg, err
}

// bump returns a new updated_at strictly after the previous one.
func (s *GuestManageService) bump(reg *entities.GuestRegistration) time.Time {
	now := s.clock.Now()
	if !now.After(reg.UpdatedAt) {
		now = reg.UpdatedAt.Add(time.Microsecond)
	}
	return now
}

// Summary returns the minimal, masked view.
func (s *GuestManageService) Summary(ctx context.Context, token string) (*models.ManageSummary, error) {
	var out *models.ManageSummary
	err := s.manageTx(ctx, func(tx *gorm.DB) error {
		tok, reg, err := s.tokens.Resolve(tx, token)
		if err != nil {
			return err
		}
		out, err = s.summary(tx, tok, reg)
		return err
	})
	return out, err
}

func (s *GuestManageService) summary(tx *gorm.DB, tok *entities.RegistrationAccessToken, reg *entities.GuestRegistration) (*models.ManageSummary, error) {
	var f entities.Facility
	if err := tx.Where("id = ?", reg.FacilityID).Take(&f).Error; err != nil {
		return nil, err
	}
	var guests []entities.Guest
	if err := tx.Where("registration_id = ?", reg.ID).Order("is_representative DESC, id ASC").Find(&guests).Error; err != nil {
		return nil, err
	}
	planned := int(reg.PlannedGuestCount)
	out := &models.ManageSummary{
		PublicID:          reg.PublicID,
		Facility:          models.ManageFacility{Code: f.Code, Name: f.Name, Timezone: f.Timezone, MaxGuests: int(f.MaxGuests)},
		CheckInDate:       utils.CivilDateFromDB(reg.CheckInDate).String(),
		CheckOutDate:      utils.CivilDateFromDB(reg.CheckOutDate).String(),
		PlannedGuestCount: planned,
		RegisteredCount:   len(guests),
		RemainingCount:    max(planned-len(guests), 0),
		IsComplete:        len(guests) >= planned,
		BookingSource:     reg.BookingSource,
		BookingReference:  reg.BookingReference,
		EditExpiresAt:     tok.ExpiresAt,
		Version:           ETag(reg),
		Guests:            make([]models.ManageGuest, 0, len(guests)),
	}
	if reg.EstimatedArrivalTime != nil && len(*reg.EstimatedArrivalTime) >= 5 {
		hhmm := (*reg.EstimatedArrivalTime)[:5]
		out.EstimatedArrivalTime = &hhmm
	}
	for _, g := range guests {
		mg := models.ManageGuest{
			PublicID:         g.PublicID,
			IsRepresentative: g.IsRepresentative,
			FullNameMasked:   utils.MaskName(g.FullName),
			HasJapanAddress:  g.HasJapanAddress,
			PassportLast4:    g.PassportNumberLast4,
			Editable:         !g.IsRepresentative,
		}
		if g.NameKana != nil {
			m := utils.MaskName(*g.NameKana)
			mg.NameKanaMasked = &m
		}
		if g.Phone != nil {
			m := utils.MaskPhone(*g.Phone)
			mg.PhoneMasked = &m
		}
		if g.Email != nil {
			m := utils.MaskEmail(*g.Email)
			mg.EmailMasked = &m
		}
		out.Guests = append(out.Guests, mg)
	}
	return out, nil
}

// AddCompanions appends companions under the registration lock:
// registered + new must not exceed planned_guest_count (409).
func (s *GuestManageService) AddCompanions(ctx context.Context, token string, in []models.CompanionInput) (*models.ManageSummary, error) {
	var out *models.ManageSummary
	err := s.manageTx(ctx, func(tx *gorm.DB) error {
		tok, reg, err := s.tokens.Resolve(tx, token)
		if err != nil {
			return err
		}
		var errs []FieldError
		guests := make([]guestFields, len(in))
		for i, c := range in {
			guests[i] = guestFields{
				FullName: c.FullName, NameKana: c.NameKana, Address: c.Address, Phone: c.Phone,
				Email: c.Email, HasJapanAddress: c.HasJapanAddress, Nationality: c.Nationality,
				PassportNumber: c.PassportNumber,
			}
			errs = append(errs, validateGuest(guestLoc(i), &guests[i])...)
		}
		if len(in) == 0 {
			errs = append(errs, FieldError{"body.guests", "at least one guest is required"})
		}
		if len(errs) > 0 {
			return newDomainError(ErrValidation, errs...)
		}
		count, err := countGuests(tx, reg.ID)
		if err != nil {
			return err
		}
		if count+len(guests) > int(reg.PlannedGuestCount) {
			return newDomainError(ErrCapacityExceeded, FieldError{"body.guests", "would exceed planned_guest_count"})
		}
		for i := range guests {
			row, err := buildGuest(s.cipher, reg.ID, &guests[i], s.clock)
			if err != nil {
				return err
			}
			if err := tx.Create(row).Error; err != nil {
				return err
			}
		}
		reg.UpdatedAt = s.bump(reg)
		if err := tx.Model(reg).Update("updated_at", reg.UpdatedAt).Error; err != nil {
			return err
		}
		if err := writeAudit(tx, s.clock, reg.FacilityID, &reg.ID, entities.ActorGuestToken, nil, "registration.guests_added", []string{"guests"}); err != nil {
			return err
		}
		out, err = s.summary(tx, tok, reg)
		return err
	})
	return out, err
}

func notEditable(fields ...string) error {
	errs := make([]FieldError, len(fields))
	for i, f := range fields {
		errs[i] = FieldError{"body." + f, "is not editable with an edit link; ask the host"}
	}
	return newDomainError(ErrFieldNotEditable, errs...)
}

// PatchRegistration edits the allowlisted low-risk fields (open question Q6).
func (s *GuestManageService) PatchRegistration(ctx context.Context, token, ifMatchHeader string, in *models.PatchManageRegistrationBody) (*models.ManageSummary, error) {
	var out *models.ManageSummary
	err := s.manageTx(ctx, func(tx *gorm.DB) error {
		tok, reg, err := s.tokens.Resolve(tx, token)
		if err != nil {
			return err
		}
		var forbidden []string
		if in.CheckInDate != nil {
			forbidden = append(forbidden, "check_in_date")
		}
		if in.CheckOutDate != nil {
			forbidden = append(forbidden, "check_out_date")
		}
		if in.FacilityCode != nil {
			forbidden = append(forbidden, "facility_code")
		}
		if len(forbidden) > 0 {
			return notEditable(forbidden...)
		}
		if !ifMatch(ifMatchHeader, ETag(reg)) {
			return ErrPreconditionFailed
		}
		updates := map[string]any{}
		var changed []string
		if in.EstimatedArrivalTime != nil {
			t := strings.TrimSpace(*in.EstimatedArrivalTime)
			if t == "" {
				updates["estimated_arrival_time"] = nil
				reg.EstimatedArrivalTime = nil
			} else {
				if _, err := time.Parse("15:04", t); err != nil {
					return newDomainError(ErrValidation, FieldError{"body.estimated_arrival_time", "must be HH:MM"})
				}
				t += ":00"
				updates["estimated_arrival_time"] = t
				reg.EstimatedArrivalTime = &t
			}
			changed = append(changed, "estimated_arrival_time")
		}
		if in.BookingSource != nil {
			v := trimmedOrNil(in.BookingSource)
			updates["booking_source"] = v
			reg.BookingSource = v
			changed = append(changed, "booking_source")
		}
		if in.BookingReference != nil {
			v := trimmedOrNil(in.BookingReference)
			updates["booking_reference"] = v
			reg.BookingReference = v
			changed = append(changed, "booking_reference")
		}
		if in.PlannedGuestCount != nil {
			n := *in.PlannedGuestCount
			count, err := countGuests(tx, reg.ID)
			if err != nil {
				return err
			}
			if n < count {
				return newDomainError(ErrValidation, FieldError{"body.planned_guest_count", "must not be less than the registered count"})
			}
			var f entities.Facility
			if err := tx.Where("id = ?", reg.FacilityID).Take(&f).Error; err != nil {
				return err
			}
			if n > int(f.MaxGuests) {
				return newDomainError(ErrCapacityExceeded, FieldError{"body.planned_guest_count", "exceeds the facility capacity"})
			}
			updates["planned_guest_count"] = n
			reg.PlannedGuestCount = uint32(n)
			changed = append(changed, "planned_guest_count")
		}
		if len(changed) > 0 {
			reg.UpdatedAt = s.bump(reg)
			updates["updated_at"] = reg.UpdatedAt
			if err := tx.Model(&entities.GuestRegistration{}).Where("id = ?", reg.ID).Updates(updates).Error; err != nil {
				return err
			}
			if err := writeAudit(tx, s.clock, reg.FacilityID, &reg.ID, entities.ActorGuestToken, nil, "registration.updated", changed); err != nil {
				return err
			}
		}
		out, err = s.summary(tx, tok, reg)
		return err
	})
	return out, err
}

// PatchGuest corrects a companion's allowlisted fields. Representatives and
// identity fields are host-only (403 field_not_editable).
func (s *GuestManageService) PatchGuest(ctx context.Context, token, ifMatchHeader, guestPublicID string, in *models.PatchManageGuestBody) (*models.ManageSummary, error) {
	var out *models.ManageSummary
	err := s.manageTx(ctx, func(tx *gorm.DB) error {
		tok, reg, err := s.tokens.Resolve(tx, token)
		if err != nil {
			return err
		}
		var forbidden []string
		if in.IsRepresentative != nil {
			forbidden = append(forbidden, "is_representative")
		}
		if in.HasJapanAddress != nil {
			forbidden = append(forbidden, "has_japan_address")
		}
		if in.Nationality != nil {
			forbidden = append(forbidden, "nationality")
		}
		if in.PassportNumber != nil {
			forbidden = append(forbidden, "passport_number")
		}
		if len(forbidden) > 0 {
			return notEditable(forbidden...)
		}
		var g entities.Guest
		err = tx.Where("public_id = ? AND registration_id = ?", guestPublicID, reg.ID).Take(&g).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if g.IsRepresentative {
			return newDomainError(ErrFieldNotEditable, FieldError{"path.guest_public_id", "the representative can only be corrected by the host"})
		}
		if !ifMatch(ifMatchHeader, ETag(reg)) {
			return ErrPreconditionFailed
		}
		updates := map[string]any{}
		changed, errs := guestEdits("body", in.FullName, in.Address, in.NameKana, in.Phone, in.Email, updates)
		if len(errs) > 0 {
			return newDomainError(ErrValidation, errs...)
		}
		if len(changed) > 0 {
			now := s.clock.Now()
			updates["updated_at"] = now
			if err := tx.Model(&entities.Guest{}).Where("id = ?", g.ID).Updates(updates).Error; err != nil {
				return err
			}
			reg.UpdatedAt = s.bump(reg)
			if err := tx.Model(&entities.GuestRegistration{}).Where("id = ?", reg.ID).Update("updated_at", reg.UpdatedAt).Error; err != nil {
				return err
			}
			slices.Sort(changed)
			fields := make([]string, len(changed))
			for i, c := range changed {
				fields[i] = "guest." + c
			}
			if err := writeAudit(tx, s.clock, reg.FacilityID, &reg.ID, entities.ActorGuestToken, nil, "registration.updated", fields); err != nil {
				return err
			}
		}
		out, err = s.summary(tx, tok, reg)
		return err
	})
	return out, err
}
