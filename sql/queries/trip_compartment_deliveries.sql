
-- =============================================================================
-- SECTION 24 — TRIP COMPARTMENT DELIVERIES
-- =============================================================================

-- name: GetCompartmentDelivery :one
SELECT *
FROM trip_compartment_deliveries
WHERE id = $1;

-- name: GetCompartmentDeliveryByTripAndCompartment :one
SELECT *
FROM trip_compartment_deliveries
WHERE trip_id      = $1
  AND compartment_id = $2;

-- name: ListCompartmentDeliveriesByTrip :many
SELECT
    tcd.*,
    ft.name AS fuel_type_name,
    vc.compartment_number
FROM trip_compartment_deliveries tcd
JOIN fuel_types           ft ON ft.code = tcd.fuel_type_code
JOIN vehicle_compartments vc ON vc.id   = tcd.compartment_id
WHERE tcd.trip_id = $1
ORDER BY vc.compartment_number;

-- name: ListDisputedDeliveries :many
-- Supervisor view: all disputed compartment deliveries across all trips.
SELECT
    tcd.*,
    t.id          AS trip_id,
    v.plate_number,
    gs.name       AS station_name
FROM trip_compartment_deliveries tcd
JOIN trips         t  ON t.id  = tcd.trip_id
JOIN vehicles      v  ON v.id  = t.vehicle_id
LEFT JOIN gas_stations gs ON gs.id = t.destination_station_id
WHERE tcd.delivery_status = 'DISPUTED'
ORDER BY t.completed_at DESC;

-- name: CreateCompartmentDelivery :one
INSERT INTO trip_compartment_deliveries (
    trip_id, compartment_id, fuel_type_code, measurement_method
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateLoadedVolume :one
UPDATE trip_compartment_deliveries
SET
    loaded_volume_l  = $2,
    loaded_weight_kg = $3,
    updated_at       = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDeliveredVolume :one
UPDATE trip_compartment_deliveries
SET
    delivered_volume_l  = $2,
    delivered_weight_kg = $3,
    updated_at          = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateCompartmentDeliveryStatus :one
UPDATE trip_compartment_deliveries
SET delivery_status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetTripVarianceSummary :one
-- Aggregate variance across all compartments for a trip (used by reconciliation engine).
SELECT
    trip_id,
    COUNT(*)                     AS compartment_count,
    SUM(loaded_volume_l)         AS total_loaded_l,
    SUM(delivered_volume_l)      AS total_delivered_l,
    SUM(variance_l)              AS total_variance_l,
    ROUND(
        (SUM(variance_l) / NULLIF(SUM(loaded_volume_l), 0) * 100)::NUMERIC, 4
    )                            AS overall_variance_pct,
    BOOL_OR(delivery_status = 'DISPUTED') AS has_disputed
FROM trip_compartment_deliveries
WHERE trip_id = $1
GROUP BY trip_id;

-- name: ListTripLoadedVolumeByFuel :many
SELECT
    fuel_type_code,
    SUM(loaded_volume_l)::NUMERIC AS total_loaded_l
FROM trip_compartment_deliveries
WHERE trip_id = $1
GROUP BY fuel_type_code
ORDER BY fuel_type_code;

-- name: ListTripDeliveredVolumeByFuel :many
SELECT
    fuel_type_code,
    SUM(delivered_volume_l)::NUMERIC AS total_delivered_l
FROM trip_compartment_deliveries
WHERE trip_id = $1
GROUP BY fuel_type_code
ORDER BY fuel_type_code;
