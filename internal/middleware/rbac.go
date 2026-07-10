package middleware

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/adevsh/petrosync/internal/model"
)

// RequiredRole is used as middleware to enforce a minimum role for a route group.
// It checks the JWT claims (set by JWTAuth) for the required role within the
// specified scope type. The scope ID is resolved from the route parameter named
// by scopeParam (e.g., "id", "facility_id", "station_id").
//
// SYSTEM_ADMIN bypasses all scope checks — they have company-wide access.
func RequiredRole(role string, scopeType string, scopeParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "no role grants found"},
			})
			return
		}

		roles, ok := rolesVal.([]model.RoleGrant)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "invalid role grants"},
			})
			return
		}

		// Resolve scope ID from path parameter
		var scopeID *int64
		if scopeParam != "" {
			raw := c.Param(scopeParam)
			if raw != "" {
				id, err := strconv.ParseInt(raw, 10, 64)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid scope id in path"},
					})
					return
				}
				scopeID = &id
			}
		}

		// Check each role grant
		for _, r := range roles {
			// SYSTEM_ADMIN bypasses all scope checks
			if r.Role == "SYSTEM_ADMIN" {
				c.Next()
				return
			}

			// Must have the required role
			if r.Role != role {
				continue
			}

			// Must match scope type
			if r.ScopeType != scopeType && scopeType != "" {
				continue
			}

			// Scope ID must match (nil means company-wide — allowed)
			if scopeID == nil || r.ScopeID == nil || *r.ScopeID == *scopeID {
				c.Next()
				return
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": gin.H{"code": "FORBIDDEN", "message": "insufficient role or scope"},
		})
	}
}
