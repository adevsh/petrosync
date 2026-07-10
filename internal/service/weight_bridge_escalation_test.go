package service

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

type fakeWeightBridgeEscalationQuerier struct {
	settingValue string
	admins       []db.ListUsersWithCompanyRoleRow
	overdue      []db.ListOverduePendingManualApprovalsRow

	escalateCalls []db.EscalateWeightBridgeReadingParams
	escalateErr   error
}

func (q *fakeWeightBridgeEscalationQuerier) GetGlobalSetting(ctx context.Context, key string) (db.SystemSetting, error) {
	return db.SystemSetting{Key: key, Value: q.settingValue}, nil
}

func (q *fakeWeightBridgeEscalationQuerier) ListUsersWithCompanyRole(ctx context.Context, role db.UserRoleT) ([]db.ListUsersWithCompanyRoleRow, error) {
	return q.admins, nil
}

func (q *fakeWeightBridgeEscalationQuerier) ListOverduePendingManualApprovals(ctx context.Context, escalationHours int32) ([]db.ListOverduePendingManualApprovalsRow, error) {
	return q.overdue, nil
}

func (q *fakeWeightBridgeEscalationQuerier) EscalateWeightBridgeReading(ctx context.Context, arg db.EscalateWeightBridgeReadingParams) (db.WeightBridgeReading, error) {
	q.escalateCalls = append(q.escalateCalls, arg)
	if q.escalateErr != nil {
		return db.WeightBridgeReading{}, q.escalateErr
	}
	return db.WeightBridgeReading{ID: arg.ID, ApprovalStatus: db.ApprovalStatusTESCALATED}, nil
}

type fakeNotifier struct {
	reqs []SendNotificationRequest
}

func (n *fakeNotifier) Send(ctx context.Context, req SendNotificationRequest) (db.NotificationLog, error) {
	n.reqs = append(n.reqs, req)
	return db.NotificationLog{}, nil
}

func TestWeightBridgeEscalationService_Run_EscalatesAndNotifies(t *testing.T) {
	q := &fakeWeightBridgeEscalationQuerier{
		settingValue: "2",
		admins: []db.ListUsersWithCompanyRoleRow{
			{ID: 100, TelegramUserID: pgtype.Int8{Int64: 555, Valid: true}},
			{ID: 200, TelegramUserID: pgtype.Int8{Valid: false}},
		},
		overdue: []db.ListOverduePendingManualApprovalsRow{
			{
				ID:             1,
				TripID:          pgtype.Int8{Int64: 10, Valid: true},
				PlateNumber:     "B-1234-XYZ",
				ReadingType:     "GROSS",
				RecordedByName:  "Operator A",
				ApprovalStatus:  db.ApprovalStatusTPENDING,
				Method:          db.MeasurementMethodTMANUALAPPROVED,
				RecordedBy:      999,
				VehicleID:       123,
				AmbientTempCelsius: pgtype.Numeric{Valid: false},
			},
		},
	}
	notifier := &fakeNotifier{}
	svc := NewWeightBridgeEscalationService(q, notifier)

	n, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 1 {
		t.Fatalf("expected 1 escalated, got %d", n)
	}
	if len(q.escalateCalls) != 1 {
		t.Fatalf("expected 1 escalate call, got %d", len(q.escalateCalls))
	}
	if q.escalateCalls[0].ID != 1 || !q.escalateCalls[0].EscalatedTo.Valid || q.escalateCalls[0].EscalatedTo.Int64 != 100 {
		t.Fatalf("unexpected escalate call: %#v", q.escalateCalls[0])
	}
	if len(notifier.reqs) != 1 {
		t.Fatalf("expected 1 notification, got %d", len(notifier.reqs))
	}
	if notifier.reqs[0].NotificationType != db.NotificationTypeTMANUALMEASUREMENTESCALATED {
		t.Fatalf("unexpected notification type: %v", notifier.reqs[0].NotificationType)
	}
	if notifier.reqs[0].RecipientTelegramID != 555 {
		t.Fatalf("unexpected recipient telegram id: %d", notifier.reqs[0].RecipientTelegramID)
	}
}

func TestWeightBridgeEscalationService_Run_InvalidSettingValue(t *testing.T) {
	q := &fakeWeightBridgeEscalationQuerier{
		settingValue: "abc",
		admins: []db.ListUsersWithCompanyRoleRow{
			{ID: 100, TelegramUserID: pgtype.Int8{Int64: 555, Valid: true}},
		},
	}
	svc := NewWeightBridgeEscalationService(q, &fakeNotifier{})

	_, err := svc.Run(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestWeightBridgeEscalationService_Run_NoAdmins(t *testing.T) {
	q := &fakeWeightBridgeEscalationQuerier{
		settingValue: "2",
		admins:       []db.ListUsersWithCompanyRoleRow{},
		overdue: []db.ListOverduePendingManualApprovalsRow{
			{ID: 1},
		},
	}
	svc := NewWeightBridgeEscalationService(q, &fakeNotifier{})

	_, err := svc.Run(context.Background())
	if err == nil {
		t.Fatalf("expected error, got nil")
	}
}

func TestWeightBridgeEscalationService_Run_SkipsAlreadyEscalated(t *testing.T) {
	q := &fakeWeightBridgeEscalationQuerier{
		settingValue: "2",
		admins: []db.ListUsersWithCompanyRoleRow{
			{ID: 100, TelegramUserID: pgtype.Int8{Int64: 555, Valid: true}},
		},
		overdue: []db.ListOverduePendingManualApprovalsRow{
			{ID: 1},
		},
		escalateErr: pgx.ErrNoRows,
	}
	notifier := &fakeNotifier{}
	svc := NewWeightBridgeEscalationService(q, notifier)

	n, err := svc.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if n != 0 {
		t.Fatalf("expected 0 escalated, got %d", n)
	}
	if len(notifier.reqs) != 0 {
		t.Fatalf("expected 0 notifications, got %d", len(notifier.reqs))
	}
}
