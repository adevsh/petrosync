package handler

import (
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
)

func (h *TripHandler) UploadPhoto(c *gin.Context) {
	tripID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	userID := c.GetInt64("user_id")

	eventTypeStr := c.PostForm("event_type")
	eventType := db.PhotoEventT(eventTypeStr)
	if !eventType.Valid() {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid event_type"}})
		return
	}

	var compartmentID pgtype.Int8
	compartmentStr := c.PostForm("compartment_id")
	if compartmentStr != "" {
		id, err := strconv.ParseInt(compartmentStr, 10, 64)
		if err != nil || id <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid compartment_id"}})
			return
		}
		compartmentID = pgtype.Int8{Int64: id, Valid: true}
	}
	if eventType == db.PhotoEventTCOMPARTMENTSEALED && !compartmentID.Valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "compartment_id is required for COMPARTMENT_SEALED"}})
		return
	}

	takenAtStr := c.PostForm("taken_at")
	if takenAtStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "taken_at is required"}})
		return
	}
	takenAt, err := time.Parse(time.RFC3339, takenAtStr)
	if err != nil {
		takenAt, err = time.Parse(time.RFC3339Nano, takenAtStr)
	}
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "taken_at must be ISO 8601"}})
		return
	}

	notesStr := c.PostForm("notes")
	var notes pgtype.Text
	if notesStr != "" {
		notes = pgtype.Text{String: notesStr, Valid: true}
	}

	fh, err := c.FormFile("photo")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "photo file is required"}})
		return
	}
	if fh.Size <= 0 || fh.Size > 5*1024*1024 {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "photo must be a JPEG up to 5MB"}})
		return
	}

	f, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to read upload"}})
		return
	}
	defer f.Close()

	buf := make([]byte, 512)
	n, _ := f.Read(buf)
	_, _ = f.Seek(0, 0)
	contentType := http.DetectContentType(buf[:n])
	if contentType != "image/jpeg" {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "photo must be a JPEG"}})
		return
	}

	if h.photos == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "photo service not configured"}})
		return
	}

	photo, err := h.photos.UploadTripPhoto(
		c.Request.Context(),
		tripID,
		userID,
		eventType,
		compartmentID,
		takenAt,
		notes,
		f,
		fh.Size,
		contentType,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "trip not found"}})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	middleware.SetAuditAction(c, "TRIP_PHOTO_UPLOAD")
	middleware.SetAuditEntity(c, "trip_photos", photo.ID)
	middleware.SetAuditAfter(c, gin.H{
		"photo_id":      photo.ID,
		"trip_id":       photo.TripID,
		"uploaded_by":   photo.UploadedBy,
		"event_type":    photo.EventType,
		"compartment_id": func() any {
			if photo.CompartmentID.Valid {
				return photo.CompartmentID.Int64
			}
			return nil
		}(),
		"taken_at":      pgTimestamptzToPtr(photo.TakenAt),
	})
	c.JSON(http.StatusCreated, gin.H{"data": gin.H{"photo_id": photo.ID}})
}

func (h *TripHandler) ListPhotos(c *gin.Context) {
	tripID, _ := strconv.ParseInt(c.Param("id"), 10, 64)
	if h.photos == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "photo service not configured"}})
		return
	}

	photos, err := h.photos.ListTripPhotosWithURLs(c.Request.Context(), tripID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": photos})
}
