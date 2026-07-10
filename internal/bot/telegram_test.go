package bot

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/service"
	"github.com/adevsh/petrosync/internal/telegram"
)

type fakeReplier struct {
	chatID int64
	text   string
}

func (f *fakeReplier) SendMessage(ctx context.Context, chatID int64, text string) (int64, error) {
	f.chatID = chatID
	f.text = text
	return 1, nil
}

type fakeLinkStore struct {
	q *fakeLinkQuerier
}

func (s *fakeLinkStore) ExecTx(ctx context.Context, fn func(q service.TelegramLinkQuerier) error) error {
	return fn(s.q)
}

type fakeLinkQuerier struct {
	telegramUserID int64
	usedToken      string
}

func (q *fakeLinkQuerier) GetValidTelegramLinkToken(ctx context.Context, token string) (db.GetValidTelegramLinkTokenRow, error) {
	return db.GetValidTelegramLinkTokenRow{
		UserID:         10,
		Token:          token,
		Username:       "alice",
		FullName:       "Alice",
		TelegramUserID: pgtype.Int8{Valid: false},
	}, nil
}

func (q *fakeLinkQuerier) GetUserByTelegramID(ctx context.Context, telegramUserID pgtype.Int8) (db.GetUserByTelegramIDRow, error) {
	return db.GetUserByTelegramIDRow{}, pgx.ErrNoRows
}

func (q *fakeLinkQuerier) LinkTelegramAccount(ctx context.Context, arg db.LinkTelegramAccountParams) error {
	if arg.TelegramUserID.Valid {
		q.telegramUserID = arg.TelegramUserID.Int64
	}
	return nil
}

func (q *fakeLinkQuerier) UseTelegramLinkToken(ctx context.Context, token string) (db.TelegramLinkToken, error) {
	q.usedToken = token
	return db.TelegramLinkToken{Token: token}, nil
}

func TestHandleUpdate_LinkUsage(t *testing.T) {
	linkSvc := service.NewTelegramLinkService(&fakeLinkStore{q: &fakeLinkQuerier{}})
	r := &fakeReplier{}

	upd := telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			From: &telegram.User{ID: 123},
			Chat: telegram.Chat{ID: 999},
			Text: "/link",
		},
	}

	if err := HandleUpdate(context.Background(), upd, linkSvc, r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.text != "Usage: /link <token>" {
		t.Fatalf("unexpected reply: %q", r.text)
	}
}

func TestHandleUpdate_LinkSuccess(t *testing.T) {
	q := &fakeLinkQuerier{}
	linkSvc := service.NewTelegramLinkService(&fakeLinkStore{q: q})
	r := &fakeReplier{}

	token := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	upd := telegram.Update{
		UpdateID: 1,
		Message: &telegram.Message{
			From: &telegram.User{ID: 123},
			Chat: telegram.Chat{ID: 999},
			Text: "/link " + token,
		},
	}

	if err := HandleUpdate(context.Background(), upd, linkSvc, r); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if r.text != "Linked successfully." {
		t.Fatalf("unexpected reply: %q", r.text)
	}
	if q.usedToken != token || q.telegramUserID != 123 {
		t.Fatalf("expected link called, got token=%q tg=%d", q.usedToken, q.telegramUserID)
	}
}
