package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"net/http"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/middleware"
	"github.com/adevsh/petrosync/internal/model"
	"github.com/adevsh/petrosync/internal/service"
)

const dashboardSessionCookie = "petrosync_session"

var (
	_, dashboardHandlerFile, _, _ = runtime.Caller(0)
	dashboardTemplateRoot         = filepath.Clean(filepath.Join(filepath.Dir(dashboardHandlerFile), "..", "..", "templates"))
)

type dashboardAuthService interface {
	LoginDashboard(ctx context.Context, username, password string, sessionTTL time.Duration) (*service.DashboardLoginResult, error)
	ChangePassword(ctx context.Context, userID int64, currentPassword, newPassword string) error
}

type dashboardSessionStore interface {
	GetSession(ctx context.Context, sessionID string) (*model.SessionData, error)
	SaveSession(ctx context.Context, sessionID string, data model.SessionData) error
	DeleteSession(ctx context.Context, sessionID string) error
}

type dashboardDataQuerier interface {
	GetCompanyWideDashboardSummary(ctx context.Context) ([]db.GetCompanyWideDashboardSummaryRow, error)
	GetFacilityDashboardSummary(ctx context.Context, id int64) (db.GetFacilityDashboardSummaryRow, error)
	ListActiveTripsWithLatestGPS(ctx context.Context) ([]db.ListActiveTripsWithLatestGPSRow, error)
	ListAllActiveDepots(ctx context.Context) ([]db.ListAllActiveDepotsRow, error)
	GetFacility(ctx context.Context, id int64) (db.GetFacilityRow, error)
	GetDepot(ctx context.Context, id int64) (db.GetDepotRow, error)
	ListFacilitiesByRefinery(ctx context.Context, refineryID int64) ([]db.ListFacilitiesByRefineryRow, error)
	ListRefineries(ctx context.Context) ([]db.Refinery, error)
	GetStation(ctx context.Context, id int64) (db.GetStationRow, error)
	GetVehicle(ctx context.Context, id int64) (db.GetVehicleRow, error)
	GetStationInventorySnapshot(ctx context.Context, stationID int64) ([]db.GetStationInventorySnapshotRow, error)
	ListAllActiveStations(ctx context.Context) ([]db.ListAllActiveStationsRow, error)
	ListAllActiveStationsByRefineryScope(ctx context.Context, refineryID int64) ([]db.ListAllActiveStationsByRefineryScopeRow, error)
	ListAllActiveStationsByStationScope(ctx context.Context, id int64) ([]db.ListAllActiveStationsByStationScopeRow, error)
	ListAllCompartmentsByVehicle(ctx context.Context, vehicleID int64) ([]db.VehicleCompartment, error)
	ListAllOpenMaintenance(ctx context.Context) ([]db.ListAllOpenMaintenanceRow, error)
	ListMaintenanceByVehicle(ctx context.Context, vehicleID int64) ([]db.VehicleMaintenanceRecord, error)
	ListStationTanksBelowReorderThreshold(ctx context.Context) ([]db.ListStationTanksBelowReorderThresholdRow, error)
	ListStationsServedByFacility(ctx context.Context, facilityID int64) ([]db.ListStationsServedByFacilityRow, error)
	ListTripsByStatus(ctx context.Context, status db.TripStatusT) ([]db.Trip, error)
	ListTripsByVehicle(ctx context.Context, arg db.ListTripsByVehicleParams) ([]db.Trip, error)
	ListVehiclesByStatus(ctx context.Context, status db.VehicleStatusT) ([]db.ListVehiclesByStatusRow, error)
	ListVehiclesByStatusAndDepot(ctx context.Context, arg db.ListVehiclesByStatusAndDepotParams) ([]db.ListVehiclesByStatusAndDepotRow, error)
	ListVehiclesByStatusAndFacility(ctx context.Context, arg db.ListVehiclesByStatusAndFacilityParams) ([]db.ListVehiclesByStatusAndFacilityRow, error)
	ListVehiclesByStatusAndRefinery(ctx context.Context, arg db.ListVehiclesByStatusAndRefineryParams) ([]db.ListVehiclesByStatusAndRefineryRow, error)
	ListVehiclesWithMaintenanceOrExpiryDue(ctx context.Context) ([]db.ListVehiclesWithMaintenanceOrExpiryDueRow, error)
}

// DashboardHandler serves the server-rendered dashboard entry pages.
type DashboardHandler struct {
	auth          dashboardAuthService
	sessions      dashboardSessionStore
	queries       dashboardDataQuerier
	workflowData  dashboardWorkflowQuerier
	workflow      dashboardWorkflowService
	tripPhotos    dashboardTripPhotoLister
	notifications dashboardNotificationService
	sessionTTL    time.Duration
	secureCookie  bool
	userAdmin     *UserHandler
	resetPw       *ResetPasswordHandler
}

// DashboardBreadcrumb renders the breadcrumb trail in the shared layout.
type DashboardBreadcrumb struct {
	Label string
	URL   string
}

type dashboardNavItem struct {
	Label string
	URL   string
}

type dashboardMetric struct {
	Label string
	Value string
	Help  string
}

type dashboardCompanyFacilitySummary struct {
	FacilityID        int64
	FacilityCode      string
	FacilityName      string
	RefineryCode      string
	ActiveTrips       int64
	AvailableVehicles int64
}

type dashboardFacilitySummary struct {
	FacilityID            int64
	FacilityName          string
	ActiveTrips           int64
	AvailableVehicles     int64
	VehiclesInMaintenance int64
}

type dashboardActiveTrip struct {
	ID               int64
	Status           string
	PlateNumber      string
	DriverName       string
	DestinationName  string
	OriginFacilityID int64
	HasLocation      bool
	Latitude         float64
	Longitude        float64
	SpeedLabel       string
	LastGpsLabel     string
}

type dashboardLiveMap struct {
	Scope        string
	WSPath       string
	SeedJSON     template.JS
	EmptyMessage string
}

type dashboardScopeReference struct {
	ID     int64
	Label  string
	Detail string
}

type dashboardScopeReferenceSection struct {
	ScopeType string
	Items     []dashboardScopeReference
}

type dashboardPageData struct {
	Title              string
	CurrentPath        string
	Authenticated      bool
	Session            *model.SessionData
	Breadcrumbs        []DashboardBreadcrumb
	NavItems           []dashboardNavItem
	FormError          string
	Username           string
	Heading            string
	Description        string
	Users              []userResponse
	SelectedUser       *userResponse
	RoleGrants         []roleGrantResponse
	RoleOptions        []string
	ScopeOptions       []string
	ScopeReferences    []dashboardScopeReferenceSection
	Metrics            []dashboardMetric
	CompanyRows        []dashboardCompanyFacilitySummary
	Facility           *dashboardFacilitySummary
	ActiveTrips        []dashboardActiveTrip
	LiveMap            *dashboardLiveMap
	Stations           []dashboardStationRow
	Station            *dashboardStationDetailView
	StationTanks       []dashboardStationTankView
	StationDeliveries  []dashboardStationDeliveryView
	FleetVehicles      []dashboardFleetVehicleRow
	FleetMaintenance   []dashboardFleetMaintenanceView
	FleetAttention     []dashboardFleetAttentionView
	Vehicle            *dashboardVehicleDetailView
	VehicleTrips       []dashboardVehicleTripView
	VehicleMaintenance []dashboardVehicleMaintenanceView
	Compartments       []dashboardVehicleCompartmentView
	DeliveryOrders     []dashboardDeliveryOrderRow
	DeliveryOrder      *dashboardDeliveryOrderDetailView
	DeliveryOrderItems []dashboardDeliveryOrderItemView
	VehicleOptions     []dashboardSelectOption
	DriverOptions      []dashboardSelectOption
	Trips              []dashboardTripRow
	Trip               *dashboardTripDetailView
	TripEvents         []dashboardTripEventView
	WeightBridge       []dashboardWeightBridgeReadingView
	Seals              []dashboardTripSealView
	Photos             []dashboardTripPhotoView
	ScopeNotice        string
	WeightBridgeForm   *dashboardWeightBridgeFormView
}

// NewDashboardHandler creates a handler for dashboard pages and session flows.
func NewDashboardHandler(auth dashboardAuthService, sessions dashboardSessionStore, queries dashboardDataQuerier, sessionTTL time.Duration, secureCookie bool) *DashboardHandler {
	if sessionTTL <= 0 {
		sessionTTL = 8 * time.Hour
	}
	return &DashboardHandler{
		auth:         auth,
		sessions:     sessions,
		queries:      queries,
		sessionTTL:   sessionTTL,
		secureCookie: secureCookie,
	}
}

// WithUserAdmin mounts dashboard user-management pages and action endpoints.
func (h *DashboardHandler) WithUserAdmin(userAdmin *UserHandler, resetPw *ResetPasswordHandler) *DashboardHandler {
	h.userAdmin = userAdmin
	h.resetPw = resetPw
	return h
}

// WithWorkflowPages mounts dashboard workflow pages on existing trip, DO, and photo services.
func (h *DashboardHandler) WithWorkflowPages(workflowData dashboardWorkflowQuerier, workflow dashboardWorkflowService, tripPhotos dashboardTripPhotoLister, notifications dashboardNotificationService) *DashboardHandler {
	h.workflowData = workflowData
	h.workflow = workflow
	h.tripPhotos = tripPhotos
	h.notifications = notifications
	return h
}

// RegisterDashboardRoutes mounts the server-rendered dashboard entry routes.
func RegisterDashboardRoutes(router *gin.Engine, dashboard *DashboardHandler, sessions dashboardSessionStore) {
	router.Static("/static", "./static")

	public := router.Group("/")
	{
		public.GET("/login", dashboard.ShowLogin)
		public.POST("/login", dashboard.Login)
	}

	protected := router.Group("/")
	protected.Use(middleware.SessionPageAuth(sessions, "/login"))
	protected.Use(middleware.RequirePasswordChange("/change-password", "/logout"))
	{
		protected.POST("/logout", dashboard.Logout)
		protected.GET("/change-password", dashboard.ShowChangePassword)
		protected.POST("/change-password", dashboard.ChangePassword)
		protected.GET("/", dashboard.Home)
		protected.GET("/delivery-orders", dashboard.DeliveryOrders)
		protected.GET("/facilities/:id/delivery-orders", dashboard.FacilityDeliveryOrders)
		protected.GET("/delivery-orders/:id", dashboard.DeliveryOrderDetail)
		protected.GET("/facilities/:id", dashboard.FacilityLanding)
		protected.GET("/stations", dashboard.Stations)
		protected.GET("/stations/:id", dashboard.StationLanding)
		protected.GET("/fleet", dashboard.FleetLanding)
		protected.GET("/fleet/vehicles/:id", dashboard.VehicleDetail)
		protected.GET("/trips", dashboard.Trips)
		protected.GET("/trips/:id", dashboard.TripDetail)
		protected.GET("/trips/:id/weight-bridge/new", dashboard.ShowWeightBridgeEntry)
		protected.POST("/trips/:id/weight-bridge", dashboard.CreateWeightBridgeEntry)
	}

	partials := protected.Group("/dashboard-partials")
	{
		partials.GET("/facilities/:id/delivery-orders/table", dashboard.DeliveryOrderTable)
		partials.GET("/delivery-orders/:id/detail", dashboard.DeliveryOrderDetailFragment)
		partials.POST("/delivery-orders/:id/approve-row", dashboard.ApproveDeliveryOrderRow)
		partials.POST("/delivery-orders/:id/approve-detail", dashboard.ApproveDeliveryOrderDetail)
		partials.POST("/delivery-orders/:id/assign", dashboard.AssignDeliveryOrder)
	}

	if dashboard.userAdmin != nil {
		userPages := protected.Group("/users")
		userPages.Use(dashboard.requireUserAdminPage())
		{
			userPages.GET("", dashboard.UserList)
			userPages.GET("/new", dashboard.ShowUserCreate)
			userPages.GET("/:id", dashboard.UserDetail)
		}

		userAPI := protected.Group("/dashboard-api/users")
		userAPI.Use(middleware.RequiredRole(nil, "SYSTEM_ADMIN", "COMPANY", ""))
		{
			userAPI.GET("", dashboard.userAdmin.ListUsers)
			userAPI.POST("", dashboard.userAdmin.CreateUser)
			userAPI.GET("/:id", dashboard.userAdmin.GetUser)
			userAPI.GET("/:id/roles", dashboard.userAdmin.ListRoles)
			userAPI.POST("/:id/roles", dashboard.userAdmin.GrantRole)
			userAPI.DELETE("/:id/roles", dashboard.userAdmin.RevokeRole)
			if dashboard.resetPw != nil {
				userAPI.POST("/:id/reset-password", dashboard.resetPw.ResetPassword)
			}
		}
	}
}

func (h *DashboardHandler) ShowLogin(c *gin.Context) {
	if session := currentDashboardSession(c); session != nil {
		c.Redirect(http.StatusSeeOther, dashboardLoginRedirectPath(session))
		return
	}
	if session, _ := h.sessionFromCookie(c); session != nil {
		c.Redirect(http.StatusSeeOther, dashboardLoginRedirectPath(session))
		return
	}

	h.render(c, http.StatusOK, "login.html", dashboardPageData{
		Title:       "Sign In",
		CurrentPath: c.Request.URL.Path,
		Heading:     "PetroSync Dashboard",
		Description: "Sign in with your PetroSync account to continue.",
	})
}

func (h *DashboardHandler) Login(c *gin.Context) {
	username := strings.TrimSpace(c.PostForm("username"))
	password := c.PostForm("password")
	if username == "" || password == "" {
		h.render(c, http.StatusBadRequest, "login.html", dashboardPageData{
			Title:       "Sign In",
			CurrentPath: c.Request.URL.Path,
			Heading:     "PetroSync Dashboard",
			Description: "Sign in with your PetroSync account to continue.",
			FormError:   "username and password are required",
			Username:    username,
		})
		return
	}

	result, err := h.auth.LoginDashboard(c.Request.Context(), username, password, h.sessionTTL)
	if err != nil {
		status := http.StatusInternalServerError
		message := "authentication failed"
		switch {
		case errors.Is(err, service.ErrInvalidCredentials):
			status = http.StatusUnauthorized
			message = "invalid username or password"
		case errors.Is(err, service.ErrUserInactive):
			status = http.StatusForbidden
			message = "user account is inactive"
		}

		h.render(c, status, "login.html", dashboardPageData{
			Title:       "Sign In",
			CurrentPath: c.Request.URL.Path,
			Heading:     "PetroSync Dashboard",
			Description: "Sign in with your PetroSync account to continue.",
			FormError:   message,
			Username:    username,
		})
		return
	}

	h.setSessionCookie(c, result.SessionID)
	c.Redirect(http.StatusSeeOther, dashboardLoginRedirectPath(&result.Session))
}

func (h *DashboardHandler) Logout(c *gin.Context) {
	if sessionID := currentDashboardSessionID(c); sessionID != "" {
		_ = h.sessions.DeleteSession(c.Request.Context(), sessionID)
	}
	h.clearSessionCookie(c)
	c.Redirect(http.StatusSeeOther, "/login")
}

func (h *DashboardHandler) ShowChangePassword(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	h.render(c, http.StatusOK, "change_password.html", h.pageData(c, session, dashboardPageData{
		Title:       "Change Password",
		Heading:     "Change your password",
		Description: "This account must set a new password before dashboard access continues.",
		Breadcrumbs: []DashboardBreadcrumb{{Label: "Change Password", URL: "/change-password"}},
	}))
}

func (h *DashboardHandler) ChangePassword(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	currentPassword := c.PostForm("current_password")
	newPassword := c.PostForm("new_password")
	if currentPassword == "" || len(newPassword) < 8 {
		h.render(c, http.StatusBadRequest, "change_password.html", h.pageData(c, session, dashboardPageData{
			Title:       "Change Password",
			Heading:     "Change your password",
			Description: "This account must set a new password before dashboard access continues.",
			FormError:   "current password and a new password of at least 8 characters are required",
			Breadcrumbs: []DashboardBreadcrumb{{Label: "Change Password", URL: "/change-password"}},
		}))
		return
	}

	if err := h.auth.ChangePassword(c.Request.Context(), session.UserID, currentPassword, newPassword); err != nil {
		status := http.StatusInternalServerError
		message := "password change failed"
		if errors.Is(err, service.ErrPasswordMismatch) {
			status = http.StatusBadRequest
			message = "current password is incorrect"
		}

		h.render(c, status, "change_password.html", h.pageData(c, session, dashboardPageData{
			Title:       "Change Password",
			Heading:     "Change your password",
			Description: "This account must set a new password before dashboard access continues.",
			FormError:   message,
			Breadcrumbs: []DashboardBreadcrumb{{Label: "Change Password", URL: "/change-password"}},
		}))
		return
	}

	updatedSession := *session
	updatedSession.ForcePasswordChange = false
	if sessionID := currentDashboardSessionID(c); sessionID != "" {
		if err := h.sessions.SaveSession(c.Request.Context(), sessionID, updatedSession); err != nil {
			h.clearSessionCookie(c)
			c.Redirect(http.StatusSeeOther, "/login")
			return
		}
	}

	c.Redirect(http.StatusSeeOther, dashboardLandingPath(updatedSession.RoleGrants))
}

func (h *DashboardHandler) Home(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	landingPath := dashboardLandingPath(session.RoleGrants)
	if landingPath != "/" {
		c.Redirect(http.StatusSeeOther, landingPath)
		return
	}

	if h.queries == nil {
		c.String(http.StatusInternalServerError, "dashboard data unavailable")
		return
	}

	companySummary, err := h.queries.GetCompanyWideDashboardSummary(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "dashboard data unavailable")
		return
	}
	activeTrips, err := h.queries.ListActiveTripsWithLatestGPS(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "dashboard data unavailable")
		return
	}
	tripRows, mapSeed, err := buildDashboardActiveTrips(activeTrips)
	if err != nil {
		c.String(http.StatusInternalServerError, "dashboard data unavailable")
		return
	}

	h.render(c, http.StatusOK, "home.html", h.pageData(c, session, dashboardPageData{
		Title:       "Operations Overview",
		Heading:     "Operations Overview",
		Description: "Company-wide snapshot for refinery administrators across facilities, vehicles, and active trips.",
		Breadcrumbs: []DashboardBreadcrumb{{Label: "Overview", URL: "/"}},
		Metrics:     buildCompanyMetrics(companySummary),
		CompanyRows: buildCompanySummaryRows(companySummary),
		ActiveTrips: tripRows,
		LiveMap: &dashboardLiveMap{
			Scope:        "company",
			WSPath:       "/ws/trips/active",
			SeedJSON:     template.JS(mapSeed),
			EmptyMessage: "No active trip positions are available yet.",
		},
	}))
}

func (h *DashboardHandler) FacilityLanding(c *gin.Context) {
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

	if h.queries == nil {
		c.String(http.StatusInternalServerError, "dashboard data unavailable")
		return
	}

	summary, err := h.queries.GetFacilityDashboardSummary(c.Request.Context(), facilityID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			c.String(http.StatusNotFound, "facility not found")
			return
		}
		c.String(http.StatusInternalServerError, "dashboard data unavailable")
		return
	}

	activeTrips, err := h.queries.ListActiveTripsWithLatestGPS(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "dashboard data unavailable")
		return
	}
	filteredTrips := filterDashboardTripsByFacility(activeTrips, facilityID)
	tripRows, mapSeed, err := buildDashboardActiveTrips(filteredTrips)
	if err != nil {
		c.String(http.StatusInternalServerError, "dashboard data unavailable")
		return
	}

	h.render(c, http.StatusOK, "facility.html", h.pageData(c, session, dashboardPageData{
		Title:       summary.FacilityName + " Dashboard",
		Heading:     summary.FacilityName,
		Description: "Scoped operational view for facility managers and operators.",
		Breadcrumbs: []DashboardBreadcrumb{
			{Label: summary.FacilityName, URL: c.Request.URL.Path},
		},
		Metrics: buildFacilityMetrics(summary),
		Facility: &dashboardFacilitySummary{
			FacilityID:            summary.FacilityID,
			FacilityName:          summary.FacilityName,
			ActiveTrips:           summary.ActiveTrips,
			AvailableVehicles:     summary.AvailableVehicles,
			VehiclesInMaintenance: summary.VehiclesInMaintenance,
		},
		ActiveTrips: tripRows,
		LiveMap: &dashboardLiveMap{
			Scope:        "facility",
			WSPath:       "/ws/trips/active",
			SeedJSON:     template.JS(mapSeed),
			EmptyMessage: "No live GPS points are available for this facility.",
		},
	}))
}

func (h *DashboardHandler) StationLanding(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	data, status, err := h.buildStationDetailPageData(c, session)
	if err != nil {
		c.String(status, err.Error())
		return
	}
	h.render(c, http.StatusOK, "station.html", data)
}

func (h *DashboardHandler) FleetLanding(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	data, status, err := h.buildFleetPageData(c, session)
	if err != nil {
		c.String(status, err.Error())
		return
	}
	h.render(c, http.StatusOK, "fleet.html", data)
}

func (h *DashboardHandler) pageData(c *gin.Context, session *model.SessionData, base dashboardPageData) dashboardPageData {
	base.CurrentPath = c.Request.URL.Path
	base.Authenticated = session != nil
	base.Session = session
	base.NavItems = dashboardNav(session)
	return base
}

func (h *DashboardHandler) render(c *gin.Context, status int, page string, data dashboardPageData) {
	h.renderTemplate(c, status, page, "page", data)
}

func (h *DashboardHandler) renderTemplate(c *gin.Context, status int, page, templateName string, data dashboardPageData) {
	tmpl, err := parseDashboardTemplate(page)
	if err != nil {
		c.String(http.StatusInternalServerError, "template error")
		return
	}

	c.Status(status)
	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.ExecuteTemplate(c.Writer, templateName, data); err != nil {
		c.String(http.StatusInternalServerError, "template error")
	}
}

func (h *DashboardHandler) setSessionCookie(c *gin.Context, sessionID string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(dashboardSessionCookie, sessionID, int(h.sessionTTL.Seconds()), "/", "", h.secureCookie, true)
}

func (h *DashboardHandler) clearSessionCookie(c *gin.Context) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(dashboardSessionCookie, "", -1, "/", "", h.secureCookie, true)
}

func (h *DashboardHandler) sessionFromCookie(c *gin.Context) (*model.SessionData, string) {
	sessionID, err := c.Cookie(dashboardSessionCookie)
	if err != nil {
		return nil, ""
	}
	session, err := h.sessions.GetSession(c.Request.Context(), sessionID)
	if err != nil || session == nil {
		return nil, ""
	}
	return session, sessionID
}

func parseDashboardTemplate(page string) (*template.Template, error) {
	partials, err := filepath.Glob(filepath.Join(dashboardTemplateRoot, "partials", "*.html"))
	if err != nil {
		return nil, err
	}

	files := []string{
		filepath.Join(dashboardTemplateRoot, "layout", "base.html"),
	}
	files = append(files, partials...)
	files = append(files, filepath.Join(dashboardTemplateRoot, "pages", page))

	return template.ParseFiles(files...)
}

func currentDashboardSession(c *gin.Context) *model.SessionData {
	sessionVal, exists := c.Get("session")
	if !exists {
		return nil
	}
	session, ok := sessionVal.(model.SessionData)
	if !ok {
		return nil
	}
	return &session
}

func currentDashboardSessionID(c *gin.Context) string {
	sessionID, _ := c.Get("session_id")
	value, _ := sessionID.(string)
	return value
}

func dashboardLoginRedirectPath(session *model.SessionData) string {
	if session == nil {
		return "/login"
	}
	if session.ForcePasswordChange {
		return "/change-password"
	}
	return dashboardLandingPath(session.RoleGrants)
}

func dashboardLandingPath(roles []model.RoleGrant) string {
	for _, role := range roles {
		if role.Role == "SYSTEM_ADMIN" || role.Role == "REFINERY_ADMIN" {
			return "/"
		}
	}
	for _, role := range roles {
		if (role.Role == "FACILITY_OPERATOR" || role.Role == "FACILITY_MANAGER") && role.ScopeType == "FACILITY" && role.ScopeID != nil {
			return "/facilities/" + strconv.FormatInt(*role.ScopeID, 10)
		}
	}
	for _, role := range roles {
		if role.Role == "STATION_MANAGER" && role.ScopeType == "STATION" && role.ScopeID != nil {
			return "/stations/" + strconv.FormatInt(*role.ScopeID, 10)
		}
	}
	for _, role := range roles {
		if role.Role == "DEPOT_STAFF" {
			return "/fleet"
		}
	}
	return "/"
}

func dashboardNav(session *model.SessionData) []dashboardNavItem {
	if session == nil {
		return nil
	}

	landing := dashboardLandingPath(session.RoleGrants)
	items := make([]dashboardNavItem, 0, 5)
	switch {
	case landing == "/":
		items = append(items, dashboardNavItem{Label: "Overview", URL: "/"})
	case strings.HasPrefix(landing, "/facilities/"):
		items = append(items, dashboardNavItem{Label: "Facility", URL: landing})
	case strings.HasPrefix(landing, "/stations/"):
		items = append(items, dashboardNavItem{Label: "Station", URL: landing})
	case landing == "/fleet":
		items = append(items, dashboardNavItem{Label: "Fleet", URL: landing})
	}
	if canViewStationPages(session.RoleGrants) && !strings.HasPrefix(landing, "/stations/") {
		items = append(items, dashboardNavItem{Label: "Stations", URL: "/stations"})
	}
	if canViewFleet(session.RoleGrants) && landing != "/fleet" {
		items = append(items, dashboardNavItem{Label: "Fleet", URL: "/fleet"})
	}
	if canManageUsers(session.RoleGrants) {
		items = append(items, dashboardNavItem{Label: "Users", URL: "/users"})
	}
	if canViewDeliveryOrderPages(session.RoleGrants) {
		items = append(items, dashboardNavItem{Label: "Delivery Orders", URL: "/delivery-orders"})
	}
	if canViewTripPages(session.RoleGrants) {
		items = append(items, dashboardNavItem{Label: "Trips", URL: "/trips"})
	}
	items = append(items, dashboardNavItem{Label: "Change Password", URL: "/change-password"})
	return items
}

func canViewFacility(roles []model.RoleGrant, facilityID int64) bool {
	for _, role := range roles {
		if role.Role == "SYSTEM_ADMIN" || role.Role == "REFINERY_ADMIN" {
			return true
		}
		if (role.Role == "FACILITY_OPERATOR" || role.Role == "FACILITY_MANAGER") && role.ScopeType == "FACILITY" && role.ScopeID != nil && *role.ScopeID == facilityID {
			return true
		}
	}
	return false
}

func buildCompanyMetrics(rows []db.GetCompanyWideDashboardSummaryRow) []dashboardMetric {
	var totalActiveTrips int64
	var totalAvailableVehicles int64
	for _, row := range rows {
		totalActiveTrips += row.ActiveTrips
		totalAvailableVehicles += row.AvailableVehicles
	}

	return []dashboardMetric{
		{Label: "Facilities", Value: strconv.Itoa(len(rows)), Help: "Active facilities in the company summary."},
		{Label: "Active Trips", Value: strconv.FormatInt(totalActiveTrips, 10), Help: "Trips currently loading, in transit, arrived, or unloading."},
		{Label: "Available Vehicles", Value: strconv.FormatInt(totalAvailableVehicles, 10), Help: "Vehicles marked available across all facilities."},
	}
}

func buildCompanySummaryRows(rows []db.GetCompanyWideDashboardSummaryRow) []dashboardCompanyFacilitySummary {
	out := make([]dashboardCompanyFacilitySummary, 0, len(rows))
	for _, row := range rows {
		out = append(out, dashboardCompanyFacilitySummary{
			FacilityID:        row.FacilityID,
			FacilityCode:      row.FacilityCode,
			FacilityName:      row.FacilityName,
			RefineryCode:      row.RefineryCode,
			ActiveTrips:       row.ActiveTrips,
			AvailableVehicles: row.AvailableVehicles,
		})
	}
	return out
}

func buildFacilityMetrics(summary db.GetFacilityDashboardSummaryRow) []dashboardMetric {
	return []dashboardMetric{
		{Label: "Active Trips", Value: strconv.FormatInt(summary.ActiveTrips, 10), Help: "Trips still in progress for this facility."},
		{Label: "Available Vehicles", Value: strconv.FormatInt(summary.AvailableVehicles, 10), Help: "Vehicles ready for dispatch from this facility."},
		{Label: "Maintenance", Value: strconv.FormatInt(summary.VehiclesInMaintenance, 10), Help: "Vehicles currently unavailable due to maintenance."},
	}
}

func filterDashboardTripsByFacility(rows []db.ListActiveTripsWithLatestGPSRow, facilityID int64) []db.ListActiveTripsWithLatestGPSRow {
	filtered := make([]db.ListActiveTripsWithLatestGPSRow, 0, len(rows))
	for _, row := range rows {
		if row.OriginFacilityID == facilityID {
			filtered = append(filtered, row)
		}
	}
	return filtered
}

func buildDashboardActiveTrips(rows []db.ListActiveTripsWithLatestGPSRow) ([]dashboardActiveTrip, string, error) {
	type liveTripSeed struct {
		TripID           int64      `json:"trip_id"`
		OriginFacilityID int64      `json:"origin_facility_id"`
		Status           string     `json:"status"`
		PlateNumber      string     `json:"plate_number"`
		DriverName       string     `json:"driver_name"`
		DestinationName  string     `json:"destination_name"`
		Lat              *float64   `json:"lat,omitempty"`
		Lng              *float64   `json:"lng,omitempty"`
		SpeedKmh         *float64   `json:"speed_kmh,omitempty"`
		LastGpsAt        *time.Time `json:"last_gps_at,omitempty"`
	}

	out := make([]dashboardActiveTrip, 0, len(rows))
	seed := make([]liveTripSeed, 0, len(rows))
	for _, row := range rows {
		destinationName := "Destination pending"
		if row.DestinationName.Valid && strings.TrimSpace(row.DestinationName.String) != "" {
			destinationName = row.DestinationName.String
		}

		lat, hasLat := numericToFloat64(row.LastLat)
		lng, hasLng := numericToFloat64(row.LastLng)
		speed, _ := numericToFloat64(row.LastSpeedKmh)
		lastGps := pgTimestamptzToPtr(row.LastGpsAt)

		trip := dashboardActiveTrip{
			ID:               row.ID,
			Status:           string(row.Status),
			PlateNumber:      row.PlateNumber,
			DriverName:       row.DriverName,
			DestinationName:  destinationName,
			OriginFacilityID: row.OriginFacilityID,
			SpeedLabel:       "-",
			LastGpsLabel:     "Awaiting GPS fix",
		}
		if speed != nil {
			trip.SpeedLabel = fmt.Sprintf("%.1f km/h", *speed)
		}
		if lastGps != nil {
			trip.LastGpsLabel = lastGps.UTC().Format("02 Jan 2006 15:04 MST")
		}
		if hasLat && hasLng {
			trip.HasLocation = true
			trip.Latitude = *lat
			trip.Longitude = *lng
		}

		out = append(out, trip)
		seed = append(seed, liveTripSeed{
			TripID:           row.ID,
			OriginFacilityID: row.OriginFacilityID,
			Status:           string(row.Status),
			PlateNumber:      row.PlateNumber,
			DriverName:       row.DriverName,
			DestinationName:  destinationName,
			Lat:              lat,
			Lng:              lng,
			SpeedKmh:         speed,
			LastGpsAt:        lastGps,
		})
	}

	payload, err := json.Marshal(seed)
	if err != nil {
		return nil, "", err
	}

	return out, string(payload), nil
}

func canViewStation(roles []model.RoleGrant, stationID int64) bool {
	for _, role := range roles {
		if role.Role == "SYSTEM_ADMIN" || role.Role == "REFINERY_ADMIN" {
			return true
		}
		if role.Role == "STATION_MANAGER" && role.ScopeType == "STATION" && role.ScopeID != nil && *role.ScopeID == stationID {
			return true
		}
	}
	return false
}

func canViewStationPages(roles []model.RoleGrant) bool {
	for _, role := range roles {
		switch role.Role {
		case "SYSTEM_ADMIN", "REFINERY_ADMIN", "FACILITY_MANAGER", "FACILITY_OPERATOR", "STATION_MANAGER":
			return true
		}
	}
	return false
}

func canViewFleet(roles []model.RoleGrant) bool {
	for _, role := range roles {
		switch role.Role {
		case "SYSTEM_ADMIN", "REFINERY_ADMIN", "FACILITY_MANAGER", "FACILITY_OPERATOR", "DEPOT_STAFF":
			return true
		}
	}
	return false
}

func canManageUsers(roles []model.RoleGrant) bool {
	for _, role := range roles {
		if role.Role == "SYSTEM_ADMIN" {
			return true
		}
	}
	return false
}
