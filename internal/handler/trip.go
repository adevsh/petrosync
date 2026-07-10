package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

// TripHandler handles trip endpoints.
type TripHandler struct {
	querier *db.Queries
}

func NewTripHandler(querier *db.Queries) *TripHandler {
	return &TripHandler{querier: querier}
}

func (h *TripHandler) ListActive(c *gin.Context) {
	trips, err := h.querier.ListActiveTrips(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if trips == nil { trips = []db.ListActiveTripsRow{} }
	c.JSON(http.StatusOK, gin.H{"data": trips})
}

func (h *TripHandler) Get(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	trip, err := h.querier.GetTripWithDetails(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "trip not found"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": trip})
}

func (h *TripHandler) ListEvents(c *gin.Context) {
	tripID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	events, err := h.querier.ListTripEventsByTrip(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if events == nil { events = []db.TripEvent{} }
	c.JSON(http.StatusOK, gin.H{"data": events})
}

func (h *TripHandler) CreateEvent(c *gin.Context) {
	tripID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	var req db.InsertTripEventParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	req.TripID = tripID
	event, err := h.querier.InsertTripEvent(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": event})
}

func (h *TripHandler) ListCompartmentDeliveries(c *gin.Context) {
	tripID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	deliveries, err := h.querier.ListCompartmentDeliveriesByTrip(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if deliveries == nil { deliveries = []db.ListCompartmentDeliveriesByTripRow{} }
	c.JSON(http.StatusOK, gin.H{"data": deliveries})
}

func (h *TripHandler) ListSeals(c *gin.Context) {
	tripID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	seals, err := h.querier.ListSealsByTrip(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if seals == nil { seals = []db.ListSealsByTripRow{} }
	c.JSON(http.StatusOK, gin.H{"data": seals})
}

// ── Weight Bridge ─────────────────────────────────────────────────────

func (h *TripHandler) ListWeightBridge(c *gin.Context) {
	tripID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	readings, err := h.querier.ListWeightBridgeReadingsByTrip(c.Request.Context(), pgtype.Int8{Int64: tripID, Valid: true})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	if readings == nil { readings = []db.WeightBridgeReading{} }
	c.JSON(http.StatusOK, gin.H{"data": readings})
}

func (h *TripHandler) CreateWeightBridge(c *gin.Context) {
	var req db.CreateWeightBridgeReadingParams
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}
	reading, err := h.querier.CreateWeightBridgeReading(c.Request.Context(), req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"data": reading})
}

func (h *TripHandler) ApproveWeightBridge(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")
	reading, err := h.querier.ApproveWeightBridgeReading(c.Request.Context(), db.ApproveWeightBridgeReadingParams{
		ID: id, ApprovedBy: pgtype.Int8{Int64: userID, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "CONFLICT", "message": "reading not in PENDING or ESCALATED state"}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": reading})
}
