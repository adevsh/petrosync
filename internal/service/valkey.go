// Package service — Valkey client wrapper for sessions, caching, and pub/sub.
package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/valkey-io/valkey-go"

	"github.com/adevsh/petrosync/internal/model"
)

// ValkeyService wraps the Valkey client for session management,
// caching, and pub/sub messaging.
type ValkeyService struct {
	client valkey.Client
}

// NewValkeyService creates a ValkeyService connected to the given address.
func NewValkeyService(ctx context.Context, addr string) (*ValkeyService, error) {
	client, err := valkey.NewClient(valkey.ClientOption{
		InitAddress: []string{addr},
	})
	if err != nil {
		return nil, fmt.Errorf("valkey: %w", err)
	}

	if err := client.Do(ctx, client.B().Ping().Build()).Error(); err != nil {
		client.Close()
		return nil, fmt.Errorf("valkey ping: %w", err)
	}

	return &ValkeyService{client: client}, nil
}

// Close shuts down the Valkey client.
func (v *ValkeyService) Close() {
	v.client.Close()
}

// Client returns the underlying Valkey client for advanced usage.
func (v *ValkeyService) Client() valkey.Client {
	return v.client
}

// ── Sessions ──────────────────────────────────────────────────────────

// SessionData is stored in Valkey for dashboard sessions.
type SessionData = model.SessionData

// RoleGrantJSON is the JSON-serializable role grant.
type RoleGrantJSON = model.RoleGrant

// SaveSession stores a dashboard session with an 8-hour TTL.
func (v *ValkeyService) SaveSession(ctx context.Context, sessionID string, data model.SessionData) error {
	payload, err := json.Marshal(data)
	if err != nil {
		return fmt.Errorf("marshal session: %w", err)
	}
	key := fmt.Sprintf("sess:%s", sessionID)
	ttl := 8 * time.Hour
	if !data.ExpiresAt.IsZero() {
		ttl = time.Until(data.ExpiresAt)
		if ttl <= 0 {
			ttl = time.Second
		}
	}
	return v.client.Do(ctx, v.client.B().Set().Key(key).Value(string(payload)).Px(ttl).Build()).Error()
}

// GetSession retrieves a dashboard session. Returns nil if not found or expired.
func (v *ValkeyService) GetSession(ctx context.Context, sessionID string) (*model.SessionData, error) {
	key := fmt.Sprintf("sess:%s", sessionID)
	result, err := v.client.Do(ctx, v.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		return nil, nil // not found
	}
	var data model.SessionData
	if err := json.Unmarshal([]byte(result), &data); err != nil {
		return nil, fmt.Errorf("unmarshal session: %w", err)
	}
	if time.Now().After(data.ExpiresAt) {
		return nil, nil // expired
	}
	return &data, nil
}

// DeleteSession removes a dashboard session (logout).
func (v *ValkeyService) DeleteSession(ctx context.Context, sessionID string) error {
	key := fmt.Sprintf("sess:%s", sessionID)
	return v.client.Do(ctx, v.client.B().Del().Key(key).Build()).Error()
}

// ── Refresh Tokens ────────────────────────────────────────────────────

// SaveRefreshToken stores a JWT refresh token with a 30-day TTL.
func (v *ValkeyService) SaveRefreshToken(ctx context.Context, token string, userID int64) error {
	key := fmt.Sprintf("jwt:refresh:%s", token)
	return v.client.Do(ctx, v.client.B().Set().Key(key).Value(fmt.Sprintf("%d", userID)).Px(30*24*time.Hour).Build()).Error()
}

// GetRefreshToken validates a refresh token and returns the user ID.
// Returns 0 if not found or expired.
func (v *ValkeyService) GetRefreshToken(ctx context.Context, token string) (int64, error) {
	key := fmt.Sprintf("jwt:refresh:%s", token)
	result, err := v.client.Do(ctx, v.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		return 0, nil
	}
	var userID int64
	if _, err := fmt.Sscanf(result, "%d", &userID); err != nil {
		return 0, fmt.Errorf("invalid refresh token value: %w", err)
	}
	return userID, nil
}

// DeleteRefreshToken invalidates a refresh token.
func (v *ValkeyService) DeleteRefreshToken(ctx context.Context, token string) error {
	key := fmt.Sprintf("jwt:refresh:%s", token)
	return v.client.Do(ctx, v.client.B().Del().Key(key).Build()).Error()
}

// ── RBAC Cache ─────────────────────────────────────────────────────────

func (v *ValkeyService) GetRoleGrants(ctx context.Context, userID int64) ([]model.RoleGrant, bool, error) {
	key := fmt.Sprintf("rbac:%d", userID)
	result, err := v.client.Do(ctx, v.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		return nil, false, nil
	}
	var grants []model.RoleGrant
	if err := json.Unmarshal([]byte(result), &grants); err != nil {
		return nil, false, fmt.Errorf("unmarshal rbac cache: %w", err)
	}
	if grants == nil {
		grants = []model.RoleGrant{}
	}
	return grants, true, nil
}

func (v *ValkeyService) SetRoleGrants(ctx context.Context, userID int64, grants []model.RoleGrant, ttl time.Duration) error {
	payload, err := json.Marshal(grants)
	if err != nil {
		return fmt.Errorf("marshal rbac cache: %w", err)
	}
	key := fmt.Sprintf("rbac:%d", userID)
	return v.client.Do(ctx, v.client.B().Set().Key(key).Value(string(payload)).Px(ttl).Build()).Error()
}

func (v *ValkeyService) DeleteRoleGrants(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("rbac:%d", userID)
	return v.client.Do(ctx, v.client.B().Del().Key(key).Build()).Error()
}

func (v *ValkeyService) GetUserActive(ctx context.Context, userID int64) (bool, bool, error) {
	key := fmt.Sprintf("user:active:%d", userID)
	result, err := v.client.Do(ctx, v.client.B().Get().Key(key).Build()).ToString()
	if err != nil {
		return false, false, nil
	}
	switch result {
	case "1":
		return true, true, nil
	case "0":
		return false, true, nil
	default:
		return false, false, fmt.Errorf("invalid user active cache value")
	}
}

func (v *ValkeyService) SetUserActive(ctx context.Context, userID int64, active bool, ttl time.Duration) error {
	key := fmt.Sprintf("user:active:%d", userID)
	val := "0"
	if active {
		val = "1"
	}
	return v.client.Do(ctx, v.client.B().Set().Key(key).Value(val).Px(ttl).Build()).Error()
}

func (v *ValkeyService) DeleteUserActive(ctx context.Context, userID int64) error {
	key := fmt.Sprintf("user:active:%d", userID)
	return v.client.Do(ctx, v.client.B().Del().Key(key).Build()).Error()
}

// ── Pub/Sub ───────────────────────────────────────────────────────────

// Publish sends a message on a Valkey channel (used for GPS → WebSocket bridge).
func (v *ValkeyService) Publish(ctx context.Context, channel, message string) error {
	return v.client.Do(ctx, v.client.B().Publish().Channel(channel).Message(message).Build()).Error()
}
