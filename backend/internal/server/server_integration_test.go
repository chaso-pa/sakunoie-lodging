package server_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"net/textproto"
	"regexp"
	"strings"
	"testing"

	"github.com/chaso-pa/sakunoie-lodging-backend/internal/config"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/routes"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/server"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/session"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/storage"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/testutil"
	"github.com/gin-gonic/gin"
)

const inviteCode = "SAKU-26-ABCD"

type app struct {
	handler http.Handler
	db      *testutil.DB
}

// newApp builds the whole application on an empty database, with fresh rate
// limiters. Tests in this package must not run in parallel: they share the
// database name.
func newApp(t *testing.T, images storage.ImageUploader) *app {
	t.Helper()
	return newAppWith(t, &routes.Deps{Images: images})
}

// newAppWith is newApp with the caller's Config, External and Logger. DB
// and Sessions are always the test database's.
func newAppWith(t *testing.T, deps *routes.Deps) *app {
	t.Helper()
	gin.SetMode(gin.TestMode)
	testDB := testutil.NewDB(t, "server")
	sm := session.New(testDB.Sessions, false)
	deps.DB = testDB.Gorm
	deps.Sessions = sm
	if deps.Config == nil {
		deps.Config = &config.Config{}
	}
	deps.Config.InviteCode = inviteCode
	if deps.Images == nil {
		deps.Images = storage.NotConfigured{}
	}
	if deps.Logger == nil {
		deps.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	}
	r, err := server.NewRouter(deps)
	if err != nil {
		t.Fatal(err)
	}
	// Wrapped exactly as cmd/server does.
	return &app{handler: sm.LoadAndSave(r), db: testDB}
}

// client is one browser: it keeps the session cookie between requests.
type client struct {
	t          *testing.T
	app        *app
	token      string
	remoteAddr string
	forwarded  string
}

func (a *app) client(t *testing.T) *client {
	return &client{t: t, app: a, remoteAddr: "127.0.0.1:40000"}
}

type response struct {
	*httptest.ResponseRecorder
}

func (r response) json(t *testing.T) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(r.Body.Bytes(), &v); err != nil {
		t.Fatalf("response is not JSON: %s", r.Body)
	}
	return v
}

func (c *client) send(method, path, contentType string, body io.Reader) response {
	c.t.Helper()
	req := httptest.NewRequest(method, path, body)
	req.RemoteAddr = c.remoteAddr
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	if c.forwarded != "" {
		req.Header.Set("X-Forwarded-For", c.forwarded)
	}
	if c.token != "" {
		req.AddCookie(&http.Cookie{Name: session.CookieName, Value: c.token})
	}
	w := httptest.NewRecorder()
	c.app.handler.ServeHTTP(w, req)
	for _, ck := range w.Result().Cookies() {
		if ck.Name != session.CookieName {
			continue
		}
		if ck.MaxAge < 0 {
			c.token = ""
		} else {
			c.token = ck.Value
		}
	}
	return response{w}
}

func (c *client) get(path string) response {
	return c.send(http.MethodGet, path, "", nil)
}

func (c *client) postJSON(path string, body any) response {
	c.t.Helper()
	b, err := json.Marshal(body)
	if err != nil {
		c.t.Fatal(err)
	}
	return c.send(http.MethodPost, path, "application/json", bytes.NewReader(b))
}

type formFile struct {
	contentType string
	data        []byte
}

func (c *client) postSignup(fields map[string]string, avatar *formFile) response {
	c.t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	for k, v := range fields {
		mw.WriteField(k, v)
	}
	if avatar != nil {
		h := textproto.MIMEHeader{}
		h.Set("Content-Disposition", `form-data; name="avatar"; filename="avatar.jpg"`)
		h.Set("Content-Type", avatar.contentType)
		part, _ := mw.CreatePart(h)
		part.Write(avatar.data)
	}
	mw.Close()
	return c.send(http.MethodPost, "/api/auth/signup", mw.FormDataContentType(), &buf)
}

func (c *client) verifyInvite(code string) response {
	return c.postJSON("/api/auth/verify-invite", map[string]string{"code": code})
}

// signup runs verify-invite and signup, and returns the member id and the
// transfer code.
func (c *client) signup(nickname string) (memberID, transferCode string) {
	c.t.Helper()
	if r := c.verifyInvite(inviteCode); r.Code != http.StatusOK {
		c.t.Fatalf("verify-invite = %d %s", r.Code, r.Body)
	}
	r := c.postSignup(map[string]string{"nickname": nickname}, nil)
	if r.Code != http.StatusOK {
		c.t.Fatalf("signup = %d %s", r.Code, r.Body)
	}
	body := r.json(c.t)
	return body["member"].(map[string]any)["id"].(string), body["transfer_code"].(string)
}

func wantStatus(t *testing.T, r response, want int) {
	t.Helper()
	if r.Code != want {
		t.Fatalf("status = %d, want %d; body: %s", r.Code, want, r.Body)
	}
}

func (a *app) sessionExists(t *testing.T, token string) bool {
	t.Helper()
	var n int
	if err := a.db.Sessions.QueryRow("SELECT COUNT(*) FROM sessions WHERE token = ?", token).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n > 0
}

func TestHealthIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})
	r := a.client(t).get("/health")
	wantStatus(t, r, http.StatusOK)
	if strings.TrimSpace(r.Body.String()) != `{"status":"ok"}` {
		t.Fatalf("body = %s", r.Body)
	}
}

// verify-invite → signup → me → logout → 401.
func TestAuthFlowIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})
	c := a.client(t)

	wantStatus(t, c.get("/api/auth/me"), http.StatusUnauthorized)

	r := c.verifyInvite("saku-26-abcd") // case-insensitive
	wantStatus(t, r, http.StatusOK)
	if r.json(t)["valid"] != true {
		t.Fatalf("body = %s", r.Body)
	}
	inviteToken := c.token
	if inviteToken == "" {
		t.Fatal("verify-invite did not start a session")
	}
	// The invite flag alone is not a login.
	wantStatus(t, c.get("/api/auth/me"), http.StatusUnauthorized)

	r = c.postSignup(map[string]string{"nickname": "旅するMika", "language": "en"}, nil)
	wantStatus(t, r, http.StatusOK)
	body := r.json(t)
	member := body["member"].(map[string]any)
	if member["nickname"] != "旅するMika" || member["preferred_language"] != "en" || member["avatar_url"] != "" || member["id"] == "" {
		t.Fatalf("member = %v", member)
	}
	code, _ := body["transfer_code"].(string)
	if !regexp.MustCompile(`^[A-Z0-9]{4}-[A-Z0-9]{4}-[A-Z0-9]{4}$`).MatchString(code) {
		t.Fatalf("transfer_code = %q", code)
	}

	// Session fixation: signup issues a new token and drops the old session.
	if c.token == inviteToken {
		t.Fatal("session token was not renewed by signup")
	}
	if a.sessionExists(t, inviteToken) {
		t.Fatal("the pre-signup session still exists")
	}

	r = c.get("/api/auth/me")
	wantStatus(t, r, http.StatusOK)
	if got := r.json(t)["member"].(map[string]any)["id"]; got != member["id"] {
		t.Fatalf("me id = %v, want %v", got, member["id"])
	}

	// invite_verified is consumed: the same session cannot sign up again.
	wantStatus(t, c.postSignup(map[string]string{"nickname": "Second"}, nil), http.StatusForbidden)

	loggedInToken := c.token
	r = c.postJSON("/api/auth/logout", nil)
	wantStatus(t, r, http.StatusOK)
	if r.json(t)["ok"] != true {
		t.Fatalf("body = %s", r.Body)
	}
	if c.token != "" {
		t.Fatal("logout did not clear the cookie")
	}
	wantStatus(t, c.get("/api/auth/me"), http.StatusUnauthorized)

	// Replaying the old cookie after logout does not work either.
	c.token = loggedInToken
	wantStatus(t, c.get("/api/auth/me"), http.StatusUnauthorized)
}

func TestWrongInviteCodeIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})
	c := a.client(t)

	r := c.verifyInvite("SAKU-26-XXXX")
	wantStatus(t, r, http.StatusUnauthorized)
	if r.json(t)["detail"] != "invalid invite code" {
		t.Fatalf("body = %s", r.Body)
	}
	wantStatus(t, c.postSignup(map[string]string{"nickname": "Mika"}, nil), http.StatusForbidden)
}

// Invite bypass: signup without verify-invite is 403 and creates nobody.
func TestSignupWithoutInviteIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})
	var before, after int64
	a.db.Gorm.Model(&models.Member{}).Count(&before)

	r := a.client(t).postSignup(map[string]string{"nickname": "Intruder"}, nil)
	wantStatus(t, r, http.StatusForbidden)
	if r.json(t)["detail"] != "invite code not verified" {
		t.Fatalf("body = %s", r.Body)
	}

	a.db.Gorm.Model(&models.Member{}).Count(&after)
	if after != before {
		t.Fatalf("member count went from %d to %d", before, after)
	}
}

func TestSignupValidationIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})
	c := a.client(t)
	wantStatus(t, c.verifyInvite(inviteCode), http.StatusOK)

	wantStatus(t, c.postSignup(map[string]string{"nickname": strings.Repeat("あ", 31)}, nil), http.StatusUnprocessableEntity)
	wantStatus(t, c.postSignup(map[string]string{"nickname": "   "}, nil), http.StatusUnprocessableEntity)
	wantStatus(t, c.postSignup(map[string]string{"nickname": "Mika", "language": "fr"}, nil), http.StatusUnprocessableEntity)
	wantStatus(t, c.postSignup(map[string]string{}, nil), http.StatusUnprocessableEntity)
	// A rejected attempt keeps the invite flag, so a corrected one succeeds.
	wantStatus(t, c.postSignup(map[string]string{"nickname": strings.Repeat("あ", 30)}, nil), http.StatusOK)
}

// Restore on a second device returns the same member and invalidates the
// first device's session through session_version.
func TestTransferRestoreIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})
	phone := a.client(t)
	memberID, code := phone.signup("Mika")
	wantStatus(t, phone.get("/api/auth/me"), http.StatusOK)
	phoneToken := phone.token

	laptop := a.client(t)
	// Give the laptop a session first, to see its token renewed on restore.
	wantStatus(t, laptop.verifyInvite(inviteCode), http.StatusOK)
	beforeRestore := laptop.token

	r := laptop.postJSON("/api/auth/transfer", map[string]string{"transfer_code": "WRONG-CODE-0000"})
	wantStatus(t, r, http.StatusUnauthorized)
	if r.json(t)["detail"] != "invalid transfer code" {
		t.Fatalf("body = %s", r.Body)
	}

	// Lowercase and without hyphens is accepted.
	r = laptop.postJSON("/api/auth/transfer", map[string]string{"transfer_code": strings.ToLower(strings.ReplaceAll(code, "-", ""))})
	wantStatus(t, r, http.StatusOK)
	if got := r.json(t)["member"].(map[string]any)["id"]; got != memberID {
		t.Fatalf("restored member id = %v, want %v", got, memberID)
	}
	if _, leaked := r.json(t)["transfer_code"]; leaked {
		t.Fatal("transfer must not return the transfer code")
	}
	if laptop.token == beforeRestore || a.sessionExists(t, beforeRestore) {
		t.Fatal("session token was not renewed by transfer")
	}

	r = laptop.get("/api/auth/me")
	wantStatus(t, r, http.StatusOK)
	if got := r.json(t)["member"].(map[string]any)["id"]; got != memberID {
		t.Fatalf("me id on the new device = %v, want %v", got, memberID)
	}

	// The old device is logged out on its next request.
	r = phone.get("/api/auth/me")
	wantStatus(t, r, http.StatusUnauthorized)
	if r.json(t)["detail"] != "session expired" {
		t.Fatalf("body = %s", r.Body)
	}
	if a.sessionExists(t, phoneToken) {
		t.Fatal("the old device's session row still exists")
	}

	var member models.Member
	if err := a.db.Gorm.First(&member, "id = ?", memberID).Error; err != nil {
		t.Fatal(err)
	}
	if member.SessionVersion != 2 {
		t.Fatalf("session_version = %d, want 2", member.SessionVersion)
	}
}

func TestVerifyInviteRateLimitIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})

	// Behind the trusted proxy (RemoteAddr 127.0.0.1) the client is whoever
	// X-Forwarded-For names.
	alice := a.client(t)
	alice.forwarded = "203.0.113.10"
	for i := 1; i <= 5; i++ {
		wantStatus(t, alice.verifyInvite("WRONG"), http.StatusUnauthorized)
	}
	r := alice.verifyInvite(inviteCode)
	wantStatus(t, r, http.StatusTooManyRequests)
	if r.json(t)["detail"] != "rate limit exceeded" {
		t.Fatalf("body = %s", r.Body)
	}

	// A different client behind the same proxy has its own allowance.
	bob := a.client(t)
	bob.forwarded = "203.0.113.11"
	wantStatus(t, bob.verifyInvite(inviteCode), http.StatusOK)

	// A forged left-most entry does not help: the right-most address not
	// belonging to a trusted proxy is used.
	alice.forwarded = "10.9.9.9, 203.0.113.10"
	wantStatus(t, alice.verifyInvite(inviteCode), http.StatusTooManyRequests)

	// Reaching Go directly from an untrusted address: the header is ignored,
	// so changing it does not buy a new allowance.
	direct := a.client(t)
	direct.remoteAddr = "198.51.100.20:5000"
	for i := 1; i <= 5; i++ {
		direct.forwarded = "203.0.113." + string(rune('0'+i))
		wantStatus(t, direct.verifyInvite("WRONG"), http.StatusUnauthorized)
	}
	direct.forwarded = "203.0.113.99"
	wantStatus(t, direct.verifyInvite(inviteCode), http.StatusTooManyRequests)

	// The invite limit is not shared with other endpoints.
	wantStatus(t, alice.get("/api/auth/me"), http.StatusUnauthorized)
}

func TestTransferRateLimitIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})
	c := a.client(t)
	c.forwarded = "203.0.113.30"
	wrong := map[string]string{"transfer_code": "AAAA-BBBB-CCCC"}

	for i := 1; i <= 3; i++ {
		wantStatus(t, c.postJSON("/api/auth/transfer", wrong), http.StatusUnauthorized)
	}
	wantStatus(t, c.postJSON("/api/auth/transfer", wrong), http.StatusTooManyRequests)

	// Transfer attempts do not use up the verify-invite allowance.
	wantStatus(t, c.verifyInvite(inviteCode), http.StatusOK)
}

type recordingUploader struct{ got []byte }

func (u *recordingUploader) UploadImage(_ context.Context, kind string, r io.Reader) (string, error) {
	u.got, _ = io.ReadAll(r)
	return "http://storage.test/" + kind + "/avatar.jpg", nil
}

func (u *recordingUploader) DeleteImage(context.Context, string) error { return nil }

func TestSignupWithAvatarIntegration(t *testing.T) {
	avatar := &formFile{contentType: "image/jpeg", data: []byte("\xff\xd8\xff\xe0 not a real jpeg")}

	t.Run("uploader wired", func(t *testing.T) {
		uploader := &recordingUploader{}
		c := newApp(t, uploader).client(t)
		wantStatus(t, c.verifyInvite(inviteCode), http.StatusOK)

		r := c.postSignup(map[string]string{"nickname": "Mika"}, avatar)
		wantStatus(t, r, http.StatusOK)
		if got := r.json(t)["member"].(map[string]any)["avatar_url"]; got != "http://storage.test/avatars/avatar.jpg" {
			t.Fatalf("avatar_url = %v", got)
		}
		if !bytes.Equal(uploader.got, avatar.data) {
			t.Fatal("uploader did not receive the uploaded bytes")
		}
	})

	t.Run("storage not configured", func(t *testing.T) {
		a := newApp(t, storage.NotConfigured{})
		c := a.client(t)
		wantStatus(t, c.verifyInvite(inviteCode), http.StatusOK)
		var before, after int64
		a.db.Gorm.Model(&models.Member{}).Count(&before)

		r := c.postSignup(map[string]string{"nickname": "Mika"}, avatar)
		wantStatus(t, r, http.StatusServiceUnavailable)
		if r.json(t)["detail"] != "image storage is not configured" {
			t.Fatalf("body = %s", r.Body)
		}
		a.db.Gorm.Model(&models.Member{}).Count(&after)
		if after != before {
			t.Fatal("a member was created although the avatar could not be stored")
		}
		// The same session can still sign up without the avatar.
		wantStatus(t, c.postSignup(map[string]string{"nickname": "Mika"}, nil), http.StatusOK)
	})

	t.Run("wrong file type", func(t *testing.T) {
		c := newApp(t, &recordingUploader{}).client(t)
		wantStatus(t, c.verifyInvite(inviteCode), http.StatusOK)
		gif := &formFile{contentType: "image/gif", data: []byte("GIF89a")}
		wantStatus(t, c.postSignup(map[string]string{"nickname": "Mika"}, gif), http.StatusUnprocessableEntity)
	})
}

// An upload over 10MB is cut off by MaxBodySize and answered with 422.
func TestOversizedUploadIntegration(t *testing.T) {
	uploader := &recordingUploader{}
	c := newApp(t, uploader).client(t)
	wantStatus(t, c.verifyInvite(inviteCode), http.StatusOK)

	big := &formFile{contentType: "image/jpeg", data: make([]byte, 11<<20)}
	r := c.postSignup(map[string]string{"nickname": "Mika"}, big)
	wantStatus(t, r, http.StatusUnprocessableEntity)
	if !strings.Contains(r.Body.String(), "request body too large") {
		t.Fatalf("body = %s", r.Body)
	}
	if uploader.got != nil {
		t.Fatal("the oversized file reached the uploader")
	}
}

// JSON endpoints refuse the content types an HTML form can send cross-site.
func TestJSONEndpointsRejectFormContentTypesIntegration(t *testing.T) {
	c := newApp(t, storage.NotConfigured{}).client(t)
	for _, contentType := range []string{"text/plain", "application/x-www-form-urlencoded"} {
		r := c.send(http.MethodPost, "/api/auth/verify-invite", contentType, strings.NewReader(`{"code":"`+inviteCode+`"}`))
		if r.Code != http.StatusUnsupportedMediaType {
			t.Errorf("Content-Type %s = %d, want 415; body: %s", contentType, r.Code, r.Body)
		}
	}
}

func TestOpenAPIDocumentsIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})

	r := a.client(t).get("/api/openapi.json")
	wantStatus(t, r, http.StatusOK)
	for _, path := range []string{`"/auth/verify-invite"`, `"/auth/signup"`, `"/auth/transfer"`, `"/auth/logout"`} {
		if !strings.Contains(r.Body.String(), path) {
			t.Errorf("public OpenAPI is missing %s", path)
		}
	}

	// The auth API's documents sit on the auth group, behind RequireAuth.
	wantStatus(t, a.client(t).get("/api/openapi/auth.json"), http.StatusUnauthorized)
	c := a.client(t)
	c.signup("Mika")
	r = c.get("/api/openapi/auth.json")
	wantStatus(t, r, http.StatusOK)
	if !strings.Contains(r.Body.String(), `"/auth/me"`) {
		t.Error("auth OpenAPI is missing /auth/me")
	}
}

// In the default deployment the Next.js proxy forwards no client address:
// every guest arrives as 127.0.0.1. They share a ceiling of 30 per minute
// instead of each other's 5, so one guest cannot keep the others out.
func TestVerifyInviteSharedAddressIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})
	c := a.client(t) // RemoteAddr 127.0.0.1, no X-Forwarded-For

	for i := 1; i <= 6; i++ {
		wantStatus(t, c.verifyInvite("WRONG"), http.StatusUnauthorized)
	}
	// Another guest behind the same proxy is not locked out by those.
	wantStatus(t, a.client(t).verifyInvite(inviteCode), http.StatusOK)
	for i := 8; i <= 30; i++ {
		wantStatus(t, c.verifyInvite("WRONG"), http.StatusUnauthorized)
	}
	r := c.verifyInvite(inviteCode)
	wantStatus(t, r, http.StatusTooManyRequests)
	// The 429 has the same shape as every other error.
	body := r.json(t)
	if body["title"] != "Too Many Requests" || body["status"] != 429.0 || body["detail"] != "rate limit exceeded" {
		t.Fatalf("429 body = %s", r.Body)
	}
	if ct := r.Header().Get("Content-Type"); !strings.HasPrefix(ct, "application/json") {
		t.Fatalf("429 Content-Type = %q", ct)
	}

	// Next.js dev names its loopback peer ::ffff:127.0.0.1: same bucket.
	mapped := a.client(t)
	mapped.forwarded = "::ffff:127.0.0.1"
	wantStatus(t, mapped.verifyInvite(inviteCode), http.StatusTooManyRequests)

	// A client whose address is forwarded has its own allowance.
	real := a.client(t)
	real.forwarded = "203.0.113.40"
	wantStatus(t, real.verifyInvite(inviteCode), http.StatusOK)
}

func TestTransferSharedAddressIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})
	owner := a.client(t)
	owner.forwarded = "203.0.113.50"
	memberID, code := owner.signup("Mika")
	wrong := map[string]string{"transfer_code": "AAAA-BBBB-CCCC"}

	// Six consecutive failures from the shared address: no lockout, and the
	// per-client limit of 3 does not apply either.
	attacker := a.client(t)
	for i := 1; i <= 6; i++ {
		wantStatus(t, attacker.postJSON("/api/auth/transfer", wrong), http.StatusUnauthorized)
	}
	// Another guest behind the same proxy can still restore.
	guest := a.client(t)
	r := guest.postJSON("/api/auth/transfer", map[string]string{"transfer_code": code})
	wantStatus(t, r, http.StatusOK)
	if got := r.json(t)["member"].(map[string]any)["id"]; got != memberID {
		t.Fatalf("restored member = %v, want %v", got, memberID)
	}

	// The shared ceiling is 30 per minute: 7 used so far.
	for i := 8; i <= 30; i++ {
		wantStatus(t, attacker.postJSON("/api/auth/transfer", wrong), http.StatusUnauthorized)
	}
	r = attacker.postJSON("/api/auth/transfer", wrong)
	wantStatus(t, r, http.StatusTooManyRequests)
	if r.json(t)["detail"] != "rate limit exceeded" {
		t.Fatalf("body = %s", r.Body)
	}
}

// A client that reaches the server with its own address keeps the strict
// behaviour: 5 invite attempts per minute, 3 restores per minute, and a
// lockout after 5 consecutive failed restores.
func TestPerClientLimitsWithPublicAddressIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})

	invite := a.client(t)
	invite.remoteAddr = "198.51.100.30:5000"
	for i := 1; i <= 5; i++ {
		wantStatus(t, invite.verifyInvite("WRONG"), http.StatusUnauthorized)
	}
	wantStatus(t, invite.verifyInvite(inviteCode), http.StatusTooManyRequests)
	// The address is untrusted, so a forged X-Forwarded-For is ignored: it
	// neither buys a new bucket nor moves the client into the shared one.
	for _, forged := range []string{"203.0.113.77", "127.0.0.1", "::ffff:127.0.0.1", "::1"} {
		invite.forwarded = forged
		if r := invite.verifyInvite(inviteCode); r.Code != http.StatusTooManyRequests {
			t.Errorf("X-Forwarded-For %s from a public address = %d, want 429", forged, r.Code)
		}
	}
	// Other addresses, and the shared one, are unaffected.
	other := a.client(t)
	other.remoteAddr = "198.51.100.31:5000"
	wantStatus(t, other.verifyInvite(inviteCode), http.StatusOK)
	wantStatus(t, a.client(t).verifyInvite(inviteCode), http.StatusOK)
}

// Over HTTP only the 3 per minute limit can be reached within one test: the
// lockout needs 5 consecutive failures, which that limit spreads over more
// than a minute. The lockout itself (5 failures from a public address, none
// from the shared one) is tested with a controlled clock in
// internal/middlewares.
func TestTransferLimitWithPublicAddressIntegration(t *testing.T) {
	a := newApp(t, storage.NotConfigured{})
	owner := a.client(t)
	owner.forwarded = "203.0.113.60"
	_, code := owner.signup("Mika")
	wrong := map[string]string{"transfer_code": "AAAA-BBBB-CCCC"}

	c := a.client(t)
	c.remoteAddr = "198.51.100.40:5000"
	for i := 1; i <= 3; i++ {
		wantStatus(t, c.postJSON("/api/auth/transfer", wrong), http.StatusUnauthorized)
	}
	// 4th within the minute: the per-client limiter.
	r := c.postJSON("/api/auth/transfer", wrong)
	wantStatus(t, r, http.StatusTooManyRequests)
	if r.json(t)["detail"] != "rate limit exceeded" {
		t.Fatalf("body = %s", r.Body)
	}
	// A forged header does not move it to another bucket or to the shared one.
	c.forwarded = "127.0.0.1"
	wantStatus(t, c.postJSON("/api/auth/transfer", map[string]string{"transfer_code": code}), http.StatusTooManyRequests)
}
