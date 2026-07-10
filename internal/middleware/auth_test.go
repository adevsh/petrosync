package middleware

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/model"
)

type fakeRoleQuerier struct {
	getUserCalls  int
	getRolesCalls int

	userActive bool
	roles      []db.UserRoleGrant
}

func (f *fakeRoleQuerier) GetActiveRolesForUser(ctx context.Context, userID int64) ([]db.UserRoleGrant, error) {
	f.getRolesCalls++
	return f.roles, nil
}

func (f *fakeRoleQuerier) GetUser(ctx context.Context, id int64) (db.GetUserRow, error) {
	f.getUserCalls++
	return db.GetUserRow{ID: id, Active: f.userActive}, nil
}

type fakeRoleCache struct {
	roleGrants map[int64][]model.RoleGrant
	active     map[int64]bool
}

func (f *fakeRoleCache) GetRoleGrants(ctx context.Context, userID int64) ([]model.RoleGrant, bool, error) {
	v, ok := f.roleGrants[userID]
	if !ok {
		return nil, false, nil
	}
	return v, true, nil
}

func (f *fakeRoleCache) SetRoleGrants(ctx context.Context, userID int64, roles []model.RoleGrant, ttl time.Duration) error {
	f.roleGrants[userID] = roles
	return nil
}

func (f *fakeRoleCache) DeleteRoleGrants(ctx context.Context, userID int64) error {
	delete(f.roleGrants, userID)
	return nil
}

func (f *fakeRoleCache) GetUserActive(ctx context.Context, userID int64) (bool, bool, error) {
	v, ok := f.active[userID]
	if !ok {
		return false, false, nil
	}
	return v, true, nil
}

func (f *fakeRoleCache) SetUserActive(ctx context.Context, userID int64, active bool, ttl time.Duration) error {
	f.active[userID] = active
	return nil
}

func (f *fakeRoleCache) DeleteUserActive(ctx context.Context, userID int64) error {
	delete(f.active, userID)
	return nil
}

func makeToken(t *testing.T, secret string, userID int64) string {
	t.Helper()
	now := time.Now()
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, &Claims{
		UserID: userID,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(30 * time.Minute)),
		},
	})
	signed, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	return signed
}

func TestJWTAuth_UsesCacheWhenPresent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"

	q := &fakeRoleQuerier{
		userActive: true,
		roles: []db.UserRoleGrant{
			{Role: db.UserRoleTSYSTEMADMIN, ScopeType: db.RoleScopeTCOMPANY},
		},
	}
	cache := &fakeRoleCache{
		roleGrants: map[int64][]model.RoleGrant{
			1: {{Role: "SYSTEM_ADMIN", ScopeType: "COMPANY"}},
		},
		active: map[int64]bool{
			1: true,
		},
	}

	r := gin.New()
	r.Use(JWTAuth(secret, q, cache))
	r.GET("/x", func(c *gin.Context) {
		roles, _ := c.Get("roles")
		c.JSON(http.StatusOK, gin.H{"roles": roles})
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, secret, 1))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if q.getRolesCalls != 0 {
		t.Fatalf("expected no roles DB calls, got %d", q.getRolesCalls)
	}

	var body struct {
		Roles []model.RoleGrant `json:"roles"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Roles) != 1 || body.Roles[0].Role != "SYSTEM_ADMIN" {
		t.Fatalf("unexpected roles: %#v", body.Roles)
	}
}

func TestJWTAuth_LoadsFromDBWhenCacheMiss(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"

	q := &fakeRoleQuerier{
		userActive: true,
		roles: []db.UserRoleGrant{
			{Role: db.UserRoleTSYSTEMADMIN, ScopeType: db.RoleScopeTCOMPANY},
		},
	}
	cache := &fakeRoleCache{
		roleGrants: map[int64][]model.RoleGrant{},
		active:     map[int64]bool{},
	}

	r := gin.New()
	r.Use(JWTAuth(secret, q, cache))
	r.GET("/x", func(c *gin.Context) {
		roles, _ := c.Get("roles")
		c.JSON(http.StatusOK, gin.H{"roles": roles})
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, secret, 1))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if q.getRolesCalls != 1 {
		t.Fatalf("expected 1 roles DB call, got %d", q.getRolesCalls)
	}
	if _, ok := cache.roleGrants[1]; !ok {
		t.Fatalf("expected roles cached")
	}
	if _, ok := cache.active[1]; !ok {
		t.Fatalf("expected active cached")
	}
}

func TestJWTAuth_BlocksInactiveUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"

	q := &fakeRoleQuerier{
		userActive: true,
	}
	cache := &fakeRoleCache{
		roleGrants: map[int64][]model.RoleGrant{},
		active: map[int64]bool{
			1: false,
		},
	}

	r := gin.New()
	r.Use(JWTAuth(secret, q, cache))
	r.GET("/x", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	req.Header.Set("Authorization", "Bearer "+makeToken(t, secret, 1))
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d: %s", w.Code, w.Body.String())
	}
}

func TestJWTQueryAuth_MissingToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"

	r := gin.New()
	r.Use(JWTQueryAuth(secret, nil, nil))
	r.GET("/ws", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"ok": true})
	})

	req := httptest.NewRequest(http.MethodGet, "/ws", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", w.Code, w.Body.String())
	}
}

func TestJWTQueryAuth_AllowsValidToken(t *testing.T) {
	gin.SetMode(gin.TestMode)
	secret := "test-secret"

	r := gin.New()
	r.Use(JWTQueryAuth(secret, nil, nil))
	r.GET("/ws", func(c *gin.Context) {
		userID, _ := c.Get("user_id")
		c.JSON(http.StatusOK, gin.H{"user_id": userID})
	})

	req := httptest.NewRequest(http.MethodGet, "/ws?token="+makeToken(t, secret, 42), nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	var body struct {
		UserID int64 `json:"user_id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.UserID != 42 {
		t.Fatalf("expected user_id 42, got %d", body.UserID)
	}
}
