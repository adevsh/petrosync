package handler

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/auditlog"
	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
)

type captureAuditSink struct {
	ch chan db.InsertAuditLogParams
}

func (s *captureAuditSink) InsertAuditLog(ctx context.Context, arg db.InsertAuditLogParams) (db.InsertAuditLogRow, error) {
	select {
	case s.ch <- arg:
	default:
	}
	return db.InsertAuditLogRow{ID: 1, CreatedAt: pgtype.Timestamptz{Time: time.Now(), Valid: true}}, nil
}

func TestAuditTrail_CapturesBeforeAfterAndIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	now := time.Now()
	q := &fakeUserQuerier{
		createUserRow: db.CreateUserRow{
			ID:                  10,
			Username:            "alice",
			FullName:            "Alice",
			ForcePasswordChange: true,
			Active:              true,
			CreatedAt:           pgtype.Timestamptz{Time: now, Valid: true},
			UpdatedAt:           pgtype.Timestamptz{Time: now, Valid: true},
		},
	}
	h := NewUserHandler(q, nil)

	sink := &captureAuditSink{ch: make(chan db.InsertAuditLogParams, 1)}
	writer := auditlog.NewAsyncWriter(sink)

	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Set("user_id", int64(99))
		c.Next()
	})
	r.Use(middleware.AuditTrail(writer))
	r.POST("/users", h.CreateUser)

	body := `{"username":"alice","full_name":"Alice","password":"password123"}`
	req := httptest.NewRequest(http.MethodPost, "/users", strings.NewReader(body))
	req.RemoteAddr = "203.0.113.10:1234"
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "ua-test")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}

	select {
	case p := <-sink.ch:
		if p.Action != "USER_CREATE" {
			t.Fatalf("unexpected action: %q", p.Action)
		}
		if p.EntityType != "users" {
			t.Fatalf("unexpected entity_type: %q", p.EntityType)
		}
		if !p.EntityID.Valid || p.EntityID.Int64 != 10 {
			t.Fatalf("unexpected entity_id: %#v", p.EntityID)
		}
		if !p.UserID.Valid || p.UserID.Int64 != 99 {
			t.Fatalf("unexpected user_id: %#v", p.UserID)
		}
		if p.IpAddress == nil || *p.IpAddress != netip.MustParseAddr("203.0.113.10") {
			t.Fatalf("unexpected ip_address: %#v", p.IpAddress)
		}
		if !p.UserAgent.Valid || p.UserAgent.String != "ua-test" {
			t.Fatalf("unexpected user_agent: %#v", p.UserAgent)
		}
		if len(p.AfterState) == 0 || !strings.Contains(string(p.AfterState), `"username":"alice"`) {
			t.Fatalf("unexpected after_state: %s", string(p.AfterState))
		}
		if len(p.BeforeState) != 0 {
			t.Fatalf("expected before_state empty, got: %s", string(p.BeforeState))
		}
	case <-time.After(500 * time.Millisecond):
		t.Fatalf("expected audit log write")
	}
}

