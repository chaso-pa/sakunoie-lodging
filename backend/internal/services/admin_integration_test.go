package services_test

import (
	"bytes"
	"context"
	"encoding/csv"
	"encoding/json"
	"errors"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
)

const adminUID = "test-admin-uid-a"

func boolp(b bool) *bool { return &b }

// adminScope grants the owner role on saku-no-ie only and returns the scope
// the auth service builds for it.
func (e *env) adminScope(t *testing.T, codes ...string) services.AdminScope {
	t.Helper()
	if len(codes) == 0 {
		codes = []string{testutil.FacilitySaku}
	}
	testutil.GrantAdmin(t, e.db, adminUID, services.RoleOwner, codes...)
	auth := services.NewAdminAuthService(e.db, testutil.FakeVerifier{"tok-a": adminUID})
	scope, err := auth.Authenticate(context.Background(), "tok-a")
	if err != nil {
		t.Fatal(err)
	}
	return *scope
}

func honoluluBody(guests ...models.GuestInput) *models.CreateGuestRegistrationBody {
	return testutil.CreateBody(testutil.FacilityHonolulu, "2026-12-01", "2026-12-03", 1, guests...)
}

func (e *env) auditRows(t *testing.T, action string) []entities.AuditLog {
	t.Helper()
	var rows []entities.AuditLog
	if err := e.db.Where("action = ?", action).Order("id").Find(&rows).Error; err != nil {
		t.Fatal(err)
	}
	return rows
}

func auditFields(t *testing.T, row entities.AuditLog) []string {
	t.Helper()
	var fields []string
	if row.ChangedFieldsJSON == nil {
		return nil
	}
	if err := json.Unmarshal([]byte(*row.ChangedFieldsJSON), &fields); err != nil {
		t.Fatalf("changed_fields_json is not a field-name array: %s", *row.ChangedFieldsJSON)
	}
	return fields
}

func TestIntegrationAdminAuthenticate(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	auth := services.NewAdminAuthService(e.db, testutil.FakeVerifier{
		"tok-owner": "uid-owner", "tok-staff": "uid-staff", "tok-unknown": "uid-unknown",
	})

	for name, tok := range map[string]string{"empty": "", "invalid": "forged"} {
		if _, err := auth.Authenticate(ctx, tok); err != services.ErrUnauthenticated {
			t.Errorf("%s token: got %v, want ErrUnauthenticated", name, err)
		}
	}
	// No admin_users row: no auto-provisioning.
	expectErr(t, func() error { _, err := auth.Authenticate(ctx, "tok-unknown"); return err }(), services.ErrNotAuthorized)
	if testutil.CountRows(t, e.db, "admin_users") != 0 {
		t.Fatal("an admin user was auto-provisioned")
	}
	// Only the owner role grants access.
	testutil.GrantAdmin(t, e.db, "uid-staff", "staff", testutil.FacilitySaku)
	expectErr(t, func() error { _, err := auth.Authenticate(ctx, "tok-staff"); return err }(), services.ErrNotAuthorized)

	id := testutil.GrantAdmin(t, e.db, "uid-owner", services.RoleOwner, testutil.FacilitySaku)
	scope, err := auth.Authenticate(ctx, "tok-owner")
	if err != nil {
		t.Fatal(err)
	}
	saku := testutil.FacilityID(t, e.db, testutil.FacilitySaku)
	if scope.AdminUserID != id || !slices.Equal(scope.FacilityIDs, []uint64{saku}) {
		t.Fatalf("scope = %+v", scope)
	}
}

// Contract §4.8: an admin of facility A cannot list, read, change or export
// facility B, whatever the request parameters say.
func TestIntegrationAdminFacilityIsolation(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	scope := e.adminScope(t)
	own := e.mustCreate(t, newKey(), sakuBody(1))
	other := e.mustCreate(t, newKey(), honoluluBody())
	hono := testutil.FacilityID(t, e.db, testutil.FacilityHonolulu)

	facs, err := e.admin.Facilities(ctx, scope)
	if err != nil || len(facs) != 1 || facs[0].Code != testutil.FacilitySaku {
		t.Fatalf("facilities = %+v, %v", facs, err)
	}

	_, err = e.admin.List(ctx, scope, services.AdminListFilter{FacilityID: hono})
	expectErr(t, err, services.ErrForbidden)
	list, err := e.admin.List(ctx, scope, services.AdminListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].PublicID != own.Body.PublicID {
		t.Fatalf("list leaked another facility: %+v", list)
	}

	_, err = e.admin.Detail(ctx, scope, other.Body.PublicID, false)
	expectErr(t, err, services.ErrNotFound)
	_, err = e.admin.Patch(ctx, scope, other.Body.PublicID, "*", &models.AdminPatchRegistrationBody{BookingReference: strp("X")})
	expectErr(t, err, services.ErrNotFound)
	_, err = e.admin.VerifyIdentity(ctx, scope, other.Body.PublicID)
	expectErr(t, err, services.ErrNotFound)
	_, err = e.admin.ReissueEditLink(ctx, scope, other.Body.PublicID)
	expectErr(t, err, services.ErrNotFound)
	_, err = e.admin.Export(ctx, scope, services.ExportFilter{FacilityID: hono})
	expectErr(t, err, services.ErrForbidden)

	// The other facility's registration is untouched.
	r := e.registration(t, other.Body.PublicID)
	if r.IdentityVerifiedAt != nil || r.BookingReference == nil || *r.BookingReference != "TEST-REF-0001" {
		t.Fatalf("other facility changed: %+v", r)
	}
	if n := testutil.CountRows(t, e.db, "audit_logs"); n != 2 { // the two registration.created rows
		t.Fatalf("audit rows = %d; refused operations must not write audit rows", n)
	}

	// A zero scope (a bug) fails closed.
	_, err = e.admin.List(ctx, services.AdminScope{}, services.AdminListFilter{})
	expectErr(t, err, services.ErrNotAuthorized)
	_, err = e.admin.Detail(ctx, services.AdminScope{}, own.Body.PublicID, false)
	expectErr(t, err, services.ErrNotAuthorized)
}

func TestIntegrationAdminSoftDeletedInvisible(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	scope := e.adminScope(t)
	res := e.mustCreate(t, newKey(), sakuBody(1))
	if err := e.db.Model(&entities.GuestRegistration{}).Where("public_id = ?", res.Body.PublicID).
		Update("deleted_at", e.clock.Now()).Error; err != nil {
		t.Fatal(err)
	}
	list, err := e.admin.List(ctx, scope, services.AdminListFilter{})
	if err != nil || list.Total != 0 || list.Metrics.Registrations != 0 {
		t.Fatalf("soft-deleted row listed: %+v %v", list, err)
	}
	_, err = e.admin.Detail(ctx, scope, res.Body.PublicID, false)
	expectErr(t, err, services.ErrNotFound)
	out, err := e.admin.Export(ctx, scope, services.ExportFilter{FacilityID: testutil.FacilityID(t, e.db, testutil.FacilitySaku)})
	if err != nil || strings.Contains(string(out.CSV), res.Body.PublicID) {
		t.Fatalf("soft-deleted row exported (%v)", err)
	}
}

func TestIntegrationAdminListFiltersAndMetrics(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	scope := e.adminScope(t, testutil.FacilitySaku, testutil.FacilityHonolulu)

	incomplete := sakuBody(2)
	incomplete.BookingReference = strp("REF-100%_SALE")
	a := e.mustCreate(t, newKey(), incomplete)

	complete := testutil.CreateBody(testutil.FacilitySaku, "2026-12-10", "2026-12-12", 2,
		testutil.Representative(), testutil.ForeignGuest("Test Searchable Companion", "TESTPP2222"))
	complete.BookingReference = strp("REF-200")
	b := e.mustCreate(t, newKey(), complete)
	c := e.mustCreate(t, newKey(), honoluluBody())
	if _, err := e.admin.VerifyIdentity(ctx, scope, c.Body.PublicID); err != nil {
		t.Fatal(err)
	}

	list, err := e.admin.List(ctx, scope, services.AdminListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	want := models.AdminLedgerMetrics{Registrations: 3, Guests: 4, Incomplete: 1, Verified: 1}
	if list.Metrics != want || list.Total != 3 || list.PageSize != services.AdminPageSize {
		t.Fatalf("metrics = %+v total=%d", list.Metrics, list.Total)
	}
	// Newest check-in first.
	if list.Items[0].PublicID != b.Body.PublicID {
		t.Fatalf("order: %s first", list.Items[0].PublicID)
	}
	for _, it := range list.Items {
		if it.PublicID == a.Body.PublicID && (it.IsComplete || it.RegisteredCount != 1 || it.RemainingCount != 1 ||
			it.RepresentativeName != "Test Guest A" || it.Facility.Code != testutil.FacilitySaku) {
			t.Fatalf("row a = %+v", it)
		}
		if it.PublicID == c.Body.PublicID && (!it.IdentityVerified || it.IdentityVerifiedAt == nil) {
			t.Fatalf("row c = %+v", it)
		}
	}

	ids := func(f services.AdminListFilter) []string {
		t.Helper()
		l, err := e.admin.List(ctx, scope, f)
		if err != nil {
			t.Fatal(err)
		}
		var out []string
		for _, it := range l.Items {
			out = append(out, it.PublicID)
		}
		slices.Sort(out)
		return out
	}
	sorted := func(s ...string) []string { slices.Sort(s); return s }

	// Search: a companion's name, and LIKE wildcards are literal.
	if got := ids(services.AdminListFilter{Q: "searchable"}); !slices.Equal(got, sorted(b.Body.PublicID)) {
		t.Errorf("q=searchable → %v", got)
	}
	if got := ids(services.AdminListFilter{Q: "100%_"}); !slices.Equal(got, sorted(a.Body.PublicID)) {
		t.Errorf("q=100%%_ → %v", got)
	}
	if got := ids(services.AdminListFilter{Q: "%"}); !slices.Equal(got, sorted(a.Body.PublicID)) {
		t.Errorf("q=%% must match a literal percent only: %v", got)
	}
	if got := ids(services.AdminListFilter{Q: "_"}); !slices.Equal(got, sorted(a.Body.PublicID)) {
		t.Errorf("q=_ must match a literal underscore only: %v", got)
	}
	// Check-in date range (inclusive).
	if got := ids(services.AdminListFilter{From: "2026-12-10", To: "2026-12-10"}); !slices.Equal(got, sorted(b.Body.PublicID)) {
		t.Errorf("date range → %v", got)
	}
	if got := ids(services.AdminListFilter{FacilityID: testutil.FacilityID(t, e.db, testutil.FacilityHonolulu)}); !slices.Equal(got, sorted(c.Body.PublicID)) {
		t.Errorf("facility filter → %v", got)
	}
	_, err = e.admin.List(ctx, scope, services.AdminListFilter{From: "2026-12-10", To: "2026-12-01"})
	expectErr(t, err, services.ErrValidation)
	// Past the last page: empty items, same metrics.
	l2, err := e.admin.List(ctx, scope, services.AdminListFilter{Page: 2})
	if err != nil || len(l2.Items) != 0 || l2.Total != 3 {
		t.Fatalf("page 2 = %+v %v", l2, err)
	}
}

func TestIntegrationAdminDetailMaskingAndReveal(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	scope := e.adminScope(t)
	res := e.mustCreate(t, newKey(), sakuBody(2, testutil.Representative(), testutil.ForeignGuest("Test Foreign Guest", "TESTPP1234")))

	d, err := e.admin.Detail(ctx, scope, res.Body.PublicID, false)
	if err != nil {
		t.Fatal(err)
	}
	rep, foreign := d.Guests[0], d.Guests[1]
	if !rep.IsRepresentative || rep.FullName != "Test Guest A" || rep.Address != "1-2***" ||
		*rep.Phone != "****01" || *rep.Email != "t***@example.test" || d.Revealed {
		t.Fatalf("masked detail = %+v", rep)
	}
	if foreign.PassportLast4 == nil || *foreign.PassportLast4 != "1234" {
		t.Fatalf("passport last4 = %v", foreign.PassportLast4)
	}
	if d.ActiveEditLinks != 1 || d.EditLinkExpiresAt == nil || d.Version == "" || d.CapacitySnapshot != 4 {
		t.Fatalf("detail meta = %+v", d)
	}
	if n := len(e.auditRows(t, services.AuditAdminRevealed)); n != 0 {
		t.Fatal("masked read must not be audited as a reveal")
	}

	d, err = e.admin.Detail(ctx, scope, res.Body.PublicID, true)
	if err != nil {
		t.Fatal(err)
	}
	if d.Guests[0].Address != "1-2-3 Test Town, Saku, Nagano" || *d.Guests[0].Phone != "000-0000-0001" || !d.Revealed {
		t.Fatalf("revealed detail = %+v", d.Guests[0])
	}
	js, _ := json.Marshal(d)
	if strings.Contains(string(js), "TESTPP1234") || strings.Contains(string(js), "ciphertext") {
		t.Fatal("detail exposes the passport number")
	}
	rows := e.auditRows(t, services.AuditAdminRevealed)
	if len(rows) != 1 || rows[0].ActorType != entities.ActorAdmin || rows[0].ActorAdminUserID == nil ||
		*rows[0].ActorAdminUserID != scope.AdminUserID {
		t.Fatalf("reveal audit = %+v", rows)
	}
}

func TestIntegrationAdminPatch(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	scope := e.adminScope(t)
	res := e.mustCreate(t, newKey(), sakuBody(3, testutil.Representative(), testutil.ForeignGuest("Test Foreign Guest", "TESTPP1234")))
	d, err := e.admin.Detail(ctx, scope, res.Body.PublicID, false)
	if err != nil {
		t.Fatal(err)
	}
	repID, foreignID := d.Guests[0].PublicID, d.Guests[1].PublicID

	// Stale If-Match → 412.
	_, err = e.admin.Patch(ctx, scope, res.Body.PublicID, `"r0"`, &models.AdminPatchRegistrationBody{BookingReference: strp("X")})
	expectErr(t, err, services.ErrPreconditionFailed)

	// Validation failures change nothing.
	for name, body := range map[string]*models.AdminPatchRegistrationBody{
		"dates":          {CheckOutDate: strp("2026-12-01")},
		"below_count":    {PlannedGuestCount: intp(1)},
		"unknown_guest":  {Guests: []models.AdminPatchGuest{{PublicID: "00000000-0000-4000-8000-000000000000", Phone: strp("1")}}},
		"rep_false":      {Guests: []models.AdminPatchGuest{{PublicID: repID, IsRepresentative: boolp(false)}}},
		"no_passport":    {Guests: []models.AdminPatchGuest{{PublicID: repID, HasJapanAddress: boolp(false), Nationality: strp("Testland")}}},
		"no_nationality": {Guests: []models.AdminPatchGuest{{PublicID: foreignID, Nationality: strp("")}}},
		"bad_email":      {Guests: []models.AdminPatchGuest{{PublicID: foreignID, Email: strp("not-an-email")}}},
		"bad_phone":      {Guests: []models.AdminPatchGuest{{PublicID: foreignID, Phone: strp("call me")}}},
	} {
		_, err := e.admin.Patch(ctx, scope, res.Body.PublicID, d.Version, body)
		if !errors.Is(err, services.ErrValidation) {
			t.Errorf("%s: got %v, want validation error", name, err)
		}
	}
	// Above the current facility max (4) needs capacity_override.
	_, err = e.admin.Patch(ctx, scope, res.Body.PublicID, d.Version, &models.AdminPatchRegistrationBody{PlannedGuestCount: intp(5)})
	expectErr(t, err, services.ErrCapacityExceeded)
	if n := len(e.auditRows(t, services.AuditAdminUpdated)); n != 0 {
		t.Fatalf("failed patches wrote %d audit rows", n)
	}

	// A real correction: dates, count with override, booking ref, phone,
	// passport re-encryption and a representative change.
	d2, err := e.admin.Patch(ctx, scope, res.Body.PublicID, `"`+d.Version+`"`, &models.AdminPatchRegistrationBody{
		CheckOutDate:      strp("2026-12-04"),
		PlannedGuestCount: intp(5),
		CapacityOverride:  true,
		BookingReference:  strp("TEST-REF-CHANGED"),
		Guests: []models.AdminPatchGuest{
			{PublicID: foreignID, PassportNumber: strp("testpp 5678"), Phone: strp("000-1111-2222"), IsRepresentative: boolp(true)},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if d2.CheckOutDate != "2026-12-04" || d2.PlannedGuestCount != 5 || d2.CapacitySnapshot != 4 ||
		*d2.BookingReference != "TEST-REF-CHANGED" || d2.Version == d.Version {
		t.Fatalf("patched = %+v", d2)
	}
	var guests []entities.Guest
	e.db.Where("registration_id = ?", e.registration(t, res.Body.PublicID).ID).Order("id").Find(&guests)
	reps := 0
	for _, g := range guests {
		if g.IsRepresentative {
			reps++
			if g.PublicID != foreignID {
				t.Fatal("representative not moved")
			}
		}
		if g.PublicID == foreignID {
			pt, err := testutil.PassportCipher().Decrypt(g.PassportNumberCiphertext, []byte(g.PublicID))
			if err != nil || pt != "TESTPP5678" || *g.PassportNumberLast4 != "5678" {
				t.Fatalf("passport not re-encrypted: %q %v", pt, err)
			}
		}
	}
	if reps != 1 {
		t.Fatalf("representatives = %d", reps)
	}

	rows := e.auditRows(t, services.AuditAdminUpdated)
	if len(rows) != 1 || rows[0].ActorType != entities.ActorAdmin || *rows[0].ActorAdminUserID != scope.AdminUserID {
		t.Fatalf("audit = %+v", rows)
	}
	got := auditFields(t, rows[0])
	wantFields := []string{"booking_reference", "capacity_override", "check_out_date", "guest.is_representative",
		"guest.passport_number", "guest.phone", "planned_guest_count"}
	if !slices.Equal(got, wantFields) {
		t.Fatalf("audit fields = %v, want %v", got, wantFields)
	}
	for _, v := range []string{"TESTPP5678", "5678", "TEST-REF-CHANGED", "000-1111-2222", "2026-12-04"} {
		if strings.Contains(*rows[0].ChangedFieldsJSON, v) {
			t.Fatalf("audit row contains a value: %s", *rows[0].ChangedFieldsJSON)
		}
	}

	// has_japan_address=true discards the stored passport (data minimization).
	_, err = e.admin.Patch(ctx, scope, res.Body.PublicID, d2.Version, &models.AdminPatchRegistrationBody{
		Guests: []models.AdminPatchGuest{{PublicID: foreignID, HasJapanAddress: boolp(true)}},
	})
	if err != nil {
		t.Fatal(err)
	}
	var g entities.Guest
	e.db.Where("public_id = ?", foreignID).Take(&g)
	if g.PassportNumberCiphertext != nil || g.PassportNumberLast4 != nil || !g.HasJapanAddress {
		t.Fatalf("passport kept after has_japan_address=true: %+v", g.PassportNumberLast4)
	}
}

func TestIntegrationAdminVerifyIdentity(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	scope := e.adminScope(t)
	res := e.mustCreate(t, newKey(), sakuBody(1))

	d, err := e.admin.VerifyIdentity(ctx, scope, res.Body.PublicID)
	if err != nil || !d.IdentityVerified || d.IdentityVerifiedAt == nil {
		t.Fatalf("verify = %+v %v", d, err)
	}
	r := e.registration(t, res.Body.PublicID)
	if r.IdentityVerifiedBy == nil || *r.IdentityVerifiedBy != scope.AdminUserID || !r.IdentityVerifiedAt.Equal(e.clock.Now()) {
		t.Fatalf("row = %+v", r)
	}
	first := *r.IdentityVerifiedAt
	e.clock.Advance(time.Hour)
	if _, err := e.admin.VerifyIdentity(ctx, scope, res.Body.PublicID); err != nil {
		t.Fatal(err)
	}
	if r := e.registration(t, res.Body.PublicID); !r.IdentityVerifiedAt.Equal(first) {
		t.Fatal("re-verification overwrote the first verification")
	}
	rows := e.auditRows(t, services.AuditIdentityVerified)
	if len(rows) != 1 || rows[0].ActorType != entities.ActorAdmin || rows[0].RegistrationID == nil || *rows[0].RegistrationID != r.ID {
		t.Fatalf("verify audit = %+v", rows)
	}
}

// Reissue revokes every earlier token, including tokens minted by
// idempotent replays, and the new one works.
func TestIntegrationAdminReissueRevokesAllTokens(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	scope := e.adminScope(t)
	key, body := newKey(), sakuBody(1)
	first := e.mustCreate(t, key, body)
	replay := e.mustCreate(t, key, body)
	if replay.Body.EditToken == nil || *replay.Body.EditToken == *first.Body.EditToken {
		t.Fatal("replay did not mint an additional token")
	}
	regID := e.registration(t, first.Body.PublicID).ID
	if n := e.activeTokens(t, regID); n != 2 {
		t.Fatalf("active tokens before reissue = %d", n)
	}

	out, err := e.admin.ReissueEditLink(ctx, scope, first.Body.PublicID)
	if err != nil {
		t.Fatal(err)
	}
	if out.EditToken == "" || out.Expired || out.EditTokenExpiresAt.IsZero() {
		t.Fatalf("reissue = %+v", out)
	}
	if n := e.activeTokens(t, regID); n != 1 {
		t.Fatalf("active tokens after reissue = %d", n)
	}
	for _, old := range []string{*first.Body.EditToken, *replay.Body.EditToken} {
		_, err := e.manage.Summary(ctx, old)
		expectErr(t, err, services.ErrTokenInvalid)
	}
	if _, err := e.manage.Summary(ctx, out.EditToken); err != nil {
		t.Fatalf("new token: %v", err)
	}
	rows := e.auditRows(t, services.AuditEditLinkReissued)
	if len(rows) != 1 || rows[0].ChangedFieldsJSON != nil || rows[0].ActorType != entities.ActorAdmin {
		t.Fatalf("reissue audit = %+v", rows)
	}

	// A checkout day that has ended still reissues, flagged as expired.
	e.clock.Advance(30 * 24 * time.Hour)
	out, err = e.admin.ReissueEditLink(ctx, scope, first.Body.PublicID)
	if err != nil || !out.Expired {
		t.Fatalf("late reissue = %+v %v", out, err)
	}
}

func TestIntegrationAdminExportCSV(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	scope := e.adminScope(t)
	saku := testutil.FacilityID(t, e.db, testutil.FacilitySaku)

	injected := sakuBody(2, testutil.Representative(), testutil.ForeignGuest("@Test Formula Guest", "TESTPP1234"))
	injected.BookingReference = strp("=SUM(1)")
	a := e.mustCreate(t, newKey(), injected)
	later := testutil.CreateBody(testutil.FacilitySaku, "2026-12-20", "2026-12-22", 1)
	later.BookingReference = strp("+cmd|' /C calc'!A0")
	b := e.mustCreate(t, newKey(), later)
	if _, err := e.admin.VerifyIdentity(ctx, scope, a.Body.PublicID); err != nil {
		t.Fatal(err)
	}

	out, err := e.admin.Export(ctx, scope, services.ExportFilter{FacilityID: saku, From: "2026-12-01", To: "2026-12-01"})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(out.CSV, []byte("\xEF\xBB\xBF")) {
		t.Fatal("missing UTF-8 BOM")
	}
	if out.Filename != "registrations-saku-no-ie-2026-12-01-2026-12-01.csv" {
		t.Fatalf("filename = %s", out.Filename)
	}
	recs, err := csv.NewReader(bytes.NewReader(out.CSV[3:])).ReadAll()
	if err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(recs[0], services.ExportColumns) || len(recs) != 3 {
		t.Fatalf("csv = %q", recs)
	}
	col := func(name string) int { return slices.Index(services.ExportColumns, name) }
	rep, guest := recs[1], recs[2]
	if rep[col("booking_reference")] != "'=SUM(1)" || rep[col("registration_public_id")] != a.Body.PublicID ||
		rep[col("is_representative")] != "true" || rep[col("guest_index")] != "1" || rep[col("identity_verified_at")] == "" {
		t.Fatalf("rep row = %q", rep)
	}
	if guest[col("full_name")] != "'@Test Formula Guest" || guest[col("passport_last4")] != "1234" ||
		guest[col("has_japan_address")] != "false" || guest[col("registered_count")] != "2" {
		t.Fatalf("guest row = %q", guest)
	}
	if bytes.Contains(out.CSV, []byte("TESTPP1234")) || bytes.Contains(out.CSV, []byte(b.Body.PublicID)) {
		t.Fatal("CSV contains a full passport number or a row outside the date range")
	}

	// Without a date filter both registrations are exported.
	all, err := e.admin.Export(ctx, scope, services.ExportFilter{FacilityID: saku})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(all.CSV, []byte(`'+cmd|' /C calc'!A0`)) || all.Filename != "registrations-saku-no-ie-start-end.csv" {
		t.Fatalf("unfiltered export:\n%s", all.CSV)
	}

	rows := e.auditRows(t, services.AuditRegistrationsExpo)
	if len(rows) != 2 || rows[0].RegistrationID != nil || rows[0].FacilityID != saku || *rows[0].ActorAdminUserID != scope.AdminUserID {
		t.Fatalf("export audit = %+v", rows)
	}
	var meta map[string]any
	if err := json.Unmarshal([]byte(*rows[0].ChangedFieldsJSON), &meta); err != nil {
		t.Fatal(err)
	}
	if meta["from"] != "2026-12-01" || meta["to"] != "2026-12-01" || meta["row_count"] != float64(2) ||
		meta["registration_count"] != float64(1) || meta["facility_id"] != float64(saku) {
		t.Fatalf("export audit meta = %v", meta)
	}
	for _, v := range []string{"Test", "SUM", "1234", a.Body.PublicID} {
		if strings.Contains(*rows[0].ChangedFieldsJSON, v) {
			t.Fatalf("export audit contains content: %s", *rows[0].ChangedFieldsJSON)
		}
	}
}
