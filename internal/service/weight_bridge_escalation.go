package service

import (
	"context"
	"errors"
	"fmt"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

type WeightBridgeEscalationQuerier interface {
	EscalateWeightBridgeReading(ctx context.Context, arg db.EscalateWeightBridgeReadingParams) (db.WeightBridgeReading, error)
	GetGlobalSetting(ctx context.Context, key string) (db.SystemSetting, error)
	ListOverduePendingManualApprovals(ctx context.Context, escalationHours int32) ([]db.ListOverduePendingManualApprovalsRow, error)
	ListUsersWithCompanyRole(ctx context.Context, role db.UserRoleT) ([]db.ListUsersWithCompanyRoleRow, error)
}

type WeightBridgeEscalationNotifier interface {
	Send(ctx context.Context, req SendNotificationRequest) (db.NotificationLog, error)
}

type WeightBridgeEscalationService struct {
	q        WeightBridgeEscalationQuerier
	notifier WeightBridgeEscalationNotifier
}

func NewWeightBridgeEscalationService(q WeightBridgeEscalationQuerier, notifier WeightBridgeEscalationNotifier) *WeightBridgeEscalationService {
	return &WeightBridgeEscalationService{q: q, notifier: notifier}
}

func (s *WeightBridgeEscalationService) Run(ctx context.Context) (int, error) {
	setting, err := s.q.GetGlobalSetting(ctx, "approval_escalation_hours")
	if err != nil {
		return 0, err
	}
	hours64, err := strconv.ParseInt(setting.Value, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid approval_escalation_hours: %w", err)
	}
	escalationHours := int32(hours64)
	if escalationHours <= 0 {
		return 0, fmt.Errorf("invalid approval_escalation_hours: %d", escalationHours)
	}

	admins, err := s.q.ListUsersWithCompanyRole(ctx, db.UserRoleTREFINERYADMIN)
	if err != nil {
		return 0, err
	}
	if len(admins) == 0 {
		return 0, fmt.Errorf("no active REFINERY_ADMIN users in COMPANY scope")
	}

	overdue, err := s.q.ListOverduePendingManualApprovals(ctx, escalationHours)
	if err != nil {
		return 0, err
	}

	assigneeID := admins[0].ID
	escalatedCount := 0
	for _, row := range overdue {
		_, err := s.q.EscalateWeightBridgeReading(ctx, db.EscalateWeightBridgeReadingParams{
			ID:          row.ID,
			EscalatedTo: pgtype.Int8{Int64: assigneeID, Valid: true},
		})
		if errors.Is(err, pgx.ErrNoRows) {
			continue
		}
		if err != nil {
			return escalatedCount, err
		}
		escalatedCount++

		if s.notifier == nil {
			continue
		}

		tripIDPtr := int64PtrFromPG(row.TripID)
		messageText := formatManualMeasurementEscalationMessage(row)
		for _, admin := range admins {
			if !admin.TelegramUserID.Valid {
				continue
			}
			adminID := admin.ID
			_, sendErr := s.notifier.Send(ctx, SendNotificationRequest{
				TripID:              tripIDPtr,
				RecipientTelegramID: admin.TelegramUserID.Int64,
				RecipientUserID:     &adminID,
				NotificationType:    db.NotificationTypeTMANUALMEASUREMENTESCALATED,
				MessageText:         messageText,
			})
			if sendErr != nil && !errors.Is(sendErr, ErrTelegramNotConfigured) {
				continue
			}
		}
	}

	return escalatedCount, nil
}

func int64PtrFromPG(v pgtype.Int8) *int64 {
	if !v.Valid {
		return nil
	}
	out := v.Int64
	return &out
}

func formatManualMeasurementEscalationMessage(row db.ListOverduePendingManualApprovalsRow) string {
	tripID := "N/A"
	if row.TripID.Valid {
		tripID = strconv.FormatInt(row.TripID.Int64, 10)
	}
	return fmt.Sprintf(
		"Manual weight bridge approval escalated\nReading ID: %d\nTrip ID: %s\nVehicle: %s\nType: %s\nRecorded by: %s",
		row.ID,
		tripID,
		row.PlateNumber,
		row.ReadingType,
		row.RecordedByName,
	)
}

