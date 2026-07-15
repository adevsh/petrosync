package service

import (
	"context"
	"fmt"
	"log"
	"runtime/debug"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

type LoggedJobFunc func(context.Context) error

func NewLoggedJob(logger *log.Logger, name string, timeout time.Duration, fn LoggedJobFunc) func() {
	if logger == nil {
		logger = log.Default()
	}

	return func() {
		start := time.Now()
		logger.Printf("job %s started", name)
		defer func() {
			if r := recover(); r != nil {
				logger.Printf("job %s panic: %v\n%s", name, r, debug.Stack())
			}
		}()

		ctx, cancel := context.WithTimeout(context.Background(), timeout)
		defer cancel()

		if err := fn(ctx); err != nil {
			logger.Printf("job %s failed after %s: %v", name, time.Since(start).Round(time.Millisecond), err)
			return
		}
		logger.Printf("job %s completed in %s", name, time.Since(start).Round(time.Millisecond))
	}
}

type ExpiryNotificationService struct {
	q             *db.Queries
	notifications *NotificationCoordinator
}

func NewExpiryNotificationService(q *db.Queries, notifications *NotificationCoordinator) *ExpiryNotificationService {
	return &ExpiryNotificationService{q: q, notifications: notifications}
}

func (s *ExpiryNotificationService) NotifyExpiringLicenses(ctx context.Context) (int, error) {
	rows, err := s.q.ListDriversWithExpiringLicense(ctx)
	if err != nil {
		return 0, err
	}

	sent := 0
	for _, row := range rows {
		if err := s.notifications.NotifyDriverLicenseExpiring(ctx, row); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}

func (s *ExpiryNotificationService) NotifyExpiringKeur(ctx context.Context) (int, error) {
	rows, err := s.q.ListVehiclesWithExpiringKeur(ctx)
	if err != nil {
		return 0, err
	}

	sent := 0
	for _, row := range rows {
		if err := s.notifications.NotifyVehicleKeurExpiring(ctx, row); err != nil {
			return sent, err
		}
		sent++
	}
	return sent, nil
}

type RouteDeviationAlertService struct {
	q             *db.Queries
	notifications *NotificationCoordinator
}

func NewRouteDeviationAlertService(q *db.Queries, notifications *NotificationCoordinator) *RouteDeviationAlertService {
	return &RouteDeviationAlertService{q: q, notifications: notifications}
}

func (s *RouteDeviationAlertService) CheckRouteDeviations(ctx context.Context) (int, error) {
	minutes, err := s.q.GetGlobalSetting(ctx, "route_deviation_alert_minutes")
	if err != nil {
		return 0, err
	}

	rows, err := s.q.ListUnnotifiedDeviationsAboveThreshold(ctx, pgtype.Text{String: minutes.Value, Valid: true})
	if err != nil {
		return 0, err
	}

	count := 0
	for _, row := range rows {
		if err := s.notifications.NotifyRouteDeviationEscalated(ctx, row); err != nil {
			return count, err
		}
		if err := s.q.MarkDeviationTelegramNotified(ctx, row.ID); err != nil {
			return count, err
		}
		count++
	}

	return count, nil
}

type RouteDeviationWorkerService struct {
	monitor *RouteDeviationMonitorService
	alerts  *RouteDeviationAlertService
}

func NewRouteDeviationWorkerService(monitor *RouteDeviationMonitorService, alerts *RouteDeviationAlertService) *RouteDeviationWorkerService {
	return &RouteDeviationWorkerService{monitor: monitor, alerts: alerts}
}

func (s *RouteDeviationWorkerService) Run(ctx context.Context) (int, int, int, error) {
	created, resolved, err := s.monitor.Check(ctx)
	if err != nil {
		return created, resolved, 0, err
	}
	alerted, err := s.alerts.CheckRouteDeviations(ctx)
	if err != nil {
		return created, resolved, alerted, err
	}
	return created, resolved, alerted, nil
}

type TelegramLinkCleanupService struct {
	q *db.Queries
}

func NewTelegramLinkCleanupService(q *db.Queries) *TelegramLinkCleanupService {
	return &TelegramLinkCleanupService{q: q}
}

func (s *TelegramLinkCleanupService) CleanupExpired(ctx context.Context) (int64, error) {
	return s.q.DeleteExpiredTelegramLinkTokens(ctx)
}

func LogJobCount(logger *log.Logger, name string, count int) {
	if logger == nil {
		logger = log.Default()
	}
	if count > 0 {
		logger.Printf("job %s affected %d record(s)", name, count)
	}
}

func LogJobCount64(logger *log.Logger, name string, count int64) {
	if logger == nil {
		logger = log.Default()
	}
	if count > 0 {
		logger.Printf("job %s affected %d record(s)", name, count)
	}
}

func JobResultError(name string, count int, err error) error {
	if err == nil {
		return nil
	}
	return fmt.Errorf("%s after %d record(s): %w", name, count, err)
}
