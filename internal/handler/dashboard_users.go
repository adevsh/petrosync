package handler

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/adevsh/petrosync/internal/db"
)

func (h *DashboardHandler) requireUserAdminPage() gin.HandlerFunc {
	return func(c *gin.Context) {
		session := currentDashboardSession(c)
		if session == nil {
			c.Redirect(http.StatusSeeOther, "/login")
			c.Abort()
			return
		}
		if !canManageUsers(session.RoleGrants) {
			c.String(http.StatusForbidden, "forbidden")
			c.Abort()
			return
		}
		c.Next()
	}
}

func (h *DashboardHandler) UserList(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	if h.userAdmin == nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	rows, err := h.userAdmin.querier.ListUsers(c.Request.Context())
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load users")
		return
	}

	users := make([]userResponse, 0, len(rows))
	for _, row := range rows {
		users = append(users, userFromListRow(row))
	}

	h.render(c, http.StatusOK, "users.html", h.pageData(c, session, dashboardPageData{
		Title:       "Users",
		Heading:     "User Management",
		Description: "Manage dashboard users, create accounts, and open role assignment or password reset actions.",
		Breadcrumbs: []DashboardBreadcrumb{{Label: "Users", URL: "/users"}},
		Users:       users,
	}))
}

func (h *DashboardHandler) ShowUserCreate(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	if h.userAdmin == nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	h.render(c, http.StatusOK, "user_new.html", h.pageData(c, session, dashboardPageData{
		Title:       "Create User",
		Heading:     "Create User",
		Description: "Create a dashboard account. New users are marked for password change on first sign-in.",
		Breadcrumbs: []DashboardBreadcrumb{
			{Label: "Users", URL: "/users"},
			{Label: "Create User", URL: "/users/new"},
		},
	}))
}

func (h *DashboardHandler) UserDetail(c *gin.Context) {
	session := currentDashboardSession(c)
	if session == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	if h.userAdmin == nil {
		c.String(http.StatusNotFound, "not found")
		return
	}

	userID, err := strconv.ParseInt(c.Param("id"), 10, 64)
	if err != nil || userID <= 0 {
		c.String(http.StatusBadRequest, "invalid user id")
		return
	}

	userRow, err := h.userAdmin.querier.GetUser(c.Request.Context(), userID)
	if err != nil {
		c.String(http.StatusNotFound, "user not found")
		return
	}
	roleRows, err := h.userAdmin.querier.GetActiveRolesForUser(c.Request.Context(), userID)
	if err != nil {
		c.String(http.StatusInternalServerError, "failed to load roles")
		return
	}

	roleGrants := make([]roleGrantResponse, 0, len(roleRows))
	for _, row := range roleRows {
		roleGrants = append(roleGrants, roleGrantFromRow(row))
	}

	selectedUser := userFromGetRow(userRow)
	h.render(c, http.StatusOK, "user_detail.html", h.pageData(c, session, dashboardPageData{
		Title:           selectedUser.FullName,
		Heading:         selectedUser.FullName,
		Description:     "Assign roles and reset this user's password using the existing admin actions.",
		Breadcrumbs:     []DashboardBreadcrumb{{Label: "Users", URL: "/users"}, {Label: selectedUser.Username, URL: c.Request.URL.Path}},
		SelectedUser:    &selectedUser,
		RoleGrants:      roleGrants,
		RoleOptions:     dashboardRoleOptions(),
		ScopeOptions:    dashboardScopeOptions(),
		ScopeReferences: h.buildUserScopeReferences(c),
	}))
}

func dashboardRoleOptions() []string {
	values := db.AllUserRoleTValues()
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}
	return out
}

func dashboardScopeOptions() []string {
	values := db.AllRoleScopeTValues()
	out := make([]string, 0, len(values))
	for _, value := range values {
		out = append(out, string(value))
	}
	return out
}

func (h *DashboardHandler) buildUserScopeReferences(c *gin.Context) []dashboardScopeReferenceSection {
	if h == nil || h.queries == nil {
		return nil
	}

	sections := make([]dashboardScopeReferenceSection, 0, 4)
	refineryLabels := make(map[int64]string)

	if refineries, err := h.queries.ListRefineries(c.Request.Context()); err == nil {
		items := make([]dashboardScopeReference, 0, len(refineries))
		for _, refinery := range refineries {
			label := strings.TrimSpace(refinery.Code)
			if name := strings.TrimSpace(refinery.Name); name != "" {
				label = label + " - " + name
			}
			refineryLabels[refinery.ID] = label
			items = append(items, dashboardScopeReference{
				ID:     refinery.ID,
				Label:  label,
				Detail: strings.TrimSpace(refinery.RegionCode),
			})
		}
		if len(items) > 0 {
			sections = append(sections, dashboardScopeReferenceSection{ScopeType: "REFINERY", Items: items})
		}

		facilityItems := make([]dashboardScopeReference, 0)
		for _, refinery := range refineries {
			facilities, facilityErr := h.queries.ListFacilitiesByRefinery(c.Request.Context(), refinery.ID)
			if facilityErr != nil {
				continue
			}
			for _, facility := range facilities {
				detail := refineryLabels[facility.RefineryID]
				if detail == "" {
					detail = "Refinery #" + strconv.FormatInt(facility.RefineryID, 10)
				}
				facilityItems = append(facilityItems, dashboardScopeReference{
					ID:     facility.ID,
					Label:  strings.TrimSpace(facility.Code) + " - " + strings.TrimSpace(facility.Name),
					Detail: detail,
				})
			}
		}
		if len(facilityItems) > 0 {
			sections = append(sections, dashboardScopeReferenceSection{ScopeType: "FACILITY", Items: facilityItems})
		}
	}

	if depots, err := h.queries.ListAllActiveDepots(c.Request.Context()); err == nil {
		items := make([]dashboardScopeReference, 0, len(depots))
		for _, depot := range depots {
			items = append(items, dashboardScopeReference{
				ID:     depot.ID,
				Label:  strings.TrimSpace(depot.Code) + " - " + strings.TrimSpace(depot.Name),
				Detail: strings.TrimSpace(depot.FacilityCode) + " - " + strings.TrimSpace(depot.FacilityName),
			})
		}
		if len(items) > 0 {
			sections = append(sections, dashboardScopeReferenceSection{ScopeType: "DEPOT", Items: items})
		}
	}

	if stations, err := h.queries.ListAllActiveStations(c.Request.Context()); err == nil {
		items := make([]dashboardScopeReference, 0, len(stations))
		for _, station := range stations {
			detail := "Facility #" + strconv.FormatInt(station.PrimaryFacilityID, 10)
			if region := strings.TrimSpace(station.RegionCode); region != "" {
				detail = detail + " · " + region
			}
			items = append(items, dashboardScopeReference{
				ID:     station.ID,
				Label:  strings.TrimSpace(station.Code) + " - " + strings.TrimSpace(station.Name),
				Detail: detail,
			})
		}
		if len(items) > 0 {
			sections = append(sections, dashboardScopeReferenceSection{ScopeType: "STATION", Items: items})
		}
	}

	return sections
}
