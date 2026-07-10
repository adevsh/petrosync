package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

// DeliveryOrderHandler handles delivery order endpoints.
type DeliveryOrderHandler struct {
	querier *db.Queries
}

func NewDeliveryOrderHandler(querier *db.Queries) *DeliveryOrderHandler {
	return &DeliveryOrderHandler{querier: querier}
}

func (h *DeliveryOrderHandler) ListByFacility(c *gin.Context) {
	facilityID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	dos, err := h.querier.ListDOsByOriginFacility(c.Request.Context(), facilityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if dos == nil { dos = []db.DeliveryOrder{} }
	c.JSON(http.StatusOK, gin.H{"data": dos})
}

func (h *DeliveryOrderHandler) ListDispatchQueue(c *gin.Context) {
	facilityID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	dos, err := h.querier.ListDOsForDispatchQueue(c.Request.Context(), facilityID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if dos == nil { dos = []db.ListDOsForDispatchQueueRow{} }
	c.JSON(http.StatusOK, gin.H{"data": dos})
}

func (h *DeliveryOrderHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	do, err := h.querier.GetDeliveryOrder(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "delivery order not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": do})
}

func (h *DeliveryOrderHandler) Create(c *gin.Context) {
	var req db.CreateDeliveryOrderParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	do, err := h.querier.CreateDeliveryOrder(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": do})
}

// Approve transitions a DO from PENDING_APPROVAL to APPROVED.
func (h *DeliveryOrderHandler) Approve(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")
	do, err := h.querier.ApproveDeliveryOrder(c.Request.Context(), db.ApproveDeliveryOrderParams{
		ID: id, ApprovedBy: pgtype.Int8{Int64: userID, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "CONFLICT", "message": "DO not in PENDING_APPROVAL state"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": do})
}

// AssignVehicleAndDriver assigns vehicle and driver to an approved DO.
func (h *DeliveryOrderHandler) AssignVehicleAndDriver(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req struct {
		VehicleID int64 `json:"vehicle_id" binding:"required"`
		DriverID  int64 `json:"driver_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	do, err := h.querier.AssignVehicleAndDriverToDO(c.Request.Context(), db.AssignVehicleAndDriverToDOParams{
		ID: id,
		AssignedVehicleID: pgtype.Int8{Int64: req.VehicleID, Valid: true},
		AssignedDriverID:  pgtype.Int8{Int64: req.DriverID, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "CONFLICT", "message": "DO not in APPROVED state"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": do})
}

// Cancel cancels a delivery order that hasn't started yet.
func (h *DeliveryOrderHandler) Cancel(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	do, err := h.querier.CancelDeliveryOrder(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "CONFLICT", "message": "DO cannot be cancelled"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": do})
}

// ListItems returns all line items for a delivery order.
func (h *DeliveryOrderHandler) ListItems(c *gin.Context) {
	doID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	items, err := h.querier.ListDOItemsByDO(c.Request.Context(), doID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if items == nil { items = []db.ListDOItemsByDORow{} }
	c.JSON(http.StatusOK, gin.H{"data": items})
}

// CreateItem adds a line item to a delivery order.
func (h *DeliveryOrderHandler) CreateItem(c *gin.Context) {
	var req db.CreateDeliveryOrderItemParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	item, err := h.querier.CreateDeliveryOrderItem(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": item})
}
