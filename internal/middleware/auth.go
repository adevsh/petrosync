// Package middleware provides HTTP middleware for the PetroSync API.
// Middleware is applied in order: auth → rbac → audit.
package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/adevsh/petrosync/internal/model"
)

// Claims represents the JWT payload for Android clients.
type Claims struct {
	UserID int64             `json:"sub"`
	Roles  []model.RoleGrant `json:"roles"`
	jwt.RegisteredClaims
}

// JWTAuth validates the Bearer token from the Authorization header.
func JWTAuth(secret string) gin.HandlerFunc {
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

		c.Set("user_id", claims.UserID)
		c.Set("roles", claims.Roles)
		c.Next()
	}
}
