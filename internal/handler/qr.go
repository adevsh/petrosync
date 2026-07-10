package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
)

// QRHandler handles QR code validation endpoints.
type QRHandler struct {
	querier *db.Queries
}

func NewQRHandler(querier *db.Queries) *QRHandler { return &QRHandler{querier: querier} }

type qrValidateReq struct {
	TripID    int64  `json:"trip_id" binding:"required"`
	QRPayload string `json:"qr_payload" binding:"required"`
	Context   string `json:"context" binding:"required"` // LOADING_BAY or STATION
}

// Validate checks that a scanned QR payload matches the expected trip context.
func (h *QRHandler) Validate(c *gin.Context) {
	middleware.SkipAudit(c)
	var req qrValidateReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error()}})
		return
	}

	switch req.Context {
	case "LOADING_BAY":
		bay, err := h.querier.GetLoadingBayByQRPayload(c.Request.Context(), req.QRPayload)
		if err != nil || bay.FacilityID == 0 {
			c.JSON(http.StatusOK, gin.H{"data": gin.H{"valid": false, "reason": "QR_INVALID"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"valid": true, "location_name": bay.BayCode}})
	case "STATION":
		station, err := h.querier.GetStationByQRPayload(c.Request.Context(), req.QRPayload)
		if err != nil {
			c.JSON(http.StatusOK, gin.H{"data": gin.H{"valid": false, "reason": "QR_INVALID"}})
			return
		}
		c.JSON(http.StatusOK, gin.H{"data": gin.H{"valid": true, "location_name": station.StationName}})
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "context must be LOADING_BAY or STATION"}})
	}
}
