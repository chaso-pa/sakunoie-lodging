package services_test

import (
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/chaso-pa/sakunoie-lodging/backend/internal/services"
)

const testProject = "demo-sakunoie"

func b64JSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return base64.RawURLEncoding.EncodeToString(b)
}

func validClaims(project string) map[string]any {
	now := time.Now().Unix()
	return map[string]any{
		"iss":       "https://securetoken.google.com/" + project,
		"aud":       project,
		"sub":       "test-uid-1",
		"user_id":   "test-uid-1",
		"iat":       now - 60,
		"auth_time": now - 60,
		"exp":       now + 3600,
	}
}

// unsignedJWT builds an alg:none token like the Auth Emulator issues.
func unsignedJWT(t *testing.T, claims map[string]any) string {
	return b64JSON(t, map[string]any{"alg": "none", "typ": "JWT"}) + "." + b64JSON(t, claims) + "."
}

// forgedJWT signs valid-looking claims with a locally generated RSA key.
func forgedJWT(t *testing.T, claims map[string]any) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	signing := b64JSON(t, map[string]any{"alg": "RS256", "typ": "JWT", "kid": "forged-test-kid"}) + "." + b64JSON(t, claims)
	sum := sha256.Sum256([]byte(signing))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatal(err)
	}
	return signing + "." + base64.RawURLEncoding.EncodeToString(sig)
}

// stubEmulator answers the emulator's accounts:lookup, which VerifyIDToken
// calls in emulator mode to check that the user exists and is enabled. It
// runs on loopback, so the test needs no network and never skips.
func stubEmulator(t *testing.T) *atomic.Int32 {
	t.Helper()
	var calls atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		if r.URL.Path != "/identitytoolkit.googleapis.com/v1/projects/"+testProject+"/accounts:lookup" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"kind":"identitytoolkit#GetAccountInfoResponse","users":[{"localId":"test-uid-1","disabled":false}]}`))
	}))
	t.Cleanup(srv.Close)
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", srv.Listener.Addr().String())
	return &calls
}

// Offline emulator-mode claim checks (plan §4 Unit,
// admin_auth_emulator_claims): signatures are skipped in emulator mode, but
// iss, aud, exp and sub are still enforced by VerifyIDToken.
func TestAdminAuthEmulatorClaims(t *testing.T) {
	stubEmulator(t)
	v, err := services.NewFirebaseVerifier(context.Background(), testProject)
	if err != nil {
		t.Fatal(err)
	}
	uid, err := v.VerifyIDToken(context.Background(), unsignedJWT(t, validClaims(testProject)))
	if err != nil || uid != "test-uid-1" {
		t.Fatalf("valid emulator token: uid=%q err=%v", uid, err)
	}

	cases := map[string]func(map[string]any){
		"wrong_aud":   func(c map[string]any) { c["aud"] = "demo-other" },
		"wrong_iss":   func(c map[string]any) { c["iss"] = "https://securetoken.google.com/demo-other" },
		"expired":     func(c map[string]any) { c["exp"] = time.Now().Add(-time.Hour).Unix() },
		"empty_sub":   func(c map[string]any) { c["sub"] = ""; c["user_id"] = "" },
		"future_iat":  func(c map[string]any) { c["iat"] = time.Now().Add(time.Hour).Unix() },
		"missing_aud": func(c map[string]any) { delete(c, "aud") },
	}
	for name, mutate := range cases {
		t.Run(name, func(t *testing.T) {
			c := validClaims(testProject)
			mutate(c)
			if uid, err := v.VerifyIDToken(context.Background(), unsignedJWT(t, c)); err == nil {
				t.Fatalf("accepted (uid=%q)", uid)
			}
		})
	}
	if _, err := v.VerifyIDToken(context.Background(), "not-a-jwt"); err == nil {
		t.Fatal("garbage accepted")
	}
}

func TestNewFirebaseVerifierRefusesEmulatorForNonDemoProject(t *testing.T) {
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "127.0.0.1:1")
	if _, err := services.NewFirebaseVerifier(context.Background(), "real-project"); err == nil {
		t.Fatal("emulator mode accepted for a non-demo project")
	}
	if _, err := services.NewFirebaseVerifier(context.Background(), ""); err == nil {
		t.Fatal("empty project accepted")
	}
}

// With the emulator env UNSET, unsigned tokens are rejected outright.
func TestAdminAuthRejectsUnsignedWithoutEmulator(t *testing.T) {
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	v, err := services.NewFirebaseVerifier(context.Background(), testProject)
	if err != nil {
		t.Fatalf("no-ADC verifier must initialize (option.WithoutAuthentication): %v", err)
	}
	if uid, err := v.VerifyIDToken(context.Background(), unsignedJWT(t, validClaims(testProject))); err == nil {
		t.Fatalf("unsigned token accepted (uid=%q)", uid)
	}
}

const googleCertsURL = "https://www.googleapis.com/robot/v1/metadata/x509/securetoken@system.gserviceaccount.com"

// With the emulator env UNSET, a token signed by a key Google never issued
// is rejected by the signature check against Google's public certs. This is
// the only test allowed to skip: it needs outbound HTTPS and skips when the
// cert endpoint is unreachable, so a network failure can never pass for a
// signature rejection.
func TestAdminAuthRejectsForgedSignatureWithoutEmulator(t *testing.T) {
	t.Setenv("FIREBASE_AUTH_EMULATOR_HOST", "")
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(googleCertsURL)
	if err != nil {
		t.Skip("SKIP (offline): Google cert endpoint unreachable: forged-signature test needs network")
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Skipf("SKIP (offline): Google cert endpoint returned %d", resp.StatusCode)
	}
	v, err := services.NewFirebaseVerifier(context.Background(), testProject)
	if err != nil {
		t.Fatal(err)
	}
	if uid, err := v.VerifyIDToken(context.Background(), forgedJWT(t, validClaims(testProject))); err == nil {
		t.Fatalf("forged token accepted (uid=%q)", uid)
	}
}
