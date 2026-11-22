package services

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Audit actions of the invitation console.
const (
	AuditInvitationCreated = "invitation.created"
	AuditInvitationRevoked = "invitation.revoked"
)

// InvitationPageSize is the admin invitation list page size.
const InvitationPageSize = 50

// InvitationService issues, lists and revokes per-booking invitations
// (admin), and resolves the current one for the wizard (public). The raw
// token is returned once by Issue; only its SHA-256 is stored.
type InvitationService struct {
	db    *gorm.DB
	clock Clock
}

func NewInvitationService(db *gorm.DB, clock Clock) *InvitationService {
	return &InvitationService{db: db, clock: clock}
}

// InvitationURLPath is the locale-less wizard path of a facility (the
// frontend's paths.registration.wizard).
func InvitationURLPath(facilityCode string) string {
	return "/stays/" + facilityCode + "/registration"
}

// invitationStatus derives the status; used wins over revoked (a used
// invitation cannot be revoked) and revoked over expired.
func invitationStatus(inv *entities.RegistrationInvitation, now time.Time) string {
	switch {
	case inv.UsedAt != nil:
		return models.InvitationUsed
	case inv.RevokedAt != nil:
		return models.InvitationRevoked
	case now.After(inv.ExpiresAt):
		return models.InvitationExpired
	}
	return models.InvitationActive
}

// AdminInvitationInput is the validated-on-issue request.
type AdminInvitationInput struct {
	FacilityID       uint64
	CheckInDate      string
	CheckOutDate     string
	BookingReference *string
	Note             *string
}

// Issue creates an invitation for a facility in scope and returns the raw
// token once. The check-out date must be after check-in and not in the past
// in facility time; the expiry is the end of that day (EditTokenExpiry).
func (s *InvitationService) Issue(ctx context.Context, scope AdminScope, in AdminInvitationInput) (*models.CreateAdminInvitationResult, error) {
	if err := checkScope(scope); err != nil {
		return nil, err
	}
	if !scope.Has(in.FacilityID) {
		return nil, ErrForbidden
	}
	var out *models.CreateAdminInvitationResult
	err := readCommittedTx(ctx, s.db, func(tx *gorm.DB) error {
		var f entities.Facility
		err := tx.Where("id = ? AND is_active = ?", in.FacilityID, true).Take(&f).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrFacilityUnavailable
		}
		if err != nil {
			return err
		}
		checkIn, checkOut, err := s.validateDates(&f, in.CheckInDate, in.CheckOutDate)
		if err != nil {
			return err
		}
		expiresAt, err := utils.EditTokenExpiry(checkOut, f.Timezone)
		if err != nil {
			return err
		}
		raw, hash, err := newToken()
		if err != nil {
			return err
		}
		now := s.clock.Now()
		inv := entities.RegistrationInvitation{
			PublicID:         uuid.NewString(),
			FacilityID:       f.ID,
			TokenHash:        hash,
			CheckInDate:      checkIn.ToDB(),
			CheckOutDate:     checkOut.ToDB(),
			BookingReference: trimmedOrNil(in.BookingReference),
			Note:             trimmedOrNil(in.Note),
			ExpiresAt:        expiresAt,
			CreatedBy:        scope.AdminUserID,
			CreatedAt:        now,
			UpdatedAt:        now,
		}
		if err := tx.Create(&inv).Error; err != nil {
			return err
		}
		// Metadata only: never the token, the note or the booking code.
		if err := writeAuditMeta(tx, s.clock, f.ID, nil, entities.ActorAdmin, &scope.AdminUserID,
			AuditInvitationCreated, map[string]string{"invitation_public_id": inv.PublicID}); err != nil {
			return err
		}
		out = &models.CreateAdminInvitationResult{
			Invitation:        adminInvitation(&inv, &f, nil, now),
			InvitationToken:   raw,
			InvitationURLPath: InvitationURLPath(f.Code),
		}
		return nil
	})
	return out, err
}

func (s *InvitationService) validateDates(f *entities.Facility, in, out string) (utils.CivilDate, utils.CivilDate, error) {
	var errs []FieldError
	checkIn, err := utils.ParseCivilDate(in)
	if err != nil {
		errs = append(errs, FieldError{"body.check_in_date", "must be a date (YYYY-MM-DD)"})
	}
	checkOut, err := utils.ParseCivilDate(out)
	if err != nil {
		errs = append(errs, FieldError{"body.check_out_date", "must be a date (YYYY-MM-DD)"})
	}
	if len(errs) == 0 {
		if checkOut.Compare(checkIn) <= 0 {
			errs = append(errs, FieldError{"body.check_out_date", "must be after check_in_date"})
		}
		loc, err := time.LoadLocation(f.Timezone)
		if err != nil {
			return checkIn, checkOut, err
		}
		if checkOut.Compare(utils.CivilDateIn(s.clock.Now(), loc)) < 0 {
			errs = append(errs, FieldError{"body.check_out_date", "must not be in the past"})
		}
	}
	if len(errs) > 0 {
		return checkIn, checkOut, newDomainError(ErrValidation, errs...)
	}
	return checkIn, checkOut, nil
}

func adminInvitation(inv *entities.RegistrationInvitation, f *entities.Facility, regPublicID *string, now time.Time) models.AdminInvitation {
	return models.AdminInvitation{
		PublicID:             inv.PublicID,
		Facility:             models.AdminFacilityRef{ID: f.ID, Code: f.Code, Name: f.Name},
		CheckInDate:          utils.CivilDateFromDB(inv.CheckInDate).String(),
		CheckOutDate:         utils.CivilDateFromDB(inv.CheckOutDate).String(),
		BookingReference:     inv.BookingReference,
		Note:                 inv.Note,
		Status:               invitationStatus(inv, now),
		ExpiresAt:            inv.ExpiresAt,
		UsedAt:               inv.UsedAt,
		RevokedAt:            inv.RevokedAt,
		RegistrationPublicID: regPublicID,
		CreatedAt:            inv.CreatedAt,
	}
}

// InvitationListFilter is the admin list filter. Zero values mean "no filter".
type InvitationListFilter struct {
	FacilityID uint64
	Status     string
	Page       int
}

// List returns one page of invitations in scope, newest first. Tokens and
// hashes are never selected into the response.
func (s *InvitationService) List(ctx context.Context, scope AdminScope, f InvitationListFilter) (*models.AdminInvitationList, error) {
	if err := checkScope(scope); err != nil {
		return nil, err
	}
	ids, err := facilitiesFilter(scope, f.FacilityID)
	if err != nil {
		return nil, err
	}
	now := s.clock.Now()
	q := s.db.WithContext(ctx).Model(&entities.RegistrationInvitation{}).Where("facility_id IN ?", ids)
	switch f.Status {
	case "":
	case models.InvitationUsed:
		q = q.Where("used_at IS NOT NULL")
	case models.InvitationRevoked:
		q = q.Where("used_at IS NULL AND revoked_at IS NOT NULL")
	case models.InvitationExpired:
		q = q.Where("used_at IS NULL AND revoked_at IS NULL AND expires_at < ?", now)
	case models.InvitationActive:
		q = q.Where("used_at IS NULL AND revoked_at IS NULL AND expires_at >= ?", now)
	default:
		return nil, newDomainError(ErrValidation, FieldError{"query.status", "is not a known status"})
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, err
	}
	page := max(f.Page, 1)
	var rows []entities.RegistrationInvitation
	if err := q.Order("created_at DESC, id DESC").Limit(InvitationPageSize).Offset((page - 1) * InvitationPageSize).
		Find(&rows).Error; err != nil {
		return nil, err
	}
	out := &models.AdminInvitationList{
		Items: make([]models.AdminInvitation, 0, len(rows)), Page: page, PageSize: InvitationPageSize, Total: int(total),
	}
	if len(rows) == 0 {
		return out, nil
	}
	facs, err := facilityMap(s.db.WithContext(ctx), scope.FacilityIDs)
	if err != nil {
		return nil, err
	}
	regIDs := []uint64{}
	for i := range rows {
		if rows[i].RegistrationID != nil {
			regIDs = append(regIDs, *rows[i].RegistrationID)
		}
	}
	regPublic := map[uint64]string{}
	if len(regIDs) > 0 {
		var regs []entities.GuestRegistration
		if err := s.db.WithContext(ctx).Scopes(notDeleted).Select("id, public_id").Where("id IN ?", regIDs).
			Find(&regs).Error; err != nil {
			return nil, err
		}
		for _, r := range regs {
			regPublic[r.ID] = r.PublicID
		}
	}
	for i := range rows {
		inv := &rows[i]
		fac := facs[inv.FacilityID]
		var regPublicID *string
		if inv.RegistrationID != nil {
			if p, ok := regPublic[*inv.RegistrationID]; ok {
				regPublicID = &p
			}
		}
		out.Items = append(out.Items, adminInvitation(inv, &fac, regPublicID, now))
	}
	return out, nil
}

// Revoke revokes an unused invitation in scope (another facility's is 404).
// A used invitation is already consumed: 409 invitation_used. Revoking an
// already revoked invitation changes nothing and writes no audit row.
func (s *InvitationService) Revoke(ctx context.Context, scope AdminScope, publicID string) (*models.AdminInvitation, error) {
	if err := checkScope(scope); err != nil {
		return nil, err
	}
	var out models.AdminInvitation
	err := readCommittedTx(ctx, s.db, func(tx *gorm.DB) error {
		var inv entities.RegistrationInvitation
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("public_id = ? AND facility_id IN ?", publicID, scope.FacilityIDs).Take(&inv).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrNotFound
		}
		if err != nil {
			return err
		}
		if inv.UsedAt != nil {
			return ErrInvitationUsed
		}
		now := s.clock.Now()
		if inv.RevokedAt == nil {
			inv.RevokedAt, inv.UpdatedAt = &now, now
			if err := tx.Model(&entities.RegistrationInvitation{}).Where("id = ?", inv.ID).
				Updates(map[string]any{"revoked_at": now, "updated_at": now}).Error; err != nil {
				return err
			}
			if err := writeAuditMeta(tx, s.clock, inv.FacilityID, nil, entities.ActorAdmin, &scope.AdminUserID,
				AuditInvitationRevoked, map[string]string{"invitation_public_id": inv.PublicID}); err != nil {
				return err
			}
		}
		var f entities.Facility
		if err := tx.Where("id = ?", inv.FacilityID).Take(&f).Error; err != nil {
			return err
		}
		out = adminInvitation(&inv, &f, nil, now)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return &out, nil
}

// Current resolves the invitation of the wizard (public, read-only) and
// returns only what prefills the stay.
func (s *InvitationService) Current(ctx context.Context, raw string) (*models.CurrentInvitation, error) {
	inv, err := findInvitation(s.db.WithContext(ctx), raw, false)
	if err != nil {
		return nil, err
	}
	if err := checkInvitationUsable(inv, s.clock.Now()); err != nil {
		return nil, err
	}
	var f entities.Facility
	err = s.db.WithContext(ctx).Where("id = ? AND is_active = ?", inv.FacilityID, true).Take(&f).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvitationInvalid
	}
	if err != nil {
		return nil, err
	}
	return &models.CurrentInvitation{
		FacilityCode: f.Code,
		CheckInDate:  utils.CivilDateFromDB(inv.CheckInDate).String(),
		CheckOutDate: utils.CivilDateFromDB(inv.CheckOutDate).String(),
		ExpiresAt:    inv.ExpiresAt,
	}, nil
}

// findInvitation looks an invitation up by the hash of its raw token,
// optionally FOR UPDATE. Missing → ErrInvitationRequired; malformed,
// unknown or revoked → ErrInvitationInvalid. Expiry and use are left to
// the caller (checkInvitationUsable), because an idempotent replay accepts
// the invitation it already used.
func findInvitation(tx *gorm.DB, raw string, lock bool) (*entities.RegistrationInvitation, error) {
	if strings.TrimSpace(raw) == "" {
		return nil, ErrInvitationRequired
	}
	h, ok := hashToken(raw)
	if !ok {
		return nil, ErrInvitationInvalid
	}
	q := tx.Where("token_hash = ?", h)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var inv entities.RegistrationInvitation
	err := q.Take(&inv).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrInvitationInvalid
	}
	if err != nil {
		return nil, err
	}
	if inv.RevokedAt != nil {
		return nil, ErrInvitationInvalid
	}
	return &inv, nil
}

// checkInvitationUsable: expired (now after the end of the check-out day)
// → ErrInvitationExpired, already used → ErrInvitationUsed.
func checkInvitationUsable(inv *entities.RegistrationInvitation, now time.Time) error {
	if now.After(inv.ExpiresAt) {
		return ErrInvitationExpired
	}
	if inv.UsedAt != nil {
		return ErrInvitationUsed
	}
	return nil
}
