package service

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

type fakeTelegramLinkStore struct {
	q *fakeTelegramLinkQuerier
}

func (s *fakeTelegramLinkStore) ExecTx(ctx context.Context, fn func(q TelegramLinkQuerier) error) error {
	return fn(s.q)
}

type fakeTelegramLinkQuerier struct {
	getValidToken     string
	getValidRow       db.GetValidTelegramLinkTokenRow
	getValidErr       error
	getUserByTgCalled bool
	useCalled         bool
	linkCalled        bool

	linkedUserID     int64
	linkedTelegramID int64
	usedToken        string
}

func (q *fakeTelegramLinkQuerier) GetValidTelegramLinkToken(ctx context.Context, token string) (db.GetValidTelegramLinkTokenRow, error) {
	q.getValidToken = token
	return q.getValidRow, q.getValidErr
}

func (q *fakeTelegramLinkQuerier) GetUserByTelegramID(ctx context.Context, telegramUserID pgtype.Int8) (db.GetUserByTelegramIDRow, error) {
	q.getUserByTgCalled = true
	return db.GetUserByTelegramIDRow{}, pgx.ErrNoRows
}

func (q *fakeTelegramLinkQuerier) LinkTelegramAccount(ctx context.Context, arg db.LinkTelegramAccountParams) error {
	q.linkCalled = true
	q.linkedUserID = arg.ID
	if arg.TelegramUserID.Valid {
		q.linkedTelegramID = arg.TelegramUserID.Int64
	}
	return nil
}

func (q *fakeTelegramLinkQuerier) UseTelegramLinkToken(ctx context.Context, token string) (db.TelegramLinkToken, error) {
	q.useCalled = true
	q.usedToken = token
	return db.TelegramLinkToken{Token: token}, nil
}

func TestTelegramLinkService_LinkByToken_Success(t *testing.T) {
	q := &fakeTelegramLinkQuerier{
		getValidRow: db.GetValidTelegramLinkTokenRow{
			UserID:         10,
			Token:          "t",
			Username:       "alice",
			FullName:       "Alice",
			TelegramUserID: pgtype.Int8{Valid: false},
		},
	}
	svc := NewTelegramLinkService(&fakeTelegramLinkStore{q: q})

	token := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	row, err := svc.LinkByToken(context.Background(), 123, token)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if row.UserID != 10 {
		t.Fatalf("unexpected user id: %d", row.UserID)
	}
	if !q.useCalled || q.usedToken != token {
		t.Fatalf("expected token used")
	}
	if !q.linkCalled || q.linkedUserID != 10 || q.linkedTelegramID != 123 {
		t.Fatalf("unexpected link: user=%d tg=%d", q.linkedUserID, q.linkedTelegramID)
	}
}

func TestTelegramLinkService_LinkByToken_InvalidToken(t *testing.T) {
	q := &fakeTelegramLinkQuerier{
		getValidErr: pgx.ErrNoRows,
	}
	svc := NewTelegramLinkService(&fakeTelegramLinkStore{q: q})

	_, err := svc.LinkByToken(context.Background(), 123, "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa")
	if err != ErrInvalidOrExpiredLinkToken {
		t.Fatalf("expected ErrInvalidOrExpiredLinkToken, got %v", err)
	}
}
