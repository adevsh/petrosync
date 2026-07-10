package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/adevsh/petrosync/internal/db"
)

// StorageTankHandler handles facility storage tank endpoints.
type StorageTankHandler struct {
	querier *db.Queries
}

func NewStorageTankHandler(querier *db.Queries) *StorageTankHandler {
	return &StorageTankHandler{querier: querier}
}

func (h *StorageTankHandler) ListByFacility(c *gin.Context) {
	facilityID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	tanks, err := h.querier.ListStorageTanksByFacility(c.Request.Context(), facilityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if tanks == nil { tanks = []db.FacilityStorageTank{} }
	c.JSON(http.StatusOK, gin.H{"data": tanks})
}

func (h *StorageTankHandler) GetAvailableVolume(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	tank, err := h.querier.GetStorageTankAvailableVolume(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "tank not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tank})
}
