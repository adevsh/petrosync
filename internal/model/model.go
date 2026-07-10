// Package model defines shared types used across middleware, service, and handler packages.
package model

import "time"

// RoleGrant is the role+scope grant carried in JWT claims and session data.
type RoleGrant struct {
	Role      string `json:"role"`
	ScopeType string `json:"scope_type"`
	ScopeID   *int64 `json:"scope_id"`
}

// SessionData is stored in Valkey for dashboard sessions.
type SessionData struct {
	UserID     int64       `json:"user_id"`
	RoleGrants []RoleGrant `json:"role_grants"`
	ExpiresAt  time.Time   `json:"expires_at"`
}
