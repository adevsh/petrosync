package handler

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/adevsh/petrosync/internal/middleware"
	"github.com/adevsh/petrosync/internal/model"
	"github.com/adevsh/petrosync/internal/service"
)

// AuthHandler handles authentication endpoints.
type AuthHandler struct {
	authService *service.AuthService
	jwtSecret   string
}

// NewAuthHandler creates an AuthHandler.
func NewAuthHandler(authService *service.AuthService, jwtSecret string) *AuthHandler {
	return &AuthHandler{authService: authService, jwtSecret: jwtSecret}
}

type loginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

type loginResponse struct {
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	UserID       int64  `json:"user_id"`
	FullName     string `json:"full_name"`
}

// Login authenticates a user and returns JWT + refresh token.
func (h *AuthHandler) Login(c *gin.Context) {
	var req loginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "username and password are required"},
		})
		return
	}

	user, accessToken, refreshToken, err := h.authService.Login(c.Request.Context(), req.Username, req.Password)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			c.JSON(http.StatusUnauthorized, gin.H{
				"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid username or password"},
			})
		case errors.Is(err, service.ErrUserInactive):
			c.JSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "user account is inactive"},
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "INTERNAL_ERROR", "message": "authentication failed"},
			})
		}
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": loginResponse{
			AccessToken:  accessToken,
			RefreshToken: refreshToken,
			UserID:       user.ID,
			FullName:     user.FullName,
		},
	})
}

type refreshRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Refresh issues a new access token from a valid refresh token.
func (h *AuthHandler) Refresh(c *gin.Context) {
	var req refreshRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "refresh_token is required"},
		})
		return
	}

	accessToken, refreshToken, err := h.authService.Refresh(c.Request.Context(), req.RefreshToken)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "UNAUTHORIZED", "message": "invalid or expired refresh token"},
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
		},
	})
}

type logoutRequest struct {
	RefreshToken string `json:"refresh_token" binding:"required"`
}

// Logout invalidates the refresh token.
func (h *AuthHandler) Logout(c *gin.Context) {
	var req logoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "refresh_token is required"},
		})
		return
	}

	_ = h.authService.Logout(c.Request.Context(), req.RefreshToken)
	middleware.SetAuditAction(c, "AUTH_LOGOUT")
	middleware.SetAuditEntity(c, "auth", 0)
	middleware.SetAuditAfter(c, gin.H{"logged_out": true})
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "logged out"}})
}

type changePasswordRequest struct {
	CurrentPassword string `json:"current_password" binding:"required"`
	NewPassword     string `json:"new_password" binding:"required,min=8"`
}

// ChangePassword updates the authenticated user's password.
func (h *AuthHandler) ChangePassword(c *gin.Context) {
	var req changePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "current_password and new_password (min 8 chars) are required"},
		})
		return
	}

	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"error": gin.H{"code": "UNAUTHORIZED", "message": "not authenticated"},
		})
		return
	}

	uid := userID.(int64)
	middleware.SetAuditAction(c, "AUTH_CHANGE_PASSWORD")
	middleware.SetAuditEntity(c, "users", uid)
	middleware.SetAuditBefore(c, gin.H{"user_id": uid})

	err := h.authService.ChangePassword(c.Request.Context(), uid, req.CurrentPassword, req.NewPassword)
	if err != nil {
		switch {
		case errors.Is(err, service.ErrPasswordMismatch):
			c.JSON(http.StatusBadRequest, gin.H{
				"error": gin.H{"code": "VALIDATION_ERROR", "message": "current password is incorrect"},
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "INTERNAL_ERROR", "message": "password change failed"},
			})
		}
		return
	}

	middleware.SetAuditAfter(c, gin.H{"password_changed": true})
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "password changed"}})
}

// Ensure AuthHandler compiles.
var _ model.RoleGrant
