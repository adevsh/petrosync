package handler

import (
	"context"
	"crypto/rand"
	"errors"
	"math/big"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
)

type UserQuerier interface {
	CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error)
	GetUser(ctx context.Context, id int64) (db.GetUserRow, error)
	ListUsers(ctx context.Context) ([]db.ListUsersRow, error)
	SetUserActive(ctx context.Context, arg db.SetUserActiveParams) (db.SetUserActiveRow, error)
	UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.UpdateUserRow, error)

	GetActiveRolesForUser(ctx context.Context, userID int64) ([]db.UserRoleGrant, error)
	GrantRole(ctx context.Context, arg db.GrantRoleParams) (db.UserRoleGrant, error)
	RevokeRole(ctx context.Context, arg db.RevokeRoleParams) error
}

type UserCache interface {
	DeleteRoleGrants(ctx context.Context, userID int64) error
	DeleteUserActive(ctx context.Context, userID int64) error
}

type UserHandler struct {
	querier UserQuerier
	cache   UserCache
}

func NewUserHandler(querier UserQuerier, cache UserCache) *UserHandler {
	return &UserHandler{querier: querier, cache: cache}
}

type userResponse struct {
	ID                  int64      `json:"id"`
	Username            string     `json:"username"`
	FullName            string     `json:"full_name"`
	TelegramUserID      *int64     `json:"telegram_user_id"`
	TelegramLinkedAt    *time.Time `json:"telegram_linked_at"`
	ForcePasswordChange bool       `json:"force_password_change"`
	Active              bool       `json:"active"`
	LastLoginAt         *time.Time `json:"last_login_at"`
	CreatedAt           *time.Time `json:"created_at"`
	UpdatedAt           *time.Time `json:"updated_at"`
}

func pgInt8ToPtr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}

func pgTimestamptzToPtr(v pgtype.Timestamptz) *time.Time {
	if !v.Valid {
		return nil
	}
	t := v.Time
	return &t
}

func userFromCreateRow(r db.CreateUserRow) userResponse {
	return userResponse{
		ID:                  r.ID,
		Username:            r.Username,
		FullName:            r.FullName,
		TelegramUserID:      pgInt8ToPtr(r.TelegramUserID),
		TelegramLinkedAt:    pgTimestamptzToPtr(r.TelegramLinkedAt),
		ForcePasswordChange: r.ForcePasswordChange,
		Active:              r.Active,
		LastLoginAt:         pgTimestamptzToPtr(r.LastLoginAt),
		CreatedAt:           pgTimestamptzToPtr(r.CreatedAt),
		UpdatedAt:           pgTimestamptzToPtr(r.UpdatedAt),
	}
}

func userFromGetRow(r db.GetUserRow) userResponse {
	return userResponse{
		ID:                  r.ID,
		Username:            r.Username,
		FullName:            r.FullName,
		TelegramUserID:      pgInt8ToPtr(r.TelegramUserID),
		TelegramLinkedAt:    pgTimestamptzToPtr(r.TelegramLinkedAt),
		ForcePasswordChange: r.ForcePasswordChange,
		Active:              r.Active,
		LastLoginAt:         pgTimestamptzToPtr(r.LastLoginAt),
		CreatedAt:           pgTimestamptzToPtr(r.CreatedAt),
		UpdatedAt:           pgTimestamptzToPtr(r.UpdatedAt),
	}
}

func userFromListRow(r db.ListUsersRow) userResponse {
	return userResponse{
		ID:                  r.ID,
		Username:            r.Username,
		FullName:            r.FullName,
		TelegramUserID:      pgInt8ToPtr(r.TelegramUserID),
		TelegramLinkedAt:    pgTimestamptzToPtr(r.TelegramLinkedAt),
		ForcePasswordChange: r.ForcePasswordChange,
		Active:              r.Active,
		LastLoginAt:         pgTimestamptzToPtr(r.LastLoginAt),
		CreatedAt:           pgTimestamptzToPtr(r.CreatedAt),
		UpdatedAt:           pgTimestamptzToPtr(r.UpdatedAt),
	}
}

func userFromSetActiveRow(r db.SetUserActiveRow) userResponse {
	return userResponse{
		ID:                  r.ID,
		Username:            r.Username,
		FullName:            r.FullName,
		TelegramUserID:      pgInt8ToPtr(r.TelegramUserID),
		TelegramLinkedAt:    pgTimestamptzToPtr(r.TelegramLinkedAt),
		ForcePasswordChange: r.ForcePasswordChange,
		Active:              r.Active,
		LastLoginAt:         pgTimestamptzToPtr(r.LastLoginAt),
		CreatedAt:           pgTimestamptzToPtr(r.CreatedAt),
		UpdatedAt:           pgTimestamptzToPtr(r.UpdatedAt),
	}
}

func userFromUpdateRow(r db.UpdateUserRow) userResponse {
	return userResponse{
		ID:                  r.ID,
		Username:            r.Username,
		FullName:            r.FullName,
		TelegramUserID:      pgInt8ToPtr(r.TelegramUserID),
		TelegramLinkedAt:    pgTimestamptzToPtr(r.TelegramLinkedAt),
		ForcePasswordChange: r.ForcePasswordChange,
		Active:              r.Active,
		LastLoginAt:         pgTimestamptzToPtr(r.LastLoginAt),
		CreatedAt:           pgTimestamptzToPtr(r.CreatedAt),
		UpdatedAt:           pgTimestamptzToPtr(r.UpdatedAt),
	}
}

func (h *UserHandler) ListUsers(c *gin.Context) {
	rows, err := h.querier.ListUsers(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	out := make([]userResponse, 0, len(rows))
	for _, r := range rows {
		out = append(out, userFromListRow(r))
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

func (h *UserHandler) GetUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid user id"}})
		return
	}
	u, err := h.querier.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "user not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": userFromGetRow(u)})
}

type createUserRequest struct {
	Username string `json:"username" binding:"required"`
	FullName string `json:"full_name" binding:"required"`
	Password string `json:"password" binding:"required,min=8"`
}

func (h *UserHandler) CreateUser(c *gin.Context) {
	var req createUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to hash password"}})
		return
	}

	row, err := h.querier.CreateUser(c.Request.Context(), db.CreateUserParams{
		Username:            req.Username,
		PasswordHash:        string(hashed),
		FullName:            req.FullName,
		ForcePasswordChange: true,
	})
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "CONFLICT", "message": "username already exists"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	middleware.SetAuditAction(c, "USER_CREATE")
	middleware.SetAuditEntity(c, "users", row.ID)
	middleware.SetAuditAfter(c, userFromCreateRow(row))
	c.JSON(http.StatusCreated, gin.H{"data": userFromCreateRow(row)})
}

type updateUserRequest struct {
	Username *string `json:"username"`
	FullName *string `json:"full_name"`
	Active   *bool   `json:"active"`
}

func (h *UserHandler) UpdateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid user id"}})
		return
	}

	var req updateUserRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	if req.Username == nil && req.FullName == nil && req.Active == nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "no fields to update"}})
		return
	}

	var updated userResponse
	loaded, err := h.querier.GetUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "user not found"}})
		return
	}
	middleware.SetAuditAction(c, "USER_UPDATE")
	middleware.SetAuditEntity(c, "users", id)
	middleware.SetAuditBefore(c, userFromGetRow(loaded))
	updated = userFromGetRow(loaded)

	if req.Username != nil || req.FullName != nil {
		username := updated.Username
		fullName := updated.FullName
		if req.Username != nil {
			username = *req.Username
		}
		if req.FullName != nil {
			fullName = *req.FullName
		}

		row, err := h.querier.UpdateUser(c.Request.Context(), db.UpdateUserParams{
			ID:       id,
			Username: username,
			FullName: fullName,
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "CONFLICT", "message": "username already exists"}})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		updated = userFromUpdateRow(row)
	}

	if req.Active != nil {
		row, err := h.querier.SetUserActive(c.Request.Context(), db.SetUserActiveParams{
			ID:     id,
			Active: *req.Active,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		updated = userFromSetActiveRow(row)
		if h.cache != nil {
			_ = h.cache.DeleteUserActive(c.Request.Context(), id)
		}
	}

	middleware.SetAuditAfter(c, updated)
	c.JSON(http.StatusOK, gin.H{"data": updated})
}

func (h *UserHandler) DeactivateUser(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid user id"}})
		return
	}

	if loaded, err := h.querier.GetUser(c.Request.Context(), id); err == nil {
		middleware.SetAuditBefore(c, userFromGetRow(loaded))
	}
	row, err := h.querier.SetUserActive(c.Request.Context(), db.SetUserActiveParams{
		ID:     id,
		Active: false,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if h.cache != nil {
		_ = h.cache.DeleteUserActive(c.Request.Context(), id)
	}
	middleware.SetAuditAction(c, "USER_DEACTIVATE")
	middleware.SetAuditEntity(c, "users", id)
	middleware.SetAuditAfter(c, userFromSetActiveRow(row))
	c.JSON(http.StatusOK, gin.H{"data": userFromSetActiveRow(row)})
}

type roleGrantResponse struct {
	ID        int64      `json:"id"`
	UserID    int64      `json:"user_id"`
	Role      string     `json:"role"`
	ScopeType string     `json:"scope_type"`
	ScopeID   *int64     `json:"scope_id"`
	GrantedBy *int64     `json:"granted_by"`
	GrantedAt *time.Time `json:"granted_at"`
	RevokedAt *time.Time `json:"revoked_at"`
}

func roleGrantFromRow(r db.UserRoleGrant) roleGrantResponse {
	return roleGrantResponse{
		ID:        r.ID,
		UserID:    r.UserID,
		Role:      string(r.Role),
		ScopeType: string(r.ScopeType),
		ScopeID:   pgInt8ToPtr(r.ScopeID),
		GrantedBy: pgInt8ToPtr(r.GrantedBy),
		GrantedAt: pgTimestamptzToPtr(r.GrantedAt),
		RevokedAt: pgTimestamptzToPtr(r.RevokedAt),
	}
}

func (h *UserHandler) ListRoles(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid user id"}})
		return
	}
	grants, err := h.querier.GetActiveRolesForUser(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	out := make([]roleGrantResponse, 0, len(grants))
	for _, g := range grants {
		out = append(out, roleGrantFromRow(g))
	}
	c.JSON(http.StatusOK, gin.H{"data": out})
}

type roleChangeRequest struct {
	Role      string `json:"role" binding:"required"`
	ScopeType string `json:"scope_type" binding:"required"`
	ScopeID   *int64 `json:"scope_id"`
}

func (h *UserHandler) GrantRole(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid user id"}})
		return
	}

	var req roleChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	role := db.UserRoleT(req.Role)
	if !role.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid role"}})
		return
	}

	scopeType := db.RoleScopeT(req.ScopeType)
	if !scopeType.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid scope_type"}})
		return
	}

	scopeID := pgtype.Int8{Valid: false}
	if scopeType == db.RoleScopeTCOMPANY {
		scopeID = pgtype.Int8{Int64: 0, Valid: true}
	} else {
		if req.ScopeID == nil || *req.ScopeID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "scope_id is required for this scope_type"}})
			return
		}
		scopeID = pgtype.Int8{Int64: *req.ScopeID, Valid: true}
	}

	grantedByVal, ok := c.Get("user_id")
	if !ok {
		c.JSON(http.StatusUnauthorized, gin.H{"error": gin.H{"code": "UNAUTHORIZED", "message": "not authenticated"}})
		return
	}
	grantedBy := grantedByVal.(int64)

	row, err := h.querier.GrantRole(c.Request.Context(), db.GrantRoleParams{
		UserID:    targetUserID,
		Role:      role,
		ScopeType: scopeType,
		ScopeID:   scopeID,
		GrantedBy: pgtype.Int8{Int64: grantedBy, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if h.cache != nil {
		_ = h.cache.DeleteRoleGrants(c.Request.Context(), targetUserID)
	}
	middleware.SetAuditAction(c, "USER_ROLE_GRANT")
	middleware.SetAuditEntity(c, "user_role_grants", row.ID)
	middleware.SetAuditAfter(c, roleGrantFromRow(row))
	c.JSON(http.StatusCreated, gin.H{"data": roleGrantFromRow(row)})
}

func (h *UserHandler) RevokeRole(c *gin.Context) {
	targetUserID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid user id"}})
		return
	}

	var req roleChangeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	role := db.UserRoleT(req.Role)
	if !role.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid role"}})
		return
	}

	scopeType := db.RoleScopeT(req.ScopeType)
	if !scopeType.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid scope_type"}})
		return
	}

	scopeID := pgtype.Int8{Int64: 0, Valid: true}
	if scopeType != db.RoleScopeTCOMPANY {
		if req.ScopeID == nil || *req.ScopeID <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "scope_id is required for this scope_type"}})
			return
		}
		scopeID = pgtype.Int8{Int64: *req.ScopeID, Valid: true}
	}

	if err := h.querier.RevokeRole(c.Request.Context(), db.RevokeRoleParams{
		UserID:    targetUserID,
		Role:      role,
		ScopeType: scopeType,
		ScopeID:   scopeID,
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if h.cache != nil {
		_ = h.cache.DeleteRoleGrants(c.Request.Context(), targetUserID)
	}
	middleware.SetAuditAction(c, "USER_ROLE_REVOKE")
	middleware.SetAuditEntity(c, "users", targetUserID)
	middleware.SetAuditBefore(c, req)
	middleware.SetAuditAfter(c, gin.H{"revoked": true})
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "role revoked"}})
}

// ── Password Reset ────────────────────────────────────────────────────

type UserPasswordStore interface {
	GetUser(ctx context.Context, id int64) (db.GetUserRow, error)
	GetUserByUsername(ctx context.Context, username string) (db.User, error)
	UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) error
	SetForcePasswordChange(ctx context.Context, id int64) error
	GetUserPasswordHash(ctx context.Context, id int64) (string, error)
}

type NotifyReset interface {
	SendTelegramDM(ctx context.Context, telegramUserID int64, message string) error
}

type ResetPasswordHandler struct {
	store    UserPasswordStore
	notifier NotifyReset
}

func NewResetPasswordHandler(store UserPasswordStore, notifier NotifyReset) *ResetPasswordHandler {
	return &ResetPasswordHandler{store: store, notifier: notifier}
}

type resetPasswordRequest struct {
	UserID int64 `json:"user_id" binding:"required"`
}

func (h *ResetPasswordHandler) ResetPassword(c *gin.Context) {
	var req resetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	user, err := h.store.GetUser(c.Request.Context(), req.UserID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "user not found"}})
		return
	}

	tempPassword, err := generateTempPassword(12)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to generate password"}})
		return
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(tempPassword), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to hash password"}})
		return
	}

	if err := h.store.UpdateUserPassword(c.Request.Context(), db.UpdateUserPasswordParams{
		ID: req.UserID, PasswordHash: string(hashed),
	}); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	_ = h.store.SetForcePasswordChange(c.Request.Context(), req.UserID)

	// Notify via Telegram DM if linked
	linked := user.TelegramUserID.Valid && user.TelegramUserID.Int64 > 0
	if linked && h.notifier != nil {
		msg := "Your PetroSync password has been reset. Temporary password: " + tempPassword +
			"\nYou will be prompted to change it on next login."
		_ = h.notifier.SendTelegramDM(c.Request.Context(), user.TelegramUserID.Int64, msg)
	}

	middleware.SetAuditAction(c, "USER_PASSWORD_RESET")
	middleware.SetAuditEntity(c, "users", req.UserID)

	c.JSON(http.StatusOK, gin.H{"data": gin.H{
		"message":         "password reset",
		"telegram_linked": linked,
		"temp_password":   tempPassword,
	}})
}

func generateTempPassword(length int) (string, error) {
	const chars = "abcdefghjkmnpqrstuvwxyzABCDEFGHJKMNPQRSTUVWXYZ23456789"
	b := make([]byte, length)
	for i := range b {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", err
		}
		b[i] = chars[n.Int64()]
	}
	return string(b), nil
}
