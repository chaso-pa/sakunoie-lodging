package services

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"strings"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// AdminPageSize is the ledger page size.
const AdminPageSize = 50

// Audit actions written by the admin console.
const (
	AuditAdminUpdated      = "admin.registration_updated"
	AuditAdminRevealed     = "admin.registration_revealed"
	AuditIdentityVerified  = "registration.identity_verified"
	AuditEditLinkReissued  = "registration.edit_link_reissued"
	AuditRegistrationsExpo = "registrations.exported"
)

// AdminRegistrationService implements the admin ledger (contract §5 Admin).
// Every method takes an AdminScope right after ctx, and every query is
// limited to scope.FacilityIDs and to rows that are not soft-deleted.
type AdminRegistrationService struct {
	db     *gorm.DB
	clock  Clock
	tokens *TokenService
	cipher *utils.PassportCipher
}

func NewAdminRegistrationService(db *gorm.DB, clock Clock, tokens *TokenService, cipher *utils.PassportCipher) *AdminRegistrationService {
	return &AdminRegistrationService{db: db, clock: clock, tokens: tokens, cipher: cipher}
}

// checkScope fails closed on an empty scope (it would otherwise match
// nothing with `IN (NULL)`, but an empty scope is always a bug).
func checkScope(scope AdminScope) error {
	if scope.AdminUserID == 0 || len(scope.FacilityIDs) == 0 {
		return ErrNotAuthorized
	}
	return nil
}

func (s *AdminRegistrationService) tx(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return s.db.WithContext(ctx).Transaction(fn, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
}

// Facilities lists the facilities in scope.
func (s *AdminRegistrationService) Facilities(ctx context.Context, scope AdminScope) ([]models.AdminFacility, error) {
	if err := checkScope(scope); err != nil {
		return nil, err
	}
	var rows []entities.Facility
	if err := s.db.WithContext(ctx).Where("id IN ?", scope.FacilityIDs).Order("id").Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make([]models.AdminFacility, len(rows))
	for i := range rows {
		out[i] = adminFacility(&rows[i])
	}
	return out, nil
}

func adminFacility(f *entities.Facility) models.AdminFacility {
	return models.AdminFacility{ID: f.ID, Code: f.Code, Name: f.Name, Timezone: f.Timezone, MaxGuests: int(f.MaxGuests), IsActive: f.IsActive}
}

// AdminListFilter is the ledger filter. Zero values mean "no filter".
type AdminListFilter struct {
	FacilityID uint64
	From, To   string
	Q          string
	Page       int
}

// dateRange parses the optional inclusive check-in date range.
func dateRange(from, to string) (*utils.CivilDate, *utils.CivilDate, error) {
	var f, t *utils.CivilDate
	var errs []FieldError
	if from != "" {
		d, err := utils.ParseCivilDate(from)
		if err != nil {
			errs = append(errs, FieldError{"query.from", "must be a date (YYYY-MM-DD)"})
		} else {
			f = &d
		}
	}
	if to != "" {
		d, err := utils.ParseCivilDate(to)
		if err != nil {
			errs = append(errs, FieldError{"query.to", "must be a date (YYYY-MM-DD)"})
		} else {
			t = &d
		}
	}
	if f != nil && t != nil && f.Compare(*t) > 0 {
		errs = append(errs, FieldError{"query.to", "must not be before from"})
	}
	if len(errs) > 0 {
		return nil, nil, newDomainError(ErrValidation, errs...)
	}
	return f, t, nil
}

// likePattern escapes LIKE wildcards with '!' (ESCAPE '!').
func likePattern(q string) string {
	r := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_")
	return "%" + r.Replace(q) + "%"
}

// scopedRegistrations is the base ledger query for the given facilities.
func scopedRegistrations(db *gorm.DB, facilityIDs []uint64, from, to *utils.CivilDate, q string) *gorm.DB {
	db = db.Model(&entities.GuestRegistration{}).Scopes(notDeleted).
		Where("guest_registrations.facility_id IN ?", facilityIDs)
	if from != nil {
		db = db.Where("guest_registrations.check_in_date >= ?", from.ToDB())
	}
	if to != nil {
		db = db.Where("guest_registrations.check_in_date <= ?", to.ToDB())
	}
	if q = strings.TrimSpace(q); q != "" {
		p := likePattern(q)
		db = db.Where("(guest_registrations.booking_reference LIKE ? ESCAPE '!' OR EXISTS ("+
			"SELECT 1 FROM guests g WHERE g.registration_id = guest_registrations.id "+
			"AND (g.full_name LIKE ? ESCAPE '!' OR g.name_kana LIKE ? ESCAPE '!')))", p, p, p)
	}
	return db
}

// facilitiesFilter resolves an optional facility filter against the scope:
// an unauthorized facility is 403, never an empty result.
func facilitiesFilter(scope AdminScope, facilityID uint64) ([]uint64, error) {
	if facilityID == 0 {
		return scope.FacilityIDs, nil
	}
	if !scope.Has(facilityID) {
		return nil, ErrForbidden
	}
	return []uint64{facilityID}, nil
}

// List returns one ledger page plus metrics over the whole filter.
func (s *AdminRegistrationService) List(ctx context.Context, scope AdminScope, f AdminListFilter) (*models.AdminRegistrationList, error) {
	if err := checkScope(scope); err != nil {
		return nil, err
	}
	ids, err := facilitiesFilter(scope, f.FacilityID)
	if err != nil {
		return nil, err
	}
	from, to, err := dateRange(f.From, f.To)
	if err != nil {
		return nil, err
	}
	page := max(f.Page, 1)
	db := s.db.WithContext(ctx)

	var m struct {
		Registrations, Guests, Incomplete, Verified int64
	}
	if err := scopedRegistrations(db, ids, from, to, f.Q).
		Joins("LEFT JOIN (SELECT registration_id, COUNT(*) AS cnt FROM guests GROUP BY registration_id) gc ON gc.registration_id = guest_registrations.id").
		Select("COUNT(*) AS registrations, " +
			"COALESCE(SUM(gc.cnt), 0) AS guests, " +
			"COALESCE(SUM(COALESCE(gc.cnt, 0) < guest_registrations.planned_guest_count), 0) AS incomplete, " +
			"COALESCE(SUM(guest_registrations.identity_verified_at IS NOT NULL), 0) AS verified").
		Scan(&m).Error; err != nil {
		return nil, err
	}

	var regs []entities.GuestRegistration
	if err := scopedRegistrations(db, ids, from, to, f.Q).
		Order("guest_registrations.check_in_date DESC, guest_registrations.id DESC").
		Limit(AdminPageSize).Offset((page - 1) * AdminPageSize).
		Find(&regs).Error; err != nil {
		return nil, err
	}
	out := &models.AdminRegistrationList{
		Items:    make([]models.AdminRegistrationRow, 0, len(regs)),
		Page:     page,
		PageSize: AdminPageSize,
		Total:    int(m.Registrations),
		Metrics: models.AdminLedgerMetrics{
			Registrations: int(m.Registrations), Guests: int(m.Guests),
			Incomplete: int(m.Incomplete), Verified: int(m.Verified),
		},
	}
	if len(regs) == 0 {
		return out, nil
	}
	regIDs := make([]uint64, len(regs))
	for i := range regs {
		regIDs[i] = regs[i].ID
	}
	counts, err := guestCounts(db, regIDs)
	if err != nil {
		return nil, err
	}
	var reps []entities.Guest
	if err := db.Where("registration_id IN ? AND is_representative = ?", regIDs, true).Find(&reps).Error; err != nil {
		return nil, err
	}
	repName := make(map[uint64]string, len(reps))
	for _, g := range reps {
		repName[g.RegistrationID] = g.FullName
	}
	facs, err := facilityMap(db, scope.FacilityIDs)
	if err != nil {
		return nil, err
	}
	for i := range regs {
		r := &regs[i]
		fac := facs[r.FacilityID]
		planned, count := int(r.PlannedGuestCount), counts[r.ID]
		out.Items = append(out.Items, models.AdminRegistrationRow{
			PublicID:           r.PublicID,
			Facility:           models.AdminFacilityRef{ID: fac.ID, Code: fac.Code, Name: fac.Name},
			CheckInDate:        utils.CivilDateFromDB(r.CheckInDate).String(),
			CheckOutDate:       utils.CivilDateFromDB(r.CheckOutDate).String(),
			BookingSource:      r.BookingSource,
			BookingReference:   r.BookingReference,
			RepresentativeName: repName[r.ID],
			PlannedGuestCount:  planned,
			RegisteredCount:    count,
			RemainingCount:     max(planned-count, 0),
			IsComplete:         count >= planned,
			IdentityVerified:   r.IdentityVerifiedAt != nil,
			IdentityVerifiedAt: r.IdentityVerifiedAt,
			SubmittedAt:        r.SubmittedAt,
		})
	}
	return out, nil
}

func guestCounts(db *gorm.DB, regIDs []uint64) (map[uint64]int, error) {
	var rows []struct {
		RegistrationID uint64
		Cnt            int
	}
	if err := db.Model(&entities.Guest{}).Select("registration_id, COUNT(*) AS cnt").
		Where("registration_id IN ?", regIDs).Group("registration_id").Scan(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint64]int, len(rows))
	for _, r := range rows {
		out[r.RegistrationID] = r.Cnt
	}
	return out, nil
}

func facilityMap(db *gorm.DB, ids []uint64) (map[uint64]entities.Facility, error) {
	var rows []entities.Facility
	if err := db.Where("id IN ?", ids).Find(&rows).Error; err != nil {
		return nil, err
	}
	out := make(map[uint64]entities.Facility, len(rows))
	for _, f := range rows {
		out[f.ID] = f
	}
	return out, nil
}

// loadScoped finds a registration by public ID inside the scope. Anything
// else, including another facility's registration, is 404 so the response
// never reveals that it exists (contract §5).
func loadScoped(tx *gorm.DB, scope AdminScope, publicID string, lock bool) (*entities.GuestRegistration, error) {
	q := tx.Scopes(notDeleted).Where("public_id = ? AND facility_id IN ?", publicID, scope.FacilityIDs)
	if lock {
		q = q.Clauses(clause.Locking{Strength: "UPDATE"})
	}
	var reg entities.GuestRegistration
	err := q.Take(&reg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &reg, nil
}

// Detail returns one registration. Address, phone and email are masked
// unless reveal is set; a reveal is audited.
func (s *AdminRegistrationService) Detail(ctx context.Context, scope AdminScope, publicID string, reveal bool) (*models.AdminRegistrationDetail, error) {
	if err := checkScope(scope); err != nil {
		return nil, err
	}
	var out *models.AdminRegistrationDetail
	err := s.tx(ctx, func(tx *gorm.DB) error {
		reg, err := loadScoped(tx, scope, publicID, false)
		if err != nil {
			return err
		}
		if reveal {
			if err := writeAudit(tx, s.clock, reg.FacilityID, &reg.ID, entities.ActorAdmin, &scope.AdminUserID,
				AuditAdminRevealed, []string{"guest.address", "guest.email", "guest.phone"}); err != nil {
				return err
			}
		}
		out, err = s.detail(tx, reg, reveal)
		return err
	})
	return out, err
}

func (s *AdminRegistrationService) detail(tx *gorm.DB, reg *entities.GuestRegistration, reveal bool) (*models.AdminRegistrationDetail, error) {
	var f entities.Facility
	if err := tx.Where("id = ?", reg.FacilityID).Take(&f).Error; err != nil {
		return nil, err
	}
	var guests []entities.Guest
	if err := tx.Where("registration_id = ?", reg.ID).Order("is_representative DESC, id ASC").Find(&guests).Error; err != nil {
		return nil, err
	}
	var links []entities.RegistrationAccessToken
	if err := tx.Where("registration_id = ? AND revoked_at IS NULL AND expires_at >= ?", reg.ID, s.clock.Now()).
		Find(&links).Error; err != nil {
		return nil, err
	}
	planned := int(reg.PlannedGuestCount)
	out := &models.AdminRegistrationDetail{
		PublicID:           reg.PublicID,
		Facility:           adminFacility(&f),
		CheckInDate:        utils.CivilDateFromDB(reg.CheckInDate).String(),
		CheckOutDate:       utils.CivilDateFromDB(reg.CheckOutDate).String(),
		PlannedGuestCount:  planned,
		CapacitySnapshot:   int(reg.CapacitySnapshot),
		RegisteredCount:    len(guests),
		RemainingCount:     max(planned-len(guests), 0),
		IsComplete:         len(guests) >= planned,
		BookingSource:      reg.BookingSource,
		BookingReference:   reg.BookingReference,
		TermsVersion:       reg.TermsVersion,
		PrivacyVersion:     reg.PrivacyVersion,
		SubmittedAt:        reg.SubmittedAt,
		IdentityVerified:   reg.IdentityVerifiedAt != nil,
		IdentityVerifiedAt: reg.IdentityVerifiedAt,
		ActiveEditLinks:    len(links),
		Revealed:           reveal,
		Version:            ETag(reg),
		Guests:             make([]models.AdminGuest, 0, len(guests)),
	}
	if reg.EstimatedArrivalTime != nil && len(*reg.EstimatedArrivalTime) >= 5 {
		hhmm := (*reg.EstimatedArrivalTime)[:5]
		out.EstimatedArrivalTime = &hhmm
	}
	for i := range links {
		if out.EditLinkExpiresAt == nil || links[i].ExpiresAt.After(*out.EditLinkExpiresAt) {
			out.EditLinkExpiresAt = &links[i].ExpiresAt
		}
	}
	for _, g := range guests {
		ag := models.AdminGuest{
			PublicID:         g.PublicID,
			IsRepresentative: g.IsRepresentative,
			FullName:         g.FullName,
			NameKana:         g.NameKana,
			Address:          g.Address,
			Phone:            g.Phone,
			Email:            g.Email,
			HasJapanAddress:  g.HasJapanAddress,
			Nationality:      g.Nationality,
			PassportLast4:    g.PassportNumberLast4,
		}
		if !reveal {
			ag.Address = utils.MaskAddress(g.Address)
			if g.Phone != nil {
				m := utils.MaskPhone(*g.Phone)
				ag.Phone = &m
			}
			if g.Email != nil {
				m := utils.MaskEmail(*g.Email)
				ag.Email = &m
			}
		}
		out.Guests = append(out.Guests, ag)
	}
	return out, nil
}

// nextUpdatedAt returns a new updated_at strictly after prev, so the ETag
// always changes.
func nextUpdatedAt(clock Clock, prev time.Time) time.Time {
	now := clock.Now()
	if !now.After(prev) {
		now = prev.Add(time.Microsecond)
	}
	return now
}

// guestChange is the pending update of one guest.
type guestChange struct {
	row     *entities.Guest
	updates map[string]any
}

// Patch applies a host correction under the registration lock. Dates,
// counts, the representative and passport numbers may all change; the
// capacity check uses the current facility max_guests unless
// capacity_override is set. Changed field names are audited.
func (s *AdminRegistrationService) Patch(ctx context.Context, scope AdminScope, publicID, ifMatchHeader string, in *models.AdminPatchRegistrationBody) (*models.AdminRegistrationDetail, error) {
	if err := checkScope(scope); err != nil {
		return nil, err
	}
	var out *models.AdminRegistrationDetail
	err := s.tx(ctx, func(tx *gorm.DB) error {
		reg, err := loadScoped(tx, scope, publicID, true)
		if err != nil {
			return err
		}
		if !ifMatch(ifMatchHeader, ETag(reg)) {
			return ErrPreconditionFailed
		}
		var f entities.Facility
		if err := tx.Where("id = ?", reg.FacilityID).Take(&f).Error; err != nil {
			return err
		}
		var guests []entities.Guest
		if err := tx.Where("registration_id = ?", reg.ID).Order("id").Find(&guests).Error; err != nil {
			return err
		}

		regUpdates, changed, errs, err := s.registrationChanges(reg, &f, len(guests), in)
		if err != nil {
			return err
		}
		guestChanges, guestFieldsChanged, gerrs := s.guestChanges(guests, in.Guests)
		errs = append(errs, gerrs...)
		if len(errs) > 0 {
			return newDomainError(ErrValidation, errs...)
		}
		changed = append(changed, guestFieldsChanged...)
		if len(changed) == 0 {
			out, err = s.detail(tx, reg, false)
			return err
		}
		for _, gc := range guestChanges {
			if len(gc.updates) == 0 {
				continue
			}
			gc.updates["updated_at"] = s.clock.Now()
			if err := tx.Model(&entities.Guest{}).Where("id = ?", gc.row.ID).Updates(gc.updates).Error; err != nil {
				return err
			}
		}
		reg.UpdatedAt = nextUpdatedAt(s.clock, reg.UpdatedAt)
		regUpdates["updated_at"] = reg.UpdatedAt
		if err := tx.Model(&entities.GuestRegistration{}).Where("id = ?", reg.ID).Updates(regUpdates).Error; err != nil {
			return err
		}
		slices.Sort(changed)
		changed = slices.Compact(changed)
		if err := writeAudit(tx, s.clock, reg.FacilityID, &reg.ID, entities.ActorAdmin, &scope.AdminUserID, AuditAdminUpdated, changed); err != nil {
			return err
		}
		reg, err = loadScoped(tx, scope, publicID, false)
		if err != nil {
			return err
		}
		out, err = s.detail(tx, reg, false)
		return err
	})
	return out, err
}

// registrationChanges validates the registration-level fields.
func (s *AdminRegistrationService) registrationChanges(reg *entities.GuestRegistration, f *entities.Facility, registered int, in *models.AdminPatchRegistrationBody) (map[string]any, []string, []FieldError, error) {
	updates := map[string]any{}
	var changed []string
	var errs []FieldError

	checkIn := utils.CivilDateFromDB(reg.CheckInDate)
	checkOut := utils.CivilDateFromDB(reg.CheckOutDate)
	if in.CheckInDate != nil {
		d, err := utils.ParseCivilDate(*in.CheckInDate)
		if err != nil {
			errs = append(errs, FieldError{"body.check_in_date", "must be a date (YYYY-MM-DD)"})
		} else if d != checkIn {
			checkIn = d
			updates["check_in_date"] = d.ToDB()
			changed = append(changed, "check_in_date")
		}
	}
	if in.CheckOutDate != nil {
		d, err := utils.ParseCivilDate(*in.CheckOutDate)
		if err != nil {
			errs = append(errs, FieldError{"body.check_out_date", "must be a date (YYYY-MM-DD)"})
		} else if d != checkOut {
			checkOut = d
			updates["check_out_date"] = d.ToDB()
			changed = append(changed, "check_out_date")
		}
	}
	if checkOut.Compare(checkIn) <= 0 {
		errs = append(errs, FieldError{"body.check_out_date", "must be after check_in_date"})
	}
	if in.EstimatedArrivalTime != nil {
		t := strings.TrimSpace(*in.EstimatedArrivalTime)
		if t == "" {
			updates["estimated_arrival_time"] = nil
		} else if _, err := time.Parse("15:04", t); err != nil {
			errs = append(errs, FieldError{"body.estimated_arrival_time", "must be HH:MM"})
		} else {
			updates["estimated_arrival_time"] = t + ":00"
		}
		changed = append(changed, "estimated_arrival_time")
	}
	if in.BookingSource != nil {
		updates["booking_source"] = trimmedOrNil(in.BookingSource)
		changed = append(changed, "booking_source")
	}
	if in.BookingReference != nil {
		updates["booking_reference"] = trimmedOrNil(in.BookingReference)
		changed = append(changed, "booking_reference")
	}
	if in.PlannedGuestCount != nil {
		n := *in.PlannedGuestCount
		switch {
		case n < 1:
			errs = append(errs, FieldError{"body.planned_guest_count", "must be at least 1"})
		case n < registered:
			errs = append(errs, FieldError{"body.planned_guest_count", "must not be less than the registered count"})
		case n > int(f.MaxGuests) && !in.CapacityOverride:
			if len(errs) == 0 {
				return nil, nil, nil, newDomainError(ErrCapacityExceeded,
					FieldError{"body.planned_guest_count", "exceeds the facility capacity; set capacity_override to allow"})
			}
		default:
			if n > int(f.MaxGuests) {
				changed = append(changed, "capacity_override")
			}
			updates["planned_guest_count"] = n
			changed = append(changed, "planned_guest_count")
		}
	}
	return updates, changed, errs, nil
}

// guestChanges validates guest corrections, including a representative
// change and passport re-encryption. Only field names are returned for the
// audit row.
func (s *AdminRegistrationService) guestChanges(guests []entities.Guest, in []models.AdminPatchGuest) ([]*guestChange, []string, []FieldError) {
	byID := make(map[string]*guestChange, len(guests))
	all := make([]*guestChange, len(guests))
	for i := range guests {
		gc := &guestChange{row: &guests[i], updates: map[string]any{}}
		byID[guests[i].PublicID] = gc
		all[i] = gc
	}
	var changed []string
	var errs []FieldError
	seen := map[string]bool{}
	newRep := ""
	for i, p := range in {
		loc := guestLoc(i)
		gc, ok := byID[p.PublicID]
		if !ok {
			errs = append(errs, FieldError{loc + ".public_id", "is not a guest of this registration"})
			continue
		}
		if seen[p.PublicID] {
			errs = append(errs, FieldError{loc + ".public_id", "appears more than once"})
			continue
		}
		seen[p.PublicID] = true
		g := gc.row
		if p.IsRepresentative != nil {
			switch {
			case !*p.IsRepresentative:
				errs = append(errs, FieldError{loc + ".is_representative", "only true is accepted; mark another guest as the representative instead"})
			case newRep != "":
				errs = append(errs, FieldError{loc + ".is_representative", "only one guest can be the representative"})
			case !g.IsRepresentative:
				newRep = p.PublicID
			}
		}
		if p.FullName != nil {
			if v := strings.TrimSpace(*p.FullName); v == "" {
				errs = append(errs, FieldError{loc + ".full_name", "is required"})
			} else {
				gc.updates["full_name"] = v
				changed = append(changed, "guest.full_name")
			}
		}
		if p.Address != nil {
			if v := strings.TrimSpace(*p.Address); v == "" {
				errs = append(errs, FieldError{loc + ".address", "is required"})
			} else {
				gc.updates["address"] = v
				changed = append(changed, "guest.address")
			}
		}
		for name, v := range map[string]*string{"name_kana": p.NameKana, "phone": p.Phone, "email": p.Email} {
			if v != nil {
				gc.updates[name] = trimmedOrNil(v)
				changed = append(changed, "guest."+name)
			}
		}

		// Identity fields: the resulting state must satisfy the same
		// conditional rule as at creation.
		hasJapan := g.HasJapanAddress
		if p.HasJapanAddress != nil && *p.HasJapanAddress != g.HasJapanAddress {
			hasJapan = *p.HasJapanAddress
			gc.updates["has_japan_address"] = hasJapan
			changed = append(changed, "guest.has_japan_address")
		}
		nationality := g.Nationality
		if p.Nationality != nil {
			nationality = trimmedOrNil(p.Nationality)
			gc.updates["nationality"] = nationality
			changed = append(changed, "guest.nationality")
		}
		if hasJapan {
			// Data minimization: no passport number for a Japan address.
			if g.PassportNumberCiphertext != nil {
				gc.updates["passport_number_ciphertext"] = nil
				gc.updates["passport_number_last4"] = nil
				changed = append(changed, "guest.passport_number")
			}
			continue
		}
		if nationality == nil {
			errs = append(errs, FieldError{loc + ".nationality", "is required when has_japan_address is false"})
		}
		switch {
		case p.PassportNumber != nil:
			n := utils.NormalizePassport(*p.PassportNumber)
			if !passportRe.MatchString(n) {
				errs = append(errs, FieldError{loc + ".passport_number", "must be 6-20 letters or digits"})
				continue
			}
			ct, err := s.cipher.Encrypt(n, []byte(g.PublicID))
			if err != nil {
				errs = append(errs, FieldError{loc + ".passport_number", "could not be stored"})
				continue
			}
			gc.updates["passport_number_ciphertext"] = ct
			gc.updates["passport_number_last4"] = utils.PassportLast4(n)
			changed = append(changed, "guest.passport_number")
		case g.PassportNumberCiphertext == nil:
			errs = append(errs, FieldError{loc + ".passport_number", "is required when has_japan_address is false"})
		}
	}
	if newRep != "" {
		for _, gc := range all {
			switch {
			case gc.row.PublicID == newRep:
				gc.updates["is_representative"] = true
			case gc.row.IsRepresentative:
				gc.updates["is_representative"] = false
			}
		}
		changed = append(changed, "guest.is_representative")
	}
	return all, changed, errs
}

// VerifyIdentity records the in-person identity check by this admin. It is
// idempotent: a registration that is already verified keeps its first
// verification and no new audit row is written.
func (s *AdminRegistrationService) VerifyIdentity(ctx context.Context, scope AdminScope, publicID string) (*models.AdminRegistrationDetail, error) {
	if err := checkScope(scope); err != nil {
		return nil, err
	}
	var out *models.AdminRegistrationDetail
	err := s.tx(ctx, func(tx *gorm.DB) error {
		reg, err := loadScoped(tx, scope, publicID, true)
		if err != nil {
			return err
		}
		if reg.IdentityVerifiedAt == nil {
			now := s.clock.Now()
			reg.UpdatedAt = nextUpdatedAt(s.clock, reg.UpdatedAt)
			reg.IdentityVerifiedAt = &now
			reg.IdentityVerifiedBy = &scope.AdminUserID
			if err := tx.Model(&entities.GuestRegistration{}).Where("id = ?", reg.ID).Updates(map[string]any{
				"identity_verified_at": now,
				"identity_verified_by": scope.AdminUserID,
				"updated_at":           reg.UpdatedAt,
			}).Error; err != nil {
				return err
			}
			if err := writeAudit(tx, s.clock, reg.FacilityID, &reg.ID, entities.ActorAdmin, &scope.AdminUserID,
				AuditIdentityVerified, []string{"identity_verified_at", "identity_verified_by"}); err != nil {
				return err
			}
		}
		out, err = s.detail(tx, reg, false)
		return err
	})
	return out, err
}

// ReissueEditLink revokes every token of the registration and returns a new
// one, valid until the end of the current checkout day (contract §3.4).
func (s *AdminRegistrationService) ReissueEditLink(ctx context.Context, scope AdminScope, publicID string) (*models.ReissueEditLinkResult, error) {
	if err := checkScope(scope); err != nil {
		return nil, err
	}
	var out *models.ReissueEditLinkResult
	err := s.tx(ctx, func(tx *gorm.DB) error {
		reg, err := loadScoped(tx, scope, publicID, true)
		if err != nil {
			return err
		}
		var f entities.Facility
		if err := tx.Where("id = ?", reg.FacilityID).Take(&f).Error; err != nil {
			return err
		}
		raw, exp, err := s.tokens.Reissue(tx, &f, reg)
		if err != nil {
			return err
		}
		if err := writeAudit(tx, s.clock, reg.FacilityID, &reg.ID, entities.ActorAdmin, &scope.AdminUserID,
			AuditEditLinkReissued, nil); err != nil {
			return err
		}
		out = &models.ReissueEditLinkResult{EditToken: raw, EditTokenExpiresAt: exp, Expired: s.clock.Now().After(exp)}
		return nil
	})
	return out, err
}
