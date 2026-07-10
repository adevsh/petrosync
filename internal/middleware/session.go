package middleware

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/adevsh/petrosync/internal/model"
)

// SessionStore is the interface for dashboard session operations.
type SessionStore interface {
	GetSession(ctx context.Context, sessionID string) (*model.SessionData, error)
}

// SessionAuth validates the dashboard session cookie against the store.
func SessionAuth(store SessionStore) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("petrosync_session")
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "not authenticated"},
			})
			return
		}

		session, err := store.GetSession(c.Request.Context(), sessionID)
		if err != nil || session == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "session expired or invalid"},
			})
			return
		}

		roles := make([]model.RoleGrant, len(session.RoleGrants))
		copy(roles, session.RoleGrants)

		c.Set("user_id", session.UserID)
		c.Set("roles", roles)
		c.Next()
	}
}
