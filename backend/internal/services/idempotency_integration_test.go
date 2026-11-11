package services_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
	"gorm.io/gorm"
)

func TestIntegrationIdempotentReplayIssuesAdditionalToken(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	key := newKey()
	first := e.mustCreate(t, key, sakuBody(2))
	second := e.mustCreate(t, key, sakuBody(2))

	if !second.Replayed || first.Replayed {
		t.Fatal("replay flag")
	}
	if second.Body.PublicID != first.Body.PublicID {
		t.Fatal("replay created a different registration")
	}
	if second.Body.EditToken == nil || *second.Body.EditToken == *first.Body.EditToken {
		t.Fatal("replay must mint an additional token")
	}
	if second.Body.RegisteredCount != 1 || second.Body.PlannedGuestCount != 2 || second.Body.Code != "" {
		t.Fatalf("replay body %+v", second.Body)
	}
	for _, tok := range []string{*first.Body.EditToken, *second.Body.EditToken} {
		if _, err := e.manage.Summary(ctx, tok); err != nil {
			t.Fatalf("token not valid: %v", err)
		}
	}
	if n := testutil.CountRows(t, e.db, "guest_registrations"); n != 1 {
		t.Fatalf("registrations = %d", n)
	}

	changed := sakuBody(3)
	_, err := e.reg.Create(ctx, key, changed)
	expectErr(t, err, services.ErrIdempotencyConflict)
}

func TestIntegrationIdempotencyDoubleSubmitRace(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	// Hold Tx1 open so the second request blocks on the key_hash unique
	// index, then gets 1062 and replays in a new transaction.
	e.reg.BeforeCommit = func(*gorm.DB) error { time.Sleep(300 * time.Millisecond); return nil }

	key := newKey()
	var wg sync.WaitGroup
	start := make(chan struct{})
	results := make([]*services.CreateResult, 2)
	errs := make([]error, 2)
	for i := range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results[i], errs[i] = e.reg.Create(ctx, key, sakuBody(2))
		}()
	}
	close(start)
	wg.Wait()

	for i := range 2 {
		if errs[i] != nil {
			t.Fatalf("request %d: %v", i, errs[i])
		}
		if results[i].Body.EditToken == nil {
			t.Fatalf("request %d has no token", i)
		}
		if _, err := e.manage.Summary(ctx, *results[i].Body.EditToken); err != nil {
			t.Fatalf("token %d invalid: %v", i, err)
		}
	}
	if results[0].Body.PublicID != results[1].Body.PublicID {
		t.Fatal("two registrations for one key")
	}
	if results[0].Replayed == results[1].Replayed {
		t.Fatal("exactly one response must be a replay")
	}
	if n := testutil.CountRows(t, e.db, "guest_registrations"); n != 1 {
		t.Fatalf("registrations = %d", n)
	}
}

func TestIntegrationIdempotencyTokenCap(t *testing.T) {
	e := setup(t)
	key := newKey()
	first := e.mustCreate(t, key, sakuBody(2))
	e.mustCreate(t, key, sakuBody(2))
	e.mustCreate(t, key, sakuBody(2))
	reg := e.registration(t, first.Body.PublicID)
	if n := e.activeTokens(t, reg.ID); n != 3 {
		t.Fatalf("active tokens = %d", n)
	}
	capped := e.mustCreate(t, key, sakuBody(2))
	if !capped.Replayed || capped.Body.EditToken != nil || capped.Body.EditTokenExpiresAt != nil ||
		capped.Body.Code != models.EditLinkUnavailable {
		t.Fatalf("capped replay %+v", capped.Body)
	}
	if n := e.activeTokens(t, reg.ID); n != 3 {
		t.Fatalf("active tokens after cap = %d", n)
	}
}

func TestIntegrationIdempotencyAfterReissue(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	key := newKey()
	first := e.mustCreate(t, key, sakuBody(2))
	replayToken := e.mustCreate(t, key, sakuBody(2)).Body.EditToken

	hostToken := e.reissue(t, first.Body.PublicID)
	// Reissue revokes all prior tokens, including the replay-issued one.
	for _, tok := range []string{*first.Body.EditToken, *replayToken} {
		_, err := e.manage.Summary(ctx, tok)
		expectErr(t, err, services.ErrTokenInvalid)
	}
	if _, err := e.manage.Summary(ctx, hostToken); err != nil {
		t.Fatal(err)
	}

	before := testutil.CountRows(t, e.db, "registration_access_tokens")
	res := e.mustCreate(t, key, sakuBody(2))
	if res.Body.EditToken != nil || res.Body.Code != models.EditLinkUnavailable {
		t.Fatalf("replay after reissue minted a token: %+v", res.Body)
	}
	if after := testutil.CountRows(t, e.db, "registration_access_tokens"); after != before {
		t.Fatalf("token rows %d -> %d", before, after)
	}
}

func TestIntegrationIdempotencyExpiredKeyReuse(t *testing.T) {
	e := setup(t)
	key := newKey()
	first := e.mustCreate(t, key, sakuBody(2))
	e.clock.Advance(services.IdempotencyTTL + time.Second)

	second := e.mustCreate(t, key, sakuBody(2))
	if second.Replayed || second.Body.PublicID == first.Body.PublicID {
		t.Fatal("expired key must create a new registration")
	}
	if n := testutil.CountRows(t, e.db, "api_idempotency_keys"); n != 1 {
		t.Fatalf("key rows = %d (must be overwritten)", n)
	}
	var row entities.APIIdempotencyKey
	e.db.Take(&row)
	if row.ResponseJSON == nil || *row.ResponseJSON != `{"registration_public_id":"`+second.Body.PublicID+`"}` {
		t.Fatalf("response_json = %v", row.ResponseJSON)
	}
	if !row.ExpiresAt.After(e.clock.Now()) {
		t.Fatal("expires_at not renewed")
	}
	// A different body is accepted on an expired key too (it is a new key).
	e.clock.Advance(services.IdempotencyTTL + time.Second)
	if res := e.mustCreate(t, key, sakuBody(3)); res.Replayed {
		t.Fatal("expected a new registration")
	}
}

func TestIntegrationIdempotencyJanitor(t *testing.T) {
	e := setup(t)
	ctx := context.Background()
	e.mustCreate(t, newKey(), sakuBody(1))
	e.mustCreate(t, newKey(), sakuBody(1))
	e.clock.Advance(services.IdempotencyTTL + time.Minute)
	keep := e.mustCreate(t, newKey(), sakuBody(1))

	n, err := e.idem.DeleteExpired(ctx)
	if err != nil || n != 2 {
		t.Fatalf("deleted %d, %v", n, err)
	}
	if left := testutil.CountRows(t, e.db, "api_idempotency_keys"); left != 1 {
		t.Fatalf("rows left = %d", left)
	}
	if testutil.CountRows(t, e.db, "guest_registrations") != 3 || keep.Body.PublicID == "" {
		t.Fatal("janitor must not touch registrations")
	}
}
