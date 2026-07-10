package service

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

type fakeNotificationStore struct {
	arg *db.InsertNotificationParams
}

func (s *fakeNotificationStore) InsertNotification(ctx context.Context, arg db.InsertNotificationParams) (db.NotificationLog, error) {
	s.arg = &arg
	return db.NotificationLog{
		ID:                  1,
		RecipientTelegramID: arg.RecipientTelegramID,
		DeliveryStatus:      arg.DeliveryStatus,
		ErrorMessage:        arg.ErrorMessage,
		TelegramMessageID:   arg.TelegramMessageID,
	}, nil
}

type fakeTelegramSender struct {
	messageID int64
	err       error
}

func (s *fakeTelegramSender) SendMessage(ctx context.Context, chatID int64, text string) (int64, error) {
	return s.messageID, s.err
}

func TestNotificationService_Send_Success(t *testing.T) {
	store := &fakeNotificationStore{}
	sender := &fakeTelegramSender{messageID: 77}
	svc := NewNotificationService(store, sender)

	tripID := int64(10)
	userID := int64(20)
	row, err := svc.Send(context.Background(), SendNotificationRequest{
		TripID:              &tripID,
		RecipientTelegramID: 123,
		RecipientUserID:     &userID,
		NotificationType:    db.NotificationTypeTDOAPPROVED,
		MessageText:         "hello",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.arg == nil {
		t.Fatalf("expected InsertNotification called")
	}
	if store.arg.DeliveryStatus != "SUCCESS" {
		t.Fatalf("unexpected status: %s", store.arg.DeliveryStatus)
	}
	if !store.arg.TelegramMessageID.Valid || store.arg.TelegramMessageID.Int64 != 77 {
		t.Fatalf("unexpected message id: %#v", store.arg.TelegramMessageID)
	}
	if store.arg.ErrorMessage.Valid {
		t.Fatalf("expected error_message null")
	}
	if row.DeliveryStatus != "SUCCESS" {
		t.Fatalf("unexpected row status: %s", row.DeliveryStatus)
	}
}

func TestNotificationService_Send_Failed(t *testing.T) {
	store := &fakeNotificationStore{}
	sendErr := errors.New("boom")
	sender := &fakeTelegramSender{err: sendErr}
	svc := NewNotificationService(store, sender)

	row, err := svc.Send(context.Background(), SendNotificationRequest{
		RecipientTelegramID: 123,
		NotificationType:    db.NotificationTypeTDOAPPROVED,
		MessageText:         "hello",
	})
	if err == nil || err.Error() != "boom" {
		t.Fatalf("expected send error, got %v", err)
	}
	if store.arg == nil {
		t.Fatalf("expected InsertNotification called")
	}
	if store.arg.DeliveryStatus != "FAILED" {
		t.Fatalf("unexpected status: %s", store.arg.DeliveryStatus)
	}
	if store.arg.TelegramMessageID.Valid {
		t.Fatalf("expected message id null on failure")
	}
	if !store.arg.ErrorMessage.Valid || store.arg.ErrorMessage.String != "boom" {
		t.Fatalf("unexpected error message: %#v", store.arg.ErrorMessage)
	}
	if row.ErrorMessage != (pgtype.Text{String: "boom", Valid: true}) {
		t.Fatalf("unexpected row error message: %#v", row.ErrorMessage)
	}
}
