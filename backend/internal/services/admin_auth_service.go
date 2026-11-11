package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"strings"

	firebase "firebase.google.com/go/v4"
	"firebase.google.com/go/v4/auth"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/entities"
	"github.com/chaso-pa/sakunoie-lodging/backend/internal/utils"
	"google.golang.org/api/option"
	"gorm.io/gorm"
)

// RoleOwner is the only admin role the MVP honours. Rows with any other role
// grant nothing (plan Phase 4; staff is a follow-up).
const RoleOwner = "owner"

// TokenVerifier verifies a Firebase ID token and returns its uid.
type TokenVerifier interface {
	VerifyIDToken(ctx context.Context, idToken string) (uid string, err error)
}

type firebaseVerifier struct{ client *auth.Client }

// NewFirebaseVerifier builds a verifier on firebase-admin-go
// auth.Client.VerifyIDToken (plan Decision A), which checks the RS256
// signature against Google's public certs, iss, aud, exp/iat/auth_time and a
// non-empty sub.
//
// option.WithoutAuthentication: verification needs only the public certs,
// and without it app.Auth() fails when no Application Default Credentials
// exist (Phase 4 spike, CLAUDE.md §13). The library reads
// FIREBASE_AUTH_EMULATOR_HOST itself; in that mode it skips the signature
// check and asks the emulator whether the user exists and is enabled. That
// mode is refused for non-demo projects here and in internal/config.
func NewFirebaseVerifier(ctx context.Context, projectID string) (TokenVerifier, error) {
	if projectID == "" {
		return nil, errors.New("admin auth: FIREBASE_PROJECT_ID is required")
	}
	if os.Getenv("FIREBASE_AUTH_EMULATOR_HOST") != "" && !strings.HasPrefix(projectID, "demo-") {
		return nil, errors.New("admin auth: refusing emulator mode for a non-demo project")
	}
	app, err := firebase.NewApp(ctx, &firebase.Config{ProjectID: projectID}, option.WithoutAuthentication())
	if err != nil {
		return nil, fmt.Errorf("admin auth: firebase app: %w", err)
	}
	client, err := app.Auth(ctx)
	if err != nil {
		return nil, fmt.Errorf("admin auth: firebase auth client: %w", err)
	}
	return &firebaseVerifier{client: client}, nil
}

func (v *firebaseVerifier) VerifyIDToken(ctx context.Context, idToken string) (string, error) {
	tok, err := v.client.VerifyIDToken(ctx, idToken)
	if err != nil {
		return "", err
	}
	return tok.UID, nil
}

// AdminScope is the server-side authorization of one admin request: the
// facilities the admin owns. Every admin service method takes it, and every
// admin query is restricted to FacilityIDs (contract §4.8).
type AdminScope struct {
	AdminUserID uint64
	FacilityIDs []uint64
}

// Has reports whether the scope includes facilityID.
func (s AdminScope) Has(facilityID uint64) bool { return slices.Contains(s.FacilityIDs, facilityID) }

// AdminAuthService turns a bearer ID token into an AdminScope.
type AdminAuthService struct {
	db       *gorm.DB
	verifier TokenVerifier
}

func NewAdminAuthService(db *gorm.DB, verifier TokenVerifier) *AdminAuthService {
	return &AdminAuthService{db: db, verifier: verifier}
}

// Authenticate verifies the ID token (ErrUnauthenticated on any failure),
// then maps uid → admin_users → owner rows of admin_facility_access. An
// unknown uid or an admin without facilities is ErrNotAuthorized; there is
// no auto-provisioning.
func (s *AdminAuthService) Authenticate(ctx context.Context, idToken string) (*AdminScope, error) {
	if idToken == "" {
		return nil, ErrUnauthenticated
	}
	uid, err := s.verifier.VerifyIDToken(ctx, idToken)
	if err != nil || uid == "" {
		// The token and the library error text are never logged.
		slog.WarnContext(ctx, "admin token rejected",
			"request_id", utils.RequestID(ctx), "reason", tokenRejectReason(err))
		return nil, ErrUnauthenticated
	}
	db := s.db.WithContext(ctx)
	var user entities.AdminUser
	err = db.Where("firebase_uid = ?", uid).Take(&user).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrNotAuthorized
	}
	if err != nil {
		return nil, err
	}
	var ids []uint64
	if err := db.Model(&entities.AdminFacilityAccess{}).
		Where("admin_user_id = ? AND role = ?", user.ID, RoleOwner).
		Order("facility_id").Pluck("facility_id", &ids).Error; err != nil {
		return nil, err
	}
	if len(ids) == 0 {
		return nil, ErrNotAuthorized
	}
	return &AdminScope{AdminUserID: user.ID, FacilityIDs: ids}, nil
}

func tokenRejectReason(err error) string {
	switch {
	case err == nil:
		return "empty_uid"
	case auth.IsIDTokenExpired(err):
		return "expired"
	case auth.IsIDTokenRevoked(err):
		return "revoked"
	case auth.IsUserDisabled(err):
		return "user_disabled"
	case auth.IsUserNotFound(err):
		return "user_not_found"
	case auth.IsIDTokenInvalid(err):
		return "invalid"
	}
	return "other"
}
