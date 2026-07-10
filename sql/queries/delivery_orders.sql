
-- =============================================================================
-- SECTION 19 — DELIVERY ORDERS
-- =============================================================================

-- name: GetDeliveryOrder :one
SELECT *
FROM delivery_orders
WHERE id = $1;

-- name: GetDeliveryOrderByNumber :one
SELECT *
FROM delivery_orders
WHERE do_number = $1;

-- name: ListDOsByOriginFacility :many
SELECT *
FROM delivery_orders
WHERE origin_facility_id = $1
ORDER BY scheduled_date DESC, created_at DESC;

-- name: ListDOsByStatus :many
SELECT
    dor.*,
    v.plate_number  AS vehicle_plate,
    u.full_name     AS raised_by_name
FROM delivery_orders dor
LEFT JOIN vehicles   v ON v.id = dor.assigned_vehicle_id
LEFT JOIN users      u ON u.id = dor.raised_by
WHERE dor.status = $1
ORDER BY dor.scheduled_date ASC, dor.created_at ASC;

-- name: ListDOsForDispatchQueue :many
-- Approved DOs awaiting or having vehicle assignment, scoped to a facility.
SELECT
    dor.*,
    gs.name  AS destination_name,
    gs.code  AS destination_code
FROM delivery_orders dor
LEFT JOIN gas_stations gs ON gs.id = dor.destination_station_id
WHERE dor.origin_facility_id = $1
  AND dor.status IN ('APPROVED', 'ASSIGNED')
ORDER BY dor.scheduled_date ASC, dor.created_at ASC;

-- name: ListDOsByDriver :many
SELECT *
FROM delivery_orders
WHERE assigned_driver_id = $1
ORDER BY scheduled_date DESC;

-- name: ListDOsByVehicle :many
SELECT *
FROM delivery_orders
WHERE assigned_vehicle_id = $1
ORDER BY scheduled_date DESC;

-- name: GetNextDOSequenceNumber :one
-- Returns the current max sequence for a facility prefix in the current year.
-- Application formats this into DO number: DO-{RU}-{YYYY}-{seq:05d}.
SELECT COALESCE(
    MAX(
        CAST(SPLIT_PART(do_number, '-', 4) AS INTEGER)
    ), 0
) AS last_seq
FROM delivery_orders
WHERE do_number LIKE $1
  AND EXTRACT(YEAR FROM created_at) = EXTRACT(YEAR FROM NOW());

-- name: CreateDeliveryOrder :one
INSERT INTO delivery_orders (
    do_number, origin_facility_id, destination_type,
    destination_station_id, destination_facility_id,
    scheduled_date, notes, raised_by
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: UpdateDOStatus :one
UPDATE delivery_orders
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ApproveDeliveryOrder :one
UPDATE delivery_orders
SET
    status      = 'APPROVED',
    approved_by = $2,
    approved_at = NOW(),
    updated_at  = NOW()
WHERE id = $1
  AND status = 'PENDING_APPROVAL'
RETURNING *;

-- name: AssignVehicleAndDriverToDO :one
UPDATE delivery_orders
SET
    status              = 'ASSIGNED',
    assigned_vehicle_id = $2,
    assigned_driver_id  = $3,
    assigned_at         = NOW(),
    updated_at          = NOW()
WHERE id = $1
  AND status = 'APPROVED'
RETURNING *;

-- name: CancelDeliveryOrder :one
UPDATE delivery_orders
SET status = 'CANCELLED', updated_at = NOW()
WHERE id = $1
  AND status NOT IN ('IN_PROGRESS', 'DELIVERED', 'RECONCILED', 'CLOSED')
RETURNING *;
