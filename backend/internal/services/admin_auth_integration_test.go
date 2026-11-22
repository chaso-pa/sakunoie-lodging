package services_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/testutil"
	"github.com/google/uuid"
)

// emulatorUser signs up a synthetic user on the running Auth Emulator and
// returns its uid and ID token.
func emulatorUser(t *testing.T, host string) (uid, idToken string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"email":             "test-" + uuid.NewString()[:8] + "@example.test",
		"password":          "test-password-1",
		"returnSecureToken": true,
	})
	resp, err := http.Post("http://"+host+"/identitytoolkit.googleapis.com/v1/accounts:signUp?key=test-api-key",
		"application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatalf("emulator signUp: %v", err)
	}
	defer resp.Body.Close()
	var out struct{ LocalID, IDToken string }
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil || out.IDToken == "" {
		t.Fatalf("emulator signUp: status %d %v", resp.StatusCode, err)
	}
	return out.LocalID, out.IDToken
}

func deleteEmulatorUser(t *testing.T, host, project, uid string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"localId": uid})
	req, _ := http.NewRequest(http.MethodPost, "http://"+host+"/identitytoolkit.googleapis.com/v1/projects/"+project+"/accounts:delete", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer owner")
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("emulator delete: %v %v", resp, err)
	}
	resp.Body.Close()
}

// Live Auth Emulator: tokens it issues pass VerifyIDToken and map to a
// scope only through admin_facility_access; a token for another audience
// and a deleted user are rejected. Runs in the final verification with the
// emulator up; skipped only when FIREBASE_AUTH_EMULATOR_HOST is unset.
func TestIntegrationAdminAuthEmulator(t *testing.T) {
	host := os.Getenv("FIREBASE_AUTH_EMULATOR_HOST")
	if host == "" {
		t.Skip("SKIP: needs the Auth Emulator (FIREBASE_AUTH_EMULATOR_HOST, project demo-sakunoie)")
	}
	project := os.Getenv("FIREBASE_PROJECT_ID")
	if project == "" {
		project = testProject
	}
	e := setup(t)
	ctx := context.Background()
	verifier, err := services.NewFirebaseVerifier(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	auth := services.NewAdminAuthService(e.db, verifier)

	uid, tok := emulatorUser(t, host)
	_, err = auth.Authenticate(ctx, tok)
	expectErr(t, err, services.ErrNotAuthorized) // valid token, no grant
	id := testutil.GrantAdmin(t, e.db, uid, services.RoleOwner, testutil.FacilitySaku)
	scope, err := auth.Authenticate(ctx, tok)
	if err != nil || scope.AdminUserID != id || len(scope.FacilityIDs) != 1 {
		t.Fatalf("scope = %+v %v", scope, err)
	}

	// Wrong audience: a verifier for demo-other rejects a demo-sakunoie token.
	other, err := services.NewFirebaseVerifier(ctx, "demo-other")
	if err != nil {
		t.Fatal(err)
	}
	_, err = services.NewAdminAuthService(e.db, other).Authenticate(ctx, tok)
	expectErr(t, err, services.ErrUnauthenticated)

	// A deleted user's token is rejected (emulator lookup).
	deleteEmulatorUser(t, host, project, uid)
	_, err = auth.Authenticate(ctx, tok)
	expectErr(t, err, services.ErrUnauthenticated)
}
