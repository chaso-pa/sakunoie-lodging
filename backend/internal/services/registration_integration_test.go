package services_test

import (
	"context"
	"errors"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"gorm.io/gorm"
)

// §4.3 atomic create.
func TestIntegrationCreateIsAtomic(t *testing.T) {
	e := setup(t)
	e.reg.BeforeCommit = func(*gorm.DB) error { return errors.New("injected failure after guest inserts") }
	body := sakuBody(3, testutil.Representative(), testutil.ForeignGuest("Test Guest B", "TESTPP0001"))
	if _, err := e.reg.Create(context.Background(), newKey(), body); err == nil {
		t.Fatal("expected injected failure")
	}
	for _, table := range []string{"guest_registrations", "guests", "registration_access_tokens", "api_idempotency_keys", "audit_logs"} {
		if n := testutil.CountRows(t, e.db, table); n != 0 {
			t.Errorf("%s has %d rows after rollback", table, n)
		}
	}
}

// §4.2 exactly one representative.
func TestIntegrationExactlyOneRepresentative(t *testing.T) {
	e := setup(t)
	comp := testutil.Representative()
	comp.IsRepresentative = false

	_, err := e.reg.Create(context.Background(), newKey(), sakuBody(2, comp))
	expectErr(t, err, services.ErrValidation)

	rep2 := testutil.Representative()
	_, err = e.reg.Create(context.Background(), newKey(), sakuBody(2, testutil.Representative(), rep2))
	expectErr(t, err, services.ErrValidation)

	res := e.mustCreate(t, newKey(), sakuBody(3, testutil.Representative()))
	if _, err := e.manage.AddCompanions(context.Background(), *res.Body.EditToken,
		[]models.CompanionInput{testutil.Companion("Test Guest B"), testutil.Companion("Test Guest C")}); err != nil {
		t.Fatal(err)
	}
	reg := e.registration(t, res.Body.PublicID)
	var reps int64
	e.db.Model(&entities.Guest{}).Where("registration_id = ? AND is_representative = ?", reg.ID, true).Count(&reps)
	if reps != 1 {
		t.Fatalf("representatives = %d", reps)
	}
}

// §4.1 counts and capacity.
func TestIntegrationCountsAndCapacity(t *testing.T) {
	e := setup(t)
	ctx := context.Background()

	empty := sakuBody(2)
	empty.Guests = nil
	_, err := e.reg.Create(ctx, newKey(), empty)
	expectErr(t, err, services.ErrValidation)

	comp := testutil.Representative()
	comp.IsRepresentative = false
	_, err = e.reg.Create(ctx, newKey(), sakuBody(1, testutil.Representative(), comp))
	expectErr(t, err, services.ErrValidation)

	_, err = e.reg.Create(ctx, newKey(), sakuBody(5))
	expectErr(t, err, services.ErrCapacityExceeded)

	same := sakuBody(1)
	same.CheckOutDate = same.CheckInDate
	_, err = e.reg.Create(ctx, newKey(), same)
	expectErr(t, err, services.ErrValidation)

	past := testutil.CreateBody(testutil.FacilitySaku, "2026-11-01", "2026-11-03", 1)
	_, err = e.reg.Create(ctx, newKey(), past)
	expectErr(t, err, services.ErrValidation)

	noConsent := sakuBody(1)
	noConsent.TermsAccepted = false
	_, err = e.reg.Create(ctx, newKey(), noConsent)
	expectErr(t, err, services.ErrValidation)

	oldTerms := sakuBody(1)
	oldTerms.TermsVersion = "2020-01-01"
	_, err = e.reg.Create(ctx, newKey(), oldTerms)
	expectErr(t, err, services.ErrValidation)

	noPassport := testutil.ForeignGuest("Test Guest B", "TESTPP0001")
	noPassport.IsRepresentative = true
	noPassport.PassportNumber = nil
	_, err = e.reg.Create(ctx, newKey(), sakuBody(1, noPassport))
	expectErr(t, err, services.ErrValidation)
	if f := services.FieldsOf(err); len(f) != 1 || f[0].Location != "body.guests[0].passport_number" {
		t.Fatalf("fields = %+v", f)
	}

	inactive := sakuBody(1)
	inactive.FacilityCode = "no-such-facility"
	_, err = e.reg.Create(ctx, newKey(), inactive)
	expectErr(t, err, services.ErrFacilityUnavailable)

	// Partial registration: planned 3, one guest.
	res := e.mustCreate(t, newKey(), sakuBody(3))
	if res.Body.RegisteredCount != 1 || res.Body.RemainingCount != 2 || res.Body.IsComplete || res.Replayed {
		t.Fatalf("unexpected result %+v", res.Body)
	}
	reg := e.registration(t, res.Body.PublicID)
	if reg.CapacitySnapshot != 4 || reg.PlannedGuestCount != 3 {
		t.Fatalf("snapshot=%d planned=%d", reg.CapacitySnapshot, reg.PlannedGuestCount)
	}
	// A later facility policy change does not rewrite history.
	if err := e.db.Model(&entities.Facility{}).Where("code = ?", testutil.FacilitySaku).Update("max_guests", 2).Error; err != nil {
		t.Fatal(err)
	}
	reg = e.registration(t, res.Body.PublicID)
	if reg.CapacitySnapshot != 4 || reg.PlannedGuestCount != 3 {
		t.Fatalf("historical row changed: snapshot=%d planned=%d", reg.CapacitySnapshot, reg.PlannedGuestCount)
	}
}

// §4.9 passport ciphertext at rest, last4 only.
func TestIntegrationPassportEncryptedAtRest(t *testing.T) {
	e := setup(t)
	rep := testutil.ForeignGuest("Test Guest P", "testpp 9876")
	rep.IsRepresentative = true
	res := e.mustCreate(t, newKey(), sakuBody(1, rep))

	reg := e.registration(t, res.Body.PublicID)
	var g entities.Guest
	if err := e.db.Where("registration_id = ?", reg.ID).Take(&g).Error; err != nil {
		t.Fatal(err)
	}
	if len(g.PassportNumberCiphertext) < 28 || string(g.PassportNumberCiphertext) == "TESTPP9876" {
		t.Fatalf("ciphertext len=%d", len(g.PassportNumberCiphertext))
	}
	var raw int64
	e.db.Raw("SELECT COUNT(*) FROM guests WHERE INSTR(passport_number_ciphertext, 'TESTPP9876') > 0").Scan(&raw)
	if raw != 0 {
		t.Fatal("plaintext found in ciphertext column")
	}
	pt, err := testutil.PassportCipher().Decrypt(g.PassportNumberCiphertext, []byte(g.PublicID))
	if err != nil || pt != "TESTPP9876" {
		t.Fatalf("decrypt: %q %v", pt, err)
	}
	if g.PassportNumberLast4 == nil || *g.PassportNumberLast4 != "9876" {
		t.Fatalf("last4 = %v", g.PassportNumberLast4)
	}
	sum, err := e.manage.Summary(context.Background(), *res.Body.EditToken)
	if err != nil {
		t.Fatal(err)
	}
	if sum.Guests[0].PassportLast4 == nil || *sum.Guests[0].PassportLast4 != "9876" {
		t.Fatalf("summary last4 = %v", sum.Guests[0].PassportLast4)
	}

	// A Japan-address guest never stores a passport even if one is sent.
	jp := testutil.Representative()
	jp.PassportNumber = strp("TESTPP0002")
	res2 := e.mustCreate(t, newKey(), sakuBody(1, jp))
	reg2 := e.registration(t, res2.Body.PublicID)
	var g2 entities.Guest
	e.db.Where("registration_id = ?", reg2.ID).Take(&g2)
	if g2.PassportNumberCiphertext != nil || g2.PassportNumberLast4 != nil {
		t.Fatal("passport stored for a Japan-address guest")
	}
}

// Dates are stored as civil dates; the token expiry is end of checkout day.
func TestIntegrationCreateStoresCivilDates(t *testing.T) {
	e := setup(t)
	res := e.mustCreate(t, newKey(), testutil.CreateBody(testutil.FacilityHonolulu, "2026-12-01", "2026-12-03", 2))
	reg := e.registration(t, res.Body.PublicID)
	if utils.CivilDateFromDB(reg.CheckOutDate).String() != "2026-12-03" {
		t.Fatalf("checkout = %v", reg.CheckOutDate)
	}
	if got := res.Body.EditTokenExpiresAt.UTC().Format("2006-01-02T15:04:05.999999Z07:00"); got != "2026-12-04T09:59:59.999999Z" {
		t.Fatalf("expiry = %s", got)
	}
	var audit entities.AuditLog
	if err := e.db.Where("registration_id = ?", reg.ID).Take(&audit).Error; err != nil {
		t.Fatal(err)
	}
	if audit.Action != "registration.created" || audit.ActorType != entities.ActorGuestToken || audit.ChangedFieldsJSON != nil {
		t.Fatalf("audit = %+v", audit)
	}
}
