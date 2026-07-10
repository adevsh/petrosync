package handler

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
)

// GPSHandler handles GPS batch ingestion and publishes to Valkey.
type GPSHandler struct {
	querier GPSQuerier
	valkey  GPSPublisher
}

type GPSQuerier interface {
	InsertGPSEvent(ctx context.Context, arg db.InsertGPSEventParams) (db.InsertGPSEventRow, error)
	CheckGPSEventUUIDExists(ctx context.Context, eventUuid uuid.UUID) (bool, error)
}

type GPSPublisher interface {
	Publish(ctx context.Context, channel, message string) error
}

func NewGPSHandler(querier GPSQuerier, valkey GPSPublisher) *GPSHandler {
	return &GPSHandler{querier: querier, valkey: valkey}
}

type gpsEvent struct {
	EventUUID      string  `json:"event_uuid" binding:"required"`
	TripID         int64   `json:"trip_id" binding:"required"`
	Latitude       float64 `json:"latitude" binding:"required"`
	Longitude      float64 `json:"longitude" binding:"required"`
	SpeedKmH       *float64 `json:"speed_kmh"`
	HeadingDeg     *float64 `json:"heading_deg"`
	AccuracyM      *float64 `json:"accuracy_m"`
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
	seen := make(map[uuid.UUID]struct{}, len(req))

	for _, evt := range req {
		eventUUID, err := uuid.Parse(evt.EventUUID)
		if err != nil {
			continue
		}
		if _, ok := seen[eventUUID]; ok {
			duplicates++
			continue
		}
		seen[eventUUID] = struct{}{}

		ts, err := time.Parse(time.RFC3339, evt.EventTimestamp)
		if err != nil {
			continue
		}

		exists, err := h.querier.CheckGPSEventUUIDExists(c.Request.Context(), eventUUID)
		if err == nil && exists {
			duplicates++
			continue
		}

		speed := pgtype.Numeric{}
		if evt.SpeedKmH != nil {
			speed = floatToNumeric(*evt.SpeedKmH)
		}
		heading := pgtype.Numeric{}
		if evt.HeadingDeg != nil {
			heading = floatToNumeric(*evt.HeadingDeg)
		}
		accuracy := pgtype.Numeric{}
		if evt.AccuracyM != nil {
			accuracy = floatToNumeric(*evt.AccuracyM)
		}

		_, err = h.querier.InsertGPSEvent(c.Request.Context(), db.InsertGPSEventParams{
			TripID:    evt.TripID,
			EventUuid: eventUUID,
			Latitude:  floatToNumeric(evt.Latitude),
			Longitude: floatToNumeric(evt.Longitude),
			SpeedKmh:   speed,
			HeadingDeg: heading,
			AccuracyM:  accuracy,
			EventTimestamp: pgtype.Timestamptz{Time: ts, Valid: true},
		})
		if err != nil {
			var pgErr *pgconn.PgError
			if errors.As(err, &pgErr) && pgErr.Code == "23505" {
				duplicates++
			}
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
		if h.valkey != nil {
			_ = h.valkey.Publish(c.Request.Context(), "ws:trip:"+strconv.FormatInt(evt.TripID, 10), string(msg))
		}
	}

	middleware.SetAuditAction(c, "GPS_BATCH_INGEST")
	middleware.SetAuditEntity(c, "gps_events", 0)
	middleware.SetAuditAfter(c, gin.H{"accepted": accepted, "duplicates": duplicates})
	c.JSON(http.StatusAccepted, gin.H{"data": gin.H{"accepted": accepted, "duplicates": duplicates}})
}

func toFloat8(v float64) pgtype.Float8 {
	if v == 0 { return pgtype.Float8{} }
	return pgtype.Float8{Float64: v, Valid: true}
}
