package middlewares

import (
	"context"
	"net/http"
	"time"

	"github.com/alexedwards/scs/v2"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/logging"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/models"
	"github.com/chaso-pa/sakunoie-lodging-backend/internal/session"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type contextKey string

const MemberIDKey contextKey = "member_id"

// RequireAuth rejects requests without a valid member session.
//
// It compares the session's session_version with the member row on every
// request (one SELECT per authenticated request, accepted at MVP scale), so
// restoring the account on another device with the transfer code invalidates
// this session on its next request.
func RequireAuth(sessionManager *scs.SessionManager, db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		memberID := sessionManager.GetString(ctx, session.KeyMemberID)
		if memberID == "" {
			abortWithProblem(c, http.StatusUnauthorized, "authentication required")
			return
		}

		sessionVersion := sessionManager.GetInt(ctx, session.KeySessionVersion)
		var member models.Member
		if err := db.WithContext(ctx).Select("session_version").Where("id = ?", memberID).First(&member).Error; err != nil {
			_ = sessionManager.Destroy(ctx)
			abortWithProblem(c, http.StatusUnauthorized, "authentication required")
			return
		}
		if member.SessionVersion != sessionVersion {
			_ = sessionManager.Destroy(ctx)
			abortWithProblem(c, http.StatusUnauthorized, "session expired")
			return
		}

		// Sliding window: every authenticated request extends the session.
		sessionManager.SetDeadline(ctx, time.Now().Add(sessionManager.Lifetime).UTC())

		// c.Set() only writes Gin's Keys, which Huma handlers cannot see. Put
		// the member id on the request context instead.
		newCtx := logging.WithMemberID(context.WithValue(ctx, MemberIDKey, memberID), memberID)
		c.Request = c.Request.WithContext(newCtx)
		c.Next()
	}
}

// GetMemberID returns the authenticated member id from a request context, or
// "" when RequireAuth has not run. Huma handlers call it with their ctx.
func GetMemberID(ctx context.Context) string {
	if v, ok := ctx.Value(MemberIDKey).(string); ok {
		return v
	}
	return ""
}

// abortWithProblem ends the request with a body shaped like Huma's errors.
func abortWithProblem(c *gin.Context, status int, detail string) {
	c.AbortWithStatusJSON(status, gin.H{
		"title":  http.StatusText(status),
		"status": status,
		"detail": detail,
	})
}
