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

const facilityGroupChatKey = "facility_group_chat_id"

type NotificationCoordinatorStore interface {
	GetDepot(ctx context.Context, id int64) (db.GetDepotRow, error)
	GetDriver(ctx context.Context, id int64) (db.GetDriverRow, error)
	GetFacilitySetting(ctx context.Context, arg db.GetFacilitySettingParams) (db.SystemSetting, error)
	GetStation(ctx context.Context, id int64) (db.GetStationRow, error)
	GetTrip(ctx context.Context, id int64) (db.GetTripRow, error)
	GetTripVarianceSummary(ctx context.Context, tripID int64) (db.GetTripVarianceSummaryRow, error)
	GetTripWithDetails(ctx context.Context, id int64) (db.GetTripWithDetailsRow, error)
	GetVehicle(ctx context.Context, id int64) (db.GetVehicleRow, error)
	ListNotificationsByRecipient(ctx context.Context, arg db.ListNotificationsByRecipientParams) ([]db.NotificationLog, error)
	ListUsersWithCompanyRole(ctx context.Context, role db.UserRoleT) ([]db.ListUsersWithCompanyRoleRow, error)
	ListUsersWithRoleInScope(ctx context.Context, arg db.ListUsersWithRoleInScopeParams) ([]db.ListUsersWithRoleInScopeRow, error)
}

type NotificationCoordinator struct {
	store NotificationCoordinatorStore
	notif *NotificationService
}

func NewNotificationCoordinator(store NotificationCoordinatorStore, notif *NotificationService) *NotificationCoordinator {
	return &NotificationCoordinator{store: store, notif: notif}
}

func (c *NotificationCoordinator) NotifyDORaised(ctx context.Context, order db.DeliveryOrder) error {
	return c.sendFacilityGroup(ctx, order.OriginFacilityID, SendNotificationRequest{
		DOID:             int64Ptr(order.ID),
		NotificationType: db.NotificationTypeTDORAISED,
		MessageText:      fmt.Sprintf("📋 DO %s raised", order.DoNumber),
	})
}

func (c *NotificationCoordinator) NotifyDOApproved(ctx context.Context, order db.DeliveryOrder) error {
	return c.sendFacilityGroup(ctx, order.OriginFacilityID, SendNotificationRequest{
		DOID:             int64Ptr(order.ID),
		NotificationType: db.NotificationTypeTDOAPPROVED,
		MessageText:      fmt.Sprintf("✅ DO %s approved", order.DoNumber),
	})
}

func (c *NotificationCoordinator) NotifyTripAssigned(ctx context.Context, order db.DeliveryOrder) error {
	if !order.AssignedDriverID.Valid {
		return nil
	}

	driver, err := c.store.GetDriver(ctx, order.AssignedDriverID.Int64)
	if err != nil {
		return err
	}
	if !driver.TelegramUserID.Valid {
		return nil
	}

	destination := "destination"
	if order.DestinationStationID.Valid {
		station, err := c.store.GetStation(ctx, order.DestinationStationID.Int64)
		if err == nil {
			destination = station.Name
		}
	}

	_, err = c.notif.Send(ctx, SendNotificationRequest{
		DOID:                int64Ptr(order.ID),
		RecipientTelegramID: driver.TelegramUserID.Int64,
		RecipientUserID:     int64Ptr(driver.UserID),
		NotificationType:    db.NotificationTypeTTRIPASSIGNED,
		MessageText:         fmt.Sprintf("🚛 Trip assigned: %s → %s", order.DoNumber, destination),
	})
	return ignoreNotificationSendError(err)
}

func (c *NotificationCoordinator) NotifyTripEvent(ctx context.Context, event db.TripEvent) error {
	switch event.EventType {
	case db.TripEventTypeTLOADINGCOMPLETED:
		return c.notifyLoadingComplete(ctx, event.TripID)
	case db.TripEventTypeTDEPARTEDFACILITY:
		return c.notifyTripDeparted(ctx, event.TripID)
	case db.TripEventTypeTDELIVERYCOMPLETED:
		return c.notifyDeliveryCompleted(ctx, event.TripID)
	default:
		return nil
	}
}

func (c *NotificationCoordinator) NotifyManualWeightBridgePending(ctx context.Context, reading db.WeightBridgeReading) error {
	if reading.Method != db.MeasurementMethodTMANUALAPPROVED || reading.ApprovalStatus != db.ApprovalStatusTPENDING {
		return nil
	}

	vehicle, err := c.store.GetVehicle(ctx, reading.VehicleID)
	if err != nil {
		return err
	}
	if !vehicle.CurrentDepotID.Valid {
		return nil
	}

	depot, err := c.store.GetDepot(ctx, vehicle.CurrentDepotID.Int64)
	if err != nil {
		return err
	}

	return c.sendUsersInScope(ctx, db.UserRoleTFACILITYMANAGER, db.RoleScopeTFACILITY, depot.PrimaryFacilityID, SendNotificationRequest{
		TripID:           int64PtrFromPG(reading.TripID),
		NotificationType: db.NotificationTypeTMANUALMEASUREMENTPENDING,
		MessageText:      fmt.Sprintf("⚖️ Manual weight pending: Trip %s", displayPGInt8(reading.TripID)),
	}, false)
}

func (c *NotificationCoordinator) NotifyRouteDeviationEscalated(ctx context.Context, row db.ListUnnotifiedDeviationsAboveThresholdRow) error {
	trip, err := c.store.GetTrip(ctx, row.TripID)
	if err != nil {
		return err
	}
	vehicle, err := c.store.GetVehicle(ctx, row.VehicleID)
	if err != nil {
		return err
	}

	minutes := 0
	if row.DurationSeconds.Valid {
		minutes = int(row.DurationSeconds.Int32 / 60)
	}
	if minutes == 0 {
		minutes = 1
	}

	return c.sendUsersInScope(ctx, db.UserRoleTFACILITYMANAGER, db.RoleScopeTFACILITY, trip.OriginFacilityID, SendNotificationRequest{
		TripID:           int64Ptr(row.TripID),
		NotificationType: db.NotificationTypeTROUTEDEVIATIONESCALATE,
		MessageText:      fmt.Sprintf("📍 Truck %s off-route for %d min", vehicle.PlateNumber, minutes),
	}, false)
}

func (c *NotificationCoordinator) NotifyDriverLicenseExpiring(ctx context.Context, row db.ListDriversWithExpiringLicenseRow) error {
	if !row.TelegramUserID.Valid {
		return nil
	}

	message := fmt.Sprintf("⏰ SIM B2 expires %s", row.SimB2Expiry.Time.Format("2006-01-02"))
	skip, err := c.alreadySentToUser(ctx, row.UserID, db.NotificationTypeTDRIVERLICENSEEXPIRING, message)
	if err != nil || skip {
		return err
	}

	_, err = c.notif.Send(ctx, SendNotificationRequest{
		RecipientTelegramID: row.TelegramUserID.Int64,
		RecipientUserID:     int64Ptr(row.UserID),
		NotificationType:    db.NotificationTypeTDRIVERLICENSEEXPIRING,
		MessageText:         message,
	})
	return ignoreNotificationSendError(err)
}

func (c *NotificationCoordinator) NotifyVehicleKeurExpiring(ctx context.Context, row db.ListVehiclesWithExpiringKeurRow) error {
	if !row.CurrentDepotID.Valid {
		return nil
	}

	message := fmt.Sprintf("⏰ Truck %s keur expires %s", row.PlateNumber, row.KeurExpiry.Time.Format("2006-01-02"))
	return c.sendUsersInScope(ctx, db.UserRoleTDEPOTSTAFF, db.RoleScopeTDEPOT, row.CurrentDepotID.Int64, SendNotificationRequest{
		NotificationType: db.NotificationTypeTVEHICLEKEUREXPIRING,
		MessageText:      message,
	}, true)
}

func (c *NotificationCoordinator) notifyLoadingComplete(ctx context.Context, tripID int64) error {
	trip, err := c.store.GetTripWithDetails(ctx, tripID)
	if err != nil {
		return err
	}
	if !trip.DestinationStationID.Valid {
		return nil
	}

	return c.sendUsersInScope(ctx, db.UserRoleTSTATIONMANAGER, db.RoleScopeTSTATION, trip.DestinationStationID.Int64, SendNotificationRequest{
		TripID:           int64Ptr(tripID),
		NotificationType: db.NotificationTypeTLOADINGCOMPLETE,
		MessageText:      fmt.Sprintf("🚛 Truck %s loaded, en route", trip.PlateNumber),
	}, false)
}

func (c *NotificationCoordinator) notifyTripDeparted(ctx context.Context, tripID int64) error {
	trip, err := c.store.GetTripWithDetails(ctx, tripID)
	if err != nil {
		return err
	}
	destination := trip.DestinationStationName.String
	if destination == "" {
		destination = "destination"
	}
	return c.sendFacilityGroup(ctx, trip.OriginFacilityID, SendNotificationRequest{
		TripID:           int64Ptr(tripID),
		DOID:             int64Ptr(trip.DoID),
		NotificationType: db.NotificationTypeTTRIPDEPARTED,
		MessageText:      fmt.Sprintf("🚛 Truck %s departed → %s", trip.PlateNumber, destination),
	})
}

func (c *NotificationCoordinator) notifyDeliveryCompleted(ctx context.Context, tripID int64) error {
	trip, err := c.store.GetTripWithDetails(ctx, tripID)
	if err != nil {
		return err
	}

	summary, err := c.store.GetTripVarianceSummary(ctx, tripID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}

	message := fmt.Sprintf(
		"✅ %dL delivered to %s. Variance: %s%%",
		summary.TotalDeliveredL,
		trip.DestinationStationName.String,
		summary.OverallVariancePct.StringFixed(2),
	)

	if err := c.sendFacilityGroup(ctx, trip.OriginFacilityID, SendNotificationRequest{
		TripID:           int64Ptr(tripID),
		DOID:             int64Ptr(trip.DoID),
		NotificationType: db.NotificationTypeTDELIVERYCOMPLETE,
		MessageText:      message,
	}); err != nil {
		return err
	}

	if trip.DestinationStationID.Valid {
		if err := c.sendUsersInScope(ctx, db.UserRoleTSTATIONMANAGER, db.RoleScopeTSTATION, trip.DestinationStationID.Int64, SendNotificationRequest{
			TripID:           int64Ptr(tripID),
			DOID:             int64Ptr(trip.DoID),
			NotificationType: db.NotificationTypeTDELIVERYCOMPLETE,
			MessageText:      message,
		}, false); err != nil {
			return err
		}
	}

	if summary.HasDisputed {
		varianceMessage := fmt.Sprintf("⚠️ Variance flagged: Trip %d variance %s%%", tripID, summary.OverallVariancePct.StringFixed(2))
		if err := c.sendFacilityGroup(ctx, trip.OriginFacilityID, SendNotificationRequest{
			TripID:           int64Ptr(tripID),
			DOID:             int64Ptr(trip.DoID),
			NotificationType: db.NotificationTypeTVARIANCEFLAGGED,
			MessageText:      varianceMessage,
		}); err != nil {
			return err
		}
		if err := c.sendUsersInCompanyRole(ctx, db.UserRoleTREFINERYADMIN, SendNotificationRequest{
			TripID:           int64Ptr(tripID),
			DOID:             int64Ptr(trip.DoID),
			NotificationType: db.NotificationTypeTVARIANCEFLAGGED,
			MessageText:      varianceMessage,
		}, false); err != nil {
			return err
		}
	}

	return nil
}

func (c *NotificationCoordinator) sendFacilityGroup(ctx context.Context, facilityID int64, req SendNotificationRequest) error {
	if c == nil || c.notif == nil {
		return nil
	}

	setting, err := c.store.GetFacilitySetting(ctx, db.GetFacilitySettingParams{
		Key:        facilityGroupChatKey,
		FacilityID: pgtype.Int8{Int64: facilityID, Valid: true},
	})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return err
	}

	chatID, err := strconv.ParseInt(setting.Value, 10, 64)
	if err != nil {
		return nil
	}
	req.RecipientTelegramID = chatID
	_, err = c.notif.Send(ctx, req)
	return ignoreNotificationSendError(err)
}

func (c *NotificationCoordinator) sendUsersInScope(ctx context.Context, role db.UserRoleT, scopeType db.RoleScopeT, scopeID int64, req SendNotificationRequest, dedupe bool) error {
	if c == nil || c.notif == nil {
		return nil
	}

	users, err := c.store.ListUsersWithRoleInScope(ctx, db.ListUsersWithRoleInScopeParams{
		Role:      role,
		ScopeType: scopeType,
		ScopeID:   pgtype.Int8{Int64: scopeID, Valid: true},
	})
	if err != nil {
		return err
	}

	for _, user := range users {
		if !user.TelegramUserID.Valid {
			continue
		}
		if dedupe {
			skip, err := c.alreadySentToUser(ctx, user.ID, req.NotificationType, req.MessageText)
			if err != nil {
				return err
			}
			if skip {
				continue
			}
		}

		sendReq := req
		sendReq.RecipientTelegramID = user.TelegramUserID.Int64
		sendReq.RecipientUserID = int64Ptr(user.ID)
		if _, err := c.notif.Send(ctx, sendReq); err != nil && !errors.Is(err, ErrTelegramNotConfigured) {
			return err
		}
	}

	return nil
}

func (c *NotificationCoordinator) sendUsersInCompanyRole(ctx context.Context, role db.UserRoleT, req SendNotificationRequest, dedupe bool) error {
	if c == nil || c.notif == nil {
		return nil
	}

	users, err := c.store.ListUsersWithCompanyRole(ctx, role)
	if err != nil {
		return err
	}

	for _, user := range users {
		if !user.TelegramUserID.Valid {
			continue
		}
		if dedupe {
			skip, err := c.alreadySentToUser(ctx, user.ID, req.NotificationType, req.MessageText)
			if err != nil {
				return err
			}
			if skip {
				continue
			}
		}

		sendReq := req
		sendReq.RecipientTelegramID = user.TelegramUserID.Int64
		sendReq.RecipientUserID = int64Ptr(user.ID)
		if _, err := c.notif.Send(ctx, sendReq); err != nil && !errors.Is(err, ErrTelegramNotConfigured) {
			return err
		}
	}

	return nil
}

func (c *NotificationCoordinator) alreadySentToUser(ctx context.Context, userID int64, typ db.NotificationTypeT, message string) (bool, error) {
	rows, err := c.store.ListNotificationsByRecipient(ctx, db.ListNotificationsByRecipientParams{
		RecipientUserID: pgtype.Int8{Int64: userID, Valid: true},
		Limit:           20,
	})
	if err != nil {
		return false, err
	}
	for _, row := range rows {
		if row.NotificationType == typ && row.MessageText == message {
			return true, nil
		}
	}
	return false, nil
}

func ignoreNotificationSendError(err error) error {
	if err == nil || errors.Is(err, ErrTelegramNotConfigured) {
		return nil
	}
	return err
}

func displayPGInt8(v pgtype.Int8) string {
	if !v.Valid {
		return "N/A"
	}
	return strconv.FormatInt(v.Int64, 10)
}

func int64Ptr(v int64) *int64 {
	return &v
}
