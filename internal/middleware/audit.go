package middleware

import (
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

// Audit writes an audit_log entry for every state-changing request.
// The handler runs after the request completes.
func Audit(querier *db.Queries, action, entityType string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Only audit successful state-changing requests
		if c.Request.Method == "GET" || c.Writer.Status() >= 300 {
			return
		}

		userID, _ := c.Get("user_id")

		params := db.InsertAuditLogParams{
			Action:     action,
			EntityType: entityType,
		}
		if uid, ok := userID.(int64); ok && uid > 0 {
			params.UserID = pgtype.Int8{Int64: uid, Valid: true}
		}

		go func() {
			_, _ = querier.InsertAuditLog(c.Request.Context(), params)
		}()
	}
}
