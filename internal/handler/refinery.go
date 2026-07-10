package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/adevsh/petrosync/internal/db"
)

// RefineryHandler handles refineries and facilities endpoints.
type RefineryHandler struct {
	querier *db.Queries
}

// NewRefineryHandler creates a RefineryHandler.
func NewRefineryHandler(querier *db.Queries) *RefineryHandler {
	return &RefineryHandler{querier: querier}
}

// ListRefineries returns all active refineries.
func (h *RefineryHandler) ListRefineries(c *gin.Context) {
	refineries, err := h.querier.ListRefineries(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to load refineries"},
		})
		return
	}
	if refineries == nil {
		refineries = []db.Refinery{}
	}
	c.JSON(http.StatusOK, gin.H{"data": refineries})
}

// GetRefinery returns a single refinery by ID.
func (h *RefineryHandler) GetRefinery(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid refinery id"},
		})
		return
	}

	refinery, err := h.querier.GetRefinery(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "NOT_FOUND", "message": "refinery not found"},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": refinery})
}

// ListFacilitiesByRefinery returns facilities for a refinery.
func (h *RefineryHandler) ListFacilitiesByRefinery(c *gin.Context) {
	refineryID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid refinery id"},
		})
		return
	}

	facilities, err := h.querier.ListFacilitiesByRefinery(c.Request.Context(), refineryID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to load facilities"},
		})
		return
	}
	if facilities == nil {
		facilities = []db.ListFacilitiesByRefineryRow{}
	}
	c.JSON(http.StatusOK, gin.H{"data": facilities})
}

// GetFacility returns a single facility by ID.
func (h *RefineryHandler) GetFacility(c *gin.Context) {
	id, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid facility id"},
		})
		return
	}

	facility, err := h.querier.GetFacility(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": gin.H{"code": "NOT_FOUND", "message": "facility not found"},
		})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": facility})
}
