
-- =============================================================================
-- SECTION 9 — VEHICLES
-- =============================================================================

-- name: GetVehicle :one
SELECT
    v.*,
    ST_X(v.current_location) AS current_longitude,
    ST_Y(v.current_location) AS current_latitude
FROM vehicles v
WHERE v.id = $1;

-- name: GetVehicleByPlate :one
SELECT
    v.*,
    ST_X(v.current_location) AS current_longitude,
    ST_Y(v.current_location) AS current_latitude
FROM vehicles v
WHERE v.plate_number = $1;

-- name: ListVehiclesByDepot :many
SELECT
    v.*,
    ST_X(v.current_location) AS current_longitude,
    ST_Y(v.current_location) AS current_latitude
FROM vehicles v
WHERE v.current_depot_id = $1
  AND v.active = TRUE
ORDER BY v.plate_number;

-- name: ListVehiclesByStatus :many
SELECT
    v.*,
    ST_X(v.current_location) AS current_longitude,
    ST_Y(v.current_location) AS current_latitude
FROM vehicles v
WHERE v.status = $1
  AND v.active = TRUE
ORDER BY v.plate_number;

-- Primary dispatch query. Finds available trucks near a facility ordered by:
-- 1. Depot proximity to origin facility (home depot first)
-- 2. Distance from current GPS location to facility
-- 3. Least recently assigned (fairness)
-- Returns up to $2 candidates. Caller filters by required fuel type separately.
-- name: ListDispatchCandidateVehicles :many
SELECT
    v.id,
    v.plate_number,
    v.total_capacity_l,
    v.tare_weight_kg,
    v.keur_expiry,
    v.current_depot_id,
    ST_X(v.current_location)  AS current_longitude,
    ST_Y(v.current_location)  AS current_latitude,
    (d.primary_facility_id = $1)::BOOLEAN  AS depot_matched,
    ROUND(
        (ST_Distance(
            v.current_location::GEOGRAPHY,
            f.location::GEOGRAPHY
        ) / 1000)::NUMERIC, 2
    ) AS distance_km
FROM vehicles v
JOIN refinery_facilities f ON f.id = $1
LEFT JOIN vehicle_depots  d ON d.id = v.current_depot_id
WHERE v.status   = 'AVAILABLE'
  AND v.active   = TRUE
  AND v.keur_expiry > CURRENT_DATE
  AND ST_DWithin(
        v.current_location::GEOGRAPHY,
        f.location::GEOGRAPHY,
        f.max_assignment_radius_km * 1000
      )
ORDER BY
    depot_matched DESC,
    distance_km   ASC,
    v.last_assigned_at ASC NULLS FIRST
LIMIT $2;

-- name: ListVehiclesWithExpiringKeur :many
-- Used for 30-day advance expiry notification.
SELECT
    v.*,
    ST_X(v.current_location) AS current_longitude,
    ST_Y(v.current_location) AS current_latitude
FROM vehicles v
WHERE v.active = TRUE
  AND v.keur_expiry BETWEEN CURRENT_DATE AND (CURRENT_DATE + INTERVAL '30 days')
ORDER BY v.keur_expiry ASC;

-- name: CreateVehicle :one
INSERT INTO vehicles (
    plate_number, chassis_number, model, manufacture_year,
    total_capacity_l, tare_weight_kg, current_depot_id,
    keur_number, keur_expiry
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: UpdateVehicleStatus :one
UPDATE vehicles
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateVehicleLocation :exec
UPDATE vehicles
SET
    current_location = ST_SetSRID(ST_MakePoint($2, $3), 4326),
    updated_at       = NOW()
WHERE id = $1;

-- name: UpdateVehicleDepot :exec
UPDATE vehicles
SET current_depot_id = $2, updated_at = NOW()
WHERE id = $1;

-- name: UpdateVehicleKeurDetails :one
UPDATE vehicles
SET
    keur_number  = $2,
    keur_expiry  = $3,
    updated_at   = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkVehicleLastAssigned :exec
UPDATE vehicles
SET last_assigned_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: DeactivateVehicle :exec
UPDATE vehicles
SET active = FALSE, updated_at = NOW()
WHERE id = $1;
