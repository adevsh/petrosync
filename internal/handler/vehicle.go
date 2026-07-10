package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
	"github.com/adevsh/petrosync/internal/model"
)

// VehicleHandler handles vehicle and compartment endpoints.
type VehicleHandler struct {
	querier *db.Queries
}

// NewVehicleHandler creates a VehicleHandler.
func NewVehicleHandler(querier *db.Queries) *VehicleHandler {
	return &VehicleHandler{querier: querier}
}

// ── Vehicles ──────────────────────────────────────────────────────────

func (h *VehicleHandler) ListVehiclesByDepot(c *gin.Context) {
	depotID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid depot id"}})
		return
	}
	vehicles, err := h.querier.ListVehiclesByDepot(c.Request.Context(), pgtype.Int8{Int64: depotID, Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if vehicles == nil {
		vehicles = []db.ListVehiclesByDepotRow{}
	}
	c.JSON(http.StatusOK, gin.H{"data": vehicles})
}

func (h *VehicleHandler) ListVehiclesByStatus(c *gin.Context) {
	status := db.VehicleStatusT(c.Query("status"))
	rolesVal, _ := c.Get("roles")
	roles, _ := rolesVal.([]model.RoleGrant)
	ctx := c.Request.Context()

	bestRank := 0
	var best model.RoleGrant
	for _, r := range roles {
		if r.Role == "SYSTEM_ADMIN" {
			best = r
			bestRank = middleware.RoleRank("SYSTEM_ADMIN")
			break
		}
		if rank := middleware.RoleRank(r.Role); rank > bestRank {
			bestRank = rank
			best = r
		}
	}

	switch {
	case best.Role == "SYSTEM_ADMIN":
		vehicles, err := h.querier.ListVehiclesByStatus(ctx, status)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": vehicles})
	case best.Role == "REFINERY_ADMIN" && best.ScopeType == "REFINERY" && best.ScopeID != nil:
		vehicles, err := h.querier.ListVehiclesByStatusAndRefinery(ctx, db.ListVehiclesByStatusAndRefineryParams{
			Status: status, RefineryID: *best.ScopeID,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": vehicles})
	case (best.Role == "FACILITY_MANAGER" || best.Role == "FACILITY_OPERATOR") && best.ScopeType == "FACILITY" && best.ScopeID != nil:
		vehicles, err := h.querier.ListVehiclesByStatusAndFacility(ctx, db.ListVehiclesByStatusAndFacilityParams{
			Status: status, PrimaryFacilityID: *best.ScopeID,
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": vehicles})
	case best.Role == "DEPOT_STAFF" && best.ScopeType == "DEPOT" && best.ScopeID != nil:
		vehicles, err := h.querier.ListVehiclesByStatusAndDepot(ctx, db.ListVehiclesByStatusAndDepotParams{
			Status: status, CurrentDepotID: pgtype.Int8{Int64: *best.ScopeID, Valid: true},
		})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": vehicles})
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "no applicable role scope"}})
	}
}

func (h *VehicleHandler) GetVehicle(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	v, err := h.querier.GetVehicle(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "vehicle not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": v})
}

func (h *VehicleHandler) CreateVehicle(c *gin.Context) {
	var req db.CreateVehicleParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	v, err := h.querier.CreateVehicle(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	middleware.SetAuditAction(c, "VEHICLE_CREATE")
	middleware.SetAuditEntity(c, "vehicles", v.ID)
	middleware.SetAuditAfter(c, v)
	c.JSON(http.StatusCreated, gin.H{"data": v})
}

// ── Vehicle Compartments ──────────────────────────────────────────────

func (h *VehicleHandler) ListCompartments(c *gin.Context) {
	vehicleID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	comps, err := h.querier.ListCompartmentsByVehicle(c.Request.Context(), vehicleID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if comps == nil {
		comps = []db.VehicleCompartment{}
	}
	c.JSON(http.StatusOK, gin.H{"data": comps})
}

func (h *VehicleHandler) CreateCompartment(c *gin.Context) {
	var req db.CreateCompartmentParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	comp, err := h.querier.CreateCompartment(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	middleware.SetAuditAction(c, "VEHICLE_COMPARTMENT_CREATE")
	middleware.SetAuditEntity(c, "vehicle_compartments", comp.ID)
	middleware.SetAuditAfter(c, comp)
	c.JSON(http.StatusCreated, gin.H{"data": comp})
}
