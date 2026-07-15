package handler

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/model"
	"github.com/adevsh/petrosync/internal/service"
)

type fakeDashboardAuthService struct {
	loginResult   *service.DashboardLoginResult
	loginErr      error
	loginUsername string
	loginPassword string

	changeErr     error
	changeUserID  int64
	changeCurrent string
	changeNew     string
}

func (f *fakeDashboardAuthService) LoginDashboard(ctx context.Context, username, password string, sessionTTL time.Duration) (*service.DashboardLoginResult, error) {
	f.loginUsername = username
	f.loginPassword = password
	if f.loginErr != nil {
		return nil, f.loginErr
	}
	return f.loginResult, nil
}

func (f *fakeDashboardAuthService) ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error {
	f.changeUserID = userID
	f.changeCurrent = currentPassword
	f.changeNew = newPassword
	return f.changeErr
}

type fakeDashboardSessionStore struct {
	sessions map[string]model.SessionData
	deleted  []string
}

type fakeDashboardDataQuerier struct {
	companySummary         []db.GetCompanyWideDashboardSummaryRow
	facilitySummary        db.GetFacilityDashboardSummaryRow
	activeTrips            []db.ListActiveTripsWithLatestGPSRow
	refineries             []db.Refinery
	facilities             map[int64]db.GetFacilityRow
	facilitiesByRefinery   map[int64][]db.ListFacilitiesByRefineryRow
	depots                 map[int64]db.GetDepotRow
	allDepots              []db.ListAllActiveDepotsRow
	stations               map[int64]db.GetStationRow
	vehicles               map[int64]db.GetVehicleRow
	allStations            []db.ListAllActiveStationsRow
	stationsByRefinery     map[int64][]db.ListAllActiveStationsByRefineryScopeRow
	stationsByStationScope map[int64][]db.ListAllActiveStationsByStationScopeRow
	stationsByFacility     map[int64][]db.ListStationsServedByFacilityRow
	stationInventory       map[int64][]db.GetStationInventorySnapshotRow
	stationTankAlerts      []db.ListStationTanksBelowReorderThresholdRow
	vehiclesByStatus       map[db.VehicleStatusT][]db.ListVehiclesByStatusRow
	vehiclesByRefinery     map[int64]map[db.VehicleStatusT][]db.ListVehiclesByStatusAndRefineryRow
	vehiclesByFacility     map[int64]map[db.VehicleStatusT][]db.ListVehiclesByStatusAndFacilityRow
	vehiclesByDepot        map[int64]map[db.VehicleStatusT][]db.ListVehiclesByStatusAndDepotRow
	vehiclesWithAttention  []db.ListVehiclesWithMaintenanceOrExpiryDueRow
	openMaintenance        []db.ListAllOpenMaintenanceRow
	maintenanceByVehicle   map[int64][]db.VehicleMaintenanceRecord
	compartmentsByVehicle  map[int64][]db.VehicleCompartment
	tripsByVehicle         map[int64][]db.Trip
	tripsByStatus          map[db.TripStatusT][]db.Trip
	companyErr             error
	facilityErr            error
	activeTripsErr         error
}

func (f *fakeDashboardDataQuerier) GetCompanyWideDashboardSummary(ctx context.Context) ([]db.GetCompanyWideDashboardSummaryRow, error) {
	if f.companyErr != nil {
		return nil, f.companyErr
	}
	return f.companySummary, nil
}

func (f *fakeDashboardDataQuerier) GetFacilityDashboardSummary(ctx context.Context, id int64) (db.GetFacilityDashboardSummaryRow, error) {
	if f.facilityErr != nil {
		return db.GetFacilityDashboardSummaryRow{}, f.facilityErr
	}
	return f.facilitySummary, nil
}

func (f *fakeDashboardDataQuerier) ListActiveTripsWithLatestGPS(ctx context.Context) ([]db.ListActiveTripsWithLatestGPSRow, error) {
	if f.activeTripsErr != nil {
		return nil, f.activeTripsErr
	}
	return f.activeTrips, nil
}

func (f *fakeDashboardDataQuerier) ListAllActiveDepots(ctx context.Context) ([]db.ListAllActiveDepotsRow, error) {
	return f.allDepots, nil
}

func (f *fakeDashboardDataQuerier) GetFacility(ctx context.Context, id int64) (db.GetFacilityRow, error) {
	row, ok := f.facilities[id]
	if !ok {
		return db.GetFacilityRow{}, errors.New("not found")
	}
	return row, nil
}

func (f *fakeDashboardDataQuerier) ListFacilitiesByRefinery(ctx context.Context, refineryID int64) ([]db.ListFacilitiesByRefineryRow, error) {
	return f.facilitiesByRefinery[refineryID], nil
}

func (f *fakeDashboardDataQuerier) ListRefineries(ctx context.Context) ([]db.Refinery, error) {
	return f.refineries, nil
}

func (f *fakeDashboardDataQuerier) GetDepot(ctx context.Context, id int64) (db.GetDepotRow, error) {
	row, ok := f.depots[id]
	if !ok {
		return db.GetDepotRow{}, errors.New("not found")
	}
	return row, nil
}

func (f *fakeDashboardDataQuerier) GetStation(ctx context.Context, id int64) (db.GetStationRow, error) {
	row, ok := f.stations[id]
	if !ok {
		return db.GetStationRow{}, errors.New("not found")
	}
	return row, nil
}

func (f *fakeDashboardDataQuerier) GetVehicle(ctx context.Context, id int64) (db.GetVehicleRow, error) {
	row, ok := f.vehicles[id]
	if !ok {
		return db.GetVehicleRow{}, errors.New("not found")
	}
	return row, nil
}

func (f *fakeDashboardDataQuerier) GetStationInventorySnapshot(ctx context.Context, stationID int64) ([]db.GetStationInventorySnapshotRow, error) {
	return f.stationInventory[stationID], nil
}

func (f *fakeDashboardDataQuerier) ListAllActiveStations(ctx context.Context) ([]db.ListAllActiveStationsRow, error) {
	return f.allStations, nil
}

func (f *fakeDashboardDataQuerier) ListAllActiveStationsByRefineryScope(ctx context.Context, refineryID int64) ([]db.ListAllActiveStationsByRefineryScopeRow, error) {
	return f.stationsByRefinery[refineryID], nil
}

func (f *fakeDashboardDataQuerier) ListAllActiveStationsByStationScope(ctx context.Context, id int64) ([]db.ListAllActiveStationsByStationScopeRow, error) {
	return f.stationsByStationScope[id], nil
}

func (f *fakeDashboardDataQuerier) ListAllCompartmentsByVehicle(ctx context.Context, vehicleID int64) ([]db.VehicleCompartment, error) {
	return f.compartmentsByVehicle[vehicleID], nil
}

func (f *fakeDashboardDataQuerier) ListAllOpenMaintenance(ctx context.Context) ([]db.ListAllOpenMaintenanceRow, error) {
	return f.openMaintenance, nil
}

func (f *fakeDashboardDataQuerier) ListMaintenanceByVehicle(ctx context.Context, vehicleID int64) ([]db.VehicleMaintenanceRecord, error) {
	return f.maintenanceByVehicle[vehicleID], nil
}

func (f *fakeDashboardDataQuerier) ListStationTanksBelowReorderThreshold(ctx context.Context) ([]db.ListStationTanksBelowReorderThresholdRow, error) {
	return f.stationTankAlerts, nil
}

func (f *fakeDashboardDataQuerier) ListStationsServedByFacility(ctx context.Context, facilityID int64) ([]db.ListStationsServedByFacilityRow, error) {
	return f.stationsByFacility[facilityID], nil
}

func (f *fakeDashboardDataQuerier) ListTripsByStatus(ctx context.Context, status db.TripStatusT) ([]db.Trip, error) {
	return f.tripsByStatus[status], nil
}

func (f *fakeDashboardDataQuerier) ListTripsByVehicle(ctx context.Context, arg db.ListTripsByVehicleParams) ([]db.Trip, error) {
	return f.tripsByVehicle[arg.VehicleID], nil
}

func (f *fakeDashboardDataQuerier) ListVehiclesByStatus(ctx context.Context, status db.VehicleStatusT) ([]db.ListVehiclesByStatusRow, error) {
	return f.vehiclesByStatus[status], nil
}

func (f *fakeDashboardDataQuerier) ListVehiclesByStatusAndDepot(ctx context.Context, arg db.ListVehiclesByStatusAndDepotParams) ([]db.ListVehiclesByStatusAndDepotRow, error) {
	if !arg.CurrentDepotID.Valid {
		return nil, nil
	}
	if rowsByStatus, ok := f.vehiclesByDepot[arg.CurrentDepotID.Int64]; ok {
		return rowsByStatus[arg.Status], nil
	}
	return nil, nil
}

func (f *fakeDashboardDataQuerier) ListVehiclesByStatusAndFacility(ctx context.Context, arg db.ListVehiclesByStatusAndFacilityParams) ([]db.ListVehiclesByStatusAndFacilityRow, error) {
	if rowsByStatus, ok := f.vehiclesByFacility[arg.PrimaryFacilityID]; ok {
		return rowsByStatus[arg.Status], nil
	}
	return nil, nil
}

func (f *fakeDashboardDataQuerier) ListVehiclesByStatusAndRefinery(ctx context.Context, arg db.ListVehiclesByStatusAndRefineryParams) ([]db.ListVehiclesByStatusAndRefineryRow, error) {
	if rowsByStatus, ok := f.vehiclesByRefinery[arg.RefineryID]; ok {
		return rowsByStatus[arg.Status], nil
	}
	return nil, nil
}

func (f *fakeDashboardDataQuerier) ListVehiclesWithMaintenanceOrExpiryDue(ctx context.Context) ([]db.ListVehiclesWithMaintenanceOrExpiryDueRow, error) {
	return f.vehiclesWithAttention, nil
}

type fakeDashboardUserQuerier struct {
	listUsers      []db.ListUsersRow
	users          map[int64]db.GetUserRow
	rolesByUser    map[int64][]db.UserRoleGrant
	createUserRow  db.CreateUserRow
	createUserArgs *db.CreateUserParams
	grantRoleRow   db.UserRoleGrant
	grantRoleArgs  *db.GrantRoleParams
	revokeRoleArgs *db.RevokeRoleParams
}

func (f *fakeDashboardUserQuerier) CreateUser(ctx context.Context, arg db.CreateUserParams) (db.CreateUserRow, error) {
	f.createUserArgs = &arg
	if f.createUserRow.ID == 0 {
		return db.CreateUserRow{
			ID:                  1,
			Username:            arg.Username,
			FullName:            arg.FullName,
			ForcePasswordChange: arg.ForcePasswordChange,
			Active:              true,
		}, nil
	}
	return f.createUserRow, nil
}

func (f *fakeDashboardUserQuerier) GetUser(ctx context.Context, id int64) (db.GetUserRow, error) {
	if row, ok := f.users[id]; ok {
		return row, nil
	}
	return db.GetUserRow{}, errors.New("not found")
}

func (f *fakeDashboardUserQuerier) ListUsers(ctx context.Context) ([]db.ListUsersRow, error) {
	return f.listUsers, nil
}

func (f *fakeDashboardUserQuerier) SetUserActive(ctx context.Context, arg db.SetUserActiveParams) (db.SetUserActiveRow, error) {
	row, err := f.GetUser(ctx, arg.ID)
	if err != nil {
		return db.SetUserActiveRow{}, err
	}
	row.Active = arg.Active
	f.users[arg.ID] = row
	return db.SetUserActiveRow{
		ID:                  row.ID,
		Username:            row.Username,
		FullName:            row.FullName,
		TelegramUserID:      row.TelegramUserID,
		TelegramLinkedAt:    row.TelegramLinkedAt,
		ForcePasswordChange: row.ForcePasswordChange,
		Active:              row.Active,
		LastLoginAt:         row.LastLoginAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}, nil
}

func (f *fakeDashboardUserQuerier) UpdateUser(ctx context.Context, arg db.UpdateUserParams) (db.UpdateUserRow, error) {
	row, err := f.GetUser(ctx, arg.ID)
	if err != nil {
		return db.UpdateUserRow{}, err
	}
	row.Username = arg.Username
	row.FullName = arg.FullName
	f.users[arg.ID] = row
	return db.UpdateUserRow{
		ID:                  row.ID,
		Username:            row.Username,
		FullName:            row.FullName,
		TelegramUserID:      row.TelegramUserID,
		TelegramLinkedAt:    row.TelegramLinkedAt,
		ForcePasswordChange: row.ForcePasswordChange,
		Active:              row.Active,
		LastLoginAt:         row.LastLoginAt,
		CreatedAt:           row.CreatedAt,
		UpdatedAt:           row.UpdatedAt,
	}, nil
}

func (f *fakeDashboardUserQuerier) GetActiveRolesForUser(ctx context.Context, userID int64) ([]db.UserRoleGrant, error) {
	return f.rolesByUser[userID], nil
}

func (f *fakeDashboardUserQuerier) GrantRole(ctx context.Context, arg db.GrantRoleParams) (db.UserRoleGrant, error) {
	f.grantRoleArgs = &arg
	if f.grantRoleRow.ID == 0 {
		return db.UserRoleGrant{
			ID:        1,
			UserID:    arg.UserID,
			Role:      arg.Role,
			ScopeType: arg.ScopeType,
			ScopeID:   arg.ScopeID,
			GrantedBy: arg.GrantedBy,
		}, nil
	}
	return f.grantRoleRow, nil
}

func (f *fakeDashboardUserQuerier) RevokeRole(ctx context.Context, arg db.RevokeRoleParams) error {
	f.revokeRoleArgs = &arg
	return nil
}

func newFakeDashboardSessionStore() *fakeDashboardSessionStore {
	return &fakeDashboardSessionStore{
		sessions: make(map[string]model.SessionData),
	}
}

func (f *fakeDashboardSessionStore) GetSession(ctx context.Context, sessionID string) (*model.SessionData, error) {
	session, ok := f.sessions[sessionID]
	if !ok {
		return nil, nil
	}
	copySession := session
	return &copySession, nil
}

func (f *fakeDashboardSessionStore) SaveSession(ctx context.Context, sessionID string, data model.SessionData) error {
	f.sessions[sessionID] = data
	return nil
}

func (f *fakeDashboardSessionStore) DeleteSession(ctx context.Context, sessionID string) error {
	delete(f.sessions, sessionID)
	f.deleted = append(f.deleted, sessionID)
	return nil
}

func newDashboardTestRouter(auth *fakeDashboardAuthService, sessions *fakeDashboardSessionStore, queries *fakeDashboardDataQuerier) *gin.Engine {
	return newDashboardTestRouterWithUserAdmin(auth, sessions, queries, nil, nil)
}

func newDashboardTestRouterWithUserAdmin(auth *fakeDashboardAuthService, sessions *fakeDashboardSessionStore, queries *fakeDashboardDataQuerier, userAdmin *UserHandler, resetPw *ResetPasswordHandler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	dashboard := NewDashboardHandler(auth, sessions, queries, time.Hour, false)
	if userAdmin != nil {
		dashboard = dashboard.WithUserAdmin(userAdmin, resetPw)
	}
	RegisterDashboardRoutes(router, dashboard, sessions)
	return router
}

func numericInt64Exp(v int64, exp int32) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(v), Exp: exp, Valid: true}
}

func mustDecimalFromString(t *testing.T, value string) decimal.Decimal {
	t.Helper()
	out, err := decimal.NewFromString(value)
	if err != nil {
		t.Fatalf("failed to parse decimal %q: %v", value, err)
	}
	return out
}

func newFakeDashboardDataQuerier() *fakeDashboardDataQuerier {
	return &fakeDashboardDataQuerier{
		facilities:             make(map[int64]db.GetFacilityRow),
		facilitiesByRefinery:   make(map[int64][]db.ListFacilitiesByRefineryRow),
		depots:                 make(map[int64]db.GetDepotRow),
		stations:               make(map[int64]db.GetStationRow),
		vehicles:               make(map[int64]db.GetVehicleRow),
		stationsByRefinery:     make(map[int64][]db.ListAllActiveStationsByRefineryScopeRow),
		stationsByStationScope: make(map[int64][]db.ListAllActiveStationsByStationScopeRow),
		stationsByFacility:     make(map[int64][]db.ListStationsServedByFacilityRow),
		stationInventory:       make(map[int64][]db.GetStationInventorySnapshotRow),
		vehiclesByStatus:       make(map[db.VehicleStatusT][]db.ListVehiclesByStatusRow),
		vehiclesByRefinery:     make(map[int64]map[db.VehicleStatusT][]db.ListVehiclesByStatusAndRefineryRow),
		vehiclesByFacility:     make(map[int64]map[db.VehicleStatusT][]db.ListVehiclesByStatusAndFacilityRow),
		vehiclesByDepot:        make(map[int64]map[db.VehicleStatusT][]db.ListVehiclesByStatusAndDepotRow),
		maintenanceByVehicle:   make(map[int64][]db.VehicleMaintenanceRecord),
		compartmentsByVehicle:  make(map[int64][]db.VehicleCompartment),
		tripsByVehicle:         make(map[int64][]db.Trip),
		tripsByStatus:          make(map[db.TripStatusT][]db.Trip),
	}
}

func TestDashboardLoginSetsCookieAndRedirects(t *testing.T) {
	facilityID := int64(7)
	auth := &fakeDashboardAuthService{
		loginResult: &service.DashboardLoginResult{
			SessionID: "sess-1",
			Session: model.SessionData{
				UserID:     42,
				FullName:   "Facility Operator",
				RoleGrants: []model.RoleGrant{{Role: "FACILITY_OPERATOR", ScopeType: "FACILITY", ScopeID: &facilityID}},
				ExpiresAt:  time.Now().Add(time.Hour),
			},
		},
	}
	sessions := newFakeDashboardSessionStore()
	router := newDashboardTestRouter(auth, sessions, &fakeDashboardDataQuerier{})

	form := url.Values{}
	form.Set("username", "operator")
	form.Set("password", "secret123")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/facilities/7" {
		t.Fatalf("expected facility redirect, got %q", got)
	}
	cookie := rec.Result().Cookies()
	if len(cookie) == 0 || cookie[0].Name != dashboardSessionCookie || cookie[0].Value != "sess-1" {
		t.Fatalf("expected dashboard session cookie, got %#v", cookie)
	}
	if !cookie[0].HttpOnly {
		t.Fatalf("expected HttpOnly cookie")
	}
	if cookie[0].SameSite != http.SameSiteStrictMode {
		t.Fatalf("expected SameSiteStrict cookie, got %v", cookie[0].SameSite)
	}
	if auth.loginUsername != "operator" || auth.loginPassword != "secret123" {
		t.Fatalf("unexpected login args: %q %q", auth.loginUsername, auth.loginPassword)
	}
}

func TestDashboardProtectedRoutesRedirectForcedPasswordChange(t *testing.T) {
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-2"] = model.SessionData{
		UserID:              12,
		FullName:            "Admin User",
		RoleGrants:          []model.RoleGrant{{Role: "REFINERY_ADMIN", ScopeType: "REFINERY"}},
		ForcePasswordChange: true,
		ExpiresAt:           time.Now().Add(time.Hour),
	}

	router := newDashboardTestRouter(&fakeDashboardAuthService{}, sessions, &fakeDashboardDataQuerier{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-2"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/change-password" {
		t.Fatalf("expected change-password redirect, got %q", got)
	}
}

func TestDashboardChangePasswordClearsForceFlagAndRedirects(t *testing.T) {
	facilityID := int64(9)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-3"] = model.SessionData{
		UserID:              99,
		FullName:            "Scoped Operator",
		RoleGrants:          []model.RoleGrant{{Role: "FACILITY_OPERATOR", ScopeType: "FACILITY", ScopeID: &facilityID}},
		ForcePasswordChange: true,
		ExpiresAt:           time.Now().Add(time.Hour),
	}

	auth := &fakeDashboardAuthService{}
	router := newDashboardTestRouter(auth, sessions, &fakeDashboardDataQuerier{})

	form := url.Values{}
	form.Set("current_password", "temporary1")
	form.Set("new_password", "changed-password")

	req := httptest.NewRequest(http.MethodPost, "/change-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-3"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/facilities/9" {
		t.Fatalf("expected facility redirect, got %q", got)
	}
	if auth.changeUserID != 99 || auth.changeCurrent != "temporary1" || auth.changeNew != "changed-password" {
		t.Fatalf("unexpected change password args: user=%d current=%q new=%q", auth.changeUserID, auth.changeCurrent, auth.changeNew)
	}
	updated := sessions.sessions["sess-3"]
	if updated.ForcePasswordChange {
		t.Fatalf("expected force_password_change to be cleared")
	}
}

func TestDashboardRootRedirectsByRole(t *testing.T) {
	stationID := int64(11)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-4"] = model.SessionData{
		UserID:     77,
		FullName:   "Station Manager",
		RoleGrants: []model.RoleGrant{{Role: "STATION_MANAGER", ScopeType: "STATION", ScopeID: &stationID}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	router := newDashboardTestRouter(&fakeDashboardAuthService{}, sessions, &fakeDashboardDataQuerier{})

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-4"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusSeeOther {
		t.Fatalf("expected status %d, got %d", http.StatusSeeOther, rec.Code)
	}
	if got := rec.Header().Get("Location"); got != "/stations/11" {
		t.Fatalf("expected station redirect, got %q", got)
	}
}

func TestDashboardLoginRendersInvalidCredentials(t *testing.T) {
	auth := &fakeDashboardAuthService{loginErr: service.ErrInvalidCredentials}
	router := newDashboardTestRouter(auth, newFakeDashboardSessionStore(), &fakeDashboardDataQuerier{})

	form := url.Values{}
	form.Set("username", "bad-user")
	form.Set("password", "wrong")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "invalid username or password") {
		t.Fatalf("expected invalid credential message, got %q", body)
	}
}

func TestDashboardChangePasswordRendersMismatch(t *testing.T) {
	facilityID := int64(5)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-5"] = model.SessionData{
		UserID:              501,
		FullName:            "Operator",
		RoleGrants:          []model.RoleGrant{{Role: "FACILITY_OPERATOR", ScopeType: "FACILITY", ScopeID: &facilityID}},
		ForcePasswordChange: true,
		ExpiresAt:           time.Now().Add(time.Hour),
	}

	auth := &fakeDashboardAuthService{changeErr: service.ErrPasswordMismatch}
	router := newDashboardTestRouter(auth, sessions, &fakeDashboardDataQuerier{})

	form := url.Values{}
	form.Set("current_password", "wrong-current")
	form.Set("new_password", "new-password")

	req := httptest.NewRequest(http.MethodPost, "/change-password", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-5"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "current password is incorrect") {
		t.Fatalf("expected mismatch message, got %q", rec.Body.String())
	}
}

func TestDashboardLoginHandlesUnexpectedError(t *testing.T) {
	auth := &fakeDashboardAuthService{loginErr: errors.New("boom")}
	router := newDashboardTestRouter(auth, newFakeDashboardSessionStore(), &fakeDashboardDataQuerier{})

	form := url.Values{}
	form.Set("username", "operator")
	form.Set("password", "secret123")

	req := httptest.NewRequest(http.MethodPost, "/login", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestDashboardHomeRendersCompanyOverview(t *testing.T) {
	refineryID := int64(3)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-home"] = model.SessionData{
		UserID:     10,
		FullName:   "Refinery Admin",
		RoleGrants: []model.RoleGrant{{Role: "REFINERY_ADMIN", ScopeType: "REFINERY", ScopeID: &refineryID}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	queries := &fakeDashboardDataQuerier{
		companySummary: []db.GetCompanyWideDashboardSummaryRow{
			{FacilityID: 7, FacilityCode: "RU-V", FacilityName: "Balikpapan", RefineryCode: "RU5", ActiveTrips: 4, AvailableVehicles: 8},
		},
		activeTrips: []db.ListActiveTripsWithLatestGPSRow{
			{
				ID:               55,
				Status:           db.TripStatusTINTRANSIT,
				VehicleID:        91,
				DriverID:         11,
				OriginFacilityID: 7,
				PlateNumber:      "KT 1234 AB",
				DriverName:       "Dimas",
				DestinationName:  pgtype.Text{String: "SPBU 01", Valid: true},
				LastLat:          numericInt64Exp(-12345, -4),
				LastLng:          numericInt64Exp(1165432, -4),
				LastSpeedKmh:     numericInt64Exp(355, -1),
				LastGpsAt:        pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 8, 30, 0, 0, time.UTC), Valid: true},
			},
		},
	}
	router := newDashboardTestRouter(&fakeDashboardAuthService{}, sessions, queries)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-home"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Company-wide snapshot for refinery administrators",
		"Balikpapan",
		"/ws/trips/active",
		"active-trips-map",
		"KT 1234 AB",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %q", want, body)
		}
	}
}

func TestDashboardUserListRendersForSystemAdmin(t *testing.T) {
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-users"] = model.SessionData{
		UserID:     1,
		FullName:   "System Admin",
		RoleGrants: []model.RoleGrant{{Role: "SYSTEM_ADMIN", ScopeType: "COMPANY"}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	userQuerier := &fakeDashboardUserQuerier{
		listUsers: []db.ListUsersRow{
			{ID: 42, Username: "alice", FullName: "Alice Example", Active: true, ForcePasswordChange: true},
		},
		users:       map[int64]db.GetUserRow{},
		rolesByUser: map[int64][]db.UserRoleGrant{},
	}

	router := newDashboardTestRouterWithUserAdmin(
		&fakeDashboardAuthService{},
		sessions,
		newFakeDashboardDataQuerier(),
		NewUserHandler(userQuerier, nil),
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-users"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "User Management") || !strings.Contains(body, "Alice Example") || !strings.Contains(body, "/users/42") {
		t.Fatalf("expected user list content, got %q", body)
	}
}

func TestDashboardUserCreatePageRendersForSystemAdmin(t *testing.T) {
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-users-new"] = model.SessionData{
		UserID:     1,
		FullName:   "System Admin",
		RoleGrants: []model.RoleGrant{{Role: "SYSTEM_ADMIN", ScopeType: "COMPANY"}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	userQuerier := &fakeDashboardUserQuerier{
		users:       map[int64]db.GetUserRow{},
		rolesByUser: map[int64][]db.UserRoleGrant{},
	}

	router := newDashboardTestRouterWithUserAdmin(
		&fakeDashboardAuthService{},
		sessions,
		newFakeDashboardDataQuerier(),
		NewUserHandler(userQuerier, nil),
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/users/new", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-users-new"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Create User") || !strings.Contains(body, "create-user-form") {
		t.Fatalf("expected create user page, got %q", body)
	}
}

func TestDashboardUserDetailRendersScopeSuggestions(t *testing.T) {
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-user-detail"] = model.SessionData{
		UserID:     1,
		FullName:   "System Admin",
		RoleGrants: []model.RoleGrant{{Role: "SYSTEM_ADMIN", ScopeType: "COMPANY"}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	queries := newFakeDashboardDataQuerier()
	queries.refineries = []db.Refinery{
		{ID: 7, Code: "RU5", Name: "Balikpapan", RegionCode: "KAL"},
	}
	queries.facilitiesByRefinery[7] = []db.ListFacilitiesByRefineryRow{
		{ID: 11, RefineryID: 7, Code: "BAL", Name: "Balikpapan Terminal"},
	}
	queries.allDepots = []db.ListAllActiveDepotsRow{
		{ID: 21, Code: "DPT-BAL", Name: "Bal Terminal Depot", FacilityCode: "BAL", FacilityName: "Balikpapan Terminal"},
	}
	queries.allStations = []db.ListAllActiveStationsRow{
		{ID: 31, Code: "SPBU-01", Name: "Sepinggan", RegionCode: "KAL", PrimaryFacilityID: 11},
	}

	now := time.Now()
	userQuerier := &fakeDashboardUserQuerier{
		users: map[int64]db.GetUserRow{
			42: {
				ID:                  42,
				Username:            "alice",
				FullName:            "Alice Example",
				Active:              true,
				ForcePasswordChange: true,
				CreatedAt:           pgtype.Timestamptz{Time: now, Valid: true},
				UpdatedAt:           pgtype.Timestamptz{Time: now, Valid: true},
			},
		},
		rolesByUser: map[int64][]db.UserRoleGrant{
			42: {
				{
					ID:        100,
					UserID:    42,
					Role:      db.UserRoleTFACILITYOPERATOR,
					ScopeType: db.RoleScopeTFACILITY,
					ScopeID:   pgtype.Int8{Int64: 11, Valid: true},
				},
			},
		},
	}

	router := newDashboardTestRouterWithUserAdmin(
		&fakeDashboardAuthService{},
		sessions,
		queries,
		NewUserHandler(userQuerier, nil),
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/users/42", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-user-detail"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "Grant Role") ||
		!strings.Contains(body, "Reset Password") ||
		!strings.Contains(body, "RU5 - Balikpapan") ||
		!strings.Contains(body, "SPBU-01 - Sepinggan") {
		t.Fatalf("expected detail page with scope references, got %q", body)
	}
}

func TestDashboardUserPagesForbidNonSystemAdmin(t *testing.T) {
	refineryID := int64(7)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-no-users"] = model.SessionData{
		UserID:     2,
		FullName:   "Refinery Admin",
		RoleGrants: []model.RoleGrant{{Role: "REFINERY_ADMIN", ScopeType: "REFINERY", ScopeID: &refineryID}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	userQuerier := &fakeDashboardUserQuerier{
		users:       map[int64]db.GetUserRow{},
		rolesByUser: map[int64][]db.UserRoleGrant{},
	}

	router := newDashboardTestRouterWithUserAdmin(
		&fakeDashboardAuthService{},
		sessions,
		newFakeDashboardDataQuerier(),
		NewUserHandler(userQuerier, nil),
		nil,
	)

	req := httptest.NewRequest(http.MethodGet, "/users", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-no-users"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected status %d, got %d", http.StatusForbidden, rec.Code)
	}
	if !strings.Contains(rec.Body.String(), "forbidden") {
		t.Fatalf("expected forbidden body, got %q", rec.Body.String())
	}
}

func TestDashboardUserResetPasswordAPIUsesDashboardSessionAuth(t *testing.T) {
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-reset-user"] = model.SessionData{
		UserID:     1,
		FullName:   "System Admin",
		RoleGrants: []model.RoleGrant{{Role: "SYSTEM_ADMIN", ScopeType: "COMPANY"}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	userQuerier := &fakeDashboardUserQuerier{
		users:       map[int64]db.GetUserRow{},
		rolesByUser: map[int64][]db.UserRoleGrant{},
	}
	passwordStore := &fakePasswordStore{
		user: db.GetUserRow{
			ID:             7,
			Username:       "driver1",
			FullName:       "Driver One",
			TelegramUserID: pgtype.Int8{Int64: 777, Valid: true},
			Active:         true,
		},
	}
	notifier := &fakeResetNotifier{}

	router := newDashboardTestRouterWithUserAdmin(
		&fakeDashboardAuthService{},
		sessions,
		newFakeDashboardDataQuerier(),
		NewUserHandler(userQuerier, nil),
		NewResetPasswordHandler(passwordStore, notifier),
	)

	req := httptest.NewRequest(http.MethodPost, "/dashboard-api/users/7/reset-password", strings.NewReader(`{"user_id":7}`))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-reset-user"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, rec.Code, rec.Body.String())
	}

	var resp struct {
		Data map[string]interface{} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data["telegram_delivered"] != true {
		t.Fatalf("expected telegram_delivered=true, got %#v", resp.Data["telegram_delivered"])
	}
	if _, ok := resp.Data["temp_password"]; ok {
		t.Fatalf("expected temp_password hidden on Telegram delivery, got %#v", resp.Data)
	}
}

func TestDashboardFacilityLandingRendersScopedOverview(t *testing.T) {
	facilityID := int64(7)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-facility"] = model.SessionData{
		UserID:     12,
		FullName:   "Facility Operator",
		RoleGrants: []model.RoleGrant{{Role: "FACILITY_OPERATOR", ScopeType: "FACILITY", ScopeID: &facilityID}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	queries := &fakeDashboardDataQuerier{
		facilitySummary: db.GetFacilityDashboardSummaryRow{
			FacilityID:            facilityID,
			FacilityName:          "Balikpapan Terminal",
			ActiveTrips:           2,
			AvailableVehicles:     5,
			VehiclesInMaintenance: 1,
		},
		activeTrips: []db.ListActiveTripsWithLatestGPSRow{
			{
				ID:               55,
				Status:           db.TripStatusTUNLOADING,
				VehicleID:        91,
				DriverID:         11,
				OriginFacilityID: facilityID,
				PlateNumber:      "KT 1234 AB",
				DriverName:       "Dimas",
				DestinationName:  pgtype.Text{String: "SPBU 01", Valid: true},
				LastLat:          numericInt64Exp(-12345, -4),
				LastLng:          numericInt64Exp(1165432, -4),
				LastGpsAt:        pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 8, 30, 0, 0, time.UTC), Valid: true},
			},
			{
				ID:               56,
				Status:           db.TripStatusTINTRANSIT,
				VehicleID:        92,
				DriverID:         12,
				OriginFacilityID: 99,
				PlateNumber:      "KT 9999 ZZ",
				DriverName:       "Raka",
			},
		},
	}
	router := newDashboardTestRouter(&fakeDashboardAuthService{}, sessions, queries)

	req := httptest.NewRequest(http.MethodGet, "/facilities/7", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-facility"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Balikpapan Terminal",
		"Facility Live Map",
		"KT 1234 AB",
		"/ws/trips/active",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %q", want, body)
		}
	}
	if strings.Contains(body, "KT 9999 ZZ") {
		t.Fatalf("expected facility view to exclude other facility trip, got %q", body)
	}
}

func TestDashboardStationsListRendersScopedInventoryAlerts(t *testing.T) {
	facilityID := int64(7)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-stations"] = model.SessionData{
		UserID:     14,
		FullName:   "Facility Operator",
		RoleGrants: []model.RoleGrant{{Role: "FACILITY_OPERATOR", ScopeType: "FACILITY", ScopeID: &facilityID}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	queries := newFakeDashboardDataQuerier()
	queries.facilities[facilityID] = db.GetFacilityRow{ID: facilityID, Name: "Balikpapan Terminal"}
	queries.stationsByFacility[facilityID] = []db.ListStationsServedByFacilityRow{
		{
			ID:                19,
			Code:              "SPBU-01",
			Name:              "SPBU 01",
			RegionCode:        "KALTIM",
			PrimaryFacilityID: facilityID,
			ContactName:       pgtype.Text{String: "Siti", Valid: true},
		},
	}
	queries.stationTankAlerts = []db.ListStationTanksBelowReorderThresholdRow{
		{StationID: 19},
	}

	router := newDashboardTestRouter(&fakeDashboardAuthService{}, sessions, queries)

	req := httptest.NewRequest(http.MethodGet, "/stations", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-stations"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"SPBU 01",
		"Balikpapan Terminal",
		"1 tank below reorder level",
		"/stations/19",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %q", want, body)
		}
	}
}

func TestDashboardStationLandingRendersInventoryAndDeliveryHistory(t *testing.T) {
	facilityID := int64(7)
	stationID := int64(19)
	vehicleID := int64(91)
	driverID := int64(33)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-station-detail"] = model.SessionData{
		UserID:     19,
		FullName:   "Station Manager",
		RoleGrants: []model.RoleGrant{{Role: "STATION_MANAGER", ScopeType: "STATION", ScopeID: &stationID}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	queries := newFakeDashboardDataQuerier()
	queries.facilities[facilityID] = db.GetFacilityRow{ID: facilityID, Name: "Balikpapan Terminal"}
	queries.stations[stationID] = db.GetStationRow{
		ID:                stationID,
		Code:              "SPBU-01",
		Name:              "SPBU 01",
		RegionCode:        "KALTIM",
		PrimaryFacilityID: facilityID,
		Address:           pgtype.Text{String: "Jl. Raya 1", Valid: true},
		ContactName:       pgtype.Text{String: "Siti", Valid: true},
		ContactPhone:      pgtype.Text{String: "0812", Valid: true},
		SpbuLicenseNumber: "LIC-01",
	}
	queries.stationInventory[stationID] = []db.GetStationInventorySnapshotRow{
		{
			StationID:         stationID,
			TankCode:          "T-01",
			FuelName:          "Pertalite",
			FuelCategory:      db.FuelCategoryTGASOLINE,
			CurrentVolumeL:    numericInt64Exp(3500, 0),
			CapacityL:         numericInt64Exp(10000, 0),
			ReorderThresholdL: numericInt64Exp(4000, 0),
			FillPct:           mustDecimalFromString(t, "35.0"),
			NeedsReorder:      true,
		},
	}
	queries.tripsByStatus[db.TripStatusTDELIVERED] = []db.Trip{
		{
			ID:                   502,
			DoID:                 102,
			VehicleID:            vehicleID,
			DriverID:             driverID,
			Status:               db.TripStatusTDELIVERED,
			OriginFacilityID:     facilityID,
			DestinationType:      db.DestinationTypeTSTATION,
			DestinationStationID: pgtype.Int8{Int64: stationID, Valid: true},
			CompletedAt:          pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 9, 30, 0, 0, time.UTC), Valid: true},
			CreatedAt:            pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC), Valid: true},
		},
	}

	workflowData := newFakeDashboardWorkflowQuerier()
	workflowData.vehicles[vehicleID] = db.GetVehicleRow{ID: vehicleID, PlateNumber: "KT 1234 AB"}
	workflowData.drivers[driverID] = db.GetDriverRow{ID: driverID, FullName: "Dimas"}
	workflowData.activeTripsByStation[stationID] = []db.ListActiveTripsByStationScopeRow{
		{
			ID:                   501,
			Status:               db.TripStatusTINTRANSIT,
			VehicleID:            vehicleID,
			DriverID:             driverID,
			OriginFacilityID:     facilityID,
			DestinationStationID: pgtype.Int8{Int64: stationID, Valid: true},
			DepartedAt:           pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 8, 45, 0, 0, time.UTC), Valid: true},
			PlateNumber:          "KT 1234 AB",
			DriverName:           "Dimas",
		},
	}

	router := newDashboardWorkflowRouter(&fakeDashboardAuthService{}, sessions, queries, workflowData, &fakeDashboardWorkflowService{}, &fakeDashboardTripPhotoLister{})

	req := httptest.NewRequest(http.MethodGet, "/stations/19", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-station-detail"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Tank Levels",
		"Delivery History",
		"Pertalite",
		"35.0%",
		"Trip #501",
		"Trip #502",
		"KT 1234 AB",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %q", want, body)
		}
	}
}

func TestDashboardFleetLandingRendersVehiclesAndMaintenanceContext(t *testing.T) {
	facilityID := int64(7)
	depotID := int64(5)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-fleet"] = model.SessionData{
		UserID:     31,
		FullName:   "Facility Manager",
		RoleGrants: []model.RoleGrant{{Role: "FACILITY_MANAGER", ScopeType: "FACILITY", ScopeID: &facilityID}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	queries := newFakeDashboardDataQuerier()
	queries.facilities[facilityID] = db.GetFacilityRow{ID: facilityID, Name: "Balikpapan Terminal"}
	queries.depots[depotID] = db.GetDepotRow{ID: depotID, Name: "Depot A", PrimaryFacilityID: facilityID}
	queries.vehiclesByFacility[facilityID] = map[db.VehicleStatusT][]db.ListVehiclesByStatusAndFacilityRow{
		db.VehicleStatusTAVAILABLE: {
			{
				ID:                91,
				PlateNumber:       "KT 1234 AB",
				Model:             pgtype.Text{String: "Isuzu", Valid: true},
				Status:            db.VehicleStatusTAVAILABLE,
				CurrentDepotID:    pgtype.Int8{Int64: depotID, Valid: true},
				TotalCapacityL:    numericInt64Exp(16000, 0),
				NextInspectionDue: pgtype.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), Valid: true},
			},
		},
		db.VehicleStatusTUNDERMAINTENANCE: {
			{
				ID:                92,
				PlateNumber:       "KT 9876 ZZ",
				Model:             pgtype.Text{String: "Hino", Valid: true},
				Status:            db.VehicleStatusTUNDERMAINTENANCE,
				CurrentDepotID:    pgtype.Int8{Int64: depotID, Valid: true},
				TotalCapacityL:    numericInt64Exp(24000, 0),
				NextInspectionDue: pgtype.Date{Time: time.Date(2026, 7, 28, 0, 0, 0, 0, time.UTC), Valid: true},
			},
		},
	}
	queries.openMaintenance = []db.ListAllOpenMaintenanceRow{
		{
			VehicleID:         92,
			PlateNumber:       "KT 9876 ZZ",
			DepotName:         "Depot A",
			MaintenanceType:   "BRAKE_SERVICE",
			Description:       pgtype.Text{String: "Brake inspection", Valid: true},
			StartedAt:         pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC), Valid: true},
			EstimatedReturnAt: pgtype.Timestamptz{Time: time.Date(2026, 7, 16, 8, 0, 0, 0, time.UTC), Valid: true},
		},
	}
	queries.vehiclesWithAttention = []db.ListVehiclesWithMaintenanceOrExpiryDueRow{
		{
			ID:          92,
			PlateNumber: "KT 9876 ZZ",
			DepotName:   "Depot A",
			NoticeType:  "UNDER_MAINTENANCE",
		},
	}

	router := newDashboardTestRouter(&fakeDashboardAuthService{}, sessions, queries)

	req := httptest.NewRequest(http.MethodGet, "/fleet", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-fleet"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Fleet Overview",
		"KT 1234 AB",
		"KT 9876 ZZ",
		"Open Maintenance",
		"Attention Board",
		"Brake inspection",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %q", want, body)
		}
	}
}

func TestDashboardVehicleDetailRendersMaintenanceAndTrips(t *testing.T) {
	facilityID := int64(7)
	depotID := int64(5)
	stationID := int64(19)
	vehicleID := int64(91)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-vehicle"] = model.SessionData{
		UserID:     41,
		FullName:   "Facility Operator",
		RoleGrants: []model.RoleGrant{{Role: "FACILITY_OPERATOR", ScopeType: "FACILITY", ScopeID: &facilityID}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	queries := newFakeDashboardDataQuerier()
	queries.facilities[facilityID] = db.GetFacilityRow{ID: facilityID, Name: "Balikpapan Terminal"}
	queries.depots[depotID] = db.GetDepotRow{ID: depotID, Name: "Depot A", PrimaryFacilityID: facilityID}
	queries.vehicles[vehicleID] = db.GetVehicleRow{
		ID:                vehicleID,
		PlateNumber:       "KT 1234 AB",
		Model:             pgtype.Text{String: "Isuzu", Valid: true},
		Status:            db.VehicleStatusTAVAILABLE,
		CurrentDepotID:    pgtype.Int8{Int64: depotID, Valid: true},
		ManufactureYear:   pgtype.Int2{Int16: 2022, Valid: true},
		TotalCapacityL:    numericInt64Exp(16000, 0),
		TareWeightKg:      numericInt64Exp(8000, 0),
		KeurNumber:        pgtype.Text{String: "KR-1", Valid: true},
		KeurExpiry:        pgtype.Date{Time: time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC), Valid: true},
		NextInspectionDue: pgtype.Date{Time: time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC), Valid: true},
		LastAssignedAt:    pgtype.Timestamptz{Time: time.Date(2026, 7, 14, 8, 0, 0, 0, time.UTC), Valid: true},
		Notes:             pgtype.Text{String: "Ready for dispatch", Valid: true},
	}
	queries.maintenanceByVehicle[vehicleID] = []db.VehicleMaintenanceRecord{
		{
			VehicleID:         vehicleID,
			MaintenanceType:   "TIRE_CHECK",
			Description:       pgtype.Text{String: "Tire pressure adjusted", Valid: true},
			StartedAt:         pgtype.Timestamptz{Time: time.Date(2026, 7, 10, 8, 0, 0, 0, time.UTC), Valid: true},
			EstimatedReturnAt: pgtype.Timestamptz{Time: time.Date(2026, 7, 10, 12, 0, 0, 0, time.UTC), Valid: true},
			CompletedAt:       pgtype.Timestamptz{Time: time.Date(2026, 7, 10, 11, 0, 0, 0, time.UTC), Valid: true},
			Notes:             pgtype.Text{String: "Completed same day", Valid: true},
		},
	}
	queries.compartmentsByVehicle[vehicleID] = []db.VehicleCompartment{
		{
			VehicleID:         vehicleID,
			CompartmentNumber: 1,
			FuelTypeCode:      pgtype.Text{String: "SOLAR", Valid: true},
			CapacityL:         numericInt64Exp(8000, 0),
		},
	}
	queries.stations[stationID] = db.GetStationRow{ID: stationID, Name: "SPBU 01"}
	queries.tripsByVehicle[vehicleID] = []db.Trip{
		{
			ID:                   501,
			DoID:                 101,
			VehicleID:            vehicleID,
			Status:               db.TripStatusTDELIVERED,
			DestinationType:      db.DestinationTypeTSTATION,
			DestinationStationID: pgtype.Int8{Int64: stationID, Valid: true},
			OriginFacilityID:     facilityID,
			DepartedAt:           pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 7, 0, 0, 0, time.UTC), Valid: true},
			CompletedAt:          pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 10, 0, 0, 0, time.UTC), Valid: true},
		},
	}

	router := newDashboardTestRouter(&fakeDashboardAuthService{}, sessions, queries)

	req := httptest.NewRequest(http.MethodGet, "/fleet/vehicles/91", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-vehicle"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Maintenance History",
		"Compartments",
		"Recent Trips",
		"SPBU 01",
		"Tire pressure adjusted",
		"Compartment 1",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %q", want, body)
		}
	}
}

type fakeDashboardWorkflowQuerier struct {
	facilities             map[int64]db.GetFacilityRow
	stations               map[int64]db.GetStationRow
	vehicles               map[int64]db.GetVehicleRow
	drivers                map[int64]db.GetDriverRow
	deliveryOrders         map[int64]db.DeliveryOrder
	deliveryOrderItems     map[int64][]db.ListDOItemsByDORow
	facilityDeliveryOrders map[int64][]db.DeliveryOrder
	facilityVehicles       map[int64][]db.ListVehiclesByStatusAndFacilityRow
	depotDrivers           map[int64][]db.ListDriversByDepotRow
	trips                  map[int64]db.GetTripWithDetailsRow
	tripByDO               map[int64]db.GetTripByDORow
	activeTrips            []db.ListActiveTripsRow
	activeTripsByRefinery  map[int64][]db.ListActiveTripsByRefineryScopeRow
	activeTripsByFacility  map[int64][]db.ListActiveTripsByFacilityScopeRow
	activeTripsByStation   map[int64][]db.ListActiveTripsByStationScopeRow
	tripEvents             map[int64][]db.TripEvent
	tripSeals              map[int64][]db.ListSealsByTripRow
	weightBridge           map[int64][]db.WeightBridgeReading
	assignResult           db.DeliveryOrder
	assignErr              error
	createdWeightBridge    []db.CreateWeightBridgeReadingParams
}

func newFakeDashboardWorkflowQuerier() *fakeDashboardWorkflowQuerier {
	return &fakeDashboardWorkflowQuerier{
		facilities:             make(map[int64]db.GetFacilityRow),
		stations:               make(map[int64]db.GetStationRow),
		vehicles:               make(map[int64]db.GetVehicleRow),
		drivers:                make(map[int64]db.GetDriverRow),
		deliveryOrders:         make(map[int64]db.DeliveryOrder),
		deliveryOrderItems:     make(map[int64][]db.ListDOItemsByDORow),
		facilityDeliveryOrders: make(map[int64][]db.DeliveryOrder),
		facilityVehicles:       make(map[int64][]db.ListVehiclesByStatusAndFacilityRow),
		depotDrivers:           make(map[int64][]db.ListDriversByDepotRow),
		trips:                  make(map[int64]db.GetTripWithDetailsRow),
		tripByDO:               make(map[int64]db.GetTripByDORow),
		activeTripsByRefinery:  make(map[int64][]db.ListActiveTripsByRefineryScopeRow),
		activeTripsByFacility:  make(map[int64][]db.ListActiveTripsByFacilityScopeRow),
		activeTripsByStation:   make(map[int64][]db.ListActiveTripsByStationScopeRow),
		tripEvents:             make(map[int64][]db.TripEvent),
		tripSeals:              make(map[int64][]db.ListSealsByTripRow),
		weightBridge:           make(map[int64][]db.WeightBridgeReading),
	}
}

func (f *fakeDashboardWorkflowQuerier) GetFacility(ctx context.Context, id int64) (db.GetFacilityRow, error) {
	row, ok := f.facilities[id]
	if !ok {
		return db.GetFacilityRow{}, errors.New("not found")
	}
	return row, nil
}

func (f *fakeDashboardWorkflowQuerier) GetStation(ctx context.Context, id int64) (db.GetStationRow, error) {
	row, ok := f.stations[id]
	if !ok {
		return db.GetStationRow{}, errors.New("not found")
	}
	return row, nil
}

func (f *fakeDashboardWorkflowQuerier) GetVehicle(ctx context.Context, id int64) (db.GetVehicleRow, error) {
	row, ok := f.vehicles[id]
	if !ok {
		return db.GetVehicleRow{}, errors.New("not found")
	}
	return row, nil
}

func (f *fakeDashboardWorkflowQuerier) GetDriver(ctx context.Context, id int64) (db.GetDriverRow, error) {
	row, ok := f.drivers[id]
	if !ok {
		return db.GetDriverRow{}, errors.New("not found")
	}
	return row, nil
}

func (f *fakeDashboardWorkflowQuerier) GetDeliveryOrder(ctx context.Context, id int64) (db.DeliveryOrder, error) {
	row, ok := f.deliveryOrders[id]
	if !ok {
		return db.DeliveryOrder{}, errors.New("not found")
	}
	return row, nil
}

func (f *fakeDashboardWorkflowQuerier) GetTripByDO(ctx context.Context, doID int64) (db.GetTripByDORow, error) {
	row, ok := f.tripByDO[doID]
	if !ok {
		return db.GetTripByDORow{}, errors.New("not found")
	}
	return row, nil
}

func (f *fakeDashboardWorkflowQuerier) GetTripWithDetails(ctx context.Context, id int64) (db.GetTripWithDetailsRow, error) {
	row, ok := f.trips[id]
	if !ok {
		return db.GetTripWithDetailsRow{}, errors.New("not found")
	}
	return row, nil
}

func (f *fakeDashboardWorkflowQuerier) ListDOsByOriginFacility(ctx context.Context, originFacilityID int64) ([]db.DeliveryOrder, error) {
	return f.facilityDeliveryOrders[originFacilityID], nil
}

func (f *fakeDashboardWorkflowQuerier) ListDOItemsByDO(ctx context.Context, doID int64) ([]db.ListDOItemsByDORow, error) {
	return f.deliveryOrderItems[doID], nil
}

func (f *fakeDashboardWorkflowQuerier) ListVehiclesByStatusAndFacility(ctx context.Context, arg db.ListVehiclesByStatusAndFacilityParams) ([]db.ListVehiclesByStatusAndFacilityRow, error) {
	return f.facilityVehicles[arg.PrimaryFacilityID], nil
}

func (f *fakeDashboardWorkflowQuerier) ListDriversByDepot(ctx context.Context, homeDepotID pgtype.Int8) ([]db.ListDriversByDepotRow, error) {
	if !homeDepotID.Valid {
		return nil, nil
	}
	return f.depotDrivers[homeDepotID.Int64], nil
}

func (f *fakeDashboardWorkflowQuerier) AssignVehicleAndDriverToDO(ctx context.Context, arg db.AssignVehicleAndDriverToDOParams) (db.DeliveryOrder, error) {
	if f.assignErr != nil {
		return db.DeliveryOrder{}, f.assignErr
	}
	if f.assignResult.ID != 0 {
		f.deliveryOrders[arg.ID] = f.assignResult
		return f.assignResult, nil
	}
	row := f.deliveryOrders[arg.ID]
	row.Status = db.DoStatusTASSIGNED
	row.AssignedVehicleID = arg.AssignedVehicleID
	row.AssignedDriverID = arg.AssignedDriverID
	f.deliveryOrders[arg.ID] = row
	return row, nil
}

func (f *fakeDashboardWorkflowQuerier) ListActiveTrips(ctx context.Context) ([]db.ListActiveTripsRow, error) {
	return f.activeTrips, nil
}

func (f *fakeDashboardWorkflowQuerier) ListActiveTripsByRefineryScope(ctx context.Context, refineryID int64) ([]db.ListActiveTripsByRefineryScopeRow, error) {
	return f.activeTripsByRefinery[refineryID], nil
}

func (f *fakeDashboardWorkflowQuerier) ListActiveTripsByFacilityScope(ctx context.Context, originFacilityID int64) ([]db.ListActiveTripsByFacilityScopeRow, error) {
	return f.activeTripsByFacility[originFacilityID], nil
}

func (f *fakeDashboardWorkflowQuerier) ListActiveTripsByStationScope(ctx context.Context, destinationStationID pgtype.Int8) ([]db.ListActiveTripsByStationScopeRow, error) {
	if !destinationStationID.Valid {
		return nil, nil
	}
	return f.activeTripsByStation[destinationStationID.Int64], nil
}

func (f *fakeDashboardWorkflowQuerier) ListTripEventsByTrip(ctx context.Context, tripID int64) ([]db.TripEvent, error) {
	return f.tripEvents[tripID], nil
}

func (f *fakeDashboardWorkflowQuerier) ListSealsByTrip(ctx context.Context, tripID int64) ([]db.ListSealsByTripRow, error) {
	return f.tripSeals[tripID], nil
}

func (f *fakeDashboardWorkflowQuerier) ListWeightBridgeReadingsByTrip(ctx context.Context, tripID pgtype.Int8) ([]db.WeightBridgeReading, error) {
	if !tripID.Valid {
		return nil, nil
	}
	return f.weightBridge[tripID.Int64], nil
}

func (f *fakeDashboardWorkflowQuerier) CreateWeightBridgeReading(ctx context.Context, arg db.CreateWeightBridgeReadingParams) (db.WeightBridgeReading, error) {
	f.createdWeightBridge = append(f.createdWeightBridge, arg)
	reading := db.WeightBridgeReading{
		ID:             int64(len(f.createdWeightBridge)),
		TripID:         arg.TripID,
		VehicleID:      arg.VehicleID,
		ReadingType:    arg.ReadingType,
		WeightKg:       arg.WeightKg,
		Method:         arg.Method,
		ApprovalStatus: db.ApprovalStatusTPENDING,
		RecordedBy:     arg.RecordedBy,
		Notes:          arg.Notes,
	}
	if arg.TripID.Valid {
		f.weightBridge[arg.TripID.Int64] = append(f.weightBridge[arg.TripID.Int64], reading)
	}
	return reading, nil
}

type fakeDashboardWorkflowService struct {
	approveResult db.DeliveryOrder
	approveErr    error
	approveDOID   int64
	approveUserID int64
	onApprove     func(doID, userID int64)
}

func (f *fakeDashboardWorkflowService) ApproveDeliveryOrder(ctx context.Context, doID, userID int64) (db.DeliveryOrder, error) {
	f.approveDOID = doID
	f.approveUserID = userID
	if f.onApprove != nil {
		f.onApprove(doID, userID)
	}
	if f.approveErr != nil {
		return db.DeliveryOrder{}, f.approveErr
	}
	return f.approveResult, nil
}

type fakeDashboardTripPhotoLister struct {
	photos map[int64][]service.TripPhotoWithURL
}

func (f *fakeDashboardTripPhotoLister) ListTripPhotosWithURLs(ctx context.Context, tripID int64) ([]service.TripPhotoWithURL, error) {
	return f.photos[tripID], nil
}

func newDashboardWorkflowRouter(auth *fakeDashboardAuthService, sessions *fakeDashboardSessionStore, queries *fakeDashboardDataQuerier, workflowData *fakeDashboardWorkflowQuerier, workflow *fakeDashboardWorkflowService, photos *fakeDashboardTripPhotoLister) *gin.Engine {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	dashboard := NewDashboardHandler(auth, sessions, queries, time.Hour, false).
		WithWorkflowPages(workflowData, workflow, photos, nil)
	RegisterDashboardRoutes(router, dashboard, sessions)
	return router
}

func TestDashboardFacilityDeliveryOrdersRendersQueue(t *testing.T) {
	facilityID := int64(7)
	stationID := int64(19)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-do"] = model.SessionData{
		UserID:     21,
		FullName:   "Facility Operator",
		RoleGrants: []model.RoleGrant{{Role: "FACILITY_OPERATOR", ScopeType: "FACILITY", ScopeID: &facilityID}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	workflowData := newFakeDashboardWorkflowQuerier()
	workflowData.facilities[facilityID] = db.GetFacilityRow{ID: facilityID, Name: "Balikpapan Terminal"}
	workflowData.stations[stationID] = db.GetStationRow{ID: stationID, Name: "SPBU 01"}
	workflowData.facilityDeliveryOrders[facilityID] = []db.DeliveryOrder{
		{
			ID:                   101,
			DoNumber:             "DO-001",
			Status:               db.DoStatusTPENDINGAPPROVAL,
			OriginFacilityID:     facilityID,
			DestinationType:      db.DestinationTypeTSTATION,
			DestinationStationID: pgtype.Int8{Int64: stationID, Valid: true},
			ScheduledDate:        pgtype.Date{Time: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC), Valid: true},
		},
	}
	workflowData.deliveryOrders[101] = workflowData.facilityDeliveryOrders[facilityID][0]

	router := newDashboardWorkflowRouter(&fakeDashboardAuthService{}, sessions, &fakeDashboardDataQuerier{}, workflowData, &fakeDashboardWorkflowService{}, &fakeDashboardTripPhotoLister{})

	req := httptest.NewRequest(http.MethodGet, "/facilities/7/delivery-orders", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-do"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Balikpapan Terminal Delivery Orders",
		"DO-001",
		"Refresh Queue",
		"/dashboard-partials/delivery-orders/101/approve-row",
		"SPBU 01",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %q", want, body)
		}
	}
}

func TestDashboardApproveDeliveryOrderDetailFragmentUpdatesInPlace(t *testing.T) {
	facilityID := int64(7)
	stationID := int64(19)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-approve"] = model.SessionData{
		UserID:     42,
		FullName:   "Facility Manager",
		RoleGrants: []model.RoleGrant{{Role: "FACILITY_MANAGER", ScopeType: "FACILITY", ScopeID: &facilityID}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	workflowData := newFakeDashboardWorkflowQuerier()
	workflowData.facilities[facilityID] = db.GetFacilityRow{ID: facilityID, Name: "Balikpapan Terminal"}
	workflowData.stations[stationID] = db.GetStationRow{ID: stationID, Name: "SPBU 01"}
	workflowData.deliveryOrders[101] = db.DeliveryOrder{
		ID:                   101,
		DoNumber:             "DO-001",
		Status:               db.DoStatusTPENDINGAPPROVAL,
		OriginFacilityID:     facilityID,
		DestinationType:      db.DestinationTypeTSTATION,
		DestinationStationID: pgtype.Int8{Int64: stationID, Valid: true},
		ScheduledDate:        pgtype.Date{Time: time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC), Valid: true},
		RaisedBy:             15,
	}
	workflowData.deliveryOrderItems[101] = []db.ListDOItemsByDORow{
		{
			FuelTypeCode:     "SOLAR",
			RequestedVolumeL: numericInt64Exp(16000, 0),
			AllocatedVolumeL: numericInt64Exp(16000, 0),
		},
	}

	approved := workflowData.deliveryOrders[101]
	approved.Status = db.DoStatusTAPPROVED
	approved.ApprovedAt = pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 9, 0, 0, 0, time.UTC), Valid: true}

	workflow := &fakeDashboardWorkflowService{
		approveResult: approved,
		onApprove: func(doID, userID int64) {
			workflowData.deliveryOrders[doID] = approved
		},
	}

	router := newDashboardWorkflowRouter(&fakeDashboardAuthService{}, sessions, &fakeDashboardDataQuerier{}, workflowData, workflow, &fakeDashboardTripPhotoLister{})

	req := httptest.NewRequest(http.MethodPost, "/dashboard-partials/delivery-orders/101/approve-detail", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-approve"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	if workflow.approveDOID != 101 || workflow.approveUserID != 42 {
		t.Fatalf("unexpected approve args: do=%d user=%d", workflow.approveDOID, workflow.approveUserID)
	}

	body := rec.Body.String()
	if !strings.Contains(body, "Approved") {
		t.Fatalf("expected approved status, got %q", body)
	}
	if strings.Contains(body, "Approve</span>") {
		t.Fatalf("expected approve button to disappear after approval, got %q", body)
	}
}

func TestDashboardTripDetailRendersOperationalSections(t *testing.T) {
	facilityID := int64(7)
	stationID := int64(19)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-trip"] = model.SessionData{
		UserID:     77,
		FullName:   "Facility Operator",
		RoleGrants: []model.RoleGrant{{Role: "FACILITY_OPERATOR", ScopeType: "FACILITY", ScopeID: &facilityID}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	workflowData := newFakeDashboardWorkflowQuerier()
	workflowData.trips[501] = db.GetTripWithDetailsRow{
		ID:                   501,
		DoID:                 101,
		VehicleID:            91,
		Status:               db.TripStatusTINTRANSIT,
		OriginFacilityID:     facilityID,
		DestinationStationID: pgtype.Int8{Int64: stationID, Valid: true},
		PlateNumber:          "KT 1234 AB",
		DriverName:           "Dimas",
		DriverTelegramID:     pgtype.Int8{Int64: 998877, Valid: true},
		DestinationStationName: pgtype.Text{
			String: "SPBU 01",
			Valid:  true,
		},
		OriginFacilityName: "Balikpapan Terminal",
		DepartedAt:         pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC), Valid: true},
	}
	payload, _ := json.Marshal(map[string]any{"note": "left facility"})
	workflowData.tripEvents[501] = []db.TripEvent{
		{
			ID:             1,
			TripID:         501,
			EventUuid:      uuid.New(),
			EventType:      db.TripEventTypeTDEPARTEDFACILITY,
			EventTimestamp: pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 8, 0, 0, 0, time.UTC), Valid: true},
			ActorUserID:    pgtype.Int8{Int64: 77, Valid: true},
			Payload:        payload,
		},
	}
	workflowData.tripSeals[501] = []db.ListSealsByTripRow{
		{
			CompartmentNumber:  1,
			SealNumberIssued:   "S-001",
			IssuedByName:       "Loader",
			IssuedAt:           pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 7, 30, 0, 0, time.UTC), Valid: true},
			VerificationStatus: db.NullSealStatusT{},
		},
	}
	workflowData.weightBridge[501] = []db.WeightBridgeReading{
		{
			ID:             11,
			TripID:         pgtype.Int8{Int64: 501, Valid: true},
			VehicleID:      91,
			ReadingType:    "TARE",
			WeightKg:       numericInt64Exp(12000, 0),
			Method:         db.MeasurementMethodTMANUALAPPROVED,
			ApprovalStatus: db.ApprovalStatusTPENDING,
			RecordedBy:     77,
			CreatedAt:      pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 7, 45, 0, 0, time.UTC), Valid: true},
		},
	}

	photos := &fakeDashboardTripPhotoLister{
		photos: map[int64][]service.TripPhotoWithURL{
			501: {
				{
					TripPhoto: db.TripPhoto{
						ID:            71,
						TripID:        501,
						EventType:     db.PhotoEventTWEIGHTBRIDGETARE,
						CompartmentID: pgtype.Int8{Int64: 1, Valid: true},
						TakenAt:       pgtype.Timestamptz{Time: time.Date(2026, 7, 15, 7, 46, 0, 0, time.UTC), Valid: true},
					},
					PresignedGetURL: "https://example.com/photo.jpg",
				},
			},
		},
	}

	router := newDashboardWorkflowRouter(&fakeDashboardAuthService{}, sessions, &fakeDashboardDataQuerier{}, workflowData, &fakeDashboardWorkflowService{}, photos)

	req := httptest.NewRequest(http.MethodGet, "/trips/501", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-trip"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"Timeline",
		"Weight Bridge",
		"Seals",
		"Photos",
		"https://example.com/photo.jpg",
		"KT 1234 AB",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %q", want, body)
		}
	}
}

func TestDashboardWeightBridgeEntryRendersForm(t *testing.T) {
	facilityID := int64(7)
	stationID := int64(19)
	sessions := newFakeDashboardSessionStore()
	sessions.sessions["sess-weight"] = model.SessionData{
		UserID:     77,
		FullName:   "Facility Operator",
		RoleGrants: []model.RoleGrant{{Role: "FACILITY_OPERATOR", ScopeType: "FACILITY", ScopeID: &facilityID}},
		ExpiresAt:  time.Now().Add(time.Hour),
	}

	workflowData := newFakeDashboardWorkflowQuerier()
	workflowData.trips[501] = db.GetTripWithDetailsRow{
		ID:                     501,
		DoID:                   101,
		VehicleID:              91,
		Status:                 db.TripStatusTINTRANSIT,
		OriginFacilityID:       facilityID,
		DestinationStationID:   pgtype.Int8{Int64: stationID, Valid: true},
		PlateNumber:            "KT 1234 AB",
		DriverName:             "Dimas",
		DestinationStationName: pgtype.Text{String: "SPBU 01", Valid: true},
		OriginFacilityName:     "Balikpapan Terminal",
	}

	router := newDashboardWorkflowRouter(&fakeDashboardAuthService{}, sessions, &fakeDashboardDataQuerier{}, workflowData, &fakeDashboardWorkflowService{}, &fakeDashboardTripPhotoLister{})

	req := httptest.NewRequest(http.MethodGet, "/trips/501/weight-bridge/new", nil)
	req.AddCookie(&http.Cookie{Name: dashboardSessionCookie, Value: "sess-weight"})
	rec := httptest.NewRecorder()

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, rec.Code)
	}
	body := rec.Body.String()
	for _, want := range []string{
		"New Reading",
		"MANUAL_APPROVED",
		"WEIGHT_BRIDGE",
		"/trips/501/weight-bridge",
	} {
		if !strings.Contains(body, want) {
			t.Fatalf("expected body to contain %q, got %q", want, body)
		}
	}
}
