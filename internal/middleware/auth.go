// Package middleware provides HTTP middleware for the PetroSync API.
// Middleware is applied in order: auth → rbac → audit.
package middleware

import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/model"
)

// Claims represents the JWT payload for Android clients.
type Claims struct {
	UserID int64             `json:"sub"`
	Roles  []model.RoleGrant `json:"roles"`
	jwt.RegisteredClaims
}

type RoleQuerier interface {
	GetActiveRolesForUser(ctx context.Context, userID int64) ([]db.UserRoleGrant, error)
	GetUser(ctx context.Context, id int64) (db.GetUserRow, error)
}

type RoleCache interface {
	GetRoleGrants(ctx context.Context, userID int64) ([]model.RoleGrant, bool, error)
	SetRoleGrants(ctx context.Context, userID int64, roles []model.RoleGrant, ttl time.Duration) error
	DeleteRoleGrants(ctx context.Context, userID int64) error

	GetUserActive(ctx context.Context, userID int64) (bool, bool, error)
	SetUserActive(ctx context.Context, userID int64, active bool, ttl time.Duration) error
	DeleteUserActive(ctx context.Context, userID int64) error
}

// JWTAuth validates the Bearer token from the Authorization header.
func JWTAuth(secret string, querier RoleQuerier, cache RoleCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "missing authorization header"},
			})
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid authorization format"},
			})
			return
		}

		token, err := jwt.ParseWithClaims(parts[1], &Claims{},
			func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			},
		)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid or expired token"},
			})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid token claims"},
			})
			return
		}

		if querier != nil {
			var active bool
			var ok bool
			if cache != nil {
				cachedActive, found, err := cache.GetUserActive(c.Request.Context(), claims.UserID)
				if err == nil && found {
					active = cachedActive
					ok = true
				}
			}
			if ok && !active {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": gin.H{"code": "FORBIDDEN", "message": "user account is inactive"},
				})
				return
			}
			if !ok {
				user, err := querier.GetUser(c.Request.Context(), claims.UserID)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid token subject"},
					})
					return
				}
				if cache != nil {
					_ = cache.SetUserActive(c.Request.Context(), claims.UserID, user.Active, 5*time.Minute)
				}
				if !user.Active {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"error": gin.H{"code": "FORBIDDEN", "message": "user account is inactive"},
					})
					return
				}
			}
		}

		var roles []model.RoleGrant
		var okRoles bool
		if cache != nil {
			cached, ok, err := cache.GetRoleGrants(c.Request.Context(), claims.UserID)
			if err == nil && ok {
				roles = cached
				okRoles = true
			}
		}

		if !okRoles && querier != nil {
			grants, err := querier.GetActiveRolesForUser(c.Request.Context(), claims.UserID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to load roles"},
				})
				return
			}
			roles = make([]model.RoleGrant, 0, len(grants))
			for _, g := range grants {
				var scopeID *int64
				if g.ScopeID.Valid {
					id := g.ScopeID.Int64
					scopeID = &id
				}
				roles = append(roles, model.RoleGrant{
					Role:      string(g.Role),
					ScopeType: string(g.ScopeType),
					ScopeID:   scopeID,
				})
			}
			if cache != nil {
				_ = cache.SetRoleGrants(c.Request.Context(), claims.UserID, roles, 5*time.Minute)
			}
		}

		c.Set("user_id", claims.UserID)
		if roles == nil {
			roles = []model.RoleGrant{}
		}
		c.Set("roles", roles)
		c.Next()
	}
}

func JWTQueryAuth(secret string, querier RoleQuerier, cache RoleCache) gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenStr := c.Query("token")
		if tokenStr == "" {
			tokenStr = c.Query("access_token")
		}
		if tokenStr == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "missing token"},
			})
			return
		}

		token, err := jwt.ParseWithClaims(tokenStr, &Claims{},
			func(t *jwt.Token) (interface{}, error) {
				if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
					return nil, jwt.ErrSignatureInvalid
				}
				return []byte(secret), nil
			},
		)
		if err != nil || !token.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid or expired token"},
			})
			return
		}

		claims, ok := token.Claims.(*Claims)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid token claims"},
			})
			return
		}

		if querier != nil {
			var active bool
			var ok bool
			if cache != nil {
				cachedActive, found, err := cache.GetUserActive(c.Request.Context(), claims.UserID)
				if err == nil && found {
					active = cachedActive
					ok = true
				}
			}
			if ok && !active {
				c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
					"error": gin.H{"code": "FORBIDDEN", "message": "user account is inactive"},
				})
				return
			}
			if !ok {
				user, err := querier.GetUser(c.Request.Context(), claims.UserID)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
						"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid token subject"},
					})
					return
				}
				if cache != nil {
					_ = cache.SetUserActive(c.Request.Context(), claims.UserID, user.Active, 5*time.Minute)
				}
				if !user.Active {
					c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
						"error": gin.H{"code": "FORBIDDEN", "message": "user account is inactive"},
					})
					return
				}
			}
		}

		var roles []model.RoleGrant
		var okRoles bool
		if cache != nil {
			cached, ok, err := cache.GetRoleGrants(c.Request.Context(), claims.UserID)
			if err == nil && ok {
				roles = cached
				okRoles = true
			}
		}

		if !okRoles && querier != nil {
			grants, err := querier.GetActiveRolesForUser(c.Request.Context(), claims.UserID)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to load roles"},
				})
				return
			}
			roles = make([]model.RoleGrant, 0, len(grants))
			for _, g := range grants {
				var scopeID *int64
				if g.ScopeID.Valid {
					id := g.ScopeID.Int64
					scopeID = &id
				}
				roles = append(roles, model.RoleGrant{
					Role:      string(g.Role),
					ScopeType: string(g.ScopeType),
					ScopeID:   scopeID,
				})
			}
			if cache != nil {
				_ = cache.SetRoleGrants(c.Request.Context(), claims.UserID, roles, 5*time.Minute)
			}
		}

		c.Set("user_id", claims.UserID)
		if roles == nil {
			roles = []model.RoleGrant{}
		}
		c.Set("roles", roles)
		c.Next()
	}
}
