package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

type fakeUserQuerier struct {
	createUserArgs *db.CreateUserParams
	grantRoleArgs  *db.GrantRoleParams

	createUserRow db.CreateUserRow
	grantRoleRow  db.UserRoleGrant
}

func (f *fakeUserQuerier) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
	f.createUserArgs = &arg
	return f.createUserRow, nil
}

func (f *fakeUserQuerier) GetUser(ctx context.Context, id int64) (db.GetUserRow, error) {
	return db.GetUserRow{ID: id, Username: "u", FullName: "n", Active: true}, nil
}

func (f *fakeUserQuerier) ListUsers(ctx context.Context) ([]db.ListUsersRow, error) {
	return []db.ListUsersRow{}, nil
}

func (f *fakeUserQuerier) SetUserActive(ctx context.Context, arg db.SetUserActiveParams) (db.SetUserActiveRow, error) {
	return db.SetUserActiveRow{ID: arg.ID, Active: arg.Active}, nil
}

func (f *fakeUserQuerier) UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.UpdateUserRow, error) {
	return db.UpdateUserRow{ID: arg.ID, Username: arg.Username, FullName: arg.FullName, Active: true}, nil
}

func (f *fakeUserQuerier) GetActiveRolesForUser(ctx context.Context, userID int64) ([]db.UserRoleGrant, error) {
	return []db.UserRoleGrant{}, nil
}

func (f *fakeUserQuerier) GrantRole(ctx context.Context, arg db.GrantRoleParams) (db.UserRoleGrant, error) {
	f.grantRoleArgs = &arg
	return f.grantRoleRow, nil
}

func (f *fakeUserQuerier) RevokeRole(ctx context.Context, arg db.RevokeRoleParams) error {
	return nil
}

type fakeUserCache struct {
	deletedRBAC   []int64
	deletedActive []int64
}

func (f *fakeUserCache) DeleteRoleGrants(ctx context.Context, userID int64) error {
	f.deletedRBAC = append(f.deletedRBAC, userID)
	return nil
}

func (f *fakeUserCache) DeleteUserActive(ctx context.Context, userID int64) error {
	f.deletedActive = append(f.deletedActive, userID)
	return nil
}

func TestUserHandler_CreateUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	q := &fakeUserQuerier{
		createUserRow: db.CreateUserRow{
			ID:                  10,
			Username:            "alice",
			FullName:            "Alice",
			ForcePasswordChange: true,
			Active:              true,
			CreatedAt:           pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:           pgtype.Timestamptz{Time: now, Valid: true},
		},
	}

	h := NewUserHandler(q, nil)
	r := gin.New()
	r.POST("/users", h.CreateUser)

	body := `{"username":"alice","full_name":"Alice","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/users", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if q.createUserArgs == nil {
		t.Fatalf("expected CreateUser called")
	}
	if q.createUserArgs.Username != "alice" || q.createUserArgs.FullName != "Alice" {
		t.Fatalf("unexpected args: %#v", q.createUserArgs)
	}
	if !q.createUserArgs.ForcePasswordChange {
		t.Fatalf("expected ForcePasswordChange true")
	}
	if q.createUserArgs.PasswordHash == "password123" || q.createUserArgs.PasswordHash == "" {
		t.Fatalf("expected password hash set")
	}

	var resp struct {
		Data userResponse `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.Username != "alice" || resp.Data.ID != 10 {
		t.Fatalf("unexpected response: %#v", resp.Data)
	}
}

func TestUserHandler_GrantRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	q := &fakeUserQuerier{
		grantRoleRow: db.UserRoleGrant{
			ID:        1,
			UserID:    2,
			Role:      db.UserRoleTSYSTEMADMIN,
			ScopeType: db.RoleScopeTCOMPANY,
			ScopeID:   pgtype.Int8{Int64: 0, Valid: true},
			GrantedBy: pgtype.Int8{Int64: 99, Valid: true},
			GrantedAt: pgtype.Timestamptz{Time: now, Valid: true},
		},
	}
	cache := &fakeUserCache{}
	h := NewUserHandler(q, cache)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(99))
		c.Next()
	})
	r.POST("/users/:id/roles", h.GrantRole)

	reqBody := `{"role":"SYSTEM_ADMIN","scope_type":"COMPANY"}`
	req := httptest.NewRequest(http.MethodPost, "/users/2/roles", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if q.grantRoleArgs == nil {
		t.Fatalf("expected GrantRole called")
	}
	if q.grantRoleArgs.UserID != 2 || q.grantRoleArgs.Role != db.UserRoleTSYSTEMADMIN {
		t.Fatalf("unexpected args: %#v", q.grantRoleArgs)
	}
	if len(cache.deletedRBAC) != 1 || cache.deletedRBAC[0] != 2 {
		t.Fatalf("expected RBAC cache invalidation, got %#v", cache.deletedRBAC)
	}
}

func TestUserHandler_GrantRole_RequiresScopeIDForNonCompany(t *testing.T) {
	gin.SetMode(gin.TestMode)

	q := &fakeUserQuerier{}
	h := NewUserHandler(q, nil)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(99))
		c.Next()
	})
	r.POST("/users/:id/roles", h.GrantRole)

	reqBody := `{"role":"FACILITY_OPERATOR","scope_type":"FACILITY"}`
	req := httptest.NewRequest(http.MethodPost, "/users/2/roles", bytes.NewBufferString(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d: %s", w.Code, w.Body.String())
	}
}
