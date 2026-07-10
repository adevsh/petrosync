package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
)

// DriverHandler handles driver endpoints.
type DriverHandler struct {
	querier *db.Queries
}

// NewDriverHandler creates a DriverHandler.
func NewDriverHandler(querier *db.Queries) *DriverHandler {
	return &DriverHandler{querier: querier}
}

func (h *DriverHandler) ListDriversByDepot(c *gin.Context) {
	depotID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	drivers, err := h.querier.ListDriversByDepot(c.Request.Context(), pgtype.Int8{Int64: depotID, Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if drivers == nil {
		drivers = []db.ListDriversByDepotRow{}
	}
	c.JSON(http.StatusOK, gin.H{"data": drivers})
}

func (h *DriverHandler) GetDriver(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	d, err := h.querier.GetDriver(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "driver not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": d})
}

func (h *DriverHandler) CreateDriver(c *gin.Context) {
	var req db.CreateDriverParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	d, err := h.querier.CreateDriver(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	middleware.SetAuditAction(c, "DRIVER_CREATE")
	middleware.SetAuditEntity(c, "drivers", d.ID)
	middleware.SetAuditAfter(c, d)
	c.JSON(http.StatusCreated, gin.H{"data": d})
}

func (h *DriverHandler) StartShift(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if before, err := h.querier.GetDriver(c.Request.Context(), id); err == nil {
		middleware.SetAuditBefore(c, before)
	}
	if err := h.querier.StartDriverShift(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if after, err := h.querier.GetDriver(c.Request.Context(), id); err == nil {
		middleware.SetAuditAfter(c, after)
	}
	middleware.SetAuditAction(c, "DRIVER_SHIFT_START")
	middleware.SetAuditEntity(c, "drivers", id)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "shift started"}})
}

func (h *DriverHandler) EndShift(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if before, err := h.querier.GetDriver(c.Request.Context(), id); err == nil {
		middleware.SetAuditBefore(c, before)
	}
	if err := h.querier.EndDriverShift(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if after, err := h.querier.GetDriver(c.Request.Context(), id); err == nil {
		middleware.SetAuditAfter(c, after)
	}
	middleware.SetAuditAction(c, "DRIVER_SHIFT_END")
	middleware.SetAuditEntity(c, "drivers", id)
	c.JSON(http.StatusOK, gin.H{"data": gin.H{"message": "shift ended"}})
}
