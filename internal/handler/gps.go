package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/service"
)

// GPSHandler handles GPS batch ingestion and publishes to Valkey.
type GPSHandler struct {
	querier *db.Queries
	valkey  *service.ValkeyService
}

func NewGPSHandler(querier *db.Queries, valkey *service.ValkeyService) *GPSHandler {
	return &GPSHandler{querier: querier, valkey: valkey}
}

type gpsEvent struct {
	EventUUID      string  `json:"event_uuid" binding:"required"`
	TripID         int64   `json:"trip_id" binding:"required"`
	Latitude       float64 `json:"latitude" binding:"required"`
	Longitude      float64 `json:"longitude" binding:"required"`
	SpeedKmH       float64 `json:"speed_kmh"`
	HeadingDeg     float64 `json:"heading_deg"`
	AccuracyM      float64 `json:"accuracy_m"`
	EventTimestamp string  `json:"event_timestamp" binding:"required"`
}

type gpsBatchReq []gpsEvent

// Batch ingests GPS events and publishes to Valkey.
func (h *GPSHandler) Batch(c *gin.Context) {
	var req gpsBatchReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	accepted := 0
	duplicates := 0

	for _, evt := range req {
		ts, err := time.Parse(time.RFC3339, evt.EventTimestamp)
		if err != nil {
			continue
		}

		_, err = h.querier.InsertGPSEvent(c.Request.Context(), db.InsertGPSEventParams{
			TripID:    evt.TripID,
			EventUuid: uuid.MustParse(evt.EventUUID),
			Latitude:  pgtype.Numeric{}, // TODO: convert float64→Numeric
			Longitude: pgtype.Numeric{},
			SpeedKmh:   pgtype.Numeric{},
			HeadingDeg: pgtype.Numeric{},
			AccuracyM:  pgtype.Numeric{},
			EventTimestamp: pgtype.Timestamptz{Time: ts, Valid: true},
		})
		if err != nil {
			// Likely duplicate UUID — silently skip
			duplicates++
			continue
		}
		accepted++

		// Publish to Valkey for WebSocket fan-out
		msg, _ := json.Marshal(gin.H{
			"trip_id":     evt.TripID,
			"lat":         evt.Latitude,
			"lng":         evt.Longitude,
			"speed_kmh":   evt.SpeedKmH,
			"last_gps_at": evt.EventTimestamp,
		})
		_ = h.valkey.Publish(c.Request.Context(), "ws:trip:"+strconv.FormatInt(evt.TripID, 10), string(msg))
	}

	c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"accepted": accepted, "duplicates": duplicates}})
}

func toFloat8(v float64) pgtype.Float8 {
	if v == 0 { return pgtype.Float8{} }
	return pgtype.Float8{Float64: v, Valid: true}
}
