package session

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/alexedwards/scs/mysqlstore"
	"github.com/alexedwards/scs/v2"
)

const (
	CookieName = "sakunoie_session"
	// Lifetime is the session length. RequireAuth pushes the deadline forward
	// by this much on every authenticated request (sliding window).
	Lifetime = 30 * 24 * time.Hour

	cleanupInterval = 5 * time.Minute
)

// Session data keys.
const (
	KeyMemberID       = "member_id"
	KeySessionVersion = "session_version"
	KeyInviteVerified = "invite_verified"
)

// New builds the session manager backed by the MySQL sessions table, which is
// created by the Prisma migration. db must come from database.OpenSessionDB.
//
// The manager is applied by wrapping the whole Gin engine:
// http.ListenAndServe(addr, sessionManager.LoadAndSave(r)). It is not a Gin
// middleware.
func New(db *sql.DB, secure bool) *scs.SessionManager {
	sm := newManager(secure)
	sm.Store = mysqlstore.NewWithCleanupInterval(db, cleanupInterval)
	return sm
}

// NewInMemory builds a session manager with the same cookie settings on scs's
// default in-memory store, for tests that do not need MySQL.
func NewInMemory() *scs.SessionManager {
	return newManager(false)
}

func newManager(secure bool) *scs.SessionManager {
	sm := scs.New()
	sm.Lifetime = Lifetime
	sm.Cookie.Name = CookieName
	sm.Cookie.HttpOnly = true
	sm.Cookie.Secure = secure
	sm.Cookie.SameSite = http.SameSiteLaxMode
	sm.Cookie.Path = "/api"
	sm.Cookie.Persist = true
	return sm
}
