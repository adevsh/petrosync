package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/adevsh/petrosync/internal/db"
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
	stations, err := h.querier.ListAllActiveStations(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if stations == nil {
		stations = []db.ListAllActiveStationsRow{}
	}
	c.JSON(http.StatusOK, gin.H{"data": stations})
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
	_, err := h.querier.UpdateDipReading(c.Request.Context(), db.UpdateDipReadingParams{
		ID: id, LastDipReadingL: floatToNumeric(req.DipReadingL),
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "dip reading updated"}})
}
