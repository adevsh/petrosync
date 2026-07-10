package service

import (
	"context"
	"math/big"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
)

func numericInt64(v int64) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(v), Exp: 0, Valid: true}
}

func numericInt64Exp(v int64, exp int32) pgtype.Numeric {
	return pgtype.Numeric{Int: big.NewInt(v), Exp: exp, Valid: true}
}

type fakeTankWorkflowStore struct {
	q *fakeTankWorkflowQuerier
}

func (s *fakeTankWorkflowStore) ExecTx(ctx context.Context, fn func(q db.TankWorkflowQuerier) error) error {
	return fn(s.q)
}

type reserveCall struct {
	tankID int64
	vol    pgtype.Numeric
}

type releaseCall struct {
	tankID int64
	vol    pgtype.Numeric
}

type deductCall struct {
	tankID int64
	vol    pgtype.Numeric
}

type creditCall struct {
	tankID int64
	vol    pgtype.Numeric
}

type stationTankUpdateCall struct {
	tankID int64
	vol    pgtype.Numeric
}

type compartmentStatusCall struct {
	id     int64
	status db.CompartmentDeliveryStatusT
}

type fakeTankWorkflowQuerier struct {
	do                     db.DeliveryOrder
	doItems                []db.ListDOItemsByDORow
	tankByFuel             map[string]db.FacilityStorageTank
	stationTankByFuel      map[string]db.StationTank
	trip                   db.GetTripRow
	tripCompartmentDetails []db.ListCompartmentDeliveriesByTripRow
	tripLoadedByFuel        []db.ListTripLoadedVolumeByFuelRow
	tripDeliveredByFuel     []db.ListTripDeliveredVolumeByFuelRow
	weightBridgeReadings    []db.WeightBridgeReading
	insertTripEventResp     db.TripEvent
	mandatoryPhotoCheck     db.GetMandatoryPhotoCheckByTripRow
	vehicleCompartments     []db.VehicleCompartment
	photosByTripAndEvent    []db.TripPhoto
	reserveErr              error
	effectiveSettingValue   string

	reserveCalls []reserveCall
	releaseCalls []releaseCall
	deductCalls  []deductCall
	creditCalls  []creditCall
	stationTankUpdateCalls  []stationTankUpdateCall
	compartmentStatusCalls  []compartmentStatusCall

	updatedTripStatuses []db.TripStatusT
	updatedDOStatuses   []db.DoStatusT
}

func (q *fakeTankWorkflowQuerier) GetDeliveryOrder(ctx context.Context, id int64) (db.DeliveryOrder, error) {
	return q.do, nil
}

func (q *fakeTankWorkflowQuerier) ApproveDeliveryOrder(ctx context.Context, arg db.ApproveDeliveryOrderParams) (db.DeliveryOrder, error) {
	out := q.do
	out.Status = db.DoStatusTAPPROVED
	return out, nil
}

func (q *fakeTankWorkflowQuerier) CancelDeliveryOrder(ctx context.Context, id int64) (db.DeliveryOrder, error) {
	out := q.do
	out.Status = db.DoStatusTCANCELLED
	return out, nil
}

func (q *fakeTankWorkflowQuerier) ListDOItemsByDO(ctx context.Context, doID int64) ([]db.ListDOItemsByDORow, error) {
	return q.doItems, nil
}

func (q *fakeTankWorkflowQuerier) GetStorageTankByFacilityAndFuel(ctx context.Context, arg db.GetStorageTankByFacilityAndFuelParams) (db.FacilityStorageTank, error) {
	return q.tankByFuel[arg.FuelTypeCode], nil
}

func (q *fakeTankWorkflowQuerier) ReserveStorageTankVolume(ctx context.Context, arg db.ReserveStorageTankVolumeParams) (db.FacilityStorageTank, error) {
	q.reserveCalls = append(q.reserveCalls, reserveCall{tankID: arg.ID, vol: arg.ReservedVolumeL})
	if q.reserveErr != nil {
		return db.FacilityStorageTank{}, q.reserveErr
	}
	return db.FacilityStorageTank{ID: arg.ID}, nil
}

func (q *fakeTankWorkflowQuerier) ReleaseStorageTankReservation(ctx context.Context, arg db.ReleaseStorageTankReservationParams) (db.FacilityStorageTank, error) {
	q.releaseCalls = append(q.releaseCalls, releaseCall{tankID: arg.ID, vol: arg.ReservedVolumeL})
	return db.FacilityStorageTank{ID: arg.ID}, nil
}

func (q *fakeTankWorkflowQuerier) GetTrip(ctx context.Context, id int64) (db.GetTripRow, error) {
	return q.trip, nil
}

func (q *fakeTankWorkflowQuerier) GetMandatoryPhotoCheckByTrip(ctx context.Context, tripID int64) (db.GetMandatoryPhotoCheckByTripRow, error) {
	return q.mandatoryPhotoCheck, nil
}

func (q *fakeTankWorkflowQuerier) ListCompartmentsByVehicle(ctx context.Context, vehicleID int64) ([]db.VehicleCompartment, error) {
	return q.vehicleCompartments, nil
}

func (q *fakeTankWorkflowQuerier) ListPhotosByTripAndEvent(ctx context.Context, arg db.ListPhotosByTripAndEventParams) ([]db.TripPhoto, error) {
	return q.photosByTripAndEvent, nil
}

func (q *fakeTankWorkflowQuerier) InsertTripEvent(ctx context.Context, arg db.InsertTripEventParams) (db.TripEvent, error) {
	return q.insertTripEventResp, nil
}

func (q *fakeTankWorkflowQuerier) UpdateTripStatus(ctx context.Context, arg db.UpdateTripStatusParams) (db.Trip, error) {
	q.updatedTripStatuses = append(q.updatedTripStatuses, arg.Status)
	return db.Trip{ID: arg.ID, Status: arg.Status}, nil
}

func (q *fakeTankWorkflowQuerier) UpdateDOStatus(ctx context.Context, arg db.UpdateDOStatusParams) (db.DeliveryOrder, error) {
	q.updatedDOStatuses = append(q.updatedDOStatuses, arg.Status)
	return db.DeliveryOrder{ID: arg.ID, Status: arg.Status}, nil
}

func (q *fakeTankWorkflowQuerier) ListTripLoadedVolumeByFuel(ctx context.Context, tripID int64) ([]db.ListTripLoadedVolumeByFuelRow, error) {
	return q.tripLoadedByFuel, nil
}

func (q *fakeTankWorkflowQuerier) ListTripDeliveredVolumeByFuel(ctx context.Context, tripID int64) ([]db.ListTripDeliveredVolumeByFuelRow, error) {
	return q.tripDeliveredByFuel, nil
}

func (q *fakeTankWorkflowQuerier) ListWeightBridgeReadingsByTrip(ctx context.Context, tripID pgtype.Int8) ([]db.WeightBridgeReading, error) {
	return q.weightBridgeReadings, nil
}

func (q *fakeTankWorkflowQuerier) DeductStorageTankVolume(ctx context.Context, arg db.DeductStorageTankVolumeParams) (db.FacilityStorageTank, error) {
	q.deductCalls = append(q.deductCalls, deductCall{tankID: arg.ID, vol: arg.CurrentVolumeL})
	return db.FacilityStorageTank{ID: arg.ID}, nil
}

func (q *fakeTankWorkflowQuerier) CreditStorageTankVolume(ctx context.Context, arg db.CreditStorageTankVolumeParams) (db.FacilityStorageTank, error) {
	q.creditCalls = append(q.creditCalls, creditCall{tankID: arg.ID, vol: arg.CurrentVolumeL})
	return db.FacilityStorageTank{ID: arg.ID}, nil
}

func (q *fakeTankWorkflowQuerier) GetEffectiveSetting(ctx context.Context, arg db.GetEffectiveSettingParams) (string, error) {
	if q.effectiveSettingValue == "" {
		return "0.3", nil
	}
	return q.effectiveSettingValue, nil
}

func (q *fakeTankWorkflowQuerier) ListCompartmentDeliveriesByTrip(ctx context.Context, tripID int64) ([]db.ListCompartmentDeliveriesByTripRow, error) {
	return q.tripCompartmentDetails, nil
}

func (q *fakeTankWorkflowQuerier) UpdateCompartmentDeliveryStatus(ctx context.Context, arg db.UpdateCompartmentDeliveryStatusParams) (db.TripCompartmentDelivery, error) {
	q.compartmentStatusCalls = append(q.compartmentStatusCalls, compartmentStatusCall{id: arg.ID, status: arg.DeliveryStatus})
	return db.TripCompartmentDelivery{ID: arg.ID, DeliveryStatus: arg.DeliveryStatus}, nil
}

func (q *fakeTankWorkflowQuerier) GetStationTankByFuel(ctx context.Context, arg db.GetStationTankByFuelParams) (db.StationTank, error) {
	return q.stationTankByFuel[arg.FuelTypeCode], nil
}

func (q *fakeTankWorkflowQuerier) UpdateStationTankVolumeAfterDelivery(ctx context.Context, arg db.UpdateStationTankVolumeAfterDeliveryParams) (db.StationTank, error) {
	q.stationTankUpdateCalls = append(q.stationTankUpdateCalls, stationTankUpdateCall{tankID: arg.ID, vol: arg.CurrentVolumeL})
	return db.StationTank{ID: arg.ID}, nil
}

func TestWorkflowService_ApproveDeliveryOrder_ReservesPerItem(t *testing.T) {
	q := &fakeTankWorkflowQuerier{
		do: db.DeliveryOrder{ID: 10, OriginFacilityID: 1, Status: db.DoStatusTPENDINGAPPROVAL},
		doItems: []db.ListDOItemsByDORow{
			{FuelTypeCode: "PERTAMAX", RequestedVolumeL: numericInt64(1000)},
			{FuelTypeCode: "BIO_SOLAR", RequestedVolumeL: numericInt64(2000)},
		},
		tankByFuel: map[string]db.FacilityStorageTank{
			"PERTAMAX":  {ID: 101, FacilityID: 1, FuelTypeCode: "PERTAMAX"},
			"BIO_SOLAR": {ID: 102, FacilityID: 1, FuelTypeCode: "BIO_SOLAR"},
		},
	}
	svc := NewWorkflowService(&fakeTankWorkflowStore{q: q})

	_, err := svc.ApproveDeliveryOrder(context.Background(), 10, 99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.reserveCalls) != 2 {
		t.Fatalf("expected 2 reserve calls, got %d", len(q.reserveCalls))
	}
	if q.reserveCalls[0].tankID != 101 || q.reserveCalls[1].tankID != 102 {
		t.Fatalf("unexpected reserve calls: %#v", q.reserveCalls)
	}
}

func TestWorkflowService_ApproveDeliveryOrder_InsufficientStock(t *testing.T) {
	q := &fakeTankWorkflowQuerier{
		do: db.DeliveryOrder{ID: 10, OriginFacilityID: 1, Status: db.DoStatusTPENDINGAPPROVAL},
		doItems: []db.ListDOItemsByDORow{
			{FuelTypeCode: "PERTAMAX", RequestedVolumeL: numericInt64(1000)},
		},
		tankByFuel: map[string]db.FacilityStorageTank{
			"PERTAMAX": {ID: 101, FacilityID: 1, FuelTypeCode: "PERTAMAX"},
		},
		reserveErr: pgx.ErrNoRows,
	}
	svc := NewWorkflowService(&fakeTankWorkflowStore{q: q})

	_, err := svc.ApproveDeliveryOrder(context.Background(), 10, 99)
	if err != ErrInsufficientStock {
		t.Fatalf("expected ErrInsufficientStock, got %v", err)
	}
}

func TestWorkflowService_CancelDeliveryOrder_ReleasesOnlyWhenReserved(t *testing.T) {
	q := &fakeTankWorkflowQuerier{
		do: db.DeliveryOrder{ID: 10, OriginFacilityID: 1, Status: db.DoStatusTAPPROVED},
		doItems: []db.ListDOItemsByDORow{
			{FuelTypeCode: "PERTAMAX", RequestedVolumeL: numericInt64(1000)},
		},
		tankByFuel: map[string]db.FacilityStorageTank{
			"PERTAMAX": {ID: 101, FacilityID: 1, FuelTypeCode: "PERTAMAX"},
		},
	}
	svc := NewWorkflowService(&fakeTankWorkflowStore{q: q})

	_, err := svc.CancelDeliveryOrder(context.Background(), 10)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.releaseCalls) != 1 {
		t.Fatalf("expected 1 release call, got %d", len(q.releaseCalls))
	}
	if q.releaseCalls[0].tankID != 101 {
		t.Fatalf("unexpected release calls: %#v", q.releaseCalls)
	}
}

func TestWorkflowService_RecordTripEvent_LoadingCompleted_Deducts(t *testing.T) {
	now := time.Now()
	q := &fakeTankWorkflowQuerier{
		trip: db.GetTripRow{
			ID:               1,
			DoID:             10,
			VehicleID:        100,
			OriginFacilityID: 1,
		},
		insertTripEventResp: db.TripEvent{TripID: 1, EventType: db.TripEventTypeTLOADINGCOMPLETED},
		mandatoryPhotoCheck: db.GetMandatoryPhotoCheckByTripRow{
			HasTarePhoto:              true,
			HasGrossPhoto:             true,
			HasCompartmentSealedPhoto: true,
		},
		vehicleCompartments: []db.VehicleCompartment{
			{ID: 501, VehicleID: 100, IsActive: true},
		},
		photosByTripAndEvent: []db.TripPhoto{
			{TripID: 1, EventType: db.PhotoEventTCOMPARTMENTSEALED, CompartmentID: pgtype.Int8{Int64: 501, Valid: true}},
		},
		tripLoadedByFuel: []db.ListTripLoadedVolumeByFuelRow{
			{FuelTypeCode: "PERTAMAX", TotalLoadedL: numericInt64(1500)},
		},
		tankByFuel: map[string]db.FacilityStorageTank{
			"PERTAMAX": {ID: 101, FacilityID: 1, FuelTypeCode: "PERTAMAX"},
		},
	}
	svc := NewWorkflowService(&fakeTankWorkflowStore{q: q})

	_, err := svc.RecordTripEvent(context.Background(), 1, db.InsertTripEventParams{
		EventUuid:      uuid.New(),
		EventType:      db.TripEventTypeTLOADINGCOMPLETED,
		EventTimestamp: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.updatedTripStatuses) != 1 || q.updatedTripStatuses[0] != db.TripStatusTLOADED {
		t.Fatalf("unexpected trip status updates: %#v", q.updatedTripStatuses)
	}
	if len(q.updatedDOStatuses) != 1 || q.updatedDOStatuses[0] != db.DoStatusTINPROGRESS {
		t.Fatalf("unexpected DO status updates: %#v", q.updatedDOStatuses)
	}
	if len(q.deductCalls) != 1 || q.deductCalls[0].tankID != 101 {
		t.Fatalf("unexpected deduct calls: %#v", q.deductCalls)
	}
}

func TestWorkflowService_RecordTripEvent_LoadingCompleted_BlockedByPendingWeightBridgeApproval(t *testing.T) {
	now := time.Now()
	q := &fakeTankWorkflowQuerier{
		trip: db.GetTripRow{
			ID:               1,
			DoID:             10,
			VehicleID:        100,
			OriginFacilityID: 1,
		},
		insertTripEventResp: db.TripEvent{TripID: 1, EventType: db.TripEventTypeTLOADINGCOMPLETED},
		mandatoryPhotoCheck: db.GetMandatoryPhotoCheckByTripRow{
			HasTarePhoto:              true,
			HasGrossPhoto:             true,
			HasCompartmentSealedPhoto: true,
		},
		vehicleCompartments: []db.VehicleCompartment{
			{ID: 501, VehicleID: 100, IsActive: true},
		},
		photosByTripAndEvent: []db.TripPhoto{
			{TripID: 1, EventType: db.PhotoEventTCOMPARTMENTSEALED, CompartmentID: pgtype.Int8{Int64: 501, Valid: true}},
		},
		weightBridgeReadings: []db.WeightBridgeReading{
			{ApprovalStatus: db.ApprovalStatusTPENDING},
		},
	}
	svc := NewWorkflowService(&fakeTankWorkflowStore{q: q})

	_, err := svc.RecordTripEvent(context.Background(), 1, db.InsertTripEventParams{
		EventUuid:      uuid.New(),
		EventType:      db.TripEventTypeTLOADINGCOMPLETED,
		EventTimestamp: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != ErrApprovalRequired {
		t.Fatalf("expected ErrApprovalRequired, got %v", err)
	}
	if len(q.updatedTripStatuses) != 0 {
		t.Fatalf("unexpected trip status updates: %#v", q.updatedTripStatuses)
	}
	if len(q.updatedDOStatuses) != 0 {
		t.Fatalf("unexpected DO status updates: %#v", q.updatedDOStatuses)
	}
	if len(q.deductCalls) != 0 {
		t.Fatalf("unexpected deduct calls: %#v", q.deductCalls)
	}
}

func TestWorkflowService_RecordTripEvent_LoadingCompleted_BlockedByMissingPhotos(t *testing.T) {
	now := time.Now()
	q := &fakeTankWorkflowQuerier{
		trip: db.GetTripRow{
			ID:               1,
			DoID:             10,
			VehicleID:        100,
			OriginFacilityID: 1,
		},
		insertTripEventResp: db.TripEvent{TripID: 1, EventType: db.TripEventTypeTLOADINGCOMPLETED},
		mandatoryPhotoCheck: db.GetMandatoryPhotoCheckByTripRow{
			HasTarePhoto:              true,
			HasGrossPhoto:             true,
			HasCompartmentSealedPhoto: true,
		},
		vehicleCompartments: []db.VehicleCompartment{
			{ID: 501, VehicleID: 100, IsActive: true},
		},
		photosByTripAndEvent: []db.TripPhoto{},
	}
	svc := NewWorkflowService(&fakeTankWorkflowStore{q: q})

	_, err := svc.RecordTripEvent(context.Background(), 1, db.InsertTripEventParams{
		EventUuid:      uuid.New(),
		EventType:      db.TripEventTypeTLOADINGCOMPLETED,
		EventTimestamp: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != ErrPhotoMissing {
		t.Fatalf("expected ErrPhotoMissing, got %v", err)
	}
	if len(q.updatedTripStatuses) != 0 {
		t.Fatalf("unexpected trip status updates: %#v", q.updatedTripStatuses)
	}
}

func TestWorkflowService_RecordTripEvent_DeliveryCompleted_ReturnTrip_Credits(t *testing.T) {
	now := time.Now()
	q := &fakeTankWorkflowQuerier{
		trip: db.GetTripRow{
			ID:                    2,
			DoID:                  11,
			OriginFacilityID:      1,
			DestinationType:       db.DestinationTypeTREFINERYFACILITY,
			DestinationFacilityID: pgtype.Int8{Int64: 2, Valid: true},
			ParentTripID:          pgtype.Int8{Int64: 1, Valid: true},
		},
		insertTripEventResp: db.TripEvent{TripID: 2, EventType: db.TripEventTypeTDELIVERYCOMPLETED},
		mandatoryPhotoCheck: db.GetMandatoryPhotoCheckByTripRow{
			HasBeforePhoto: true,
			HasPumpPhoto:   true,
			HasAfterPhoto:  true,
		},
		tripDeliveredByFuel: []db.ListTripDeliveredVolumeByFuelRow{
			{FuelTypeCode: "PERTAMAX", TotalDeliveredL: numericInt64(500)},
		},
		tankByFuel: map[string]db.FacilityStorageTank{
			"PERTAMAX": {ID: 201, FacilityID: 2, FuelTypeCode: "PERTAMAX"},
		},
	}
	svc := NewWorkflowService(&fakeTankWorkflowStore{q: q})

	_, err := svc.RecordTripEvent(context.Background(), 2, db.InsertTripEventParams{
		EventUuid:      uuid.New(),
		EventType:      db.TripEventTypeTDELIVERYCOMPLETED,
		EventTimestamp: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.creditCalls) != 1 || q.creditCalls[0].tankID != 201 {
		t.Fatalf("unexpected credit calls: %#v", q.creditCalls)
	}
}

func TestWorkflowService_RecordTripEvent_DeliveryCompleted_BlockedByMissingPhotos(t *testing.T) {
	now := time.Now()
	q := &fakeTankWorkflowQuerier{
		trip: db.GetTripRow{
			ID:               2,
			DoID:             11,
			OriginFacilityID: 1,
		},
		insertTripEventResp: db.TripEvent{TripID: 2, EventType: db.TripEventTypeTDELIVERYCOMPLETED},
		mandatoryPhotoCheck: db.GetMandatoryPhotoCheckByTripRow{
			HasBeforePhoto: true,
			HasPumpPhoto:   false,
			HasAfterPhoto:  true,
		},
	}
	svc := NewWorkflowService(&fakeTankWorkflowStore{q: q})

	_, err := svc.RecordTripEvent(context.Background(), 2, db.InsertTripEventParams{
		EventUuid:      uuid.New(),
		EventType:      db.TripEventTypeTDELIVERYCOMPLETED,
		EventTimestamp: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != ErrPhotoMissing {
		t.Fatalf("expected ErrPhotoMissing, got %v", err)
	}
	if len(q.updatedTripStatuses) != 0 {
		t.Fatalf("unexpected trip status updates: %#v", q.updatedTripStatuses)
	}
}

func TestWorkflowService_RecordTripEvent_DeliveryCompleted_StationTrip_UpdatesStationTanksAndMarksDelivered(t *testing.T) {
	now := time.Now()
	q := &fakeTankWorkflowQuerier{
		trip: db.GetTripRow{
			ID:                   3,
			DoID:                 12,
			OriginFacilityID:     1,
			DestinationType:      db.DestinationTypeTSTATION,
			DestinationStationID: pgtype.Int8{Int64: 50, Valid: true},
		},
		insertTripEventResp: db.TripEvent{TripID: 3, EventType: db.TripEventTypeTDELIVERYCOMPLETED},
		mandatoryPhotoCheck: db.GetMandatoryPhotoCheckByTripRow{
			HasBeforePhoto: true,
			HasPumpPhoto:   true,
			HasAfterPhoto:  true,
		},
		tripDeliveredByFuel: []db.ListTripDeliveredVolumeByFuelRow{
			{FuelTypeCode: "PERTAMAX", TotalDeliveredL: numericInt64(500)},
		},
		stationTankByFuel: map[string]db.StationTank{
			"PERTAMAX": {ID: 901, StationID: 50, FuelTypeCode: "PERTAMAX"},
		},
		effectiveSettingValue: "0.3",
		tripCompartmentDetails: []db.ListCompartmentDeliveriesByTripRow{
			{ID: 1001, TripID: 3, VariancePct: numericInt64Exp(2, -1)},
		},
	}
	svc := NewWorkflowService(&fakeTankWorkflowStore{q: q})

	_, err := svc.RecordTripEvent(context.Background(), 3, db.InsertTripEventParams{
		EventUuid:      uuid.New(),
		EventType:      db.TripEventTypeTDELIVERYCOMPLETED,
		EventTimestamp: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.stationTankUpdateCalls) != 1 || q.stationTankUpdateCalls[0].tankID != 901 {
		t.Fatalf("unexpected station tank update calls: %#v", q.stationTankUpdateCalls)
	}
	if len(q.compartmentStatusCalls) != 1 || q.compartmentStatusCalls[0].status != db.CompartmentDeliveryStatusTDELIVERED {
		t.Fatalf("unexpected compartment status calls: %#v", q.compartmentStatusCalls)
	}
	if len(q.updatedTripStatuses) != 1 || q.updatedTripStatuses[0] != db.TripStatusTDELIVERED {
		t.Fatalf("unexpected trip status updates: %#v", q.updatedTripStatuses)
	}
	if len(q.updatedDOStatuses) != 1 || q.updatedDOStatuses[0] != db.DoStatusTDELIVERED {
		t.Fatalf("unexpected DO status updates: %#v", q.updatedDOStatuses)
	}
}

func TestWorkflowService_RecordTripEvent_DeliveryCompleted_StationTrip_DisputesWhenToleranceExceeded(t *testing.T) {
	now := time.Now()
	q := &fakeTankWorkflowQuerier{
		trip: db.GetTripRow{
			ID:                   4,
			DoID:                 13,
			OriginFacilityID:     1,
			DestinationType:      db.DestinationTypeTSTATION,
			DestinationStationID: pgtype.Int8{Int64: 51, Valid: true},
		},
		insertTripEventResp: db.TripEvent{TripID: 4, EventType: db.TripEventTypeTDELIVERYCOMPLETED},
		mandatoryPhotoCheck: db.GetMandatoryPhotoCheckByTripRow{
			HasBeforePhoto: true,
			HasPumpPhoto:   true,
			HasAfterPhoto:  true,
		},
		tripDeliveredByFuel: []db.ListTripDeliveredVolumeByFuelRow{
			{FuelTypeCode: "PERTAMAX", TotalDeliveredL: numericInt64(500)},
		},
		stationTankByFuel: map[string]db.StationTank{
			"PERTAMAX": {ID: 902, StationID: 51, FuelTypeCode: "PERTAMAX"},
		},
		effectiveSettingValue: "0.3",
		tripCompartmentDetails: []db.ListCompartmentDeliveriesByTripRow{
			{ID: 1002, TripID: 4, VariancePct: numericInt64Exp(5, -1)},
		},
	}
	svc := NewWorkflowService(&fakeTankWorkflowStore{q: q})

	_, err := svc.RecordTripEvent(context.Background(), 4, db.InsertTripEventParams{
		EventUuid:      uuid.New(),
		EventType:      db.TripEventTypeTDELIVERYCOMPLETED,
		EventTimestamp: pgtype.Timestamptz{Time: now, Valid: true},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.stationTankUpdateCalls) != 1 || q.stationTankUpdateCalls[0].tankID != 902 {
		t.Fatalf("unexpected station tank update calls: %#v", q.stationTankUpdateCalls)
	}
	if len(q.compartmentStatusCalls) != 1 || q.compartmentStatusCalls[0].status != db.CompartmentDeliveryStatusTDISPUTED {
		t.Fatalf("unexpected compartment status calls: %#v", q.compartmentStatusCalls)
	}
	if len(q.updatedTripStatuses) != 1 || q.updatedTripStatuses[0] != db.TripStatusTDISPUTED {
		t.Fatalf("unexpected trip status updates: %#v", q.updatedTripStatuses)
	}
	if len(q.updatedDOStatuses) != 1 || q.updatedDOStatuses[0] != db.DoStatusTDISPUTED {
		t.Fatalf("unexpected DO status updates: %#v", q.updatedDOStatuses)
	}
}
