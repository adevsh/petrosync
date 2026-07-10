
-- =============================================================================
-- SECTION 33 — REPORTING & CROSS-TABLE QUERIES
-- =============================================================================

-- name: GetFacilityDashboardSummary :one
-- Central ops dashboard: inventory + active trip count for one facility.
SELECT
    rf.id                                          AS facility_id,
    rf.name                                        AS facility_name,
    COUNT(DISTINCT t.id) FILTER (
        WHERE t.status NOT IN ('CLOSED','CANCELLED','RECONCILED')
    )                                              AS active_trips,
    COUNT(DISTINCT v.id) FILTER (
        WHERE v.status = 'AVAILABLE'
    )                                              AS available_vehicles,
    COUNT(DISTINCT v.id) FILTER (
        WHERE v.status = 'UNDER_MAINTENANCE'
    )                                              AS vehicles_in_maintenance
FROM refinery_facilities rf
LEFT JOIN vehicle_depots  vd ON vd.primary_facility_id = rf.id
LEFT JOIN vehicles         v ON v.current_depot_id     = vd.id AND v.active = TRUE
LEFT JOIN trips            t ON t.origin_facility_id   = rf.id
WHERE rf.id = $1
GROUP BY rf.id, rf.name;

-- name: GetCompanyWideDashboardSummary :many
-- Multi-refinery ops view: one row per facility.
SELECT
    rf.id                                          AS facility_id,
    rf.code                                        AS facility_code,
    rf.name                                        AS facility_name,
    r.code                                         AS refinery_code,
    COUNT(DISTINCT t.id) FILTER (
        WHERE t.status IN ('LOADING','LOADED','IN_TRANSIT','ARRIVED','UNLOADING')
    )                                              AS active_trips,
    COUNT(DISTINCT v.id) FILTER (
        WHERE v.status = 'AVAILABLE' AND v.active = TRUE
    )                                              AS available_vehicles
FROM refinery_facilities rf
JOIN refineries           r  ON r.id  = rf.refinery_id
LEFT JOIN vehicle_depots  vd ON vd.primary_facility_id = rf.id
LEFT JOIN vehicles         v ON v.current_depot_id = vd.id
LEFT JOIN trips            t ON t.origin_facility_id = rf.id
WHERE rf.active = TRUE
GROUP BY rf.id, rf.code, rf.name, r.code
ORDER BY r.code, rf.is_primary DESC;

-- name: GetMonthlyDeliveryStatsByFacility :many
-- Reporting: delivered volume by fuel type per facility per month.
SELECT
    DATE_TRUNC('month', t.completed_at) AS month,
    tcd.fuel_type_code,
    COUNT(DISTINCT t.id)                AS trip_count,
    SUM(tcd.loaded_volume_l)            AS total_loaded_l,
    SUM(tcd.delivered_volume_l)         AS total_delivered_l,
    SUM(tcd.variance_l)                 AS total_variance_l,
    ROUND(
        (SUM(tcd.variance_l) / NULLIF(SUM(tcd.loaded_volume_l), 0) * 100)::NUMERIC, 4
    )                                   AS overall_variance_pct
FROM trips t
JOIN trip_compartment_deliveries tcd ON tcd.trip_id = t.id
WHERE t.origin_facility_id = $1
  AND t.status             = 'CLOSED'
  AND t.completed_at      >= $2
  AND t.completed_at      <  $3
GROUP BY DATE_TRUNC('month', t.completed_at), tcd.fuel_type_code
ORDER BY month DESC, tcd.fuel_type_code;

-- name: GetDriverComplianceSummary :one
-- Compliance scoring: variance, deviation, and seal mismatch history per driver.
SELECT
    d.id                   AS driver_id,
    u.full_name,
    COUNT(DISTINCT t.id)   AS total_trips,
    COUNT(DISTINCT t.id) FILTER (
        WHERE EXISTS (
            SELECT 1 FROM trip_compartment_deliveries tcd2
            WHERE tcd2.trip_id = t.id AND tcd2.delivery_status = 'DISPUTED'
        )
    )                      AS disputed_trips,
    COUNT(DISTINCT rde.trip_id) AS trips_with_deviation,
    COUNT(DISTINCT cs.trip_id)  AS trips_with_seal_mismatch,
    ROUND(
        (1 - (COUNT(DISTINCT t.id) FILTER (
            WHERE EXISTS (
                SELECT 1 FROM trip_compartment_deliveries tcd2
                WHERE tcd2.trip_id = t.id AND tcd2.delivery_status = 'DISPUTED'
            )
        ))::NUMERIC / NULLIF(COUNT(DISTINCT t.id), 0)) * 100, 1
    )                      AS compliance_score_pct
FROM drivers d
JOIN users   u ON u.id = d.user_id
LEFT JOIN trips t ON t.driver_id = d.id AND t.status = 'CLOSED'
    AND t.completed_at >= $2
    AND t.completed_at <  $3
LEFT JOIN route_deviation_events rde ON rde.trip_id = t.id
LEFT JOIN compartment_seals cs ON cs.trip_id = t.id
    AND cs.verification_status IN ('MISMATCHED','BROKEN','MISSING')
WHERE d.id = $1
GROUP BY d.id, u.full_name;

-- name: ListPendingWeightBridgeApprovalsByFacility :many
-- Combines both PENDING and ESCALATED readings needing action from this facility's managers.
SELECT
    wbr.*,
    v.plate_number,
    uro.full_name  AS recorded_by_name,
    t.id           AS trip_id,
    dor.do_number,
    CASE
        WHEN wbr.approval_status = 'PENDING'   THEN 'FACILITY_MANAGER'
        WHEN wbr.approval_status = 'ESCALATED' THEN 'REFINERY_ADMIN'
        ELSE wbr.approval_status::TEXT
    END            AS required_approver_role
FROM weight_bridge_readings wbr
JOIN vehicles v  ON v.id  = wbr.vehicle_id
JOIN users    uro ON uro.id = wbr.recorded_by
LEFT JOIN trips t  ON t.id = wbr.trip_id
LEFT JOIN delivery_orders dor ON dor.id = t.do_id
WHERE wbr.method NOT IN ('WEIGHT_BRIDGE')
  AND wbr.approval_status IN ('PENDING', 'ESCALATED')
  AND v.current_depot_id IN (
        SELECT id FROM vehicle_depots WHERE primary_facility_id = $1
      )
ORDER BY wbr.created_at ASC;

-- name: GetStationInventorySnapshot :many
-- Full inventory snapshot for a station (all active tanks).
SELECT
    st.*,
    ft.name                  AS fuel_name,
    ft.category              AS fuel_category,
    ROUND(
        (st.current_volume_l / NULLIF(st.capacity_l, 0) * 100)::NUMERIC, 1
    )                        AS fill_pct,
    (st.current_volume_l <= st.reorder_threshold_l) AS needs_reorder
FROM station_tanks st
JOIN fuel_types    ft ON ft.code = st.fuel_type_code
WHERE st.station_id = $1
  AND st.active     = TRUE
ORDER BY ft.category, ft.ron_cn;

-- name: ListVehiclesWithMaintenanceOrExpiryDue :many
-- Operations notice board: trucks needing attention in next 30 days.
SELECT
    v.id,
    v.plate_number,
    v.status,
    v.keur_expiry,
    v.next_inspection_due,
    vd.name AS depot_name,
    rf.code AS facility_code,
    CASE
        WHEN v.status = 'UNDER_MAINTENANCE'                                       THEN 'UNDER_MAINTENANCE'
        WHEN v.keur_expiry       <= (CURRENT_DATE + INTERVAL '30 days')           THEN 'KEUR_EXPIRING'
        WHEN v.next_inspection_due <= (CURRENT_DATE + INTERVAL '30 days')         THEN 'INSPECTION_DUE'
        ELSE 'OK'
    END AS notice_type
FROM vehicles v
JOIN vehicle_depots      vd ON vd.id = v.current_depot_id
JOIN refinery_facilities rf ON rf.id = vd.primary_facility_id
WHERE v.active = TRUE
  AND (
        v.status = 'UNDER_MAINTENANCE'
        OR v.keur_expiry       <= (CURRENT_DATE + INTERVAL '30 days')
        OR v.next_inspection_due <= (CURRENT_DATE + INTERVAL '30 days')
      )
ORDER BY rf.code, vd.name, v.keur_expiry ASC NULLS LAST;

-- =============================================================================
-- END OF QUERIES
-- =============================================================================
