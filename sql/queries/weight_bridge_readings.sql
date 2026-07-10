
-- =============================================================================
-- SECTION 21 — WEIGHT BRIDGE READINGS
-- =============================================================================

-- name: GetWeightBridgeReading :one
SELECT *
FROM weight_bridge_readings
WHERE id = $1;

-- name: GetTareReadingByTrip :one
SELECT *
FROM weight_bridge_readings
WHERE trip_id      = $1
  AND reading_type = 'TARE';

-- name: GetGrossReadingByTrip :one
SELECT *
FROM weight_bridge_readings
WHERE trip_id      = $1
  AND reading_type = 'GROSS';

-- name: ListWeightBridgeReadingsByTrip :many
SELECT *
FROM weight_bridge_readings
WHERE trip_id = $1
ORDER BY created_at;

-- name: ListPendingManualApprovals :many
-- Facility manager approval queue. Scoped to a specific facility via vehicle's current depot.
SELECT
    wbr.*,
    v.plate_number,
    u.full_name AS recorded_by_name,
    t.id AS trip_id
FROM weight_bridge_readings wbr
JOIN vehicles v ON v.id = wbr.vehicle_id
JOIN users    u ON u.id = wbr.recorded_by
LEFT JOIN trips t ON t.id = wbr.trip_id
WHERE wbr.method          = 'MANUAL_APPROVED'
  AND wbr.approval_status = 'PENDING'
  AND v.current_depot_id IN (
        SELECT id FROM vehicle_depots WHERE primary_facility_id = $1
      )
ORDER BY wbr.created_at ASC;

-- name: ListEscalatedApprovals :many
-- Refinery Admin escalation queue (company-wide).
SELECT
    wbr.*,
    v.plate_number,
    u.full_name AS recorded_by_name
FROM weight_bridge_readings wbr
JOIN vehicles v ON v.id = wbr.vehicle_id
JOIN users    u ON u.id = wbr.recorded_by
WHERE wbr.method          = 'MANUAL_APPROVED'
  AND wbr.approval_status = 'ESCALATED'
ORDER BY wbr.escalated_at ASC;

-- name: CreateWeightBridgeReading :one
INSERT INTO weight_bridge_readings (
    trip_id, vehicle_id, reading_type, weight_kg,
    method, ambient_temp_celsius, recorded_by, notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: ApproveWeightBridgeReading :one
UPDATE weight_bridge_readings
SET
    approval_status = 'APPROVED',
    approved_by     = $2,
    approved_at     = NOW()
WHERE id = $1
  AND approval_status IN ('PENDING', 'ESCALATED')
RETURNING *;

-- name: EscalateWeightBridgeReading :one
-- Called by background worker when Facility Manager window expires.
UPDATE weight_bridge_readings
SET
    approval_status = 'ESCALATED',
    escalated_at    = NOW(),
    escalated_to    = $2
WHERE id = $1
  AND approval_status = 'PENDING'
RETURNING *;

-- name: RejectWeightBridgeReading :one
UPDATE weight_bridge_readings
SET
    approval_status = 'REJECTED',
    approved_by     = $2,
    approved_at     = NOW(),
    notes           = $3
WHERE id = $1
RETURNING *;
