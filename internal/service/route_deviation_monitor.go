package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/adevsh/petrosync/internal/db"
)

const defaultRouteDeviationThresholdMeters = 500

type RouteDeviationMonitorQuerier interface {
	CountTripDeviations(ctx context.Context, tripID int64) (int32, error)
	CreateDeviationEvent(ctx context.Context, arg db.CreateDeviationEventParams) (db.RouteDeviationEvent, error)
	GetGlobalSetting(ctx context.Context, key string) (db.SystemSetting, error)
	GetOpenDeviationByTrip(ctx context.Context, tripID int64) (db.RouteDeviationEvent, error)
	ListActiveTripsOffRoute(ctx context.Context) ([]db.ListActiveTripsOffRouteRow, error)
	UpdateDeviationDuration(ctx context.Context, id int64) (db.RouteDeviationEvent, error)
}

type RouteDeviationMonitorService struct {
	q RouteDeviationMonitorQuerier
}

func NewRouteDeviationMonitorService(q RouteDeviationMonitorQuerier) *RouteDeviationMonitorService {
	return &RouteDeviationMonitorService{q: q}
}

func (s *RouteDeviationMonitorService) Check(ctx context.Context) (int, int, error) {
	threshold, err := s.thresholdMeters(ctx)
	if err != nil {
		return 0, 0, err
	}

	candidates, err := s.q.ListActiveTripsOffRoute(ctx)
	if err != nil {
		return 0, 0, err
	}

	created := 0
	resolved := 0

	for _, candidate := range candidates {
		open, err := s.q.GetOpenDeviationByTrip(ctx, candidate.TripID)
		hasOpen := err == nil
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return created, resolved, err
		}

		if candidate.DeviationMeters.GreaterThanOrEqual(threshold) {
			if hasOpen {
				continue
			}

			count, err := s.q.CountTripDeviations(ctx, candidate.TripID)
			if err != nil {
				return created, resolved, err
			}

			if _, err := s.q.CreateDeviationEvent(ctx, db.CreateDeviationEventParams{
				TripID:          candidate.TripID,
				DeviationMeters: decimalToNumeric(candidate.DeviationMeters),
				OccurrenceCount: int16(count + 1),
			}); err != nil {
				return created, resolved, err
			}
			created++
			continue
		}

		if hasOpen {
			if _, err := s.q.UpdateDeviationDuration(ctx, open.ID); err != nil {
				return created, resolved, err
			}
			resolved++
		}
	}

	return created, resolved, nil
}

func (s *RouteDeviationMonitorService) thresholdMeters(ctx context.Context) (decimal.Decimal, error) {
	setting, err := s.q.GetGlobalSetting(ctx, "route_deviation_threshold_m")
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return decimal.NewFromInt(defaultRouteDeviationThresholdMeters), nil
		}
		return decimal.Decimal{}, err
	}

	value, err := decimal.NewFromString(setting.Value)
	if err != nil {
		return decimal.Decimal{}, err
	}
	return value, nil
}

func decimalToNumeric(d decimal.Decimal) pgtype.Numeric {
	return pgtype.Numeric{
		Int:   d.Coefficient(),
		Exp:   d.Exponent(),
		Valid: true,
	}
}
