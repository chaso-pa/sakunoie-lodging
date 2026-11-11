package services

import (
	"context"
	"crypto/hmac"
	"database/sql"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// errReplayRetry: the key row vanished between the 1062 and the replay
// (e.g. the janitor removed an expired row); start over.
var errReplayRetry = errors.New("idempotency replay retry")

const maxCreateAttempts = 3

// RegistrationService creates registrations (contract §4.1–4.3) with
// Decision D idempotency.
type RegistrationService struct {
	db           *gorm.DB
	clock        Clock
	tokens       *TokenService
	idem         *IdempotencyService
	cipher       *utils.PassportCipher
	policy       PolicyVersions
	BeforeCommit func(tx *gorm.DB) error // test seam: runs after all inserts, before commit
}

func NewRegistrationService(db *gorm.DB, clock Clock, tokens *TokenService, idem *IdempotencyService, cipher *utils.PassportCipher) *RegistrationService {
	return &RegistrationService{db: db, clock: clock, tokens: tokens, idem: idem, cipher: cipher, policy: CurrentPolicyVersions()}
}

// CreateResult is the create/replay outcome. Replayed=true → HTTP 200.
type CreateResult struct {
	Body     models.CreateGuestRegistrationResult
	Replayed bool
}

// Create runs Decision D:
//   - Tx1: facility read, validation, INSERT key row, registration, guests,
//     token, audit; commit.
//   - On 1062 for key_hash: roll back and replay in a NEW transaction (the
//     REPEATABLE READ snapshot of Tx1 predates the other commit).
func (s *RegistrationService) Create(ctx context.Context, idempotencyKey string, in *models.CreateGuestRegistrationBody) (*CreateResult, error) {
	keyHash := s.idem.KeyHash(idempotencyKey)
	reqHash, err := s.idem.RequestHash(in)
	if err != nil {
		return nil, err
	}
	for attempt := 0; attempt < maxCreateAttempts; attempt++ {
		res, err := s.createNew(ctx, keyHash, reqHash, in)
		if err == nil || !isDuplicateIdempotencyKey(err) {
			return res, err
		}
		res, err = s.replay(ctx, keyHash, reqHash, in)
		if errors.Is(err, errReplayRetry) {
			continue
		}
		return res, err
	}
	return nil, errors.New("idempotency: too many retries")
}

func (s *RegistrationService) createNew(ctx context.Context, keyHash, reqHash []byte, in *models.CreateGuestRegistrationBody) (*CreateResult, error) {
	var res *CreateResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		f, err := loadActiveFacility(tx, in.FacilityCode)
		if err != nil {
			return err
		}
		v, err := s.validate(f, in)
		if err != nil {
			return err
		}
		publicID := uuid.NewString()
		if err := s.idem.insertKey(tx, keyHash, reqHash, publicID); err != nil {
			return err
		}
		res, err = s.insert(tx, f, v, publicID)
		return err
	})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// replay is Tx3: lock the key row, then overwrite (expired), conflict
// (different body) or rebuild the response from DB rows with an additional
// token (same body).
func (s *RegistrationService) replay(ctx context.Context, keyHash, reqHash []byte, in *models.CreateGuestRegistrationBody) (*CreateResult, error) {
	var res *CreateResult
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var row entities.APIIdempotencyKey
		err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("key_hash = ?", keyHash).Take(&row).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errReplayRetry
		}
		if err != nil {
			return err
		}
		now := s.clock.Now()
		if row.ExpiresAt.Before(now) {
			f, err := loadActiveFacility(tx, in.FacilityCode)
			if err != nil {
				return err
			}
			v, err := s.validate(f, in)
			if err != nil {
				return err
			}
			publicID := uuid.NewString()
			if err := s.idem.overwriteKey(tx, &row, reqHash, publicID); err != nil {
				return err
			}
			res, err = s.insert(tx, f, v, publicID)
			return err
		}
		if !hmac.Equal(row.RequestHash, reqHash) {
			return ErrIdempotencyConflict
		}
		var ptr idempotencyPointer
		if row.ResponseJSON == nil || json.Unmarshal([]byte(*row.ResponseJSON), &ptr) != nil {
			return errors.New("idempotency: corrupt response pointer")
		}
		var reg entities.GuestRegistration
		err = tx.Scopes(notDeleted).Clauses(clause.Locking{Strength: "UPDATE"}).
			Where("public_id = ?", ptr.RegistrationPublicID).Take(&reg).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return ErrIdempotencyConflict
		}
		if err != nil {
			return err
		}
		var f entities.Facility
		if err := tx.Where("id = ?", reg.FacilityID).Take(&f).Error; err != nil {
			return err
		}
		count, err := countGuests(tx, reg.ID)
		if err != nil {
			return err
		}
		body := resultBody(&f, &reg, count)
		raw, exp, ok, err := s.tokens.IssueAdditional(tx, &f, &reg)
		if err != nil {
			return err
		}
		if ok {
			body.EditToken, body.EditTokenExpiresAt = &raw, &exp
		} else {
			body.Code = models.EditLinkUnavailable
		}
		res = &CreateResult{Body: body, Replayed: true}
		return nil
	}, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return nil, err
	}
	return res, nil
}

// validatedCreate is the normalized create request.
type validatedCreate struct {
	in       *models.CreateGuestRegistrationBody
	checkIn  utils.CivilDate
	checkOut utils.CivilDate
	arrival  *string
	guests   []guestFields
}

func (s *RegistrationService) validate(f *entities.Facility, in *models.CreateGuestRegistrationBody) (*validatedCreate, error) {
	var errs []FieldError
	v := &validatedCreate{in: in}

	var err error
	v.checkIn, err = utils.ParseCivilDate(in.CheckInDate)
	if err != nil {
		errs = append(errs, FieldError{"body.check_in_date", "must be a date (YYYY-MM-DD)"})
	}
	v.checkOut, err = utils.ParseCivilDate(in.CheckOutDate)
	if err != nil {
		errs = append(errs, FieldError{"body.check_out_date", "must be a date (YYYY-MM-DD)"})
	}
	if len(errs) == 0 {
		if v.checkOut.Compare(v.checkIn) <= 0 {
			errs = append(errs, FieldError{"body.check_out_date", "must be after check_in_date"})
		}
		loc, err := time.LoadLocation(f.Timezone)
		if err != nil {
			return nil, err
		}
		if v.checkOut.Compare(utils.CivilDateIn(s.clock.Now(), loc)) < 0 {
			errs = append(errs, FieldError{"body.check_out_date", "must not be in the past"})
		}
	}
	if in.EstimatedArrivalTime != nil {
		t := strings.TrimSpace(*in.EstimatedArrivalTime)
		if _, err := time.Parse("15:04", t); err != nil {
			errs = append(errs, FieldError{"body.estimated_arrival_time", "must be HH:MM"})
		} else {
			t += ":00"
			v.arrival = &t
		}
	}
	if !in.TermsAccepted {
		errs = append(errs, FieldError{"body.terms_accepted", "must be true"})
	}
	if !in.PrivacyAccepted {
		errs = append(errs, FieldError{"body.privacy_accepted", "must be true"})
	}
	if in.TermsVersion != s.policy.TermsVersion {
		errs = append(errs, FieldError{"body.terms_version", "is not the current terms version"})
	}
	if in.PrivacyVersion != s.policy.PrivacyVersion {
		errs = append(errs, FieldError{"body.privacy_version", "is not the current privacy version"})
	}
	if in.PlannedGuestCount < 1 {
		errs = append(errs, FieldError{"body.planned_guest_count", "must be at least 1"})
	}
	switch {
	case len(in.Guests) == 0:
		errs = append(errs, FieldError{"body.guests", "at least one guest is required"})
	case len(in.Guests) > in.PlannedGuestCount:
		errs = append(errs, FieldError{"body.guests", "must not exceed planned_guest_count"})
	}
	reps := 0
	for i, g := range in.Guests {
		if g.IsRepresentative {
			reps++
		}
		gf := guestFields{
			IsRepresentative: g.IsRepresentative, FullName: g.FullName, NameKana: g.NameKana,
			Address: g.Address, Phone: g.Phone, Email: g.Email, HasJapanAddress: g.HasJapanAddress,
			Nationality: g.Nationality, PassportNumber: g.PassportNumber,
		}
		errs = append(errs, validateGuest(guestLoc(i), &gf)...)
		v.guests = append(v.guests, gf)
	}
	if len(in.Guests) > 0 && reps != 1 {
		errs = append(errs, FieldError{"body.guests", "exactly one guest must be the representative"})
	}
	if len(errs) > 0 {
		return nil, newDomainError(ErrValidation, errs...)
	}
	if in.PlannedGuestCount > int(f.MaxGuests) {
		return nil, newDomainError(ErrCapacityExceeded, FieldError{"body.planned_guest_count", "exceeds the facility capacity"})
	}
	return v, nil
}

func guestLoc(i int) string { return "body.guests[" + strconv.Itoa(i) + "]" }

// insert writes the registration, guests, first token and audit row.
func (s *RegistrationService) insert(tx *gorm.DB, f *entities.Facility, v *validatedCreate, publicID string) (*CreateResult, error) {
	now := s.clock.Now()
	in := v.in
	reg := entities.GuestRegistration{
		PublicID:             publicID,
		FacilityID:           f.ID,
		CheckInDate:          v.checkIn.ToDB(),
		CheckOutDate:         v.checkOut.ToDB(),
		EstimatedArrivalTime: v.arrival,
		PlannedGuestCount:    uint32(in.PlannedGuestCount),
		CapacitySnapshot:     f.MaxGuests,
		BookingSource:        trimmedOrNil(in.BookingSource),
		BookingReference:     trimmedOrNil(in.BookingReference),
		TermsVersion:         in.TermsVersion,
		PrivacyVersion:       in.PrivacyVersion,
		TermsAcceptedAt:      now,
		PrivacyAcceptedAt:    now,
		SubmittedAt:          now,
		CreatedAt:            now,
		UpdatedAt:            now,
	}
	if err := tx.Create(&reg).Error; err != nil {
		return nil, err
	}
	for i := range v.guests {
		row, err := buildGuest(s.cipher, reg.ID, &v.guests[i], s.clock)
		if err != nil {
			return nil, err
		}
		if err := tx.Create(row).Error; err != nil {
			return nil, err
		}
	}
	exp, err := tokenExpiry(f, &reg)
	if err != nil {
		return nil, err
	}
	raw, err := s.tokens.issue(tx, reg.ID, exp)
	if err != nil {
		return nil, err
	}
	if err := writeAudit(tx, s.clock, f.ID, &reg.ID, entities.ActorGuestToken, nil, "registration.created", nil); err != nil {
		return nil, err
	}
	if s.BeforeCommit != nil {
		if err := s.BeforeCommit(tx); err != nil {
			return nil, err
		}
	}
	body := resultBody(f, &reg, len(v.guests))
	body.EditToken, body.EditTokenExpiresAt = &raw, &exp
	return &CreateResult{Body: body}, nil
}

func resultBody(f *entities.Facility, reg *entities.GuestRegistration, count int) models.CreateGuestRegistrationResult {
	planned := int(reg.PlannedGuestCount)
	return models.CreateGuestRegistrationResult{
		PublicID:          reg.PublicID,
		FacilityCode:      f.Code,
		CheckInDate:       utils.CivilDateFromDB(reg.CheckInDate).String(),
		CheckOutDate:      utils.CivilDateFromDB(reg.CheckOutDate).String(),
		PlannedGuestCount: planned,
		RegisteredCount:   count,
		RemainingCount:    max(planned-count, 0),
		IsComplete:        count >= planned,
	}
}

// countGuests counts with a locking read so it sees the latest committed
// rows regardless of the transaction snapshot.
func countGuests(tx *gorm.DB, regID uint64) (int, error) {
	var ids []uint64
	if err := tx.Model(&entities.Guest{}).Clauses(clause.Locking{Strength: "UPDATE"}).
		Where("registration_id = ?", regID).Pluck("id", &ids).Error; err != nil {
		return 0, err
	}
	return len(ids), nil
}
