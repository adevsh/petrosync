package middleware

import (
	"context"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/adevsh/petrosync/internal/db"
	"github.com/adevsh/petrosync/internal/model"
)

type ScopeQuerier interface {
	GetFacility(ctx context.Context, id int64) (db.GetFacilityRow, error)
	GetDepot(ctx context.Context, id int64) (db.GetDepotRow, error)
	GetStation(ctx context.Context, id int64) (db.GetStationRow, error)
}

var roleRank = map[string]int{
	"DRIVER":            1,
	"STATION_MANAGER":   2,
	"DEPOT_STAFF":       3,
	"FACILITY_OPERATOR": 4,
	"FACILITY_MANAGER":  5,
	"REFINERY_ADMIN":    6,
	"SYSTEM_ADMIN":      7,
}

func RoleRank(role string) int {
	r, ok := roleRank[role]
	if !ok {
		return 0
	}
	return r
}

// RequiredRole is used as middleware to enforce a minimum role for a route group.
// It checks the JWT claims (set by JWTAuth) for the required role within the
// specified scope type. The scope ID is resolved from the route parameter named
// by scopeParam (e.g., "id", "facility_id", "station_id").
//
// SYSTEM_ADMIN bypasses all scope checks — they have company-wide access.
func RequiredRole(querier ScopeQuerier, role string, scopeType string, scopeParam string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "no role grants found"},
			})
			return
		}

		roles, ok := rolesVal.([]model.RoleGrant)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "invalid role grants"},
			})
			return
		}

		requiredRank := RoleRank(role)
		if requiredRank == 0 {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": gin.H{"code": "INTERNAL_ERROR", "message": "invalid RBAC role requirement"},
			})
			return
		}

		// Resolve scope ID from path parameter
		var scopeID *int64
		if scopeParam != "" {
			raw := c.Param(scopeParam)
			if raw != "" {
				id, err := strconv.ParseInt(raw, 10, 64)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
						"error": gin.H{"code": "VALIDATION_ERROR", "message": "invalid scope id in path"},
					})
					return
				}
				scopeID = &id
			}
		}

		resolveFacilityID := func(ctx context.Context, scopeType string, scopeID int64) (int64, bool, error) {
			if querier == nil {
				return 0, false, nil
			}
			switch scopeType {
			case "FACILITY":
				return scopeID, true, nil
			case "DEPOT":
				depot, err := querier.GetDepot(ctx, scopeID)
				if err != nil {
					return 0, true, err
				}
				return depot.PrimaryFacilityID, true, nil
			case "STATION":
				station, err := querier.GetStation(ctx, scopeID)
				if err != nil {
					return 0, true, err
				}
				return station.PrimaryFacilityID, true, nil
			default:
				return 0, true, nil
			}
		}

		// Check each role grant
		for _, r := range roles {
			// SYSTEM_ADMIN bypasses all scope checks
			if r.Role == "SYSTEM_ADMIN" {
				c.Next()
				return
			}

			if RoleRank(r.Role) < requiredRank {
				continue
			}

			if scopeType == "" {
				c.Next()
				return
			}

			if r.ScopeType == scopeType {
				if scopeID == nil || r.ScopeID == nil || *r.ScopeID == *scopeID {
					c.Next()
					return
				}
				continue
			}

			if scopeID == nil || r.ScopeID == nil {
				continue
			}

			ctx := c.Request.Context()

			switch {
			case r.Role == "REFINERY_ADMIN" && r.ScopeType == "REFINERY":
				facilityID, ok, err := resolveFacilityID(ctx, scopeType, *scopeID)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to resolve facility scope"},
					})
					return
				}
				if ok {
					f, err := querier.GetFacility(ctx, facilityID)
					if err != nil {
						c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
							"error": gin.H{"code": "FORBIDDEN", "message": "invalid scope resource"},
						})
						return
					}
					if f.RefineryID == *r.ScopeID {
						c.Next()
						return
					}
				}
			case r.ScopeType == "FACILITY" && (scopeType == "DEPOT" || scopeType == "STATION"):
				facilityID, ok, err := resolveFacilityID(ctx, scopeType, *scopeID)
				if err != nil {
					c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
						"error": gin.H{"code": "INTERNAL_ERROR", "message": "failed to resolve facility scope"},
					})
					return
				}
				if ok && facilityID == *r.ScopeID {
					c.Next()
					return
				}
			}
		}

		c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
			"error": gin.H{"code": "FORBIDDEN", "message": "insufficient role or scope"},
		})
	}
}

func DisallowDriver() gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesVal, exists := c.Get("roles")
		if !exists {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "no role grants found"},
			})
			return
		}

		roles, ok := rolesVal.([]model.RoleGrant)
		if !ok {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "invalid role grants"},
			})
			return
		}

		maxRank := 0
		hasDriver := false
		for _, r := range roles {
			if r.Role == "DRIVER" {
				hasDriver = true
			}
			if rank := RoleRank(r.Role); rank > maxRank {
				maxRank = rank
			}
		}
		if hasDriver && maxRank <= RoleRank("DRIVER") {
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"error": gin.H{"code": "FORBIDDEN", "message": "driver role is not allowed on this endpoint"},
			})
			return
		}

		c.Next()
	}
}
