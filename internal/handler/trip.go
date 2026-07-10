package handler

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
	"github.com/adevsh/petrosync/internal/model"
	"github.com/adevsh/petrosync/internal/service"
)

// TripHandler handles trip endpoints.
type TripHandler struct {
	querier  *db.Queries
	workflow *service.WorkflowService
	photos   *service.TripPhotoService
}

func NewTripHandler(querier *db.Queries, workflow *service.WorkflowService, photos *service.TripPhotoService) *TripHandler {
	return &TripHandler{querier: querier, workflow: workflow, photos: photos}
}

func (h *TripHandler) ListActive(c *gin.Context) {
	rolesVal, _ := c.Get("roles")
	roles, _ := rolesVal.([]model.RoleGrant)
	userID := c.GetInt64("user_id")

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
		trips, err := h.querier.ListActiveTrips(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": trips})
	case best.Role == "REFINERY_ADMIN" && best.ScopeType == "REFINERY" && best.ScopeID != nil:
		trips, err := h.querier.ListActiveTripsByRefineryScope(ctx, *best.ScopeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": trips})
	case (best.Role == "FACILITY_MANAGER" || best.Role == "FACILITY_OPERATOR") && best.ScopeType == "FACILITY" && best.ScopeID != nil:
		trips, err := h.querier.ListActiveTripsByFacilityScope(ctx, *best.ScopeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": trips})
	case best.Role == "DEPOT_STAFF" && best.ScopeType == "DEPOT" && best.ScopeID != nil:
		depot, err := h.querier.GetDepot(ctx, *best.ScopeID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to resolve depot scope"}})
			return
		}
		trips, err := h.querier.ListActiveTripsByFacilityScope(ctx, depot.PrimaryFacilityID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": trips})
	case best.Role == "STATION_MANAGER" && best.ScopeType == "STATION" && best.ScopeID != nil:
		trips, err := h.querier.ListActiveTripsByStationScope(ctx, pgtype.Int8{Int64: *best.ScopeID, Valid: true})
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": trips})
	case best.Role == "DRIVER":
		trips, err := h.querier.ListActiveTripsByDriverUserScope(ctx, userID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": trips})
	default:
		c.JSON(http.StatusForbidden, gin.H{"error": gin.H{"code": "FORBIDDEN", "message": "no applicable role scope"}})
	}
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
	event, err := h.workflow.RecordTripEvent(c.Request.Context(), tripID, req)
	if err != nil {
		if errors.Is(err, service.ErrApprovalRequired) {
			c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "APPROVAL_REQUIRED", "message": "weight bridge approval required"}})
			return
		}
		if errors.Is(err, service.ErrPhotoMissing) {
			c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "PHOTO_MISSING", "message": "mandatory photos not uploaded for this step"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	middleware.SetAuditAction(c, "TRIP_EVENT_CREATE")
	middleware.SetAuditEntity(c, "trips", tripID)
	middleware.SetAuditAfter(c, event)
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
	middleware.SetAuditAction(c, "WEIGHT_BRIDGE_READING_CREATE")
	middleware.SetAuditEntity(c, "weight_bridge_readings", reading.ID)
	middleware.SetAuditAfter(c, reading)
	c.JSON(http.StatusCreated, gin.H{"data": reading})
}

func (h *TripHandler) ApproveWeightBridge(c *gin.Context) {
	id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")
	if before, err := h.querier.GetWeightBridgeReading(c.Request.Context(), id); err == nil {
		middleware.SetAuditBefore(c, before)
	}
	reading, err := h.querier.ApproveWeightBridgeReading(c.Request.Context(), db.ApproveWeightBridgeReadingParams{
		ID: id, ApprovedBy: pgtype.Int8{Int64: userID, Valid: true},
	})
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": gin.H{"code": "CONFLICT", "message": "reading not in PENDING or ESCALATED state"}})
		return
	}
	middleware.SetAuditAction(c, "WEIGHT_BRIDGE_READING_APPROVE")
	middleware.SetAuditEntity(c, "weight_bridge_readings", id)
	middleware.SetAuditAfter(c, reading)
	c.JSON(http.StatusOK, gin.H{"data": reading})
}
