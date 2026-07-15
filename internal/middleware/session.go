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

func loadSession(ctx context.Context, store SessionStore, sessionID string) (*model.SessionData, error) {
	if sessionID == "" {
		return nil, nil
	}
	return store.GetSession(ctx, sessionID)
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

		session, err := loadSession(c.Request.Context(), store, sessionID)
		if err != nil || session == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "session expired or invalid"},
			})
			return
		}

		roles := make([]model.RoleGrant, len(session.RoleGrants))
		copy(roles, session.RoleGrants)

		c.Set("user_id", session.UserID)
		c.Set("full_name", session.FullName)
		c.Set("roles", roles)
		c.Set("session", *session)
		c.Next()
	}
}

// SessionPageAuth redirects browser requests to the dashboard login page.
func SessionPageAuth(store SessionStore, loginPath string) gin.HandlerFunc {
	return func(c *gin.Context) {
		sessionID, err := c.Cookie("petrosync_session")
		if err != nil {
			c.Redirect(http.StatusSeeOther, loginPath)
			c.Abort()
			return
		}

		session, err := loadSession(c.Request.Context(), store, sessionID)
		if err != nil || session == nil {
			c.Redirect(http.StatusSeeOther, loginPath)
			c.Abort()
			return
		}

		roles := make([]model.RoleGrant, len(session.RoleGrants))
		copy(roles, session.RoleGrants)

		c.Set("user_id", session.UserID)
		c.Set("full_name", session.FullName)
		c.Set("roles", roles)
		c.Set("session_id", sessionID)
		c.Set("session", *session)
		c.Next()
	}
}

// RequirePasswordChange redirects authenticated dashboard sessions until the password is changed.
func RequirePasswordChange(changePasswordPath string, allowPaths ...string) gin.HandlerFunc {
	allowed := map[string]struct{}{
		changePasswordPath: {},
	}
	for _, path := range allowPaths {
		allowed[path] = struct{}{}
	}

	return func(c *gin.Context) {
		sessionVal, exists := c.Get("session")
		if !exists {
			c.Next()
			return
		}

		session, ok := sessionVal.(model.SessionData)
		if !ok || !session.ForcePasswordChange {
			c.Next()
			return
		}

		if _, ok := allowed[c.Request.URL.Path]; ok {
			c.Next()
			return
		}

		c.Redirect(http.StatusSeeOther, changePasswordPath)
		c.Abort()
	}
}
