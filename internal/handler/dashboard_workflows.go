package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
	"github.com/adevsh/petrosync/internal/model"
	"github.com/adevsh/petrosync/internal/service"
)

type dashboardWorkflowQuerier interface {
	GetFacility(ctx context.Context, id int64) (db.GetFacilityRow, error)
	GetStation(ctx context.Context, id int64) (db.GetStationRow, error)
	GetVehicle(ctx context.Context, id int64) (db.GetVehicleRow, error)
	GetDriver(ctx context.Context, id int64) (db.GetDriverRow, error)
	GetDeliveryOrder(ctx context.Context, id int64) (db.DeliveryOrder, error)
	GetTripByDO(ctx context.Context, doID int64) (db.GetTripByDORow, error)
	GetTripWithDetails(ctx context.Context, id int64) (db.GetTripWithDetailsRow, error)
	ListDOsByOriginFacility(ctx context.Context, originFacilityID int64) ([]db.DeliveryOrder, error)
	ListDOItemsByDO(ctx context.Context, doID int64) ([]db.ListDOItemsByDORow, error)
	ListVehiclesByStatusAndFacility(ctx context.Context, arg db.ListVehiclesByStatusAndFacilityParams) ([]db.ListVehiclesByStatusAndFacilityRow, error)
	ListDriversByDepot(ctx context.Context, homeDepotID pgtype.Int8) ([]db.ListDriversByDepotRow, error)
	AssignVehicleAndDriverToDO(ctx context.Context, arg db.AssignVehicleAndDriverToDOParams) (db.DeliveryOrder, error)
	ListActiveTrips(ctx context.Context) ([]db.ListActiveTripsRow, error)
	ListActiveTripsByRefineryScope(ctx context.Context, refineryID int64) ([]db.ListActiveTripsByRefineryScopeRow, error)
	ListActiveTripsByFacilityScope(ctx context.Context, originFacilityID int64) ([]db.ListActiveTripsByFacilityScopeRow, error)
	ListActiveTripsByStationScope(ctx context.Context, destinationStationID pgtype.Int8) ([]db.ListActiveTripsByStationScopeRow, error)
	ListTripEventsByTrip(ctx context.Context, tripID int64) ([]db.TripEvent, error)
	ListSealsByTrip(ctx context.Context, tripID int64) ([]db.ListSealsByTripRow, error)
	ListWeightBridgeReadingsByTrip(ctx context.Context, tripID pgtype.Int8) ([]db.WeightBridgeReading, error)
	CreateWeightBridgeReading(ctx context.Context, arg db.CreateWeightBridgeReadingParams) (db.WeightBridgeReading, error)
}

type dashboardWorkflowService interface {
	ApproveDeliveryOrder(ctx context.Context, doID, userID int64) (db.DeliveryOrder, error)
}

type dashboardTripPhotoLister interface {
	ListTripPhotosWithURLs(ctx context.Context, tripID int64) ([]service.TripPhotoWithURL, error)
}

type dashboardNotificationService interface {
	NotifyDOApproved(ctx context.Context, do db.DeliveryOrder) error
	NotifyTripAssigned(ctx context.Context, do db.DeliveryOrder) error
	NotifyManualWeightBridgePending(ctx context.Context, reading db.WeightBridgeReading) error
}

type dashboardDeliveryOrderRow struct {
	ID            int64
	DoNumber      string
	Status        string
	StatusTone    string
	Destination   string
	ScheduledDate string
	Assignment    string
	DetailURL     string
	CanApprove    bool
	CanAssign     bool
	ErrorMessage  string
}

type dashboardDeliveryOrderDetailView struct {
	ID               int64
	OriginFacilityID int64
	DoNumber         string
	Status           string
	StatusTone       string
	OriginFacility   string
	Destination      string
	ScheduledDate    string
	RaisedBy         string
	ApprovedAt       string
	AssignedVehicle  string
	AssignedDriver   string
	AssignedAt       string
	Notes            string
	CanApprove       bool
	CanAssign        bool
	ActionError      string
	ApproveURL       string
	AssignURL        string
	DetailFragment   string
	TripURL          string
}

type dashboardDeliveryOrderItemView struct {
	FuelType        string
	Compartment     string
	RequestedVolume string
	AllocatedVolume string
}

type dashboardSelectOption struct {
	ID    int64
	Label string
	Hint  string
}

type dashboardTripRow struct {
	ID          int64
	Status      string
	StatusTone  string
	PlateNumber string
	DriverName  string
	Destination string
	DepartedAt  string
	DetailURL   string
}

type dashboardTripDetailView struct {
	ID                   int64
	DoID                 int64
	DoURL                string
	Status               string
	StatusTone           string
	PlateNumber          string
	DriverName           string
	DriverTelegram       string
	OriginFacility       string
	Destination          string
	DepartedAt           string
	ArrivedAt            string
	CompletedAt          string
	WeightBridgeEntryURL string
}

type dashboardTripEventView struct {
	EventType string
	Timestamp string
	Actor     string
	Payload   string
}

type dashboardWeightBridgeReadingView struct {
	ID             int64
	ReadingType    string
	Weight         string
	Method         string
	Temperature    string
	ApprovalStatus string
	ApprovalTone   string
	RecordedBy     string
	ApprovedAt     string
	Notes          string
	CreatedAt      string
}

type dashboardTripSealView struct {
	Compartment  string
	IssuedSeal   string
	IssuedBy     string
	IssuedAt     string
	VerifiedSeal string
	VerifiedBy   string
	VerifiedAt   string
	Verification string
	Notes        string
}

type dashboardTripPhotoView struct {
	EventType   string
	Compartment string
	TakenAt     string
	URL         string
	Notes       string
}

type dashboardWeightBridgeFormView struct {
	TripID     int64
	TripLabel  string
	Action     string
	ErrorLabel string
}

type dashboardLookupCache struct {
	facilities map[int64]string
	stations   map[int64]string
	vehicles   map[int64]string
	drivers    map[int64]string
}

func (h *DashboardHandler) DeliveryOrders(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	if h.workflowData == nil {
		c.String(http.StatusNotFound, "not found")
		return
	}
	if facilityID, ok := dashboardFacilityScope(session.RoleGrants); ok {
		c.Redirect(http.StatusSeeOther, fmt.Sprintf("/facilities/%d/delivery-orders", facilityID))
		return
	}
	if !canViewDeliveryOrderPages(session.RoleGrants) {
		c.String(http.StatusForbidden, "forbidden")
		return
	}
	if h.queries == nil {
		c.String(http.StatusInternalServerError, "dashboard data unavailable")
		return
	}

	rows, err := h.queries.GetCompanyWideDashboardSummary(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "dashboard data unavailable")
		return
	}

	h.render(c, http.StatusOK, "delivery_orders.html", h.pageData(c, session, dashboardPageData{
		Title:       "Delivery Orders",
		Heading:     "Delivery Orders",
		Description: "Choose a facility to work the approval and assignment queue.",
		Breadcrumbs: []DashboardBreadcrumb{{Label: "Delivery Orders", URL: "/delivery-orders"}},
		CompanyRows: buildCompanySummaryRows(rows),
		ScopeNotice: "Facility-scoped roles jump directly to their queue. Refinery and system admins can pick a facility below.",
	}))
}

func (h *DashboardHandler) FacilityDeliveryOrders(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	facilityID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid facility id")
		return
	}
	if !canViewFacility(session.RoleGrants, facilityID) {
		c.String(http.StatusForbidden, "forbidden")
		return
	}

	data, status, err := h.buildFacilityDeliveryOrderPageData(c, session, facilityID, "")
	if err != nil {
		c.String(status, err.Error())
		return
	}
	h.render(c, http.StatusOK, "delivery_orders.html", data)
}

func (h *DashboardHandler) DeliveryOrderTable(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	facilityID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid facility id")
		return
	}
	if !canViewFacility(session.RoleGrants, facilityID) {
		c.String(http.StatusForbidden, "forbidden")
		return
	}

	data, status, err := h.buildFacilityDeliveryOrderPageData(c, session, facilityID, "")
	if err != nil {
		c.String(status, err.Error())
		return
	}
	h.renderTemplate(c, http.StatusOK, "delivery_orders.html", "delivery-order-table", data)
}

func (h *DashboardHandler) DeliveryOrderDetail(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	doID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid delivery order id")
		return
	}

	data, status, err := h.buildDeliveryOrderDetailPageData(c, session, doID, "")
	if err != nil {
		c.String(status, err.Error())
		return
	}
	h.render(c, http.StatusOK, "delivery_order_detail.html", data)
}

func (h *DashboardHandler) DeliveryOrderDetailFragment(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	doID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid delivery order id")
		return
	}

	data, status, err := h.buildDeliveryOrderDetailPageData(c, session, doID, "")
	if err != nil {
		c.String(status, err.Error())
		return
	}
	h.renderTemplate(c, http.StatusOK, "delivery_order_detail.html", "delivery-order-detail-panels", data)
}

func (h *DashboardHandler) ApproveDeliveryOrderRow(c *gin.Context) {
	h.handleApproveDeliveryOrder(c, "delivery_orders.html", "delivery-order-row")
}

func (h *DashboardHandler) ApproveDeliveryOrderDetail(c *gin.Context) {
	h.handleApproveDeliveryOrder(c, "delivery_order_detail.html", "delivery-order-detail-panels")
}

func (h *DashboardHandler) AssignDeliveryOrder(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	if h.workflowData == nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	doID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid delivery order id")
		return
	}

	currentDO, err := h.workflowData.GetDeliveryOrder(c.Request.Context(), doID)
	if err != nil {
		c.String(http.StatusNotFound, "delivery order not found")
		return
	}
	if !canViewFacility(session.RoleGrants, currentDO.OriginFacilityID) {
		c.String(http.StatusForbidden, "forbidden")
		return
	}

	vehicleID, vehicleErr := strconv.ParseInt(strings.TrimSpace(c.PostForm("vehicle_id")), 10, 64)
	driverID, driverErr := strconv.ParseInt(strings.TrimSpace(c.PostForm("driver_id")), 10, 64)
	if vehicleErr != nil || driverErr != nil || vehicleID <= 0 || driverID <= 0 {
		data, status, buildErr := h.buildDeliveryOrderDetailPageData(c, session, doID, "vehicle and driver are required")
		if buildErr != nil {
			c.String(status, buildErr.Error())
			return
		}
		h.renderTemplate(c, http.StatusOK, "delivery_order_detail.html", "delivery-order-detail-panels", data)
		return
	}

	assigned, err := h.workflowData.AssignVehicleAndDriverToDO(c.Request.Context(), db.AssignVehicleAndDriverToDOParams{
		ID:                doID,
		AssignedVehicleID: pgtype.Int8{Int64: vehicleID, Valid: true},
		AssignedDriverID:  pgtype.Int8{Int64: driverID, Valid: true},
	})
	if err != nil {
		data, status, buildErr := h.buildDeliveryOrderDetailPageData(c, session, doID, "delivery order is not ready for assignment")
		if buildErr != nil {
			c.String(status, buildErr.Error())
			return
		}
		h.renderTemplate(c, http.StatusOK, "delivery_order_detail.html", "delivery-order-detail-panels", data)
		return
	}

	if h.notifications != nil {
		go func() {
			if notifyErr := h.notifications.NotifyTripAssigned(c.Request.Context(), assigned); notifyErr != nil {
				log.Printf("trip assigned notification failed: %v", notifyErr)
			}
		}()
	}

	data, status, buildErr := h.buildDeliveryOrderDetailPageData(c, session, doID, "")
	if buildErr != nil {
		c.String(status, buildErr.Error())
		return
	}
	h.renderTemplate(c, http.StatusOK, "delivery_order_detail.html", "delivery-order-detail-panels", data)
}

func (h *DashboardHandler) Trips(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	if h.workflowData == nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	rows, heading, description, err := h.buildTripRowsForSession(c.Request.Context(), session)
	if err != nil {
		if errors.Is(err, errDashboardForbidden) {
			c.String(http.StatusForbidden, "forbidden")
			return
		}
		c.String(http.StatusInternalServerError, "dashboard data unavailable")
		return
	}

	h.render(c, http.StatusOK, "trips.html", h.pageData(c, session, dashboardPageData{
		Title:       "Trips",
		Heading:     heading,
		Description: description,
		Breadcrumbs: []DashboardBreadcrumb{{Label: "Trips", URL: "/trips"}},
		Trips:       rows,
	}))
}

func (h *DashboardHandler) TripDetail(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	tripID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid trip id")
		return
	}

	data, status, err := h.buildTripDetailPageData(c, session, tripID, "")
	if err != nil {
		c.String(status, err.Error())
		return
	}
	h.render(c, http.StatusOK, "trip_detail.html", data)
}

func (h *DashboardHandler) ShowWeightBridgeEntry(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	tripID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid trip id")
		return
	}

	data, status, err := h.buildWeightBridgeEntryPageData(c, session, tripID, "")
	if err != nil {
		c.String(status, err.Error())
		return
	}
	h.render(c, http.StatusOK, "weight_bridge_new.html", data)
}

func (h *DashboardHandler) CreateWeightBridgeEntry(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	if h.workflowData == nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	tripID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid trip id")
		return
	}

	trip, status, err := h.loadTripForAccess(c.Request.Context(), session, tripID)
	if err != nil {
		c.String(status, err.Error())
		return
	}

	readingType := strings.ToUpper(strings.TrimSpace(c.PostForm("reading_type")))
	if readingType != "TARE" && readingType != "GROSS" {
		data, buildStatus, buildErr := h.buildWeightBridgeEntryPageData(c, session, tripID, "reading type must be TARE or GROSS")
		if buildErr != nil {
			c.String(buildStatus, buildErr.Error())
			return
		}
		h.render(c, http.StatusOK, "weight_bridge_new.html", data)
		return
	}

	weightValue, err := strconv.ParseFloat(strings.TrimSpace(c.PostForm("weight_kg")), 64)
	if err != nil || weightValue <= 0 {
		data, buildStatus, buildErr := h.buildWeightBridgeEntryPageData(c, session, tripID, "weight must be a positive number")
		if buildErr != nil {
			c.String(buildStatus, buildErr.Error())
			return
		}
		h.render(c, http.StatusOK, "weight_bridge_new.html", data)
		return
	}

	method := db.MeasurementMethodT(strings.TrimSpace(c.PostForm("method")))
	if !method.Valid() {
		data, buildStatus, buildErr := h.buildWeightBridgeEntryPageData(c, session, tripID, "measurement method is invalid")
		if buildErr != nil {
			c.String(buildStatus, buildErr.Error())
			return
		}
		h.render(c, http.StatusOK, "weight_bridge_new.html", data)
		return
	}

	var ambientTemp pgtype.Numeric
	ambientRaw := strings.TrimSpace(c.PostForm("ambient_temp_celsius"))
	if ambientRaw != "" {
		value, parseErr := strconv.ParseFloat(ambientRaw, 64)
		if parseErr != nil {
			data, buildStatus, buildErr := h.buildWeightBridgeEntryPageData(c, session, tripID, "ambient temperature must be numeric")
			if buildErr != nil {
				c.String(buildStatus, buildErr.Error())
				return
			}
			h.render(c, http.StatusOK, "weight_bridge_new.html", data)
			return
		}
		ambientTemp = floatToNumeric(value)
	}

	notes := strings.TrimSpace(c.PostForm("notes"))
	var notesValue pgtype.Text
	if notes != "" {
		notesValue = pgtype.Text{String: notes, Valid: true}
	}

	reading, err := h.workflowData.CreateWeightBridgeReading(c.Request.Context(), db.CreateWeightBridgeReadingParams{
		TripID:             pgtype.Int8{Int64: tripID, Valid: true},
		VehicleID:          trip.VehicleID,
		ReadingType:        readingType,
		WeightKg:           floatToNumeric(weightValue),
		Method:             method,
		AmbientTempCelsius: ambientTemp,
		RecordedBy:         session.UserID,
		Notes:              notesValue,
	})
	if err != nil {
		data, buildStatus, buildErr := h.buildWeightBridgeEntryPageData(c, session, tripID, "weight bridge reading could not be saved")
		if buildErr != nil {
			c.String(buildStatus, buildErr.Error())
			return
		}
		h.render(c, http.StatusOK, "weight_bridge_new.html", data)
		return
	}

	if h.notifications != nil {
		go func() {
			if notifyErr := h.notifications.NotifyManualWeightBridgePending(c.Request.Context(), reading); notifyErr != nil {
				log.Printf("manual weight bridge notification failed: %v", notifyErr)
			}
		}()
	}

	c.Redirect(http.StatusSeeOther, fmt.Sprintf("/trips/%d#weight-bridge", tripID))
}

var errDashboardForbidden = errors.New("dashboard forbidden")

func (h *DashboardHandler) handleApproveDeliveryOrder(c *gin.Context, page, templateName string) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	if h.workflowData == nil || h.workflow == nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	doID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid delivery order id")
		return
	}

	currentDO, err := h.workflowData.GetDeliveryOrder(c.Request.Context(), doID)
	if err != nil {
		c.String(http.StatusNotFound, "delivery order not found")
		return
	}
	if !canViewFacility(session.RoleGrants, currentDO.OriginFacilityID) {
		c.String(http.StatusForbidden, "forbidden")
		return
	}

	_, err = h.workflow.ApproveDeliveryOrder(c.Request.Context(), doID, session.UserID)
	if err != nil {
		message := "delivery order is not pending approval"
		if errors.Is(err, service.ErrInsufficientStock) {
			message = "insufficient storage tank stock"
		}
		h.renderApproveError(c, session, currentDO, page, templateName, message)
		return
	}

	updatedDO, err := h.workflowData.GetDeliveryOrder(c.Request.Context(), doID)
	if err != nil {
		c.String(http.StatusInternalServerError, "dashboard data unavailable")
		return
	}
	if h.notifications != nil {
		go func() {
			if notifyErr := h.notifications.NotifyDOApproved(c.Request.Context(), updatedDO); notifyErr != nil {
				log.Printf("delivery order approved notification failed: %v", notifyErr)
			}
		}()
	}

	switch templateName {
	case "delivery-order-row":
		cache := newDashboardLookupCache()
		row := h.buildDeliveryOrderRow(c.Request.Context(), updatedDO, cache)
		h.renderTemplate(c, http.StatusOK, page, templateName, h.pageData(c, session, dashboardPageData{DeliveryOrders: []dashboardDeliveryOrderRow{row}}))
	default:
		data, status, buildErr := h.buildDeliveryOrderDetailPageData(c, session, doID, "")
		if buildErr != nil {
			c.String(status, buildErr.Error())
			return
		}
		h.renderTemplate(c, http.StatusOK, page, templateName, data)
	}
}

func (h *DashboardHandler) renderApproveError(c *gin.Context, session *model.SessionData, currentDO db.DeliveryOrder, page, templateName, message string) {
	switch templateName {
	case "delivery-order-row":
		cache := newDashboardLookupCache()
		row := h.buildDeliveryOrderRow(c.Request.Context(), currentDO, cache)
		row.ErrorMessage = message
		h.renderTemplate(c, http.StatusOK, page, templateName, h.pageData(c, session, dashboardPageData{DeliveryOrders: []dashboardDeliveryOrderRow{row}}))
	default:
		data, status, buildErr := h.buildDeliveryOrderDetailPageData(c, session, currentDO.ID, message)
		if buildErr != nil {
			c.String(status, buildErr.Error())
			return
		}
		h.renderTemplate(c, http.StatusOK, page, templateName, data)
	}
}

func (h *DashboardHandler) buildFacilityDeliveryOrderPageData(c *gin.Context, session *model.SessionData, facilityID int64, message string) (dashboardPageData, int, error) {
	if h.workflowData == nil {
		return dashboardPageData{}, http.StatusNotFound, errors.New("not found")
	}

	facility, err := h.workflowData.GetFacility(c.Request.Context(), facilityID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dashboardPageData{}, http.StatusNotFound, errors.New("facility not found")
		}
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}

	dos, err := h.workflowData.ListDOsByOriginFacility(c.Request.Context(), facilityID)
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}

	cache := newDashboardLookupCache()
	rows := make([]dashboardDeliveryOrderRow, 0, len(dos))
	for _, item := range dos {
		rows = append(rows, h.buildDeliveryOrderRow(c.Request.Context(), item, cache))
	}

	return h.pageData(c, session, dashboardPageData{
		Title:          facility.Name + " Delivery Orders",
		Heading:        facility.Name + " Delivery Orders",
		Description:    "Review the facility queue, approve pending orders, and open detail pages for assignment.",
		Breadcrumbs:    []DashboardBreadcrumb{{Label: "Delivery Orders", URL: "/delivery-orders"}, {Label: facility.Name, URL: c.Request.URL.Path}},
		Facility:       &dashboardFacilitySummary{FacilityID: facility.ID, FacilityName: facility.Name},
		DeliveryOrders: rows,
		ScopeNotice:    message,
	}), http.StatusOK, nil
}

func (h *DashboardHandler) buildDeliveryOrderDetailPageData(c *gin.Context, session *model.SessionData, doID int64, message string) (dashboardPageData, int, error) {
	if h.workflowData == nil {
		return dashboardPageData{}, http.StatusNotFound, errors.New("not found")
	}

	deliveryOrder, err := h.workflowData.GetDeliveryOrder(c.Request.Context(), doID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return dashboardPageData{}, http.StatusNotFound, errors.New("delivery order not found")
		}
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}
	if !canViewFacility(session.RoleGrants, deliveryOrder.OriginFacilityID) {
		return dashboardPageData{}, http.StatusForbidden, errors.New("forbidden")
	}

	originFacility, err := h.workflowData.GetFacility(c.Request.Context(), deliveryOrder.OriginFacilityID)
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}

	items, err := h.workflowData.ListDOItemsByDO(c.Request.Context(), doID)
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}

	cache := newDashboardLookupCache()
	destination := h.resolveDestinationLabel(c.Request.Context(), cache, deliveryOrder.DestinationType, deliveryOrder.DestinationStationID, deliveryOrder.DestinationFacilityID)
	assignedVehicle := "Unassigned"
	if deliveryOrder.AssignedVehicleID.Valid {
		assignedVehicle = h.resolveVehicleLabel(c.Request.Context(), cache, deliveryOrder.AssignedVehicleID.Int64)
	}
	assignedDriver := "Unassigned"
	if deliveryOrder.AssignedDriverID.Valid {
		assignedDriver = h.resolveDriverLabel(c.Request.Context(), cache, deliveryOrder.AssignedDriverID.Int64)
	}

	var tripURL string
	if trip, tripErr := h.workflowData.GetTripByDO(c.Request.Context(), doID); tripErr == nil {
		tripURL = fmt.Sprintf("/trips/%d", trip.ID)
	}

	view := &dashboardDeliveryOrderDetailView{
		ID:               deliveryOrder.ID,
		OriginFacilityID: deliveryOrder.OriginFacilityID,
		DoNumber:         deliveryOrder.DoNumber,
		Status:           humanizeEnum(string(deliveryOrder.Status)),
		StatusTone:       statusTone(string(deliveryOrder.Status)),
		OriginFacility:   originFacility.Name,
		Destination:      destination,
		ScheduledDate:    formatDate(deliveryOrder.ScheduledDate),
		RaisedBy:         fmt.Sprintf("User #%d", deliveryOrder.RaisedBy),
		ApprovedAt:       formatTimestamptz(deliveryOrder.ApprovedAt, "Awaiting approval"),
		AssignedVehicle:  assignedVehicle,
		AssignedDriver:   assignedDriver,
		AssignedAt:       formatTimestamptz(deliveryOrder.AssignedAt, "Not assigned"),
		Notes:            formatText(deliveryOrder.Notes, "No notes recorded."),
		CanApprove:       deliveryOrder.Status == db.DoStatusTPENDINGAPPROVAL,
		CanAssign:        deliveryOrder.Status == db.DoStatusTAPPROVED || deliveryOrder.Status == db.DoStatusTASSIGNED,
		ActionError:      message,
		ApproveURL:       fmt.Sprintf("/dashboard-partials/delivery-orders/%d/approve-detail", deliveryOrder.ID),
		AssignURL:        fmt.Sprintf("/dashboard-partials/delivery-orders/%d/assign", deliveryOrder.ID),
		DetailFragment:   fmt.Sprintf("/dashboard-partials/delivery-orders/%d/detail", deliveryOrder.ID),
		TripURL:          tripURL,
	}

	itemViews := make([]dashboardDeliveryOrderItemView, 0, len(items))
	for _, item := range items {
		compartment := "Any compartment"
		if item.CompartmentID.Valid {
			compartment = fmt.Sprintf("Compartment #%d", item.CompartmentID.Int64)
		}
		itemViews = append(itemViews, dashboardDeliveryOrderItemView{
			FuelType:        item.FuelTypeCode,
			Compartment:     compartment,
			RequestedVolume: formatNumeric(item.RequestedVolumeL, " L"),
			AllocatedVolume: formatNumeric(item.AllocatedVolumeL, " L"),
		})
	}

	vehicleOptions, driverOptions := h.buildAssignmentOptions(c.Request.Context(), deliveryOrder.OriginFacilityID)

	return h.pageData(c, session, dashboardPageData{
		Title:              deliveryOrder.DoNumber,
		Heading:            deliveryOrder.DoNumber,
		Description:        "Use the existing approval and assignment rules without leaving the dashboard.",
		Breadcrumbs:        []DashboardBreadcrumb{{Label: "Delivery Orders", URL: "/delivery-orders"}, {Label: originFacility.Name, URL: fmt.Sprintf("/facilities/%d/delivery-orders", originFacility.ID)}, {Label: deliveryOrder.DoNumber, URL: c.Request.URL.Path}},
		DeliveryOrder:      view,
		DeliveryOrderItems: itemViews,
		VehicleOptions:     vehicleOptions,
		DriverOptions:      driverOptions,
	}), http.StatusOK, nil
}

func (h *DashboardHandler) buildTripRowsForSession(ctx context.Context, session *model.SessionData) ([]dashboardTripRow, string, string, error) {
	if h.workflowData == nil {
		return nil, "", "", errors.New("not found")
	}

	best := dashboardBestRole(session.RoleGrants)
	switch {
	case best.Role == "SYSTEM_ADMIN":
		rows, err := h.workflowData.ListActiveTrips(ctx)
		if err != nil {
			return nil, "", "", err
		}
		out := make([]dashboardTripRow, 0, len(rows))
		for _, row := range rows {
			out = append(out, buildTripRow(row.ID, row.Status, row.PlateNumber, row.DriverName, formatDestinationName(row.DestinationName), row.DepartedAt))
		}
		return out, "Active Trips", "System-wide view of active trips and their current workflow state.", nil
	case best.Role == "REFINERY_ADMIN" && best.ScopeType == "REFINERY" && best.ScopeID != nil:
		rows, err := h.workflowData.ListActiveTripsByRefineryScope(ctx, *best.ScopeID)
		if err != nil {
			return nil, "", "", err
		}
		out := make([]dashboardTripRow, 0, len(rows))
		for _, row := range rows {
			out = append(out, buildTripRow(row.ID, row.Status, row.PlateNumber, row.DriverName, formatDestinationName(row.DestinationName), row.DepartedAt))
		}
		return out, "Refinery Trips", "Trips currently loading, moving, or unloading across the refinery scope.", nil
	case (best.Role == "FACILITY_MANAGER" || best.Role == "FACILITY_OPERATOR") && best.ScopeType == "FACILITY" && best.ScopeID != nil:
		facility, err := h.workflowData.GetFacility(ctx, *best.ScopeID)
		if err != nil {
			return nil, "", "", err
		}
		rows, err := h.workflowData.ListActiveTripsByFacilityScope(ctx, *best.ScopeID)
		if err != nil {
			return nil, "", "", err
		}
		out := make([]dashboardTripRow, 0, len(rows))
		for _, row := range rows {
			out = append(out, buildTripRow(row.ID, row.Status, row.PlateNumber, row.DriverName, formatDestinationName(row.DestinationName), row.DepartedAt))
		}
		return out, facility.Name + " Trips", "Facility-scoped trip list for dispatch and arrival monitoring.", nil
	case best.Role == "STATION_MANAGER" && best.ScopeType == "STATION" && best.ScopeID != nil:
		station, err := h.workflowData.GetStation(ctx, *best.ScopeID)
		if err != nil {
			return nil, "", "", err
		}
		rows, err := h.workflowData.ListActiveTripsByStationScope(ctx, pgtype.Int8{Int64: *best.ScopeID, Valid: true})
		if err != nil {
			return nil, "", "", err
		}
		out := make([]dashboardTripRow, 0, len(rows))
		for _, row := range rows {
			out = append(out, buildTripRow(row.ID, row.Status, row.PlateNumber, row.DriverName, formatDestinationName(row.DestinationName), row.DepartedAt))
		}
		return out, station.Name + " Trips", "Trips arriving at this station with live operational context.", nil
	default:
		return nil, "", "", errDashboardForbidden
	}
}

func (h *DashboardHandler) buildTripDetailPageData(c *gin.Context, session *model.SessionData, tripID int64, message string) (dashboardPageData, int, error) {
	trip, status, err := h.loadTripForAccess(c.Request.Context(), session, tripID)
	if err != nil {
		return dashboardPageData{}, status, err
	}

	events, err := h.workflowData.ListTripEventsByTrip(c.Request.Context(), tripID)
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}
	seals, err := h.workflowData.ListSealsByTrip(c.Request.Context(), tripID)
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}
	readings, err := h.workflowData.ListWeightBridgeReadingsByTrip(c.Request.Context(), pgtype.Int8{Int64: tripID, Valid: true})
	if err != nil {
		return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}

	var photos []service.TripPhotoWithURL
	if h.tripPhotos != nil {
		photos, err = h.tripPhotos.ListTripPhotosWithURLs(c.Request.Context(), tripID)
		if err != nil {
			return dashboardPageData{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
		}
	}

	view := &dashboardTripDetailView{
		ID:                   trip.ID,
		DoID:                 trip.DoID,
		DoURL:                fmt.Sprintf("/delivery-orders/%d", trip.DoID),
		Status:               humanizeEnum(string(trip.Status)),
		StatusTone:           statusTone(string(trip.Status)),
		PlateNumber:          trip.PlateNumber,
		DriverName:           trip.DriverName,
		DriverTelegram:       formatInt8(trip.DriverTelegramID, "Not linked"),
		OriginFacility:       trip.OriginFacilityName,
		Destination:          tripDestinationLabel(trip),
		DepartedAt:           formatTimestamptz(trip.DepartedAt, "Not departed"),
		ArrivedAt:            formatTimestamptz(trip.ArrivedAt, "Not arrived"),
		CompletedAt:          formatTimestamptz(trip.CompletedAt, "Not completed"),
		WeightBridgeEntryURL: fmt.Sprintf("/trips/%d/weight-bridge/new", trip.ID),
	}

	eventViews := make([]dashboardTripEventView, 0, len(events))
	for _, event := range events {
		payload := "-"
		if len(event.Payload) > 0 {
			var compact bytes.Buffer
			if compactErr := json.Compact(&compact, event.Payload); compactErr == nil {
				payload = compact.String()
			} else {
				payload = string(event.Payload)
			}
		}
		eventViews = append(eventViews, dashboardTripEventView{
			EventType: humanizeEnum(string(event.EventType)),
			Timestamp: formatTimestamptz(event.EventTimestamp, "-"),
			Actor:     formatInt8(event.ActorUserID, "System"),
			Payload:   payload,
		})
	}

	weightBridgeViews := make([]dashboardWeightBridgeReadingView, 0, len(readings))
	for _, reading := range readings {
		weightBridgeViews = append(weightBridgeViews, dashboardWeightBridgeReadingView{
			ID:             reading.ID,
			ReadingType:    humanizeEnum(reading.ReadingType),
			Weight:         formatNumeric(reading.WeightKg, " kg"),
			Method:         humanizeEnum(string(reading.Method)),
			Temperature:    formatNumeric(reading.AmbientTempCelsius, " C"),
			ApprovalStatus: humanizeEnum(string(reading.ApprovalStatus)),
			ApprovalTone:   statusTone(string(reading.ApprovalStatus)),
			RecordedBy:     fmt.Sprintf("User #%d", reading.RecordedBy),
			ApprovedAt:     formatTimestamptz(reading.ApprovedAt, "Awaiting review"),
			Notes:          formatText(reading.Notes, "No notes recorded."),
			CreatedAt:      formatTimestamptz(reading.CreatedAt, "-"),
		})
	}

	sealViews := make([]dashboardTripSealView, 0, len(seals))
	for _, seal := range seals {
		verification := "Pending"
		if seal.VerificationStatus.Valid {
			verification = humanizeEnum(string(seal.VerificationStatus.SealStatusT))
		}
		sealViews = append(sealViews, dashboardTripSealView{
			Compartment:  fmt.Sprintf("Compartment %d", seal.CompartmentNumber),
			IssuedSeal:   seal.SealNumberIssued,
			IssuedBy:     seal.IssuedByName,
			IssuedAt:     formatTimestamptz(seal.IssuedAt, "-"),
			VerifiedSeal: formatText(seal.SealNumberVerified, "Not verified"),
			VerifiedBy:   formatText(seal.VerifiedByName, "Not verified"),
			VerifiedAt:   formatTimestamptz(seal.VerifiedAt, "Not verified"),
			Verification: verification,
			Notes:        formatText(seal.Notes, "No notes recorded."),
		})
	}

	photoViews := make([]dashboardTripPhotoView, 0, len(photos))
	for _, photo := range photos {
		compartment := "Whole trip"
		if photo.CompartmentID.Valid {
			compartment = fmt.Sprintf("Compartment #%d", photo.CompartmentID.Int64)
		}
		photoViews = append(photoViews, dashboardTripPhotoView{
			EventType:   humanizeEnum(string(photo.EventType)),
			Compartment: compartment,
			TakenAt:     formatTimestamptz(photo.TakenAt, "-"),
			URL:         photo.PresignedGetURL,
			Notes:       formatText(photo.Notes, "No notes recorded."),
		})
	}

	description := "Track trip status, timeline events, weight bridge state, seal verification, and photo evidence."
	if message != "" {
		description = message
	}

	return h.pageData(c, session, dashboardPageData{
		Title:        fmt.Sprintf("Trip %d", trip.ID),
		Heading:      fmt.Sprintf("Trip %d", trip.ID),
		Description:  description,
		Breadcrumbs:  []DashboardBreadcrumb{{Label: "Trips", URL: "/trips"}, {Label: fmt.Sprintf("Trip %d", trip.ID), URL: c.Request.URL.Path}},
		Trip:         view,
		TripEvents:   eventViews,
		WeightBridge: weightBridgeViews,
		Seals:        sealViews,
		Photos:       photoViews,
	}), http.StatusOK, nil
}

func (h *DashboardHandler) buildWeightBridgeEntryPageData(c *gin.Context, session *model.SessionData, tripID int64, message string) (dashboardPageData, int, error) {
	trip, status, err := h.loadTripForAccess(c.Request.Context(), session, tripID)
	if err != nil {
		return dashboardPageData{}, status, err
	}

	form := &dashboardWeightBridgeFormView{
		TripID:     trip.ID,
		TripLabel:  fmt.Sprintf("%s to %s", trip.PlateNumber, tripDestinationLabel(trip)),
		Action:     fmt.Sprintf("/trips/%d/weight-bridge", trip.ID),
		ErrorLabel: message,
	}

	return h.pageData(c, session, dashboardPageData{
		Title:       "Weight Bridge Entry",
		Heading:     "Weight Bridge Entry",
		Description: "Record a new tare or gross reading using the existing trip and approval workflow.",
		Breadcrumbs: []DashboardBreadcrumb{{Label: "Trips", URL: "/trips"}, {Label: fmt.Sprintf("Trip %d", trip.ID), URL: fmt.Sprintf("/trips/%d", trip.ID)}, {Label: "Weight Bridge Entry", URL: c.Request.URL.Path}},
		Trip: &dashboardTripDetailView{
			ID:          trip.ID,
			PlateNumber: trip.PlateNumber,
			Destination: tripDestinationLabel(trip),
			DriverName:  trip.DriverName,
		},
		WeightBridgeForm: form,
	}), http.StatusOK, nil
}

func (h *DashboardHandler) loadTripForAccess(ctx context.Context, session *model.SessionData, tripID int64) (db.GetTripWithDetailsRow, int, error) {
	if h.workflowData == nil {
		return db.GetTripWithDetailsRow{}, http.StatusNotFound, errors.New("not found")
	}

	trip, err := h.workflowData.GetTripWithDetails(ctx, tripID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return db.GetTripWithDetailsRow{}, http.StatusNotFound, errors.New("trip not found")
		}
		return db.GetTripWithDetailsRow{}, http.StatusInternalServerError, errors.New("dashboard data unavailable")
	}
	if !canViewTripDetail(session.RoleGrants, trip) {
		return db.GetTripWithDetailsRow{}, http.StatusForbidden, errors.New("forbidden")
	}
	return trip, http.StatusOK, nil
}

func (h *DashboardHandler) buildAssignmentOptions(ctx context.Context, facilityID int64) ([]dashboardSelectOption, []dashboardSelectOption) {
	if h.workflowData == nil {
		return nil, nil
	}

	vehicles, err := h.workflowData.ListVehiclesByStatusAndFacility(ctx, db.ListVehiclesByStatusAndFacilityParams{
		Status:            db.VehicleStatusTAVAILABLE,
		PrimaryFacilityID: facilityID,
	})
	if err != nil {
		return nil, nil
	}

	vehicleOptions := make([]dashboardSelectOption, 0, len(vehicles))
	depotIDs := make(map[int64]struct{})
	for _, vehicle := range vehicles {
		hint := formatNumeric(vehicle.TotalCapacityL, " L")
		if vehicle.Model.Valid {
			hint = fmt.Sprintf("%s, %s", vehicle.Model.String, hint)
		}
		vehicleOptions = append(vehicleOptions, dashboardSelectOption{
			ID:    vehicle.ID,
			Label: vehicle.PlateNumber,
			Hint:  hint,
		})
		if vehicle.CurrentDepotID.Valid {
			depotIDs[vehicle.CurrentDepotID.Int64] = struct{}{}
		}
	}

	driverMap := make(map[int64]dashboardSelectOption)
	for depotID := range depotIDs {
		drivers, err := h.workflowData.ListDriversByDepot(ctx, pgtype.Int8{Int64: depotID, Valid: true})
		if err != nil {
			continue
		}
		for _, driver := range drivers {
			hint := formatDate(driver.SimB2Expiry)
			if driver.IsOnShift {
				hint = "On shift, SIM expires " + hint
			} else {
				hint = "Off shift, SIM expires " + hint
			}
			driverMap[driver.ID] = dashboardSelectOption{
				ID:    driver.ID,
				Label: driver.FullName,
				Hint:  hint,
			}
		}
	}

	driverOptions := make([]dashboardSelectOption, 0, len(driverMap))
	for _, option := range driverMap {
		driverOptions = append(driverOptions, option)
	}
	sort.Slice(driverOptions, func(i, j int) bool {
		return driverOptions[i].Label < driverOptions[j].Label
	})

	return vehicleOptions, driverOptions
}

func (h *DashboardHandler) buildDeliveryOrderRow(ctx context.Context, item db.DeliveryOrder, cache *dashboardLookupCache) dashboardDeliveryOrderRow {
	assignment := "Unassigned"
	if item.AssignedVehicleID.Valid || item.AssignedDriverID.Valid {
		parts := make([]string, 0, 2)
		if item.AssignedVehicleID.Valid {
			parts = append(parts, h.resolveVehicleLabel(ctx, cache, item.AssignedVehicleID.Int64))
		}
		if item.AssignedDriverID.Valid {
			parts = append(parts, h.resolveDriverLabel(ctx, cache, item.AssignedDriverID.Int64))
		}
		assignment = strings.Join(parts, " / ")
	}

	return dashboardDeliveryOrderRow{
		ID:            item.ID,
		DoNumber:      item.DoNumber,
		Status:        humanizeEnum(string(item.Status)),
		StatusTone:    statusTone(string(item.Status)),
		Destination:   h.resolveDestinationLabel(ctx, cache, item.DestinationType, item.DestinationStationID, item.DestinationFacilityID),
		ScheduledDate: formatDate(item.ScheduledDate),
		Assignment:    assignment,
		DetailURL:     fmt.Sprintf("/delivery-orders/%d", item.ID),
		CanApprove:    item.Status == db.DoStatusTPENDINGAPPROVAL,
		CanAssign:     item.Status == db.DoStatusTAPPROVED || item.Status == db.DoStatusTASSIGNED,
	}
}

func buildTripRow(id int64, status db.TripStatusT, plateNumber, driverName, destination string, departedAt pgtype.Timestamptz) dashboardTripRow {
	return dashboardTripRow{
		ID:          id,
		Status:      humanizeEnum(string(status)),
		StatusTone:  statusTone(string(status)),
		PlateNumber: plateNumber,
		DriverName:  driverName,
		Destination: destination,
		DepartedAt:  formatTimestamptz(departedAt, "Awaiting departure"),
		DetailURL:   fmt.Sprintf("/trips/%d", id),
	}
}

func newDashboardLookupCache() *dashboardLookupCache {
	return &dashboardLookupCache{
		facilities: make(map[int64]string),
		stations:   make(map[int64]string),
		vehicles:   make(map[int64]string),
		drivers:    make(map[int64]string),
	}
}

func (h *DashboardHandler) resolveDestinationLabel(ctx context.Context, cache *dashboardLookupCache, destinationType db.DestinationTypeT, stationID, facilityID pgtype.Int8) string {
	switch destinationType {
	case db.DestinationTypeTSTATION:
		if stationID.Valid {
			return h.resolveStationLabel(ctx, cache, stationID.Int64)
		}
	case db.DestinationTypeTREFINERYFACILITY:
		if facilityID.Valid {
			return h.resolveFacilityLabel(ctx, cache, facilityID.Int64)
		}
	}
	return "Destination pending"
}

func (h *DashboardHandler) resolveFacilityLabel(ctx context.Context, cache *dashboardLookupCache, facilityID int64) string {
	if label, ok := cache.facilities[facilityID]; ok {
		return label
	}
	label := fmt.Sprintf("Facility #%d", facilityID)
	if h.workflowData != nil {
		if facility, err := h.workflowData.GetFacility(ctx, facilityID); err == nil {
			label = facility.Name
		}
	} else if h.queries != nil {
		if facility, err := h.queries.GetFacility(ctx, facilityID); err == nil {
			label = facility.Name
		}
	}
	cache.facilities[facilityID] = label
	return label
}

func (h *DashboardHandler) resolveStationLabel(ctx context.Context, cache *dashboardLookupCache, stationID int64) string {
	if label, ok := cache.stations[stationID]; ok {
		return label
	}
	label := fmt.Sprintf("Station #%d", stationID)
	if h.workflowData != nil {
		if station, err := h.workflowData.GetStation(ctx, stationID); err == nil {
			label = station.Name
		}
	} else if h.queries != nil {
		if station, err := h.queries.GetStation(ctx, stationID); err == nil {
			label = station.Name
		}
	}
	cache.stations[stationID] = label
	return label
}

func (h *DashboardHandler) resolveVehicleLabel(ctx context.Context, cache *dashboardLookupCache, vehicleID int64) string {
	if label, ok := cache.vehicles[vehicleID]; ok {
		return label
	}
	label := fmt.Sprintf("Vehicle #%d", vehicleID)
	if h.workflowData != nil {
		if vehicle, err := h.workflowData.GetVehicle(ctx, vehicleID); err == nil {
			label = vehicle.PlateNumber
		}
	} else if h.queries != nil {
		if vehicle, err := h.queries.GetVehicle(ctx, vehicleID); err == nil {
			label = vehicle.PlateNumber
		}
	}
	cache.vehicles[vehicleID] = label
	return label
}

func (h *DashboardHandler) resolveDriverLabel(ctx context.Context, cache *dashboardLookupCache, driverID int64) string {
	if label, ok := cache.drivers[driverID]; ok {
		return label
	}
	label := fmt.Sprintf("Driver #%d", driverID)
	if h.workflowData != nil {
		if driver, err := h.workflowData.GetDriver(ctx, driverID); err == nil {
			label = driver.FullName
		}
	}
	cache.drivers[driverID] = label
	return label
}

func dashboardBestRole(roles []model.RoleGrant) model.RoleGrant {
	bestRank := 0
	var best model.RoleGrant
	for _, role := range roles {
		if role.Role == "SYSTEM_ADMIN" {
			return role
		}
		if rank := middleware.RoleRank(role.Role); rank > bestRank {
			bestRank = rank
			best = role
		}
	}
	return best
}

func dashboardFacilityScope(roles []model.RoleGrant) (int64, bool) {
	for _, role := range roles {
		if (role.Role == "FACILITY_MANAGER" || role.Role == "FACILITY_OPERATOR") && role.ScopeType == "FACILITY" && role.ScopeID != nil {
			return *role.ScopeID, true
		}
	}
	return 0, false
}

func canViewDeliveryOrderPages(roles []model.RoleGrant) bool {
	for _, role := range roles {
		switch role.Role {
		case "SYSTEM_ADMIN", "REFINERY_ADMIN", "FACILITY_MANAGER", "FACILITY_OPERATOR":
			return true
		}
	}
	return false
}

func canViewTripPages(roles []model.RoleGrant) bool {
	for _, role := range roles {
		switch role.Role {
		case "SYSTEM_ADMIN", "REFINERY_ADMIN", "FACILITY_MANAGER", "FACILITY_OPERATOR", "STATION_MANAGER":
			return true
		}
	}
	return false
}

func canViewTripDetail(roles []model.RoleGrant, trip db.GetTripWithDetailsRow) bool {
	if canViewFacility(roles, trip.OriginFacilityID) {
		return true
	}
	if trip.DestinationStationID.Valid && canViewStation(roles, trip.DestinationStationID.Int64) {
		return true
	}
	return false
}

func formatTimestamptz(value pgtype.Timestamptz, fallback string) string {
	if !value.Valid {
		return fallback
	}
	return value.Time.UTC().Format("02 Jan 2006 15:04 MST")
}

func formatDate(value pgtype.Date) string {
	if !value.Valid {
		return "-"
	}
	return value.Time.Format("02 Jan 2006")
}

func formatNumeric(value pgtype.Numeric, suffix string) string {
	floatValue, ok := numericToFloat64(value)
	if !ok || floatValue == nil {
		return "-"
	}

	text := strconv.FormatFloat(*floatValue, 'f', 2, 64)
	text = strings.TrimRight(strings.TrimRight(text, "0"), ".")
	return text + suffix
}

func formatText(value any, fallback string) string {
	switch v := any(value).(type) {
	case pgtype.Text:
		if v.Valid && strings.TrimSpace(v.String) != "" {
			return v.String
		}
	case string:
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return fallback
}

func formatInt8(value pgtype.Int8, fallback string) string {
	if !value.Valid {
		return fallback
	}
	return strconv.FormatInt(value.Int64, 10)
}

func humanizeEnum(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	parts := strings.Split(strings.ToLower(strings.ReplaceAll(value, "_", " ")), " ")
	for i, part := range parts {
		if part == "" {
			continue
		}
		parts[i] = strings.ToUpper(part[:1]) + part[1:]
	}
	return strings.Join(parts, " ")
}

func statusTone(status string) string {
	switch status {
	case "PENDING_APPROVAL", "PENDING", "ESCALATED", "LOADING", "IN_TRANSIT", "ARRIVED", "UNLOADING", "UNDER_MAINTENANCE":
		return "amber"
	case "APPROVED", "ASSIGNED", "AVAILABLE", "LOADED", "DELIVERED", "RECONCILED", "CLOSED", "COMPLETED":
		return "emerald"
	case "CANCELLED", "DECOMMISSIONED", "DISPUTED", "REJECTED":
		return "rose"
	default:
		return "slate"
	}
}

func formatDestinationName(name pgtype.Text) string {
	if name.Valid && strings.TrimSpace(name.String) != "" {
		return name.String
	}
	return "Destination pending"
}

func tripDestinationLabel(trip db.GetTripWithDetailsRow) string {
	if trip.DestinationStationName.Valid && strings.TrimSpace(trip.DestinationStationName.String) != "" {
		return trip.DestinationStationName.String
	}
	if trip.DestinationFacilityID.Valid {
		return fmt.Sprintf("Facility #%d", trip.DestinationFacilityID.Int64)
	}
	return "Destination pending"
}
