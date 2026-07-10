// Package service implements the business logic layer for PetroSync.
// No SQL, no HTTP concerns belong here — only domain rules and orchestration.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"golang.org/x/crypto/bcrypt"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
	"github.com/adevsh/petrosync/internal/model"
)

// Common errors
var (
	ErrInvalidCredentials = errors.New("invalid username or password")
	ErrUserInactive       = errors.New("user account is inactive")
	ErrPasswordMismatch   = errors.New("current password does not match")
)

// AuthService handles authentication: login, refresh, logout, password change.
type AuthService struct {
	querier   *db.Queries
	jwtSecret []byte
	valkey    *ValkeyService
}

// NewAuthService creates an AuthService.
func NewAuthService(querier *db.Queries, jwtSecret string, valkey *ValkeyService) *AuthService {
	return &AuthService{
		querier:   querier,
		jwtSecret: []byte(jwtSecret),
		valkey:    valkey,
	}
}

// Login authenticates a user and returns JWT + refresh token.
func (s *AuthService) Login(ctx context.Context, username, password string) (*db.User, string, string, error) {
	user, err := s.querier.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, "", "", ErrInvalidCredentials
	}
	if !user.Active {
		return nil, "", "", ErrUserInactive
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	// Fetch active role grants
	grants, err := s.querier.GetActiveRolesForUser(ctx, user.ID)
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to load roles: %w", err)
	}

	// Build JWT claims
	accessToken, err := s.issueJWT(user.ID, grants)
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := generateRefreshToken()
	if err != nil {
		return nil, "", "", fmt.Errorf("failed to generate refresh token: %w", err)
	}

	// Store refresh token in Valkey
	if err := s.valkey.SaveRefreshToken(ctx, refreshToken, user.ID); err != nil {
		return nil, "", "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	// Record login timestamp (best-effort)
	_ = s.querier.RecordUserLogin(ctx, user.ID)

	return &user, accessToken, refreshToken, nil
}

// issueJWT creates a signed JWT for the given user and role grants.
func (s *AuthService) issueJWT(userID int64, grants []db.UserRoleGrant) (string, error) {
	now := time.Now()
	roleClaims := make([]model.RoleGrant, len(grants))
	for i, g := range grants {
		roleClaims[i] = model.RoleGrant{
			Role:      string(g.Role),
			ScopeType: string(g.ScopeType),
			ScopeID:   pgInt8ToPtr(g.ScopeID),
		}
	}

	claims := middleware.Claims{
		UserID: userID,
		Roles:  roleClaims,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.jwtSecret)
}

// Refresh validates a refresh token and issues a new JWT.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (string, string, error) {
	userID, err := s.valkey.GetRefreshToken(ctx, refreshToken)
	if err != nil || userID == 0 {
		return "", "", errors.New("invalid or expired refresh token")
	}

	// Invalidate old token
	_ = s.valkey.DeleteRefreshToken(ctx, refreshToken)

	// Load user and roles
	user, err := s.querier.GetUser(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("user not found: %w", err)
	}

	grants, err := s.querier.GetActiveRolesForUser(ctx, userID)
	if err != nil {
		return "", "", fmt.Errorf("failed to load roles: %w", err)
	}

	accessToken, err := s.issueJWT(user.ID, grants)
	if err != nil {
		return "", "", err
	}

	newRefresh, err := generateRefreshToken()
	if err != nil {
		return "", "", err
	}

	if err := s.valkey.SaveRefreshToken(ctx, newRefresh, userID); err != nil {
		return "", "", fmt.Errorf("failed to store refresh token: %w", err)
	}

	return accessToken, newRefresh, nil
}

// Logout invalidates the refresh token.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	return s.valkey.DeleteRefreshToken(ctx, refreshToken)
}

// ChangePassword verifies the current password and sets a new one.
func (s *AuthService) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	pwHash, err := s.querier.GetUserPasswordHash(ctx, userID)
	if err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(pwHash), []byte(currentPassword)); err != nil {
		return ErrPasswordMismatch
	}

	hashed, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return fmt.Errorf("failed to hash password: %w", err)
	}

	return s.querier.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           userID,
		PasswordHash: string(hashed),
	})
}

func generateRefreshToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func pgInt8ToPtr(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	return &v.Int64
}
