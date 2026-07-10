package middleware

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/model"
)

type fakeScopeQuerier struct {
	facilities map[int64]db.GetFacilityRow
	depots     map[int64]db.GetDepotRow
	stations   map[int64]db.GetStationRow
}

func (f *fakeScopeQuerier) GetFacility(ctx context.Context, id int64) (db.GetFacilityRow, error) {
	v, ok := f.facilities[id]
	if !ok {
		return db.GetFacilityRow{}, errors.New("not found")
	}
	return v, nil
}

func (f *fakeScopeQuerier) GetDepot(ctx context.Context, id int64) (db.GetDepotRow, error) {
	v, ok := f.depots[id]
	if !ok {
		return db.GetDepotRow{}, errors.New("not found")
	}
	return v, nil
}

func (f *fakeScopeQuerier) GetStation(ctx context.Context, id int64) (db.GetStationRow, error) {
	v, ok := f.stations[id]
	if !ok {
		return db.GetStationRow{}, errors.New("not found")
	}
	return v, nil
}

func TestRequiredRole_HierarchyAndScopeMatch(t *testing.T) {
	gin.SetMode(gin.TestMode)

	facilityID := int64(10)
	q := &fakeScopeQuerier{
		facilities: map[int64]db.GetFacilityRow{
			facilityID: {ID: facilityID, RefineryID: 1},
		},
	}

	r := gin.New()
	r.GET("/facilities/:id",
		func(c *gin.Context) {
			c.Set("roles", []model.RoleGrant{
				{Role: "FACILITY_MANAGER", ScopeType: "FACILITY", ScopeID: &facilityID},
			})
		},
		RequiredRole(q, "FACILITY_OPERATOR", "FACILITY", "id"),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/facilities/10", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestRequiredRole_ScopeMismatchForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	facilityID := int64(10)
	q := &fakeScopeQuerier{
		facilities: map[int64]db.GetFacilityRow{
			11: {ID: 11, RefineryID: 1},
		},
	}

	r := gin.New()
	r.GET("/facilities/:id",
		func(c *gin.Context) {
			c.Set("roles", []model.RoleGrant{
				{Role: "FACILITY_OPERATOR", ScopeType: "FACILITY", ScopeID: &facilityID},
			})
		},
		RequiredRole(q, "FACILITY_OPERATOR", "FACILITY", "id"),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/facilities/11", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestRequiredRole_RefineryAdminImplicitFacilityAccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	facilityID := int64(10)
	refineryID := int64(5)
	q := &fakeScopeQuerier{
		facilities: map[int64]db.GetFacilityRow{
			facilityID: {ID: facilityID, RefineryID: refineryID},
		},
	}

	r := gin.New()
	r.GET("/facilities/:id",
		func(c *gin.Context) {
			c.Set("roles", []model.RoleGrant{
				{Role: "REFINERY_ADMIN", ScopeType: "REFINERY", ScopeID: &refineryID},
			})
		},
		RequiredRole(q, "FACILITY_OPERATOR", "FACILITY", "id"),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/facilities/10", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestDisallowDriver_BlocksDriverOnlyUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.New()
	r.GET("/x",
		func(c *gin.Context) {
			c.Set("roles", []model.RoleGrant{{Role: "DRIVER", ScopeType: "COMPANY"}})
		},
		DisallowDriver(),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d", w.Code)
	}
}

func TestDisallowDriver_AllowsMixedRoleUser(t *testing.T) {
	gin.SetMode(gin.TestMode)

	facilityID := int64(10)
	r := gin.New()
	r.GET("/x",
		func(c *gin.Context) {
			c.Set("roles", []model.RoleGrant{
				{Role: "DRIVER", ScopeType: "COMPANY"},
				{Role: "FACILITY_OPERATOR", ScopeType: "FACILITY", ScopeID: &facilityID},
			})
		},
		DisallowDriver(),
		func(c *gin.Context) { c.Status(http.StatusOK) },
	)

	w := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/x", nil)
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

