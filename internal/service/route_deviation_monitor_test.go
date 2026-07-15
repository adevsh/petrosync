package service

import (
	"context"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/adevsh/petrosync/internal/db"
)

type fakeRouteDeviationMonitorQuerier struct {
	settingValue string
	settingErr   error

	candidates  []db.ListActiveTripsOffRouteRow
	openByTrip  map[int64]db.RouteDeviationEvent
	countByTrip map[int64]int32

	createCalls  []db.CreateDeviationEventParams
	resolveCalls []int64
}

func (q *fakeRouteDeviationMonitorQuerier) CountTripDeviations(ctx context.Context, tripID int64) (int32, error) {
	return q.countByTrip[tripID], nil
}

func (q *fakeRouteDeviationMonitorQuerier) CreateDeviationEvent(ctx context.Context, arg db.CreateDeviationEventParams) (db.RouteDeviationEvent, error) {
	q.createCalls = append(q.createCalls, arg)
	return db.RouteDeviationEvent{TripID: arg.TripID, OccurrenceCount: arg.OccurrenceCount}, nil
}

func (q *fakeRouteDeviationMonitorQuerier) GetGlobalSetting(ctx context.Context, key string) (db.SystemSetting, error) {
	if q.settingErr != nil {
		return db.SystemSetting{}, q.settingErr
	}
	return db.SystemSetting{Key: key, Value: q.settingValue}, nil
}

func (q *fakeRouteDeviationMonitorQuerier) GetOpenDeviationByTrip(ctx context.Context, tripID int64) (db.RouteDeviationEvent, error) {
	open, ok := q.openByTrip[tripID]
	if !ok {
		return db.RouteDeviationEvent{}, pgx.ErrNoRows
	}
	return open, nil
}

func (q *fakeRouteDeviationMonitorQuerier) ListActiveTripsOffRoute(ctx context.Context) ([]db.ListActiveTripsOffRouteRow, error) {
	return q.candidates, nil
}

func (q *fakeRouteDeviationMonitorQuerier) UpdateDeviationDuration(ctx context.Context, id int64) (db.RouteDeviationEvent, error) {
	q.resolveCalls = append(q.resolveCalls, id)
	return db.RouteDeviationEvent{ID: id, ResolvedAt: pgtype.Timestamptz{Valid: true}}, nil
}

func TestRouteDeviationMonitorService_Check_CreatesDeviationWhenTripMovesOffRoute(t *testing.T) {
	q := &fakeRouteDeviationMonitorQuerier{
		settingErr: pgx.ErrNoRows,
		candidates: []db.ListActiveTripsOffRouteRow{
			{TripID: 10, DeviationMeters: decimal.NewFromInt(650)},
		},
		openByTrip:  map[int64]db.RouteDeviationEvent{},
		countByTrip: map[int64]int32{10: 1},
	}

	created, resolved, err := NewRouteDeviationMonitorService(q).Check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created != 1 || resolved != 0 {
		t.Fatalf("expected created=1 resolved=0, got %d %d", created, resolved)
	}
	if len(q.createCalls) != 1 {
		t.Fatalf("expected one create call, got %d", len(q.createCalls))
	}
	if q.createCalls[0].OccurrenceCount != 2 {
		t.Fatalf("expected occurrence_count=2, got %d", q.createCalls[0].OccurrenceCount)
	}
}

func TestRouteDeviationMonitorService_Check_ResolvesDeviationWhenTripReturnsToRoute(t *testing.T) {
	q := &fakeRouteDeviationMonitorQuerier{
		settingErr: pgx.ErrNoRows,
		candidates: []db.ListActiveTripsOffRouteRow{
			{TripID: 11, DeviationMeters: decimal.NewFromInt(100)},
		},
		openByTrip: map[int64]db.RouteDeviationEvent{
			11: {ID: 77, TripID: 11},
		},
		countByTrip: map[int64]int32{},
	}

	created, resolved, err := NewRouteDeviationMonitorService(q).Check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created != 0 || resolved != 1 {
		t.Fatalf("expected created=0 resolved=1, got %d %d", created, resolved)
	}
	if len(q.resolveCalls) != 1 || q.resolveCalls[0] != 77 {
		t.Fatalf("unexpected resolve calls: %#v", q.resolveCalls)
	}
}

func TestRouteDeviationMonitorService_Check_UsesConfiguredThreshold(t *testing.T) {
	q := &fakeRouteDeviationMonitorQuerier{
		settingValue: "200",
		candidates: []db.ListActiveTripsOffRouteRow{
			{TripID: 12, DeviationMeters: decimal.NewFromInt(199)},
		},
		openByTrip:  map[int64]db.RouteDeviationEvent{},
		countByTrip: map[int64]int32{12: 0},
	}

	created, resolved, err := NewRouteDeviationMonitorService(q).Check(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created != 0 || resolved != 0 {
		t.Fatalf("expected no changes, got created=%d resolved=%d", created, resolved)
	}
	if len(q.createCalls) != 0 {
		t.Fatalf("expected no create calls, got %#v", q.createCalls)
	}
}
