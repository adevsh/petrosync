package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
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

type fakePasswordStore struct {
	user              db.GetUserRow
	updatePasswordArg *db.UpdateUserPasswordParams
	forcePasswordID   int64
}

func (f *fakePasswordStore) GetUser(ctx context.Context, id int64) (db.GetUserRow, error) {
	if f.user.ID != id {
		return db.GetUserRow{}, errors.New("not found")
	}
	out := f.user
	if f.forcePasswordID == id {
		out.ForcePasswordChange = true
	}
	return out, nil
}

func (f *fakePasswordStore) GetUserByUsername(ctx context.Context, username string) (db.User, error) {
	return db.User{}, nil
}

func (f *fakePasswordStore) UpdateUserPassword(ctx context.Context, arg db.UpdateUserPasswordParams) error {
	f.updatePasswordArg = &arg
	return nil
}

func (f *fakePasswordStore) SetForcePasswordChange(ctx context.Context, id int64) error {
	f.forcePasswordID = id
	return nil
}

func (f *fakePasswordStore) GetUserPasswordHash(ctx context.Context, id int64) (string, error) {
	return "", nil
}

type fakeResetNotifier struct {
	telegramUserID int64
	message        string
	err            error
}

func (f *fakeResetNotifier) SendTelegramDM(ctx context.Context, telegramUserID int64, message string) error {
	f.telegramUserID = telegramUserID
	f.message = message
	return f.err
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

func TestResetPasswordHandler_HidesTempPasswordWhenTelegramDelivered(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakePasswordStore{
		user: db.GetUserRow{
			ID:             7,
			Username:       "driver1",
			FullName:       "Driver One",
			TelegramUserID: pgtype.Int8{Int64: 777, Valid: true},
			Active:         true,
		},
	}
	notifier := &fakeResetNotifier{}
	h := NewResetPasswordHandler(store, notifier)

	r := gin.New()
	r.POST("/users/:id/reset-password", h.ResetPassword)

	req := httptest.NewRequest(http.MethodPost, "/users/7/reset-password", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}
	if store.updatePasswordArg == nil || store.updatePasswordArg.ID != 7 {
		t.Fatalf("expected password update for user 7, got %#v", store.updatePasswordArg)
	}
	if store.forcePasswordID != 7 {
		t.Fatalf("expected force-password-change for user 7, got %d", store.forcePasswordID)
	}
	if notifier.telegramUserID != 777 {
		t.Fatalf("expected Telegram DM to 777, got %d", notifier.telegramUserID)
	}

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp.Data["temp_password"]; ok {
		t.Fatalf("expected temp_password hidden when Telegram delivery succeeds: %#v", resp.Data)
	}
	if resp.Data["telegram_delivered"] != true {
		t.Fatalf("expected telegram_delivered=true, got %#v", resp.Data["telegram_delivered"])
	}
}

func TestResetPasswordHandler_ShowsTempPasswordWhenTelegramUnavailable(t *testing.T) {
	gin.SetMode(gin.TestMode)

	store := &fakePasswordStore{
		user: db.GetUserRow{
			ID:             9,
			Username:       "driver2",
			FullName:       "Driver Two",
			TelegramUserID: pgtype.Int8{Int64: 999, Valid: true},
			Active:         true,
		},
	}
	notifier := &fakeResetNotifier{err: errors.New("telegram down")}
	h := NewResetPasswordHandler(store, notifier)

	r := gin.New()
	r.POST("/users/:id/reset-password", h.ResetPassword)

	req := httptest.NewRequest(http.MethodPost, "/users/9/reset-password", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if _, ok := resp.Data["temp_password"]; !ok {
		t.Fatalf("expected temp_password fallback when Telegram delivery fails: %#v", resp.Data)
	}
	if resp.Data["telegram_delivered"] != false {
		t.Fatalf("expected telegram_delivered=false, got %#v", resp.Data["telegram_delivered"])
	}
}
