package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

type fakeTelegramLinkQuerier struct {
	createArgs *db.CreateTelegramLinkTokenParams
}

func (f *fakeTelegramLinkQuerier) GetUser(ctx context.Context, id int64) (db.GetUserRow, error) {
	return db.GetUserRow{ID: id, Username: "u", FullName: "n", Active: true}, nil
}

func (f *fakeTelegramLinkQuerier) CreateTelegramLinkToken(ctx context.Context, arg db.CreateTelegramLinkTokenParams) (db.TelegramLinkToken, error) {
	f.createArgs = &arg
	now := time.Now()
	return db.TelegramLinkToken{
		ID:        1,
		UserID:    arg.UserID,
		Token:     arg.Token,
		ExpiresAt: pgtype.Timestamptz{Time: now.Add(48 * time.Hour), Valid: true},
		CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
	}, nil
}

func TestTelegramLinkTokenHandler_CreateLinkToken(t *testing.T) {
	gin.SetMode(gin.TestMode)

	q := &fakeTelegramLinkQuerier{}
	h := NewTelegramLinkTokenHandler(q)
	r := gin.New()
	r.POST("/users/:id/telegram/link-token", h.CreateLinkToken)

	req := httptest.NewRequest(http.MethodPost, "/users/10/telegram/link-token", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", w.Code, w.Body.String())
	}
	if q.createArgs == nil {
		t.Fatalf("expected CreateTelegramLinkToken called")
	}
	if q.createArgs.UserID != 10 {
		t.Fatalf("unexpected user id: %d", q.createArgs.UserID)
	}
	if len(q.createArgs.Token) != 64 {
		t.Fatalf("expected 64-char token, got %q", q.createArgs.Token)
	}

	var resp struct {
		Data struct {
			Token     string     `json:"token"`
			ExpiresAt *time.Time `json:"expires_at"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if resp.Data.Token == "" || len(resp.Data.Token) != 64 {
		t.Fatalf("unexpected token: %q", resp.Data.Token)
	}
	if resp.Data.ExpiresAt == nil {
		t.Fatalf("expected expires_at")
	}
}
