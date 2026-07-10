package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
	"github.com/adevsh/petrosync/internal/model"
)

// StationHandler handles gas station and station tank endpoints.
type StationHandler struct {
	querier *db.Queries
}

// NewStationHandler creates a StationHandler.
func NewStationHandler(querier *db.Queries) *StationHandler {
	return &StationHandler{querier: querier}
}

func (h *StationHandler) ListStations(c *gin.Context) {
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
		stations, err := h.querier.ListAllActiveStations(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": stations})
	case best.Role == "REFINERY_ADMIN" && best.ScopeType == "REFINERY" && best.ScopeID != nil:
		stations, err := h.querier.ListAllActiveStationsByRefineryScope(ctx, *best.ScopeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": stations})
	case (best.Role == "FACILITY_MANAGER" || best.Role == "FACILITY_OPERATOR") && best.ScopeType == "FACILITY" && best.ScopeID != nil:
		stations, err := h.querier.ListStationsServedByFacility(ctx, *best.ScopeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": stations})
	case best.Role == "DEPOT_STAFF" && best.ScopeType == "DEPOT" && best.ScopeID != nil:
		depot, err := h.querier.GetDepot(ctx, *best.ScopeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to resolve depot scope"}})
			return
		}
		stations, err := h.querier.ListStationsServedByFacility(ctx, depot.PrimaryFacilityID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": stations})
	case best.Role == "STATION_MANAGER" && best.ScopeType == "STATION" && best.ScopeID != nil:
		stations, err := h.querier.ListAllActiveStationsByStationScope(ctx, *best.ScopeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": stations})
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "no applicable role scope"}})
	}
}

func (h *StationHandler) GetStation(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	s, err := h.querier.GetStation(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "station not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": s})
}

func (h *StationHandler) CreateStation(c *gin.Context) {
	var req db.CreateStationParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	s, err := h.querier.CreateStation(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	middleware.SetAuditAction(c, "STATION_CREATE")
	middleware.SetAuditEntity(c, "stations", s.ID)
	middleware.SetAuditAfter(c, s)
	c.JSON(http.StatusCreated, gin.H{"data": s})
}

func (h *StationHandler) ListTanks(c *gin.Context) {
	stationID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	tanks, err := h.querier.ListStationTanksByStation(c.Request.Context(), stationID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if tanks == nil {
		tanks = []db.StationTank{}
	}
	c.JSON(http.StatusOK, gin.H{"data": tanks})
}

func (h *StationHandler) UpdateDipReading(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		DipReadingL float64 `json:"dip_reading_l" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	if before, err := h.querier.GetStationTank(c.Request.Context(), id); err == nil {
		middleware.SetAuditBefore(c, before)
	}
	_, err := h.querier.UpdateDipReading(c.Request.Context(), db.UpdateDipReadingParams{
		ID: id, LastDipReadingL: floatToNumeric(req.DipReadingL),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if after, err := h.querier.GetStationTank(c.Request.Context(), id); err == nil {
		middleware.SetAuditAfter(c, after)
	}
	middleware.SetAuditAction(c, "STATION_TANK_DIP_UPDATE")
	middleware.SetAuditEntity(c, "station_tanks", id)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "dip reading updated"}})
}
