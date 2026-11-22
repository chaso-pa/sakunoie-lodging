package services_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"gorm.io/gorm"
)

func (e *env) invitation(t *testing.T, publicID string) entities.RegistrationInvitation {
	t.Helper()
	var inv entities.RegistrationInvitation
	if err := e.db.Where("public_id = ?", publicID).Take(&inv).Error; err != nil {
		t.Fatal(err)
	}
	return inv
}

func (e *env) issue(t *testing.T, scope services.AdminScope, in services.AdminInvitationInput) *models.CreateAdminInvitationResult {
	t.Helper()
	res, err := e.invite.Issue(context.Background(), scope, in)
	if err != nil {
		t.Fatalf("issue: %v", err)
	}
	return res
}

func sakuInvite(t *testing.T, e *env) services.AdminInvitationInput {
	t.Helper()
	return services.AdminInvitationInput{
		FacilityID: testutil.FacilityID(t, e.db, testutil.FacilitySaku), CheckInDate: "2026-12-01", CheckOutDate: "2026-12-03",
	}
}

// Issue returns the raw token once, stores only its SHA-256, computes the
// expiry from the check-out day in facility time and audits without the
// token, the note or the booking code.
func TestIntegrationInvitationIssue(t *testing.T) {
	e := setup(t)
	scope := e.adminScope(t)
	in := sakuInvite(t, e)
	in.BookingReference, in.Note = strp("  HM-TEST-01 "), strp("Test Guest Name")
	res := e.issue(t, scope, in)

	raw, err := base64.RawURLEncoding.DecodeString(res.InvitationToken)
	if err != nil || len(raw) != 32 {
		t.Fatalf("token is not 32 base64url bytes: %v", err)
	}
	if res.InvitationURLPath != "/stays/saku-no-ie/registration" || strings.Contains(res.InvitationURLPath, res.InvitationToken) {
		t.Fatalf("url path = %q", res.InvitationURLPath)
	}
	inv := e.invitation(t, res.Invitation.PublicID)
	sum := sha256.Sum256(raw)
	if !slices.Equal(inv.TokenHash, sum[:]) {
		t.Fatal("token_hash is not the SHA-256 of the raw token")
	}
	if *inv.BookingReference != "HM-TEST-01" || *inv.Note != "Test Guest Name" || inv.CreatedBy != scope.AdminUserID {
		t.Fatalf("row = %+v", inv)
	}
	if got := inv.ExpiresAt.UTC().Format(time.RFC3339Nano); got != "2026-12-03T14:59:59.999999Z" {
		t.Fatalf("expires_at = %s (end of 2026-12-03 in Asia/Tokyo)", got)
	}
	if res.Invitation.Status != models.InvitationActive || res.Invitation.RegistrationPublicID != nil {
		t.Fatalf("invitation = %+v", res.Invitation)
	}

	audits := e.auditRows(t, services.AuditInvitationCreated)
	if len(audits) != 1 {
		t.Fatalf("audit rows = %d", len(audits))
	}
	a := audits[0]
	if a.ActorType != entities.ActorAdmin || a.ActorAdminUserID == nil || *a.ActorAdminUserID != scope.AdminUserID ||
		a.RegistrationID != nil || a.ChangedFieldsJSON == nil ||
		*a.ChangedFieldsJSON != `{"invitation_public_id":"`+inv.PublicID+`"}` {
		t.Fatalf("audit = %+v %v", a, a.ChangedFieldsJSON)
	}
	// The raw token is nowhere in the database.
	for _, q := range []string{
		"SELECT COUNT(*) FROM audit_logs WHERE INSTR(IFNULL(changed_fields_json,''), ?) > 0",
		"SELECT COUNT(*) FROM registration_invitations WHERE INSTR(token_hash, ?) > 0 OR INSTR(IFNULL(note,''), ?) > 0",
	} {
		var n int64
		args := []any{res.InvitationToken}
		if strings.Count(q, "?") == 2 {
			args = append(args, res.InvitationToken)
		}
		e.db.Raw(q, args...).Scan(&n)
		if n != 0 {
			t.Fatalf("raw token stored: %s", q)
		}
	}
}

func TestIntegrationInvitationIssueValidation(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	scope := e.adminScope(t)
	saku := testutil.FacilityID(t, e.db, testutil.FacilitySaku)
	hono := testutil.FacilityID(t, e.db, testutil.FacilityHonolulu)

	for name, c := range map[string]struct {
		in, out, loc string
	}{
		"same day":  {"2026-12-01", "2026-12-01", "body.check_out_date"},
		"reversed":  {"2026-12-03", "2026-12-01", "body.check_out_date"},
		"past":      {"2026-11-10", "2026-11-19", "body.check_out_date"},
		"malformed": {"2026-13-01", "2026-12-03", "body.check_in_date"},
	} {
		_, err := e.invite.Issue(ctx, scope, services.AdminInvitationInput{FacilityID: saku, CheckInDate: c.in, CheckOutDate: c.out})
		expectErr(t, err, services.ErrValidation)
		if f := services.FieldsOf(err); len(f) != 1 || f[0].Location != c.loc {
			t.Errorf("%s: fields = %+v", name, f)
		}
	}
	// Check-out today in facility time is allowed (2026-11-20 in Tokyo at baseNow).
	e.issue(t, scope, services.AdminInvitationInput{FacilityID: saku, CheckInDate: "2026-11-19", CheckOutDate: "2026-11-20"})

	_, err := e.invite.Issue(ctx, scope, services.AdminInvitationInput{FacilityID: hono, CheckInDate: "2026-12-01", CheckOutDate: "2026-12-03"})
	expectErr(t, err, services.ErrForbidden)
	_, err = e.invite.Issue(ctx, services.AdminScope{}, sakuInvite(t, e))
	expectErr(t, err, services.ErrNotAuthorized)

	if err := e.db.Model(&entities.Facility{}).Where("id = ?", saku).Update("is_active", false).Error; err != nil {
		t.Fatal(err)
	}
	_, err = e.invite.Issue(ctx, scope, sakuInvite(t, e))
	expectErr(t, err, services.ErrFacilityUnavailable)
	if n := len(e.auditRows(t, services.AuditInvitationCreated)); n != 1 {
		t.Fatalf("refused issues wrote audit rows: %d", n)
	}
}

// The derived status, the status filter, pagination totals and facility
// isolation of the list.
func TestIntegrationInvitationListAndScope(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	scope := e.adminScope(t)
	hono := testutil.FacilityID(t, e.db, testutil.FacilityHonolulu)

	active := e.issue(t, scope, sakuInvite(t, e))
	revoked := e.issue(t, scope, sakuInvite(t, e))
	used := e.issue(t, scope, sakuInvite(t, e))
	soon := e.issue(t, scope, services.AdminInvitationInput{
		FacilityID: testutil.FacilityID(t, e.db, testutil.FacilitySaku), CheckInDate: "2026-11-19", CheckOutDate: "2026-11-20",
	})
	e.issueInvitation(t, testutil.FacilityHonolulu, "2026-12-01", "2026-12-03")

	if _, err := e.invite.Revoke(ctx, scope, revoked.Invitation.PublicID); err != nil {
		t.Fatal(err)
	}
	reg, err := e.reg.Create(ctx, newKey(), used.InvitationToken, sakuBody(1))
	if err != nil {
		t.Fatal(err)
	}
	// End of 2026-11-20 in Tokyo has passed for `soon`.
	e.clock.Set(time.Date(2026, 11, 20, 15, 0, 0, 0, time.UTC))

	list, err := e.invite.List(ctx, scope, services.InvitationListFilter{})
	if err != nil {
		t.Fatal(err)
	}
	if list.Total != 4 || len(list.Items) != 4 || list.PageSize != services.InvitationPageSize {
		t.Fatalf("list leaked another facility or miscounted: total=%d items=%d", list.Total, len(list.Items))
	}
	status := map[string]string{}
	for _, it := range list.Items {
		status[it.PublicID] = it.Status
		if it.Facility.Code != testutil.FacilitySaku {
			t.Fatalf("item of another facility: %+v", it)
		}
		if it.PublicID == used.Invitation.PublicID &&
			(it.RegistrationPublicID == nil || *it.RegistrationPublicID != reg.Body.PublicID || it.UsedAt == nil) {
			t.Fatalf("used item = %+v", it)
		}
	}
	want := map[string]string{
		active.Invitation.PublicID:  models.InvitationActive,
		revoked.Invitation.PublicID: models.InvitationRevoked,
		used.Invitation.PublicID:    models.InvitationUsed,
		soon.Invitation.PublicID:    models.InvitationExpired,
	}
	for id, s := range want {
		if status[id] != s {
			t.Errorf("%s: status %q, want %q", id, status[id], s)
		}
	}
	// Newest first (equal created_at under the fixed clock: higher id first).
	if list.Items[0].PublicID != soon.Invitation.PublicID || list.Items[3].PublicID != active.Invitation.PublicID {
		t.Fatalf("order: first = %s", list.Items[0].PublicID)
	}
	for s, id := range map[string]string{
		models.InvitationActive: active.Invitation.PublicID, models.InvitationRevoked: revoked.Invitation.PublicID,
		models.InvitationUsed: used.Invitation.PublicID, models.InvitationExpired: soon.Invitation.PublicID,
	} {
		l, err := e.invite.List(ctx, scope, services.InvitationListFilter{Status: s})
		if err != nil || l.Total != 1 || l.Items[0].PublicID != id {
			t.Errorf("status %s: %+v %v", s, l, err)
		}
	}
	_, err = e.invite.List(ctx, scope, services.InvitationListFilter{Status: "bogus"})
	expectErr(t, err, services.ErrValidation)
	_, err = e.invite.List(ctx, scope, services.InvitationListFilter{FacilityID: hono})
	expectErr(t, err, services.ErrForbidden)
	_, err = e.invite.List(ctx, services.AdminScope{}, services.InvitationListFilter{})
	expectErr(t, err, services.ErrNotAuthorized)

	// The response never carries a token or hash.
	js, _ := json.Marshal(list)
	for _, needle := range []string{"token", "hash", active.InvitationToken} {
		if strings.Contains(string(js), needle) {
			t.Fatalf("list JSON contains %q", needle)
		}
	}

	// Another facility's invitation cannot be revoked (404, never 403).
	var otherInv entities.RegistrationInvitation
	e.db.Where("facility_id = ?", hono).Take(&otherInv)
	_, err = e.invite.Revoke(ctx, scope, otherInv.PublicID)
	expectErr(t, err, services.ErrNotFound)
	if e.invitation(t, otherInv.PublicID).RevokedAt != nil {
		t.Fatal("another facility's invitation was revoked")
	}
}

func TestIntegrationInvitationRevoke(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	scope := e.adminScope(t)
	res := e.issue(t, scope, sakuInvite(t, e))

	got, err := e.invite.Revoke(ctx, scope, res.Invitation.PublicID)
	if err != nil || got.Status != models.InvitationRevoked || got.RevokedAt == nil {
		t.Fatalf("revoke: %+v %v", got, err)
	}
	// Idempotent: no second audit row, first revocation kept.
	again, err := e.invite.Revoke(ctx, scope, res.Invitation.PublicID)
	if err != nil || !again.RevokedAt.Equal(*got.RevokedAt) {
		t.Fatalf("second revoke: %+v %v", again, err)
	}
	audits := e.auditRows(t, services.AuditInvitationRevoked)
	if len(audits) != 1 || audits[0].ActorType != entities.ActorAdmin || *audits[0].ActorAdminUserID != scope.AdminUserID ||
		*audits[0].ChangedFieldsJSON != `{"invitation_public_id":"`+res.Invitation.PublicID+`"}` {
		t.Fatalf("revoke audit = %+v", audits)
	}

	// A revoked invitation is invalid for the guest (401, like a revoked edit token).
	_, err = e.invite.Current(ctx, res.InvitationToken)
	expectErr(t, err, services.ErrInvitationInvalid)
	_, err = e.reg.Create(ctx, newKey(), res.InvitationToken, sakuBody(1))
	expectErr(t, err, services.ErrInvitationInvalid)

	// A used invitation is consumed: it cannot be revoked.
	used := e.issue(t, scope, sakuInvite(t, e))
	if _, err := e.reg.Create(ctx, newKey(), used.InvitationToken, sakuBody(1)); err != nil {
		t.Fatal(err)
	}
	_, err = e.invite.Revoke(ctx, scope, used.Invitation.PublicID)
	expectErr(t, err, services.ErrInvitationUsed)
	if e.invitation(t, used.Invitation.PublicID).RevokedAt != nil {
		t.Fatal("a used invitation was revoked")
	}
	_, err = e.invite.Revoke(ctx, scope, "00000000-0000-4000-8000-000000000000")
	expectErr(t, err, services.ErrNotFound)
}

// Expiry is the last microsecond of the check-out day in facility time.
func TestIntegrationInvitationExpiryAcrossTimezones(t *testing.T) {
	for _, c := range []struct {
		facility string
		expiry   time.Time
	}{
		{testutil.FacilitySaku, time.Date(2026, 12, 3, 14, 59, 59, 999999000, time.UTC)},    // Asia/Tokyo
		{testutil.FacilityHonolulu, time.Date(2026, 12, 4, 9, 59, 59, 999999000, time.UTC)}, // Pacific/Honolulu
	} {
		t.Run(c.facility, func(t *testing.T) {
			e := setup(t)
			ctx := context.Background()
			body := testutil.CreateBody(c.facility, "2026-12-01", "2026-12-03", 1)
			tok := e.issueInvitation(t, c.facility, "2026-12-01", "2026-12-03")
			tok2 := e.issueInvitation(t, c.facility, "2026-12-01", "2026-12-03")

			e.clock.Set(c.expiry)
			cur, err := e.invite.Current(ctx, tok)
			if err != nil || !cur.ExpiresAt.Equal(c.expiry) {
				t.Fatalf("at expiry: %+v %v", cur, err)
			}
			e.clock.Set(c.expiry.Add(time.Microsecond))
			_, err = e.invite.Current(ctx, tok)
			expectErr(t, err, services.ErrInvitationExpired)
			_, err = e.reg.Create(ctx, newKey(), tok, body)
			expectErr(t, err, services.ErrInvitationExpired)

			// At the last microsecond the create still succeeds.
			e.clock.Set(c.expiry)
			if _, err := e.reg.Create(ctx, newKey(), tok2, body); err != nil {
				t.Fatalf("create at expiry: %v", err)
			}
		})
	}
}

// One invitation creates at most one registration; afterwards the guest
// uses the edit link.
func TestIntegrationInvitationSingleUse(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	tok := e.issueInvitation(t, testutil.FacilitySaku, "2026-12-01", "2026-12-03")

	cur, err := e.invite.Current(ctx, tok)
	if err != nil || cur.FacilityCode != testutil.FacilitySaku || cur.CheckInDate != "2026-12-01" || cur.CheckOutDate != "2026-12-03" {
		t.Fatalf("current: %+v %v", cur, err)
	}
	res, err := e.reg.Create(ctx, newKey(), tok, sakuBody(2))
	if err != nil {
		t.Fatal(err)
	}
	var inv entities.RegistrationInvitation
	e.db.Take(&inv)
	reg := e.registration(t, res.Body.PublicID)
	if inv.UsedAt == nil || inv.RegistrationID == nil || *inv.RegistrationID != reg.ID {
		t.Fatalf("invitation not consumed: %+v", inv)
	}
	_, err = e.invite.Current(ctx, tok)
	expectErr(t, err, services.ErrInvitationUsed)
	_, err = e.reg.Create(ctx, newKey(), tok, sakuBody(2))
	expectErr(t, err, services.ErrInvitationUsed)
	if n := testutil.CountRows(t, e.db, "guest_registrations"); n != 1 {
		t.Fatalf("registrations = %d", n)
	}
	// The edit link of the registration keeps working.
	if _, err := e.manage.Summary(ctx, *res.Body.EditToken); err != nil {
		t.Fatal(err)
	}
}

// Two different Idempotency-Keys with one invitation at the same time:
// exactly one registration.
func TestIntegrationInvitationConcurrentDoubleSubmit(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	tok := e.issueInvitation(t, testutil.FacilitySaku, "2026-12-01", "2026-12-03")
	// Hold the first transaction open so the second blocks on the invitation lock.
	e.reg.BeforeCommit = func(*gorm.DB) error { time.Sleep(200 * time.Millisecond); return nil }

	var wg sync.WaitGroup
	start := make(chan struct{})
	errs := make([]error, 2)
	results := make([]*services.CreateResult, 2)
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results[i], errs[i] = e.reg.Create(ctx, newKey(), tok, sakuBody(1))
		}()
	}
	close(start)
	wg.Wait()

	ok, used := 0, 0
	for i := range 2 {
		switch {
		case errs[i] == nil:
			ok++
			if results[i].Replayed {
				t.Fatal("a different key must not replay")
			}
		case strings.Contains(errs[i].Error(), services.ErrInvitationUsed.Error()):
			used++
		default:
			t.Fatalf("request %d: %v", i, errs[i])
		}
	}
	if ok != 1 || used != 1 {
		t.Fatalf("ok=%d used=%d", ok, used)
	}
	if n := testutil.CountRows(t, e.db, "guest_registrations"); n != 1 {
		t.Fatalf("registrations = %d", n)
	}
	if n := testutil.CountRows(t, e.db, "api_idempotency_keys"); n != 1 {
		t.Fatalf("key rows = %d", n)
	}
}

// The host sets the dates and the facility; the guest cannot change them.
func TestIntegrationInvitationDatesAndFacilityMismatch(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	tok := e.issueInvitation(t, testutil.FacilitySaku, "2026-12-01", "2026-12-03")

	_, err := e.reg.Create(ctx, newKey(), tok, testutil.CreateBody(testutil.FacilitySaku, "2026-12-01", "2026-12-04", 1))
	expectErr(t, err, services.ErrInvitationDatesMismatch)
	if f := services.FieldsOf(err); len(f) != 1 || f[0].Location != "body.check_out_date" {
		t.Fatalf("fields = %+v", f)
	}
	_, err = e.reg.Create(ctx, newKey(), tok, testutil.CreateBody(testutil.FacilitySaku, "2026-12-02", "2026-12-04", 1))
	expectErr(t, err, services.ErrInvitationDatesMismatch)
	if f := services.FieldsOf(err); len(f) != 2 {
		t.Fatalf("fields = %+v", f)
	}
	_, err = e.reg.Create(ctx, newKey(), tok, testutil.CreateBody(testutil.FacilityHonolulu, "2026-12-01", "2026-12-03", 1))
	expectErr(t, err, services.ErrInvitationInvalid)

	// Nothing was written and the invitation is still usable.
	if n := testutil.CountRows(t, e.db, "guest_registrations") + testutil.CountRows(t, e.db, "api_idempotency_keys"); n != 0 {
		t.Fatalf("rows written: %d", n)
	}
	if _, err := e.reg.Create(ctx, newKey(), tok, sakuBody(1)); err != nil {
		t.Fatal(err)
	}
}

// Decision D with invitations: the same key and body by the same invitation
// replays after the invitation is used; another key is 409 invitation_used;
// the same key with another invitation is 409 idempotency_conflict.
func TestIntegrationInvitationIdempotentReplay(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	tok := e.issueInvitation(t, testutil.FacilitySaku, "2026-12-01", "2026-12-03")
	key := newKey()

	first, err := e.reg.Create(ctx, key, tok, sakuBody(2))
	if err != nil {
		t.Fatal(err)
	}
	second, err := e.reg.Create(ctx, key, tok, sakuBody(2))
	if err != nil {
		t.Fatalf("same-key replay: %v", err)
	}
	if !second.Replayed || second.Body.PublicID != first.Body.PublicID || second.Body.EditToken == nil {
		t.Fatalf("replay = %+v", second)
	}

	_, err = e.reg.Create(ctx, newKey(), tok, sakuBody(2))
	expectErr(t, err, services.ErrInvitationUsed)
	_, err = e.reg.Create(ctx, key, tok, sakuBody(3))
	expectErr(t, err, services.ErrIdempotencyConflict)

	other := e.issueInvitation(t, testutil.FacilitySaku, "2026-12-01", "2026-12-03")
	_, err = e.reg.Create(ctx, key, other, sakuBody(2))
	expectErr(t, err, services.ErrIdempotencyConflict)
	var unused int64
	e.db.Model(&entities.RegistrationInvitation{}).Where("used_at IS NULL").Count(&unused)
	if unused != 1 {
		t.Fatalf("the other invitation was consumed (unused = %d)", unused)
	}
	if n := testutil.CountRows(t, e.db, "guest_registrations"); n != 1 {
		t.Fatalf("registrations = %d", n)
	}

	// A replay after the check-out day has ended is refused with the expiry.
	exp, _ := utils.EditTokenExpiry(utils.CivilDate{Year: 2026, Month: 12, Day: 3}, "Asia/Tokyo")
	e.clock.Set(exp.Add(time.Microsecond))
	_, err = e.reg.Create(ctx, key, tok, sakuBody(2))
	expectErr(t, err, services.ErrInvitationExpired)
}

func TestIntegrationInvitationRequired(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	for name, c := range map[string]struct {
		tok  string
		want error
	}{
		"missing":   {"", services.ErrInvitationRequired},
		"blank":     {"   ", services.ErrInvitationRequired},
		"malformed": {"not-a-token", services.ErrInvitationInvalid},
		"unknown":   {base64.RawURLEncoding.EncodeToString(make([]byte, 32)), services.ErrInvitationInvalid},
	} {
		_, err := e.reg.Create(ctx, newKey(), c.tok, sakuBody(1))
		if err == nil || !strings.Contains(err.Error(), c.want.Error()) {
			t.Errorf("%s create: %v", name, err)
		}
		_, err = e.invite.Current(ctx, c.tok)
		if err == nil || !strings.Contains(err.Error(), c.want.Error()) {
			t.Errorf("%s current: %v", name, err)
		}
	}
	for _, table := range []string{"guest_registrations", "api_idempotency_keys", "audit_logs"} {
		if n := testutil.CountRows(t, e.db, table); n != 0 {
			t.Errorf("%s has %d rows", table, n)
		}
	}
}
