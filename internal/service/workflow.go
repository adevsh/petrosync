package service

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/shopspring/decimal"

	"github.com/adevsh/petrosync/internal/db"
)

var (
	ErrInsufficientStock = errors.New("insufficient stock")
	ErrApprovalRequired  = errors.New("approval required")
	ErrPhotoMissing      = errors.New("photo missing")
)

type WorkflowService struct {
	store db.TankWorkflowStore
}

func NewWorkflowService(store db.TankWorkflowStore) *WorkflowService {
	return &WorkflowService{store: store}
}

func (s *WorkflowService) ApproveDeliveryOrder(ctx context.Context, doID, userID int64) (db.DeliveryOrder, error) {
	var out db.DeliveryOrder
	err := s.store.ExecTx(ctx, func(q db.TankWorkflowQuerier) error {
		do, err := q.GetDeliveryOrder(ctx, doID)
		if err != nil {
			return err
		}

		approved, err := q.ApproveDeliveryOrder(ctx, db.ApproveDeliveryOrderParams{
			ID:         doID,
			ApprovedBy: pgtype.Int8{Int64: userID, Valid: true},
		})
		if err != nil {
			return err
		}

		items, err := q.ListDOItemsByDO(ctx, doID)
		if err != nil {
			return err
		}

		for _, item := range items {
			tank, err := q.GetStorageTankByFacilityAndFuel(ctx, db.GetStorageTankByFacilityAndFuelParams{
				FacilityID:   do.OriginFacilityID,
				FuelTypeCode: item.FuelTypeCode,
			})
			if err != nil {
				return err
			}

			_, err = q.ReserveStorageTankVolume(ctx, db.ReserveStorageTankVolumeParams{
				ID:              tank.ID,
				ReservedVolumeL: item.RequestedVolumeL,
			})
			if errors.Is(err, pgx.ErrNoRows) {
				return ErrInsufficientStock
			}
			if err != nil {
				return err
			}
		}

		out = approved
		return nil
	})
	return out, err
}

func (s *WorkflowService) CancelDeliveryOrder(ctx context.Context, doID int64) (db.DeliveryOrder, error) {
	var out db.DeliveryOrder
	err := s.store.ExecTx(ctx, func(q db.TankWorkflowQuerier) error {
		do, err := q.GetDeliveryOrder(ctx, doID)
		if err != nil {
			return err
		}

		cancelled, err := q.CancelDeliveryOrder(ctx, doID)
		if err != nil {
			return err
		}

		if do.Status == db.DoStatusTAPPROVED || do.Status == db.DoStatusTASSIGNED {
			items, err := q.ListDOItemsByDO(ctx, doID)
			if err != nil {
				return err
			}

			for _, item := range items {
				tank, err := q.GetStorageTankByFacilityAndFuel(ctx, db.GetStorageTankByFacilityAndFuelParams{
					FacilityID:   do.OriginFacilityID,
					FuelTypeCode: item.FuelTypeCode,
				})
				if err != nil {
					return err
				}

				if _, err := q.ReleaseStorageTankReservation(ctx, db.ReleaseStorageTankReservationParams{
					ID:              tank.ID,
					ReservedVolumeL: item.RequestedVolumeL,
				}); err != nil {
					return err
				}
			}
		}

		out = cancelled
		return nil
	})
	return out, err
}

func (s *WorkflowService) RecordTripEvent(ctx context.Context, tripID int64, req db.InsertTripEventParams) (db.TripEvent, error) {
	req.TripID = tripID

	var out db.TripEvent
	err := s.store.ExecTx(ctx, func(q db.TankWorkflowQuerier) error {
		trip, err := q.GetTrip(ctx, tripID)
		if err != nil {
			return err
		}

		event, err := q.InsertTripEvent(ctx, req)
		if err != nil {
			return err
		}

		if event.EventType == db.TripEventTypeTLOADINGCOMPLETED {
			check, err := q.GetMandatoryPhotoCheckByTrip(ctx, tripID)
			if err != nil {
				return err
			}
			if !check.HasTarePhoto || !check.HasGrossPhoto || !check.HasCompartmentSealedPhoto {
				return ErrPhotoMissing
			}

			compartments, err := q.ListCompartmentsByVehicle(ctx, trip.VehicleID)
			if err != nil {
				return err
			}
			sealedPhotos, err := q.ListPhotosByTripAndEvent(ctx, db.ListPhotosByTripAndEventParams{
				TripID: tripID, EventType: db.PhotoEventTCOMPARTMENTSEALED,
			})
			if err != nil {
				return err
			}
			hasByCompartment := make(map[int64]bool, len(sealedPhotos))
			for _, p := range sealedPhotos {
				if p.CompartmentID.Valid {
					hasByCompartment[p.CompartmentID.Int64] = true
				}
			}
			for _, c := range compartments {
				if !hasByCompartment[c.ID] {
					return ErrPhotoMissing
				}
			}

			readings, err := q.ListWeightBridgeReadingsByTrip(ctx, pgtype.Int8{Int64: tripID, Valid: true})
			if err != nil {
				return err
			}
			for _, r := range readings {
				if r.ApprovalStatus == db.ApprovalStatusTPENDING || r.ApprovalStatus == db.ApprovalStatusTESCALATED {
					return ErrApprovalRequired
				}
			}

			if _, err := q.UpdateTripStatus(ctx, db.UpdateTripStatusParams{ID: tripID, Status: db.TripStatusTLOADED}); err != nil {
				return err
			}
			if _, err := q.UpdateDOStatus(ctx, db.UpdateDOStatusParams{ID: trip.DoID, Status: db.DoStatusTINPROGRESS}); err != nil {
				return err
			}

			loaded, err := q.ListTripLoadedVolumeByFuel(ctx, tripID)
			if err != nil {
				return err
			}

			for _, row := range loaded {
				if db.NumericIsZero(row.TotalLoadedL) {
					continue
				}

				tank, err := q.GetStorageTankByFacilityAndFuel(ctx, db.GetStorageTankByFacilityAndFuelParams{
					FacilityID:   trip.OriginFacilityID,
					FuelTypeCode: row.FuelTypeCode,
				})
				if err != nil {
					return err
				}

				if _, err := q.DeductStorageTankVolume(ctx, db.DeductStorageTankVolumeParams{
					ID:             tank.ID,
					CurrentVolumeL: row.TotalLoadedL,
				}); err != nil {
					return err
				}
			}
		}

		if event.EventType == db.TripEventTypeTDELIVERYCOMPLETED {
			check, err := q.GetMandatoryPhotoCheckByTrip(ctx, tripID)
			if err != nil {
				return err
			}
			if !check.HasBeforePhoto || !check.HasPumpPhoto || !check.HasAfterPhoto {
				return ErrPhotoMissing
			}

			if trip.DestinationType == db.DestinationTypeTSTATION && trip.DestinationStationID.Valid {
				delivered, err := q.ListTripDeliveredVolumeByFuel(ctx, tripID)
				if err != nil {
					return err
				}

				stationID := trip.DestinationStationID.Int64
				for _, row := range delivered {
					if db.NumericIsZero(row.TotalDeliveredL) {
						continue
					}

					tank, err := q.GetStationTankByFuel(ctx, db.GetStationTankByFuelParams{
						StationID: stationID, FuelTypeCode: row.FuelTypeCode,
					})
					if err != nil {
						return err
					}

					if _, err := q.UpdateStationTankVolumeAfterDelivery(ctx, db.UpdateStationTankVolumeAfterDeliveryParams{
						ID: tank.ID, CurrentVolumeL: row.TotalDeliveredL,
					}); err != nil {
						return err
					}
				}
			}

			hasDispute, err := applyDeliveryVarianceAndDisputes(ctx, q, tripID, trip.OriginFacilityID)
			if err != nil {
				return err
			}

			tripStatus := db.TripStatusTDELIVERED
			doStatus := db.DoStatusTDELIVERED
			if hasDispute {
				tripStatus = db.TripStatusTDISPUTED
				doStatus = db.DoStatusTDISPUTED
			}

			if _, err := q.UpdateTripStatus(ctx, db.UpdateTripStatusParams{ID: tripID, Status: tripStatus}); err != nil {
				return err
			}
			if _, err := q.UpdateDOStatus(ctx, db.UpdateDOStatusParams{ID: trip.DoID, Status: doStatus}); err != nil {
				return err
			}

			if trip.ParentTripID.Valid && trip.DestinationType == db.DestinationTypeTREFINERYFACILITY && trip.DestinationFacilityID.Valid {
				delivered, err := q.ListTripDeliveredVolumeByFuel(ctx, tripID)
				if err != nil {
					return err
				}

				destFacilityID := trip.DestinationFacilityID.Int64
				for _, row := range delivered {
					if db.NumericIsZero(row.TotalDeliveredL) {
						continue
					}

					tank, err := q.GetStorageTankByFacilityAndFuel(ctx, db.GetStorageTankByFacilityAndFuelParams{
						FacilityID:   destFacilityID,
						FuelTypeCode: row.FuelTypeCode,
					})
					if err != nil {
						return err
					}

					if _, err := q.CreditStorageTankVolume(ctx, db.CreditStorageTankVolumeParams{
						ID:             tank.ID,
						CurrentVolumeL: row.TotalDeliveredL,
					}); err != nil {
						return err
					}
				}
			}
		}

		out = event
		return nil
	})

	return out, err
}

func applyDeliveryVarianceAndDisputes(ctx context.Context, q db.TankWorkflowQuerier, tripID, facilityID int64) (bool, error) {
	toleranceStr, err := q.GetEffectiveSetting(ctx, db.GetEffectiveSettingParams{
		Key:        "variance_tolerance_pct",
		FacilityID: pgtype.Int8{Int64: facilityID, Valid: true},
	})
	if err != nil {
		return false, err
	}
	tolerance, err := decimal.NewFromString(toleranceStr)
	if err != nil {
		return false, err
	}

	deliveries, err := q.ListCompartmentDeliveriesByTrip(ctx, tripID)
	if err != nil {
		return false, err
	}

	hasDispute := false
	for _, d := range deliveries {
		variancePct, ok := numericToDecimal(d.VariancePct)
		if !ok {
			continue
		}

		status := db.CompartmentDeliveryStatusTDELIVERED
		if variancePct.Abs().GreaterThan(tolerance) {
			status = db.CompartmentDeliveryStatusTDISPUTED
			hasDispute = true
		}

		if _, err := q.UpdateCompartmentDeliveryStatus(ctx, db.UpdateCompartmentDeliveryStatusParams{
			ID: d.ID, DeliveryStatus: status,
		}); err != nil {
			return false, err
		}
	}

	return hasDispute, nil
}

func numericToDecimal(n pgtype.Numeric) (decimal.Decimal, bool) {
	if !n.Valid || n.Int == nil {
		return decimal.Decimal{}, false
	}
	return decimal.NewFromBigInt(n.Int, n.Exp), true
}
