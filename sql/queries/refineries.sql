
-- =============================================================================
-- SECTION 4 — REFINERIES
-- =============================================================================

-- name: GetRefinery :one
SELECT *
FROM refineries
WHERE id = $1;

-- name: GetRefineryByCode :one
SELECT *
FROM refineries
WHERE code = $1;

-- name: ListRefineries :many
SELECT *
FROM refineries
WHERE active = TRUE
ORDER BY code;

-- name: ListRefineriesByRegion :many
SELECT *
FROM refineries
WHERE region_code = $1
  AND active = TRUE
ORDER BY code;

-- name: CreateRefinery :one
INSERT INTO refineries (code, name, region_code, commissioned_year)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: UpdateRefinery :one
UPDATE refineries
SET
    name               = $2,
    region_code        = $3,
    commissioned_year  = $4,
    updated_at         = NOW()
WHERE id = $1
RETURNING *;
