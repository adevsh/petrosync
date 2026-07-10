package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
)

type TelegramLinkQuerier interface {
	GetUser(ctx context.Context, id int64) (db.GetUserRow, error)
	CreateTelegramLinkToken(ctx context.Context, arg db.CreateTelegramLinkTokenParams) (db.TelegramLinkToken, error)
}

type TelegramLinkTokenHandler struct {
	querier TelegramLinkQuerier
}

func NewTelegramLinkTokenHandler(querier TelegramLinkQuerier) *TelegramLinkTokenHandler {
	return &TelegramLinkTokenHandler{querier: querier}
}

type telegramLinkTokenResponse struct {
	Token     string     `json:"token"`
	ExpiresAt *time.Time `json:"expires_at"`
}

func (h *TelegramLinkTokenHandler) CreateLinkToken(c *gin.Context) {
	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid user id"}})
		return
	}

	if _, err := h.querier.GetUser(c.Request.Context(), userID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": gin.H{"code": "NOT_FOUND", "message": "user not found"}})
		return
	}

	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to generate token"}})
		return
	}
	token := hex.EncodeToString(b)

	row, err := h.querier.CreateTelegramLinkToken(c.Request.Context(), db.CreateTelegramLinkTokenParams{
		UserID: userID,
		Token:  token,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error()}})
		return
	}

	middleware.SetAuditAction(c, "TELEGRAM_LINK_TOKEN_CREATE")
	middleware.SetAuditEntity(c, "telegram_link_tokens", row.ID)
	middleware.SetAuditAfter(c, gin.H{
		"token_id":    row.ID,
		"user_id":     row.UserID,
		"expires_at":  pgTimestamptzToPtr(row.ExpiresAt),
	})
	c.JSON(http.StatusCreated, gin.H{"data": telegramLinkTokenResponse{
		Token:     row.Token,
		ExpiresAt: pgTimestamptzToPtr(row.ExpiresAt),
	}})
}
