package db

import (
	"context"

	"github.com/jackc/pgx/v5/pgtype"
)

type TankWorkflowStore interface {
	ExecTx(ctx context.Context, fn func(q TankWorkflowQuerier) error) error
}

type TankWorkflowQuerier interface {
	ApproveDeliveryOrder(ctx context.Context, arg ApproveDeliveryOrderParams) (DeliveryOrder, error)
	CancelDeliveryOrder(ctx context.Context, id int64) (DeliveryOrder, error)
	CreditStorageTankVolume(ctx context.Context, arg CreditStorageTankVolumeParams) (FacilityStorageTank, error)
	DeductStorageTankVolume(ctx context.Context, arg DeductStorageTankVolumeParams) (FacilityStorageTank, error)
	GetEffectiveSetting(ctx context.Context, arg GetEffectiveSettingParams) (string, error)
	GetDeliveryOrder(ctx context.Context, id int64) (DeliveryOrder, error)
	GetMandatoryPhotoCheckByTrip(ctx context.Context, tripID int64) (GetMandatoryPhotoCheckByTripRow, error)
	GetStorageTankByFacilityAndFuel(ctx context.Context, arg GetStorageTankByFacilityAndFuelParams) (FacilityStorageTank, error)
	GetStationTankByFuel(ctx context.Context, arg GetStationTankByFuelParams) (StationTank, error)
	GetTrip(ctx context.Context, id int64) (GetTripRow, error)
	InsertTripEvent(ctx context.Context, arg InsertTripEventParams) (TripEvent, error)
	ListDOItemsByDO(ctx context.Context, doID int64) ([]ListDOItemsByDORow, error)
	ListCompartmentsByVehicle(ctx context.Context, vehicleID int64) ([]VehicleCompartment, error)
	ListCompartmentDeliveriesByTrip(ctx context.Context, tripID int64) ([]ListCompartmentDeliveriesByTripRow, error)
	ListPhotosByTripAndEvent(ctx context.Context, arg ListPhotosByTripAndEventParams) ([]TripPhoto, error)
	ListTripDeliveredVolumeByFuel(ctx context.Context, tripID int64) ([]ListTripDeliveredVolumeByFuelRow, error)
	ListTripLoadedVolumeByFuel(ctx context.Context, tripID int64) ([]ListTripLoadedVolumeByFuelRow, error)
	ListWeightBridgeReadingsByTrip(ctx context.Context, tripID pgtype.Int8) ([]WeightBridgeReading, error)
	ReleaseStorageTankReservation(ctx context.Context, arg ReleaseStorageTankReservationParams) (FacilityStorageTank, error)
	ReserveStorageTankVolume(ctx context.Context, arg ReserveStorageTankVolumeParams) (FacilityStorageTank, error)
	UpdateCompartmentDeliveryStatus(ctx context.Context, arg UpdateCompartmentDeliveryStatusParams) (TripCompartmentDelivery, error)
	UpdateDOStatus(ctx context.Context, arg UpdateDOStatusParams) (DeliveryOrder, error)
	UpdateStationTankVolumeAfterDelivery(ctx context.Context, arg UpdateStationTankVolumeAfterDeliveryParams) (StationTank, error)
	UpdateTripStatus(ctx context.Context, arg UpdateTripStatusParams) (Trip, error)
}

func NumericIsZero(n pgtype.Numeric) bool {
	if !n.Valid {
		return true
	}
	return n.Int == nil || n.Int.Sign() == 0
}
