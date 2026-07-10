package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
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
	vehicles, err := h.querier.ListVehiclesByStatus(c.Request.Context(), status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if vehicles == nil {
		vehicles = []db.ListVehiclesByStatusRow{}
	}
	c.JSON(http.StatusOK, gin.H{"data": vehicles})
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
	c.JSON(http.StatusCreated, gin.H{"data": comp})
}
