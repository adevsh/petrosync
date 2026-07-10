
-- =============================================================================
-- SECTION 2 — FUEL TYPES
-- =============================================================================

-- name: GetFuelType :one
SELECT *
FROM fuel_types
WHERE code = $1;

-- name: ListFuelTypes :many
SELECT *
FROM fuel_types
ORDER BY category, ron_cn;

-- name: ListActiveFuelTypes :many
SELECT *
FROM fuel_types
WHERE active = TRUE
ORDER BY category, ron_cn;

-- name: ListFuelTypesByCategory :many
SELECT *
FROM fuel_types
WHERE category = $1
  AND active = TRUE
ORDER BY ron_cn;

-- name: CreateFuelType :one
INSERT INTO fuel_types (
    code, name, category, ron_cn,
    density_kg_per_l_at_15c, evaporation_factor_pct, is_subsidized
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: UpdateFuelType :one
UPDATE fuel_types
SET
    name                    = $2,
    density_kg_per_l_at_15c = $3,
    evaporation_factor_pct  = $4,
    is_subsidized           = $5,
    updated_at              = NOW()
WHERE code = $1
RETURNING *;

-- name: DeactivateFuelType :exec
UPDATE fuel_types
SET active = FALSE, updated_at = NOW()
WHERE code = $1;
