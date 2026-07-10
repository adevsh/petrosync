package middleware

import (
	"encoding/json"
	"net/netip"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/auditlog"
	"github.com/adevsh/petrosync/internal/db"
)

const (
	auditKeyAction     = "audit_action"
	auditKeyEntityType = "audit_entity_type"
	auditKeyEntityID   = "audit_entity_id"
	auditKeyBefore     = "audit_before"
	auditKeyAfter      = "audit_after"
	auditKeySkip       = "audit_skip"
)

func SetAuditAction(c *gin.Context, action string) {
	c.Set(auditKeyAction, action)
}

func SetAuditEntity(c *gin.Context, entityType string, entityID int64) {
	c.Set(auditKeyEntityType, entityType)
	c.Set(auditKeyEntityID, entityID)
}

func SetAuditBefore(c *gin.Context, v any) {
	if v == nil {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	c.Set(auditKeyBefore, json.RawMessage(b))
}

func SetAuditAfter(c *gin.Context, v any) {
	if v == nil {
		return
	}
	b, err := json.Marshal(v)
	if err != nil {
		return
	}
	c.Set(auditKeyAfter, json.RawMessage(b))
}

func SkipAudit(c *gin.Context) {
	c.Set(auditKeySkip, true)
}

func AuditTrail(writer *auditlog.AsyncWriter) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Status() >= 300 {
			return
		}
		if c.GetBool(auditKeySkip) {
			return
		}

		switch c.Request.Method {
		case "POST", "PUT", "PATCH", "DELETE":
		default:
			return
		}

		action, _ := c.Get(auditKeyAction)
		actionStr, _ := action.(string)
		if actionStr == "" {
			actionStr = strings.TrimSpace(c.Request.Method + " " + c.FullPath())
		}

		entityType, _ := c.Get(auditKeyEntityType)
		entityTypeStr, _ := entityType.(string)
		if entityTypeStr == "" {
			entityTypeStr = c.FullPath()
		}

		var entityID pgtype.Int8
		if v, ok := c.Get(auditKeyEntityID); ok {
			if id, ok := v.(int64); ok && id > 0 {
				entityID = pgtype.Int8{Int64: id, Valid: true}
			}
		}

		var before json.RawMessage
		if v, ok := c.Get(auditKeyBefore); ok {
			if b, ok := v.(json.RawMessage); ok {
				before = b
			}
		}

		var after json.RawMessage
		if v, ok := c.Get(auditKeyAfter); ok {
			if b, ok := v.(json.RawMessage); ok {
				after = b
			}
		}

		userID, _ := c.Get("user_id")

		params := db.InsertAuditLogParams{
			Action:      actionStr,
			EntityType:  entityTypeStr,
			EntityID:    entityID,
			BeforeState: before,
			AfterState:  after,
		}
		if uid, ok := userID.(int64); ok && uid > 0 {
			params.UserID = pgtype.Int8{Int64: uid, Valid: true}
		}

		if ua := c.GetHeader("User-Agent"); ua != "" {
			params.UserAgent = pgtype.Text{String: ua, Valid: true}
		}
		if ip := c.ClientIP(); ip != "" {
			if addr, err := netip.ParseAddr(ip); err == nil {
				params.IpAddress = &addr
			}
		}

		writer.Write(params)
	}
}
