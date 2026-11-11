package services_test

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
)

// §4.4 concurrent additions: exactly planned - current succeed.
func TestIntegrationConcurrentCompanionAdditions(t *testing.T) {
	e := setup(t)
	res := e.mustCreate(t, newKey(), sakuBody(4))
	token := *res.Body.EditToken

	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, 10)
	for i := range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			_, errs[i] = e.manage.AddCompanions(context.Background(), token,
				[]models.CompanionInput{testutil.Companion("Test Companion")})
		}()
	}
	close(start)
	wg.Wait()

	ok, conflict := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			ok++
		case strings.Contains(err.Error(), services.ErrCapacityExceeded.Error()):
			conflict++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if ok != 3 || conflict != 7 {
		t.Fatalf("ok=%d conflict=%d, want 3/7", ok, conflict)
	}
	reg := e.registration(t, res.Body.PublicID)
	if n := testutil.CountRows(t, e.db.Where("registration_id = ?", reg.ID), "guests"); n != 4 {
		t.Fatalf("guests = %d", n)
	}
}

// §4.6 expiry at end of checkout day (±1µs) and revocation.
func TestIntegrationTokenExpiryAndRevocation(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	for _, tc := range []struct {
		facility, expiry string
	}{
		{testutil.FacilitySaku, "2026-12-03T14:59:59.999999Z"},     // Asia/Tokyo
		{testutil.FacilityHonolulu, "2026-12-04T09:59:59.999999Z"}, // Pacific/Honolulu
	} {
		e.clock.Set(baseNow)
		res := e.mustCreate(t, newKey(), testutil.CreateBody(tc.facility, "2026-12-01", "2026-12-03", 1))
		exp, _ := time.Parse(time.RFC3339Nano, tc.expiry)
		if !res.Body.EditTokenExpiresAt.Equal(exp) {
			t.Fatalf("%s expiry %s", tc.facility, res.Body.EditTokenExpiresAt)
		}
		e.clock.Set(exp)
		if _, err := e.manage.Summary(ctx, *res.Body.EditToken); err != nil {
			t.Fatalf("%s at expiry: %v", tc.facility, err)
		}
		e.clock.Set(exp.Add(time.Microsecond))
		_, err := e.manage.Summary(ctx, *res.Body.EditToken)
		expectErr(t, err, services.ErrTokenExpired)
	}

	e.clock.Set(baseNow)
	res := e.mustCreate(t, newKey(), sakuBody(2))
	for _, bad := range []string{"", "not-a-token", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA"} {
		_, err := e.manage.Summary(ctx, bad)
		expectErr(t, err, services.ErrTokenInvalid)
	}
	newTok := e.reissue(t, res.Body.PublicID)
	_, err := e.manage.Summary(ctx, *res.Body.EditToken)
	expectErr(t, err, services.ErrTokenInvalid)
	if _, err := e.manage.Summary(ctx, newTok); err != nil {
		t.Fatal(err)
	}

	// Soft-deleted registration: the token stops working.
	if err := e.db.Model(&entities.GuestRegistration{}).Where("public_id = ?", res.Body.PublicID).
		Update("deleted_at", baseNow).Error; err != nil {
		t.Fatal(err)
	}
	_, err = e.manage.Summary(ctx, newTok)
	expectErr(t, err, services.ErrTokenInvalid)

	// last_used_at is recorded.
	var used int64
	e.db.Model(&entities.RegistrationAccessToken{}).Where("last_used_at IS NOT NULL").Count(&used)
	if used == 0 {
		t.Fatal("last_used_at not recorded")
	}
}

// §4.7 token scope: companions only, allowlisted fields, no passport output.
func TestIntegrationTokenFieldScope(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	rep := testutil.ForeignGuest("Test Guest R", "TESTPP4321")
	rep.IsRepresentative = true
	res := e.mustCreate(t, newKey(), sakuBody(3, rep))
	token := *res.Body.EditToken

	sum, err := e.manage.AddCompanions(ctx, token, []models.CompanionInput{testutil.Companion("Test Guest C")})
	if err != nil {
		t.Fatal(err)
	}
	var repID, compID string
	for _, g := range sum.Guests {
		if g.IsRepresentative {
			repID = g.PublicID
		} else {
			compID = g.PublicID
		}
	}

	_, err = e.manage.PatchGuest(ctx, token, sum.Version, repID, &models.PatchManageGuestBody{FullName: strp("Changed")})
	expectErr(t, err, services.ErrFieldNotEditable)
	_, err = e.manage.PatchGuest(ctx, token, sum.Version, compID, &models.PatchManageGuestBody{PassportNumber: strp("TESTPP0003")})
	expectErr(t, err, services.ErrFieldNotEditable)
	yes := true
	_, err = e.manage.PatchGuest(ctx, token, sum.Version, compID, &models.PatchManageGuestBody{HasJapanAddress: &yes})
	expectErr(t, err, services.ErrFieldNotEditable)
	_, err = e.manage.PatchRegistration(ctx, token, sum.Version, &models.PatchManageRegistrationBody{CheckOutDate: strp("2026-12-09")})
	expectErr(t, err, services.ErrFieldNotEditable)
	_, err = e.manage.PatchGuest(ctx, token, sum.Version, "00000000-0000-4000-8000-000000000000", &models.PatchManageGuestBody{Phone: strp("1")})
	expectErr(t, err, services.ErrNotFound)

	// Stale If-Match → 412; current → OK.
	_, err = e.manage.PatchGuest(ctx, token, `"stale"`, compID, &models.PatchManageGuestBody{Phone: strp("000-0000-0009")})
	expectErr(t, err, services.ErrPreconditionFailed)
	after, err := e.manage.PatchGuest(ctx, token, `"`+sum.Version+`"`, compID,
		&models.PatchManageGuestBody{Phone: strp("000-0000-0009"), Email: strp("test-c@example.test")})
	if err != nil {
		t.Fatal(err)
	}
	if after.Version == sum.Version {
		t.Fatal("version must change after an edit")
	}
	_, err = e.manage.PatchGuest(ctx, token, sum.Version, compID, &models.PatchManageGuestBody{Phone: strp("1")})
	expectErr(t, err, services.ErrPreconditionFailed)

	// Registration allowlist: planned >= registered, <= facility max.
	_, err = e.manage.PatchRegistration(ctx, token, after.Version, &models.PatchManageRegistrationBody{PlannedGuestCount: intp(1)})
	expectErr(t, err, services.ErrValidation)
	_, err = e.manage.PatchRegistration(ctx, token, after.Version, &models.PatchManageRegistrationBody{PlannedGuestCount: intp(5)})
	expectErr(t, err, services.ErrCapacityExceeded)
	upd, err := e.manage.PatchRegistration(ctx, token, after.Version, &models.PatchManageRegistrationBody{
		PlannedGuestCount: intp(4), EstimatedArrivalTime: strp("16:30"), BookingReference: strp(""),
	})
	if err != nil {
		t.Fatal(err)
	}
	if upd.PlannedGuestCount != 4 || upd.RemainingCount != 2 || upd.BookingReference != nil ||
		upd.EstimatedArrivalTime == nil || *upd.EstimatedArrivalTime != "16:30" {
		t.Fatalf("patched summary %+v", upd)
	}

	// Manage output never carries passport numbers beyond last4.
	js, _ := json.Marshal(upd)
	s := string(js)
	if strings.Contains(s, "TESTPP4321") || strings.Contains(s, `"passport_number`) || !strings.Contains(s, `"passport_last4":"4321"`) {
		t.Fatalf("manage JSON exposes passport data: %s", s)
	}
	if strings.Contains(s, "Test Guest R") || strings.Contains(s, "Test Town") {
		t.Fatalf("manage JSON exposes unmasked PII: %s", s)
	}

	// Audit rows hold field names only.
	var audits []entities.AuditLog
	e.db.Where("action = ?", "registration.updated").Order("id").Find(&audits)
	if len(audits) != 2 || *audits[0].ChangedFieldsJSON != `["guest.email","guest.phone"]` ||
		*audits[1].ChangedFieldsJSON != `["estimated_arrival_time","booking_reference","planned_guest_count"]` {
		for _, a := range audits {
			t.Logf("audit %s %v", a.Action, *a.ChangedFieldsJSON)
		}
		t.Fatal("unexpected audit rows")
	}
	if strings.Contains(*audits[0].ChangedFieldsJSON, "example.test") {
		t.Fatal("audit stores values")
	}
}

// Lock order (deadlock 1213): a guest PATCH (token → registration before the
// fix) racing an admin reissue (registration → tokens) must never fail with
// anything but the expected 401/412 outcomes.
func TestIntegrationConcurrentManagePatchAndReissue(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	scope := e.adminScope(t)
	res := e.mustCreate(t, newKey(), sakuBody(2))
	token := *res.Body.EditToken
	sum, err := e.manage.AddCompanions(ctx, token, []models.CompanionInput{testutil.Companion("Test Companion")})
	if err != nil {
		t.Fatal(err)
	}
	companion := sum.Guests[1].PublicID

	for round := range 15 {
		sum, err := e.manage.Summary(ctx, token)
		if err != nil {
			t.Fatalf("round %d summary: %v", round, err)
		}
		var wg sync.WaitGroup
		start := make(chan struct{})
		guestErrs := make([]error, 4)
		for i := range guestErrs {
			wg.Add(1)
			go func() {
				defer wg.Done()
				<-start
				phone := "090-0000-000" + string(rune('0'+i))
				_, guestErrs[i] = e.manage.PatchGuest(ctx, token, `"`+sum.Version+`"`, companion,
					&models.PatchManageGuestBody{Phone: &phone})
			}()
		}
		var reissued *models.ReissueEditLinkResult
		var adminErr error
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			reissued, adminErr = e.admin.ReissueEditLink(ctx, scope, res.Body.PublicID)
		}()
		close(start)
		wg.Wait()

		if adminErr != nil {
			t.Fatalf("round %d reissue: %v", round, adminErr)
		}
		for i, err := range guestErrs {
			switch {
			case err == nil,
				errors.Is(err, services.ErrTokenInvalid),
				errors.Is(err, services.ErrPreconditionFailed):
			default:
				t.Fatalf("round %d guest %d: unexpected error (would be 500): %v", round, i, err)
			}
		}
		token = reissued.EditToken
	}
}
