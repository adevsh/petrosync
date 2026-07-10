
-- =============================================================================
-- SECTION 14 — DRIVERS
-- =============================================================================

-- name: GetDriver :one
SELECT
    d.*,
    u.username,
    u.full_name,
    u.telegram_user_id,
    u.active AS user_active
FROM drivers d
JOIN users u ON u.id = d.user_id
WHERE d.id = $1;

-- name: GetDriverByUserID :one
SELECT
    d.*,
    u.username,
    u.full_name,
    u.telegram_user_id,
    u.active AS user_active
FROM drivers d
JOIN users u ON u.id = d.user_id
WHERE d.user_id = $1;

-- name: ListDriversByDepot :many
SELECT
    d.*,
    u.full_name,
    u.telegram_user_id,
    u.active AS user_active
FROM drivers d
JOIN users u ON u.id = d.user_id
WHERE d.home_depot_id = $1
  AND u.active = TRUE
ORDER BY u.full_name;

-- name: ListAvailableDriversForDispatch :many
-- Available = on shift, valid SIM B2, no active trip.
SELECT
    d.*,
    u.full_name,
    u.telegram_user_id
FROM drivers d
JOIN users u ON u.id = d.user_id
WHERE d.is_on_shift  = TRUE
  AND d.sim_b2_expiry > CURRENT_DATE
  AND u.active       = TRUE
  AND NOT EXISTS (
        SELECT 1 FROM trips t
        WHERE t.driver_id = d.id
          AND t.status NOT IN ('CLOSED', 'CANCELLED', 'RECONCILED')
      )
ORDER BY u.full_name;

-- name: ListDriversWithExpiringLicense :many
-- 30-day advance warning window.
SELECT
    d.*,
    u.full_name,
    u.telegram_user_id
FROM drivers d
JOIN users u ON u.id = d.user_id
WHERE d.sim_b2_expiry BETWEEN CURRENT_DATE AND (CURRENT_DATE + INTERVAL '30 days')
  AND u.active = TRUE
ORDER BY d.sim_b2_expiry ASC;

-- name: CreateDriver :one
INSERT INTO drivers (
    user_id, employee_number, sim_b2_number, sim_b2_expiry, home_depot_id
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: StartDriverShift :exec
UPDATE drivers
SET
    is_on_shift         = TRUE,
    current_shift_start = NOW(),
    current_shift_end   = NULL,
    updated_at          = NOW()
WHERE id = $1;

-- name: EndDriverShift :exec
UPDATE drivers
SET
    is_on_shift       = FALSE,
    current_shift_end = NOW(),
    updated_at        = NOW()
WHERE id = $1;

-- name: UpdateDriverLicense :one
UPDATE drivers
SET
    sim_b2_number = $2,
    sim_b2_expiry = $3,
    updated_at    = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDriverHomeDepot :exec
UPDATE drivers
SET home_depot_id = $2, updated_at = NOW()
WHERE id = $1;
