package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

var ErrTelegramNotConfigured = errors.New("telegram bot not configured")

type TelegramSender interface {
	SendMessage(ctx context.Context, chatID int64, text string) (int64, error)
}

type NotificationStore interface {
	InsertNotification(ctx context.Context, arg db.InsertNotificationParams) (db.NotificationLog, error)
}

type NotificationService struct {
	store  NotificationStore
	sender TelegramSender
}

func NewNotificationService(store NotificationStore, sender TelegramSender) *NotificationService {
	return &NotificationService{store: store, sender: sender}
}

type SendNotificationRequest struct {
	TripID              *int64
	DOID                *int64
	RecipientTelegramID int64
	RecipientUserID     *int64
	NotificationType    db.NotificationTypeT
	MessageText         string
}

func (s *NotificationService) Send(ctx context.Context, req SendNotificationRequest) (db.NotificationLog, error) {
	var (
		messageID int64
		sendErr   error
	)
	if s.sender == nil {
		sendErr = ErrTelegramNotConfigured
	} else {
		messageID, sendErr = s.sender.SendMessage(ctx, req.RecipientTelegramID, req.MessageText)
	}

	status := "SUCCESS"
	if sendErr != nil {
		status = "FAILED"
	}

	var tripID pgtype.Int8
	if req.TripID != nil {
		tripID = pgtype.Int8{Int64: *req.TripID, Valid: true}
	}
	var doID pgtype.Int8
	if req.DOID != nil {
		doID = pgtype.Int8{Int64: *req.DOID, Valid: true}
	}
	var userID pgtype.Int8
	if req.RecipientUserID != nil {
		userID = pgtype.Int8{Int64: *req.RecipientUserID, Valid: true}
	}

	msgID := pgtype.Int8{Valid: false}
	if sendErr == nil {
		msgID = pgtype.Int8{Int64: messageID, Valid: true}
	}

	errMsg := pgtype.Text{Valid: false}
	if sendErr != nil {
		errMsg = pgtype.Text{String: sendErr.Error(), Valid: true}
	}

	logRow, err := s.store.InsertNotification(ctx, db.InsertNotificationParams{
		TripID:              tripID,
		DoID:                doID,
		RecipientTelegramID: req.RecipientTelegramID,
		RecipientUserID:     userID,
		NotificationType:    req.NotificationType,
		MessageText:         req.MessageText,
		DeliveryStatus:      status,
		TelegramMessageID:   msgID,
		ErrorMessage:        errMsg,
	})
	if err != nil {
		return db.NotificationLog{}, err
	}
	if sendErr != nil {
		return logRow, sendErr
	}
	return logRow, nil
}
