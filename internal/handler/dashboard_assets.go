package handler

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/model"
)

type dashboardStationRow struct {
	ID               int64
	Code             string
	Name             string
	Region           string
	Facility         string
	Contact          string
	InventorySummary string
	NeedsAttention   bool
	DetailURL        string
}

type dashboardStationDetailView struct {
	ID            int64
	Code          string
	Name          string
	LicenseNumber string
	Region        string
	Facility      string
	Address       string
	ContactName   string
	ContactPhone  string
}

type dashboardStationTankView struct {
	TankCode         string
	FuelName         string
	FuelCategory     string
	CurrentVolume    string
	Capacity         string
	ReorderThreshold string
	FillPct          string
	LastDip          string
	NeedsReorder     bool
}

type dashboardStationDeliveryView struct {
	TripID         int64
	Status         string
	StatusTone     string
	Vehicle        string
	Driver         string
	OriginFacility string
	Timestamp      string
	DetailURL      string
}

type dashboardFleetVehicleRow struct {
	ID                 int64
	PlateNumber        string
	Model              string
	Status             string
	StatusTone         string
	Capacity           string
	Depot              string
	NextInspection     string
	OpenMaintenance    string
	HasOpenMaintenance bool
	DetailURL          string
}

type dashboardFleetMaintenanceView struct {
	Vehicle         string
	MaintenanceType string
	Description     string
	StartedAt       string
	EstimatedReturn string
	Depot           string
}

type dashboardFleetAttentionView struct {
	Vehicle   string
	Notice    string
	DueDate   string
	Depot     string
	DetailURL string
}

type dashboardVehicleDetailView struct {
	ID              int64
	PlateNumber     string
	Status          string
	StatusTone      string
	Model           string
	ManufactureYear string
	Capacity        string
	TareWeight      string
	Depot           string
	KeurNumber      string
	KeurExpiry      string
	NextInspection  string
	LastAssignedAt  string
	Notes           string
}

type dashboardVehicleTripView struct {
	TripID      int64
	DoID        int64
	Status      string
	StatusTone  string
	Destination string
	DepartedAt  string
	CompletedAt string
	DetailURL   string
}

type dashboardVehicleMaintenanceView struct {
	MaintenanceType string
	Description     string
	StartedAt       string
	EstimatedReturn string
	CompletedAt     string
	Notes           string
	IsOpen          bool
}

type dashboardVehicleCompartmentView struct {
	Compartment string
	FuelType    string
	Capacity    string
}

type dashboardFleetVehicleSummary struct {
	ID                int64
	PlateNumber       string
	Model             pgtype.Text
	Status            db.VehicleStatusT
	TotalCapacityL    pgtype.Numeric
	CurrentDepotID    pgtype.Int8
	NextInspectionDue pgtype.Date
}

func (h *DashboardHandler) Stations(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	if !canViewStationPages(session.RoleGrants) {
		c.String(http.StatusForbidden, "forbidden")
		return
	}

	data, status, err := h.buildStationsPageData(c, session)
	if err != nil {
		c.String(status, err.Error())
		return
	}
	h.render(c, http.StatusOK, "stations.html", data)
}

func (h *DashboardHandler) VehicleDetail(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	if !canViewFleet(session.RoleGrants) {
		c.String(http.StatusForbidden, "forbidden")
		return
	}

	data, status, err := h.buildVehicleDetailPageData(c, session)
	if err != nil {
		c.String(status, err.Error())
		return
	}
	h.render(c, http.StatusOK, "vehicle.html", data)
}

func (h *DashboardHandler) buildStationsPageData(c *gin.Context, session *model.SessionData) (dashboardPageData, int, error) {
	if h.queries == nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}

	ctx := c.Request.Context()
	cache := newDashboardLookupCache()
	lowTankRows, err := h.queries.ListStationTanksBelowReorderThreshold(ctx)
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}
	lowTankCounts := make(map[int64]int)
	for _, row := range lowTankRows {
		lowTankCounts[row.StationID]++
	}

	rows, description, scopeNotice, err := h.buildStationRowsForSession(ctx, session, cache, lowTankCounts)
	if err != nil {
		return dashboardPageData{}, http.StatusForbidden, err
	}

	attentionCount := 0
	for _, row := range rows {
		if row.NeedsAttention {
			attentionCount++
		}
	}

	return h.pageData(c, session, dashboardPageData{
		Title:       "Stations",
		Heading:     "Stations",
		Description: description,
		ScopeNotice: scopeNotice,
		Breadcrumbs: []DashboardBreadcrumb{{Label: "Stations", URL: "/stations"}},
		Metrics: []dashboardMetric{
			{Label: "Stations", Value: strconv.Itoa(len(rows)), Help: "Active stations visible in this dashboard scope."},
			{Label: "Tank Alerts", Value: strconv.Itoa(attentionCount), Help: "Stations with at least one tank at or below reorder threshold."},
		},
		Stations: rows,
	}), http.StatusOK, nil
}

func (h *DashboardHandler) buildStationRowsForSession(ctx context.Context, session *model.SessionData, cache *dashboardLookupCache, lowTankCounts map[int64]int) ([]dashboardStationRow, string, string, error) {
	best := dashboardBestRole(session.RoleGrants)
	switch {
	case best.Role == "SYSTEM_ADMIN":
		rows, err := h.queries.ListAllActiveStations(ctx)
		if err != nil {
			return nil, "", "", err
		}
		return h.buildStationRowsFromAllActive(ctx, rows, cache, lowTankCounts), "Station list with low-stock context and quick access to tank and delivery details.", "Company-wide station visibility.", nil
	case best.Role == "REFINERY_ADMIN" && best.ScopeType == "REFINERY" && best.ScopeID != nil:
		rows, err := h.queries.ListAllActiveStationsByRefineryScope(ctx, *best.ScopeID)
		if err != nil {
			return nil, "", "", err
		}
		return h.buildStationRowsFromRefineryScope(ctx, rows, cache, lowTankCounts), "Stations in the refinery scope with inventory alerts and delivery context.", "Filtered to the refinery-linked stations for this user.", nil
	case (best.Role == "FACILITY_MANAGER" || best.Role == "FACILITY_OPERATOR") && best.ScopeType == "FACILITY" && best.ScopeID != nil:
		rows, err := h.queries.ListStationsServedByFacility(ctx, *best.ScopeID)
		if err != nil {
			return nil, "", "", err
		}
		facilityLabel := h.resolveFacilityLabel(ctx, cache, *best.ScopeID)
		return h.buildStationRowsFromFacilityScope(ctx, rows, cache, lowTankCounts), "Stations supplied by the current facility, with low-stock signals pulled from station tanks.", "Filtered to stations served by " + facilityLabel + ".", nil
	case best.Role == "STATION_MANAGER" && best.ScopeType == "STATION" && best.ScopeID != nil:
		rows, err := h.queries.ListAllActiveStationsByStationScope(ctx, *best.ScopeID)
		if err != nil {
			return nil, "", "", err
		}
		return h.buildStationRowsFromStationScope(ctx, rows, cache, lowTankCounts), "Single-station operational view with tank and delivery context.", "Filtered to the current station scope.", nil
	default:
		return nil, "", "", errors.New("forbidden")
	}
}

func (h *DashboardHandler) buildStationRowsFromAllActive(ctx context.Context, rows []db.ListAllActiveStationsRow, cache *dashboardLookupCache, lowTankCounts map[int64]int) []dashboardStationRow {
	out := make([]dashboardStationRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, h.buildStationRow(ctx, cache, row.ID, row.Code, row.Name, row.RegionCode, row.PrimaryFacilityID, row.ContactName, lowTankCounts[row.ID]))
	}
	return out
}

func (h *DashboardHandler) buildStationRowsFromRefineryScope(ctx context.Context, rows []db.ListAllActiveStationsByRefineryScopeRow, cache *dashboardLookupCache, lowTankCounts map[int64]int) []dashboardStationRow {
	out := make([]dashboardStationRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, h.buildStationRow(ctx, cache, row.ID, row.Code, row.Name, row.RegionCode, row.PrimaryFacilityID, row.ContactName, lowTankCounts[row.ID]))
	}
	return out
}

func (h *DashboardHandler) buildStationRowsFromFacilityScope(ctx context.Context, rows []db.ListStationsServedByFacilityRow, cache *dashboardLookupCache, lowTankCounts map[int64]int) []dashboardStationRow {
	out := make([]dashboardStationRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, h.buildStationRow(ctx, cache, row.ID, row.Code, row.Name, row.RegionCode, row.PrimaryFacilityID, row.ContactName, lowTankCounts[row.ID]))
	}
	return out
}

func (h *DashboardHandler) buildStationRowsFromStationScope(ctx context.Context, rows []db.ListAllActiveStationsByStationScopeRow, cache *dashboardLookupCache, lowTankCounts map[int64]int) []dashboardStationRow {
	out := make([]dashboardStationRow, 0, len(rows))
	for _, row := range rows {
		out = append(out, h.buildStationRow(ctx, cache, row.ID, row.Code, row.Name, row.RegionCode, row.PrimaryFacilityID, row.ContactName, lowTankCounts[row.ID]))
	}
	return out
}

func (h *DashboardHandler) buildStationRow(ctx context.Context, cache *dashboardLookupCache, id int64, code, name, region string, facilityID int64, contactName pgtype.Text, lowTankCount int) dashboardStationRow {
	inventorySummary := "No reorder alerts"
	if lowTankCount == 1 {
		inventorySummary = "1 tank below reorder level"
	} else if lowTankCount > 1 {
		inventorySummary = fmt.Sprintf("%d tanks below reorder level", lowTankCount)
	}
	return dashboardStationRow{
		ID:               id,
		Code:             code,
		Name:             name,
		Region:           region,
		Facility:         h.resolveFacilityLabel(ctx, cache, facilityID),
		Contact:          formatText(contactName, "No contact assigned"),
		InventorySummary: inventorySummary,
		NeedsAttention:   lowTankCount > 0,
		DetailURL:        fmt.Sprintf("/stations/%d", id),
	}
}

func (h *DashboardHandler) buildStationDetailPageData(c *gin.Context, session *model.SessionData) (dashboardPageData, int, error) {
	if h.queries == nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}

	stationID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return dashboardPageData{}, http.StatusBadRequest, errors.New("invalid station id")
	}

	ctx := c.Request.Context()
	station, err := h.queries.GetStation(ctx, stationID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dashboardPageData{}, http.StatusNotFound, errors.New("station not found")
		}
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}
	if !h.canViewStationPage(ctx, session.RoleGrants, station) {
		return dashboardPageData{}, http.StatusForbidden, errors.New("forbidden")
	}

	inventory, err := h.queries.GetStationInventorySnapshot(ctx, stationID)
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}

	tankViews := make([]dashboardStationTankView, 0, len(inventory))
	reorderCount := 0
	for _, row := range inventory {
		if row.NeedsReorder {
			reorderCount++
		}
		tankViews = append(tankViews, dashboardStationTankView{
			TankCode:         row.TankCode,
			FuelName:         row.FuelName,
			FuelCategory:     humanizeEnum(string(row.FuelCategory)),
			CurrentVolume:    formatNumeric(row.CurrentVolumeL, " L"),
			Capacity:         formatNumeric(row.CapacityL, " L"),
			ReorderThreshold: formatNumeric(row.ReorderThresholdL, " L"),
			FillPct:          row.FillPct.StringFixed(1) + "%",
			LastDip:          formatTimestamptz(row.LastDipAt, "No dip reading"),
			NeedsReorder:     row.NeedsReorder,
		})
	}

	deliveries, err := h.buildStationDeliveryHistory(ctx, stationID)
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}

	cache := newDashboardLookupCache()
	stationView := &dashboardStationDetailView{
		ID:            station.ID,
		Code:          station.Code,
		Name:          station.Name,
		LicenseNumber: station.SpbuLicenseNumber,
		Region:        station.RegionCode,
		Facility:      h.resolveFacilityLabel(ctx, cache, station.PrimaryFacilityID),
		Address:       formatText(station.Address, "Address not set"),
		ContactName:   formatText(station.ContactName, "No contact assigned"),
		ContactPhone:  formatText(station.ContactPhone, "No phone number"),
	}

	return h.pageData(c, session, dashboardPageData{
		Title:       station.Name,
		Heading:     station.Name,
		Description: "Tank inventory snapshot and recent delivery activity for the selected station.",
		Breadcrumbs: []DashboardBreadcrumb{
			{Label: "Stations", URL: "/stations"},
			{Label: station.Name, URL: c.Request.URL.Path},
		},
		Metrics: []dashboardMetric{
			{Label: "Active Tanks", Value: strconv.Itoa(len(tankViews)), Help: "Inventory tanks currently active at this station."},
			{Label: "Reorder Alerts", Value: strconv.Itoa(reorderCount), Help: "Tanks already at or below their reorder threshold."},
			{Label: "Recent Deliveries", Value: strconv.Itoa(len(deliveries)), Help: "Recent station deliveries collected from active and completed trips."},
		},
		Station:           stationView,
		StationTanks:      tankViews,
		StationDeliveries: deliveries,
	}), http.StatusOK, nil
}

func (h *DashboardHandler) buildStationDeliveryHistory(ctx context.Context, stationID int64) ([]dashboardStationDeliveryView, error) {
	cache := newDashboardLookupCache()

	type deliveryRecord struct {
		view      dashboardStationDeliveryView
		timestamp time.Time
	}

	records := make([]deliveryRecord, 0, 8)
	if h.workflowData != nil {
		active, err := h.workflowData.ListActiveTripsByStationScope(ctx, pgtype.Int8{Int64: stationID, Valid: true})
		if err != nil {
			return nil, err
		}
		for _, trip := range active {
			records = append(records, deliveryRecord{
				view: dashboardStationDeliveryView{
					TripID:         trip.ID,
					Status:         humanizeEnum(string(trip.Status)),
					StatusTone:     statusTone(string(trip.Status)),
					Vehicle:        trip.PlateNumber,
					Driver:         trip.DriverName,
					OriginFacility: h.resolveFacilityLabel(ctx, cache, trip.OriginFacilityID),
					Timestamp:      formatTimestamptz(trip.DepartedAt, "Awaiting departure"),
					DetailURL:      fmt.Sprintf("/trips/%d", trip.ID),
				},
				timestamp: pgTimestamptzToSortTime(trip.DepartedAt),
			})
		}
	}

	for _, status := range []db.TripStatusT{db.TripStatusTDELIVERED, db.TripStatusTRECONCILED, db.TripStatusTCLOSED} {
		completed, err := h.queries.ListTripsByStatus(ctx, status)
		if err != nil {
			return nil, err
		}
		for _, trip := range completed {
			if !trip.DestinationStationID.Valid || trip.DestinationStationID.Int64 != stationID {
				continue
			}
			records = append(records, deliveryRecord{
				view: dashboardStationDeliveryView{
					TripID:         trip.ID,
					Status:         humanizeEnum(string(trip.Status)),
					StatusTone:     statusTone(string(trip.Status)),
					Vehicle:        h.resolveVehicleLabel(ctx, cache, trip.VehicleID),
					Driver:         h.resolveDriverLabel(ctx, cache, trip.DriverID),
					OriginFacility: h.resolveFacilityLabel(ctx, cache, trip.OriginFacilityID),
					Timestamp:      formatTripMoment(trip),
					DetailURL:      fmt.Sprintf("/trips/%d", trip.ID),
				},
				timestamp: tripSortMoment(trip),
			})
		}
	}

	sort.Slice(records, func(i, j int) bool {
		return records[i].timestamp.After(records[j].timestamp)
	})
	if len(records) > 8 {
		records = records[:8]
	}

	views := make([]dashboardStationDeliveryView, 0, len(records))
	for _, record := range records {
		views = append(views, record.view)
	}
	return views, nil
}

func (h *DashboardHandler) buildFleetPageData(c *gin.Context, session *model.SessionData) (dashboardPageData, int, error) {
	if h.queries == nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}

	ctx := c.Request.Context()
	summaries, description, scopeNotice, err := h.listFleetVehiclesForSession(ctx, session)
	if err != nil {
		return dashboardPageData{}, http.StatusForbidden, err
	}

	vehicleIDs := make(map[int64]struct{}, len(summaries))
	for _, summary := range summaries {
		vehicleIDs[summary.ID] = struct{}{}
	}

	openMaintenance, err := h.queries.ListAllOpenMaintenance(ctx)
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}
	openMaintenanceCounts := make(map[int64]int)
	fleetMaintenance := make([]dashboardFleetMaintenanceView, 0)
	for _, record := range openMaintenance {
		if _, ok := vehicleIDs[record.VehicleID]; !ok {
			continue
		}
		openMaintenanceCounts[record.VehicleID]++
		fleetMaintenance = append(fleetMaintenance, dashboardFleetMaintenanceView{
			Vehicle:         record.PlateNumber,
			MaintenanceType: humanizeEnum(record.MaintenanceType),
			Description:     formatText(record.Description, "No description"),
			StartedAt:       formatTimestamptz(record.StartedAt, "-"),
			EstimatedReturn: formatTimestamptz(record.EstimatedReturnAt, "Awaiting estimate"),
			Depot:           record.DepotName,
		})
	}

	attentionRows, err := h.queries.ListVehiclesWithMaintenanceOrExpiryDue(ctx)
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}
	attentionByVehicle := make(map[int64]dashboardFleetAttentionView)
	for _, row := range attentionRows {
		if _, ok := vehicleIDs[row.ID]; !ok {
			continue
		}
		attentionByVehicle[row.ID] = dashboardFleetAttentionView{
			Vehicle:   row.PlateNumber,
			Notice:    humanizeEnum(row.NoticeType),
			DueDate:   fleetAttentionDate(row),
			Depot:     row.DepotName,
			DetailURL: fmt.Sprintf("/fleet/vehicles/%d", row.ID),
		}
	}

	depotLabels := make(map[int64]string)
	fleetVehicles := make([]dashboardFleetVehicleRow, 0, len(summaries))
	var availableCount, inFlightCount, maintenanceCount int
	for _, summary := range summaries {
		if summary.CurrentDepotID.Valid {
			if _, ok := depotLabels[summary.CurrentDepotID.Int64]; !ok {
				depotLabels[summary.CurrentDepotID.Int64] = h.resolveDepotLabel(ctx, summary.CurrentDepotID.Int64)
			}
		}

		switch summary.Status {
		case db.VehicleStatusTAVAILABLE:
			availableCount++
		case db.VehicleStatusTASSIGNED, db.VehicleStatusTINTRANSIT:
			inFlightCount++
		case db.VehicleStatusTUNDERMAINTENANCE:
			maintenanceCount++
		}

		depot := "Unassigned depot"
		if summary.CurrentDepotID.Valid {
			depot = depotLabels[summary.CurrentDepotID.Int64]
		}

		fleetVehicles = append(fleetVehicles, dashboardFleetVehicleRow{
			ID:                 summary.ID,
			PlateNumber:        summary.PlateNumber,
			Model:              formatText(summary.Model, "Model not set"),
			Status:             humanizeEnum(string(summary.Status)),
			StatusTone:         statusTone(string(summary.Status)),
			Capacity:           formatNumeric(summary.TotalCapacityL, " L"),
			Depot:              depot,
			NextInspection:     formatDate(summary.NextInspectionDue),
			OpenMaintenance:    fleetOpenMaintenanceLabel(openMaintenanceCounts[summary.ID]),
			HasOpenMaintenance: openMaintenanceCounts[summary.ID] > 0,
			DetailURL:          fmt.Sprintf("/fleet/vehicles/%d", summary.ID),
		})
	}

	sort.Slice(fleetVehicles, func(i, j int) bool {
		if fleetVehicles[i].Status != fleetVehicles[j].Status {
			return fleetVehicleStatusOrder(fleetVehicles[i].Status) < fleetVehicleStatusOrder(fleetVehicles[j].Status)
		}
		return fleetVehicles[i].PlateNumber < fleetVehicles[j].PlateNumber
	})

	fleetAttention := make([]dashboardFleetAttentionView, 0, len(attentionByVehicle))
	for _, vehicle := range fleetVehicles {
		if attention, ok := attentionByVehicle[vehicle.ID]; ok {
			fleetAttention = append(fleetAttention, attention)
		}
	}

	return h.pageData(c, session, dashboardPageData{
		Title:       "Fleet",
		Heading:     "Fleet Overview",
		Description: description,
		ScopeNotice: scopeNotice,
		Breadcrumbs: []DashboardBreadcrumb{{Label: "Fleet", URL: "/fleet"}},
		Metrics: []dashboardMetric{
			{Label: "Vehicles", Value: strconv.Itoa(len(fleetVehicles)), Help: "Active vehicles visible in this dashboard scope."},
			{Label: "Available", Value: strconv.Itoa(availableCount), Help: "Vehicles ready for assignment."},
			{Label: "On Trip", Value: strconv.Itoa(inFlightCount), Help: "Vehicles assigned or in transit."},
			{Label: "Maintenance", Value: strconv.Itoa(maintenanceCount), Help: "Vehicles currently marked under maintenance."},
		},
		FleetVehicles:    fleetVehicles,
		FleetMaintenance: fleetMaintenance,
		FleetAttention:   fleetAttention,
	}), http.StatusOK, nil
}

func (h *DashboardHandler) listFleetVehiclesForSession(ctx context.Context, session *model.SessionData) ([]dashboardFleetVehicleSummary, string, string, error) {
	best := dashboardBestRole(session.RoleGrants)
	statuses := []db.VehicleStatusT{
		db.VehicleStatusTAVAILABLE,
		db.VehicleStatusTASSIGNED,
		db.VehicleStatusTINTRANSIT,
		db.VehicleStatusTUNDERMAINTENANCE,
	}

	var description string
	var scopeNotice string
	vehicles := make(map[int64]dashboardFleetVehicleSummary)
	addVehicle := func(summary dashboardFleetVehicleSummary) {
		vehicles[summary.ID] = summary
	}

	for _, status := range statuses {
		switch {
		case best.Role == "SYSTEM_ADMIN":
			rows, err := h.queries.ListVehiclesByStatus(ctx, status)
			if err != nil {
				return nil, "", "", err
			}
			for _, row := range rows {
				addVehicle(dashboardFleetVehicleSummary{
					ID:                row.ID,
					PlateNumber:       row.PlateNumber,
					Model:             row.Model,
					Status:            row.Status,
					TotalCapacityL:    row.TotalCapacityL,
					CurrentDepotID:    row.CurrentDepotID,
					NextInspectionDue: row.NextInspectionDue,
				})
			}
			description = "Vehicle availability, inspections, and maintenance signals across the current dashboard scope."
			scopeNotice = "Company-wide fleet visibility."
		case best.Role == "REFINERY_ADMIN" && best.ScopeType == "REFINERY" && best.ScopeID != nil:
			rows, err := h.queries.ListVehiclesByStatusAndRefinery(ctx, db.ListVehiclesByStatusAndRefineryParams{
				Status:     status,
				RefineryID: *best.ScopeID,
			})
			if err != nil {
				return nil, "", "", err
			}
			for _, row := range rows {
				addVehicle(dashboardFleetVehicleSummary{
					ID:                row.ID,
					PlateNumber:       row.PlateNumber,
					Model:             row.Model,
					Status:            row.Status,
					TotalCapacityL:    row.TotalCapacityL,
					CurrentDepotID:    row.CurrentDepotID,
					NextInspectionDue: row.NextInspectionDue,
				})
			}
			description = "Vehicle availability, inspections, and maintenance signals across the current dashboard scope."
			scopeNotice = "Filtered to the refinery-linked fleet."
		case (best.Role == "FACILITY_MANAGER" || best.Role == "FACILITY_OPERATOR") && best.ScopeType == "FACILITY" && best.ScopeID != nil:
			rows, err := h.queries.ListVehiclesByStatusAndFacility(ctx, db.ListVehiclesByStatusAndFacilityParams{
				Status:            status,
				PrimaryFacilityID: *best.ScopeID,
			})
			if err != nil {
				return nil, "", "", err
			}
			for _, row := range rows {
				addVehicle(dashboardFleetVehicleSummary{
					ID:                row.ID,
					PlateNumber:       row.PlateNumber,
					Model:             row.Model,
					Status:            row.Status,
					TotalCapacityL:    row.TotalCapacityL,
					CurrentDepotID:    row.CurrentDepotID,
					NextInspectionDue: row.NextInspectionDue,
				})
			}
			description = "Vehicle availability, inspections, and maintenance signals across the current dashboard scope."
			scopeNotice = "Filtered to " + h.resolveFacilityLabel(ctx, newDashboardLookupCache(), *best.ScopeID) + "."
		case best.Role == "DEPOT_STAFF" && best.ScopeType == "DEPOT" && best.ScopeID != nil:
			rows, err := h.queries.ListVehiclesByStatusAndDepot(ctx, db.ListVehiclesByStatusAndDepotParams{
				Status:         status,
				CurrentDepotID: pgtype.Int8{Int64: *best.ScopeID, Valid: true},
			})
			if err != nil {
				return nil, "", "", err
			}
			for _, row := range rows {
				addVehicle(dashboardFleetVehicleSummary{
					ID:                row.ID,
					PlateNumber:       row.PlateNumber,
					Model:             row.Model,
					Status:            row.Status,
					TotalCapacityL:    row.TotalCapacityL,
					CurrentDepotID:    row.CurrentDepotID,
					NextInspectionDue: row.NextInspectionDue,
				})
			}
			description = "Vehicle availability, inspections, and maintenance signals across the current dashboard scope."
			scopeNotice = "Filtered to " + h.resolveDepotLabel(ctx, *best.ScopeID) + "."
		default:
			return nil, "", "", errors.New("forbidden")
		}
	}

	out := make([]dashboardFleetVehicleSummary, 0, len(vehicles))
	for _, summary := range vehicles {
		out = append(out, summary)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].PlateNumber < out[j].PlateNumber
	})
	return out, description, scopeNotice, nil
}

func (h *DashboardHandler) buildVehicleDetailPageData(c *gin.Context, session *model.SessionData) (dashboardPageData, int, error) {
	if h.queries == nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}

	vehicleID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		return dashboardPageData{}, http.StatusBadRequest, errors.New("invalid vehicle id")
	}

	ctx := c.Request.Context()
	vehicle, err := h.queries.GetVehicle(ctx, vehicleID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dashboardPageData{}, http.StatusNotFound, errors.New("vehicle not found")
		}
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}
	if !h.canViewVehiclePage(ctx, session.RoleGrants, vehicle) {
		return dashboardPageData{}, http.StatusForbidden, errors.New("forbidden")
	}

	maintenanceRows, err := h.queries.ListMaintenanceByVehicle(ctx, vehicleID)
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}
	maintenance := make([]dashboardVehicleMaintenanceView, 0, len(maintenanceRows))
	openMaintenanceCount := 0
	for _, row := range maintenanceRows {
		if !row.CompletedAt.Valid {
			openMaintenanceCount++
		}
		maintenance = append(maintenance, dashboardVehicleMaintenanceView{
			MaintenanceType: humanizeEnum(row.MaintenanceType),
			Description:     formatText(row.Description, "No description"),
			StartedAt:       formatTimestamptz(row.StartedAt, "-"),
			EstimatedReturn: formatTimestamptz(row.EstimatedReturnAt, "Awaiting estimate"),
			CompletedAt:     formatTimestamptz(row.CompletedAt, "Open"),
			Notes:           formatText(row.Notes, "No notes"),
			IsOpen:          !row.CompletedAt.Valid,
		})
	}

	compartmentRows, err := h.queries.ListAllCompartmentsByVehicle(ctx, vehicleID)
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}
	compartments := make([]dashboardVehicleCompartmentView, 0, len(compartmentRows))
	for _, row := range compartmentRows {
		compartments = append(compartments, dashboardVehicleCompartmentView{
			Compartment: fmt.Sprintf("Compartment %d", row.CompartmentNumber),
			FuelType:    formatText(row.FuelTypeCode, "Unassigned"),
			Capacity:    formatNumeric(row.CapacityL, " L"),
		})
	}

	tripRows, err := h.queries.ListTripsByVehicle(ctx, db.ListTripsByVehicleParams{
		VehicleID: vehicleID,
		Limit:     8,
	})
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}
	cache := newDashboardLookupCache()
	trips := make([]dashboardVehicleTripView, 0, len(tripRows))
	for _, trip := range tripRows {
		trips = append(trips, dashboardVehicleTripView{
			TripID:      trip.ID,
			DoID:        trip.DoID,
			Status:      humanizeEnum(string(trip.Status)),
			StatusTone:  statusTone(string(trip.Status)),
			Destination: h.resolveDestinationLabel(ctx, cache, trip.DestinationType, trip.DestinationStationID, trip.DestinationFacilityID),
			DepartedAt:  formatTimestamptz(trip.DepartedAt, "Awaiting departure"),
			CompletedAt: formatTimestamptz(trip.CompletedAt, "In progress"),
			DetailURL:   fmt.Sprintf("/trips/%d", trip.ID),
		})
	}

	depotLabel := "Unassigned depot"
	if vehicle.CurrentDepotID.Valid {
		depotLabel = h.resolveDepotLabel(ctx, vehicle.CurrentDepotID.Int64)
	}

	vehicleView := &dashboardVehicleDetailView{
		ID:              vehicle.ID,
		PlateNumber:     vehicle.PlateNumber,
		Status:          humanizeEnum(string(vehicle.Status)),
		StatusTone:      statusTone(string(vehicle.Status)),
		Model:           formatText(vehicle.Model, "Model not set"),
		ManufactureYear: formatInt2(vehicle.ManufactureYear, "-"),
		Capacity:        formatNumeric(vehicle.TotalCapacityL, " L"),
		TareWeight:      formatNumeric(vehicle.TareWeightKg, " kg"),
		Depot:           depotLabel,
		KeurNumber:      formatText(vehicle.KeurNumber, "Not set"),
		KeurExpiry:      formatDate(vehicle.KeurExpiry),
		NextInspection:  formatDate(vehicle.NextInspectionDue),
		LastAssignedAt:  formatTimestamptz(vehicle.LastAssignedAt, "Not assigned yet"),
		Notes:           formatText(vehicle.Notes, "No notes"),
	}

	return h.pageData(c, session, dashboardPageData{
		Title:       vehicle.PlateNumber,
		Heading:     vehicle.PlateNumber,
		Description: "Vehicle profile, maintenance history, compartments, and recent trips.",
		Breadcrumbs: []DashboardBreadcrumb{
			{Label: "Fleet", URL: "/fleet"},
			{Label: vehicle.PlateNumber, URL: c.Request.URL.Path},
		},
		Metrics: []dashboardMetric{
			{Label: "Open Maintenance", Value: strconv.Itoa(openMaintenanceCount), Help: "Maintenance records still open for this vehicle."},
			{Label: "Compartments", Value: strconv.Itoa(len(compartments)), Help: "Compartments configured for this vehicle."},
			{Label: "Recent Trips", Value: strconv.Itoa(len(trips)), Help: "Most recent trips associated with this vehicle."},
		},
		Vehicle:            vehicleView,
		VehicleTrips:       trips,
		VehicleMaintenance: maintenance,
		Compartments:       compartments,
	}), http.StatusOK, nil
}

func (h *DashboardHandler) canViewStationPage(ctx context.Context, roles []model.RoleGrant, station db.GetStationRow) bool {
	for _, role := range roles {
		switch {
		case role.Role == "SYSTEM_ADMIN":
			return true
		case role.Role == "REFINERY_ADMIN" && role.ScopeType == "REFINERY" && role.ScopeID != nil:
			facility, err := h.queries.GetFacility(ctx, station.PrimaryFacilityID)
			if err == nil && facility.RefineryID == *role.ScopeID {
				return true
			}
		case (role.Role == "FACILITY_MANAGER" || role.Role == "FACILITY_OPERATOR") && role.ScopeType == "FACILITY" && role.ScopeID != nil:
			if station.PrimaryFacilityID == *role.ScopeID {
				return true
			}
			stations, err := h.queries.ListStationsServedByFacility(ctx, *role.ScopeID)
			if err != nil {
				continue
			}
			for _, row := range stations {
				if row.ID == station.ID {
					return true
				}
			}
		case role.Role == "STATION_MANAGER" && role.ScopeType == "STATION" && role.ScopeID != nil && *role.ScopeID == station.ID:
			return true
		}
	}
	return false
}

func (h *DashboardHandler) canViewVehiclePage(ctx context.Context, roles []model.RoleGrant, vehicle db.GetVehicleRow) bool {
	for _, role := range roles {
		switch {
		case role.Role == "SYSTEM_ADMIN":
			return true
		case role.Role == "DEPOT_STAFF" && role.ScopeType == "DEPOT" && role.ScopeID != nil && vehicle.CurrentDepotID.Valid && vehicle.CurrentDepotID.Int64 == *role.ScopeID:
			return true
		case (role.Role == "FACILITY_MANAGER" || role.Role == "FACILITY_OPERATOR") && role.ScopeType == "FACILITY" && role.ScopeID != nil && vehicle.CurrentDepotID.Valid:
			depot, err := h.queries.GetDepot(ctx, vehicle.CurrentDepotID.Int64)
			if err == nil && depot.PrimaryFacilityID == *role.ScopeID {
				return true
			}
		case role.Role == "REFINERY_ADMIN" && role.ScopeType == "REFINERY" && role.ScopeID != nil && vehicle.CurrentDepotID.Valid:
			depot, err := h.queries.GetDepot(ctx, vehicle.CurrentDepotID.Int64)
			if err != nil {
				continue
			}
			facility, err := h.queries.GetFacility(ctx, depot.PrimaryFacilityID)
			if err == nil && facility.RefineryID == *role.ScopeID {
				return true
			}
		}
	}
	return false
}

func (h *DashboardHandler) resolveDepotLabel(ctx context.Context, depotID int64) string {
	label := fmt.Sprintf("Depot #%d", depotID)
	if h.queries == nil {
		return label
	}
	depot, err := h.queries.GetDepot(ctx, depotID)
	if err != nil {
		return label
	}
	return depot.Name
}

func fleetOpenMaintenanceLabel(count int) string {
	if count <= 0 {
		return "No open records"
	}
	if count == 1 {
		return "1 open record"
	}
	return fmt.Sprintf("%d open records", count)
}

func fleetAttentionDate(row db.ListVehiclesWithMaintenanceOrExpiryDueRow) string {
	switch row.NoticeType {
	case "KEUR_EXPIRING":
		return formatDate(row.KeurExpiry)
	case "INSPECTION_DUE":
		return formatDate(row.NextInspectionDue)
	default:
		return "Requires follow-up"
	}
}

func fleetVehicleStatusOrder(status string) int {
	switch status {
	case "Available":
		return 1
	case "Assigned":
		return 2
	case "In Transit":
		return 3
	case "Under Maintenance":
		return 4
	default:
		return 5
	}
}

func tripSortMoment(trip db.Trip) time.Time {
	if trip.CompletedAt.Valid {
		return trip.CompletedAt.Time
	}
	if trip.ArrivedAt.Valid {
		return trip.ArrivedAt.Time
	}
	if trip.DepartedAt.Valid {
		return trip.DepartedAt.Time
	}
	if trip.CreatedAt.Valid {
		return trip.CreatedAt.Time
	}
	return time.Time{}
}

func pgTimestamptzToSortTime(value pgtype.Timestamptz) time.Time {
	if !value.Valid {
		return time.Time{}
	}
	return value.Time
}

func formatTripMoment(trip db.Trip) string {
	if trip.CompletedAt.Valid {
		return trip.CompletedAt.Time.UTC().Format("02 Jan 2006 15:04 MST")
	}
	if trip.ArrivedAt.Valid {
		return trip.ArrivedAt.Time.UTC().Format("02 Jan 2006 15:04 MST")
	}
	if trip.DepartedAt.Valid {
		return trip.DepartedAt.Time.UTC().Format("02 Jan 2006 15:04 MST")
	}
	return formatTimestamptz(trip.CreatedAt, "-")
}

func formatInt2(value pgtype.Int2, fallback string) string {
	if !value.Valid {
		return fallback
	}
	return strconv.Itoa(int(value.Int16))
}
