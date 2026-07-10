-- =============================================================================
-- PetroSync — sqlc Query Definitions
-- =============================================================================
-- Compatible with: sqlc v2, pgx/v5 driver
-- Positional parameters: $1, $2, ... throughout (PostgreSQL native style)
--
-- GEOMETRY convention:
--   READ  → ST_X(col) AS longitude, ST_Y(col) AS latitude (returns FLOAT8)
--   WRITE → ST_SetSRID(ST_MakePoint($lng, $lat), 4326) in INSERT/UPDATE
--   SPATIAL queries → geometry columns used directly with PostGIS functions
--
-- APPEND-ONLY tables (trip_events, gps_events, audit_log, notification_log):
--   Only Insert queries are provided. No Update/Delete variants.
--   Enforced at DB role level — petrosync_app has no UPDATE/DELETE on these.
--
-- GENERATED COLUMNS (variance_l, variance_pct):
--   Excluded from INSERT/UPDATE statements — PostgreSQL computes them.
--
-- Recommended sqlc.yaml overrides:
--   NUMERIC   → pgtype.Numeric  (or decimal.Decimal via override)
--   JSONB     → json.RawMessage
--   INET      → netip.Addr
--   UUID      → pgtype.UUID
--   Enum types → emit_enum_valid_method: true
-- =============================================================================


-- =============================================================================
-- SECTION 1 — REGIONS
-- =============================================================================

-- name: GetRegion :one
SELECT code, name, created_at
FROM regions
WHERE code = $1;

-- name: ListRegions :many
SELECT code, name, created_at
FROM regions
ORDER BY name;


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


-- =============================================================================
-- SECTION 3 — SYSTEM SETTINGS
-- =============================================================================

-- name: GetGlobalSetting :one
SELECT *
FROM system_settings
WHERE key = $1
  AND facility_id IS NULL;

-- name: GetFacilitySetting :one
SELECT *
FROM system_settings
WHERE key = $1
  AND facility_id = $2;

-- Resolve effective value: facility-specific overrides global default.
-- name: GetEffectiveSetting :one
SELECT COALESCE(fac.value, gbl.value) AS value
FROM system_settings gbl
LEFT JOIN system_settings fac
       ON fac.key = gbl.key
      AND fac.facility_id = $2
WHERE gbl.key = $1
  AND gbl.facility_id IS NULL;

-- name: ListSettingsByFacility :many
SELECT *
FROM system_settings
WHERE facility_id = $1
ORDER BY key;

-- name: ListGlobalSettings :many
SELECT *
FROM system_settings
WHERE facility_id IS NULL
ORDER BY key;

-- name: UpsertGlobalSetting :one
INSERT INTO system_settings (facility_id, key, value, description, updated_by)
VALUES (NULL, $1, $2, $3, $4)
ON CONFLICT ON CONSTRAINT idx_system_settings_global
DO UPDATE SET
    value      = EXCLUDED.value,
    description = COALESCE(EXCLUDED.description, system_settings.description),
    updated_by  = EXCLUDED.updated_by,
    updated_at  = NOW()
RETURNING *;

-- name: UpsertFacilitySetting :one
INSERT INTO system_settings (facility_id, key, value, description, updated_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT ON CONSTRAINT idx_system_settings_facility
DO UPDATE SET
    value       = EXCLUDED.value,
    description = COALESCE(EXCLUDED.description, system_settings.description),
    updated_by  = EXCLUDED.updated_by,
    updated_at  = NOW()
RETURNING *;

-- name: DeleteFacilitySetting :exec
DELETE FROM system_settings
WHERE key = $1
  AND facility_id = $2;


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


-- =============================================================================
-- SECTION 5 — REFINERY FACILITIES
-- =============================================================================

-- name: GetFacility :one
SELECT
    rf.*,
    ST_X(rf.location) AS longitude,
    ST_Y(rf.location) AS latitude
FROM refinery_facilities rf
WHERE rf.id = $1;

-- name: GetFacilityByCode :one
SELECT
    rf.*,
    ST_X(rf.location) AS longitude,
    ST_Y(rf.location) AS latitude
FROM refinery_facilities rf
WHERE rf.code = $1;

-- name: ListFacilitiesByRefinery :many
SELECT
    rf.*,
    ST_X(rf.location) AS longitude,
    ST_Y(rf.location) AS latitude
FROM refinery_facilities rf
WHERE rf.refinery_id = $1
  AND rf.active = TRUE
ORDER BY rf.is_primary DESC, rf.name;

-- name: GetPrimaryFacilityByRefinery :one
SELECT
    rf.*,
    ST_X(rf.location) AS longitude,
    ST_Y(rf.location) AS latitude
FROM refinery_facilities rf
WHERE rf.refinery_id = $1
  AND rf.is_primary = TRUE
  AND rf.active = TRUE;

-- name: ListAllActiveFacilities :many
SELECT
    rf.*,
    r.code  AS refinery_code,
    r.name  AS refinery_name,
    ST_X(rf.location) AS longitude,
    ST_Y(rf.location) AS latitude
FROM refinery_facilities rf
JOIN refineries r ON r.id = rf.refinery_id
WHERE rf.active = TRUE
ORDER BY r.code, rf.is_primary DESC;

-- name: CreateFacility :one
INSERT INTO refinery_facilities (
    code, refinery_id, name, location, is_primary, max_assignment_radius_km, address
) VALUES (
    $1, $2, $3,
    ST_SetSRID(ST_MakePoint($4, $5), 4326),
    $6, $7, $8
)
RETURNING *;

-- name: UpdateFacility :one
UPDATE refinery_facilities
SET
    name                     = $2,
    address                  = $3,
    max_assignment_radius_km = $4,
    updated_at               = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateFacilityLocation :one
UPDATE refinery_facilities
SET
    location   = ST_SetSRID(ST_MakePoint($2, $3), 4326),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateFacility :exec
UPDATE refinery_facilities
SET active = FALSE, updated_at = NOW()
WHERE id = $1;


-- =============================================================================
-- SECTION 6 — VEHICLE DEPOTS
-- =============================================================================

-- name: GetDepot :one
SELECT
    vd.*,
    ST_X(vd.location) AS longitude,
    ST_Y(vd.location) AS latitude
FROM vehicle_depots vd
WHERE vd.id = $1;

-- name: GetDepotByCode :one
SELECT
    vd.*,
    ST_X(vd.location) AS longitude,
    ST_Y(vd.location) AS latitude
FROM vehicle_depots vd
WHERE vd.code = $1;

-- name: ListDepotsByFacility :many
SELECT
    vd.*,
    ST_X(vd.location) AS longitude,
    ST_Y(vd.location) AS latitude
FROM vehicle_depots vd
WHERE vd.primary_facility_id = $1
  AND vd.active = TRUE
ORDER BY vd.name;

-- name: ListAllActiveDepots :many
SELECT
    vd.*,
    rf.code AS facility_code,
    rf.name AS facility_name,
    ST_X(vd.location) AS longitude,
    ST_Y(vd.location) AS latitude
FROM vehicle_depots vd
JOIN refinery_facilities rf ON rf.id = vd.primary_facility_id
WHERE vd.active = TRUE
ORDER BY rf.code, vd.name;

-- name: CreateDepot :one
INSERT INTO vehicle_depots (
    code, name, primary_facility_id, location, default_truck_capacity_l
) VALUES (
    $1, $2, $3,
    ST_SetSRID(ST_MakePoint($4, $5), 4326),
    $6
)
RETURNING *;

-- name: UpdateDepot :one
UPDATE vehicle_depots
SET
    name                     = $2,
    default_truck_capacity_l = $3,
    updated_at               = NOW()
WHERE id = $1
RETURNING *;


-- =============================================================================
-- SECTION 7 — FACILITY LOADING BAYS
-- =============================================================================

-- name: GetLoadingBay :one
SELECT *
FROM facility_loading_bays
WHERE id = $1;

-- name: GetLoadingBayByQRPayload :one
SELECT flb.*, rf.id AS refinery_id
FROM facility_loading_bays flb
JOIN refinery_facilities rf ON rf.id = flb.facility_id
WHERE flb.qr_payload = $1
  AND flb.active = TRUE;

-- name: ListLoadingBaysByFacility :many
SELECT *
FROM facility_loading_bays
WHERE facility_id = $1
  AND active = TRUE
ORDER BY bay_code;

-- name: CreateLoadingBay :one
INSERT INTO facility_loading_bays (
    facility_id, bay_code, qr_payload, fuel_type_code
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateLoadingBayFuelType :one
UPDATE facility_loading_bays
SET fuel_type_code = $2
WHERE id = $1
RETURNING *;

-- name: DeactivateLoadingBay :exec
UPDATE facility_loading_bays
SET active = FALSE
WHERE id = $1;

-- name: ValidateLoadingBayQR :one
-- Validates QR payload belongs to the expected facility and is active.
SELECT flb.id, flb.facility_id, flb.bay_code, flb.fuel_type_code
FROM facility_loading_bays flb
WHERE flb.qr_payload = $1
  AND flb.facility_id = $2
  AND flb.active = TRUE;


-- =============================================================================
-- SECTION 8 — FACILITY STORAGE TANKS
-- =============================================================================

-- name: GetStorageTank :one
SELECT *
FROM facility_storage_tanks
WHERE id = $1;

-- name: GetStorageTankByFacilityAndFuel :one
SELECT *
FROM facility_storage_tanks
WHERE facility_id = $1
  AND fuel_type_code = $2
  AND active = TRUE;

-- name: ListStorageTanksByFacility :many
SELECT *
FROM facility_storage_tanks
WHERE facility_id = $1
  AND active = TRUE
ORDER BY fuel_type_code;

-- name: GetStorageTankAvailableVolume :one
SELECT
    id,
    facility_id,
    fuel_type_code,
    capacity_l,
    current_volume_l,
    reserved_volume_l,
    (current_volume_l - reserved_volume_l) AS available_volume_l
FROM facility_storage_tanks
WHERE id = $1
  AND active = TRUE;

-- name: CreateStorageTank :one
INSERT INTO facility_storage_tanks (
    facility_id, tank_code, fuel_type_code,
    capacity_l, current_volume_l, min_operational_l
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: ReserveStorageTankVolume :one
-- Atomically reserve volume at DO approval. Fails if insufficient available.
UPDATE facility_storage_tanks
SET
    reserved_volume_l = reserved_volume_l + $2,
    last_updated_at   = NOW()
WHERE id = $1
  AND active = TRUE
  AND (current_volume_l - reserved_volume_l) >= $2
RETURNING *;

-- name: ReleaseStorageTankReservation :one
-- Release reservation (on DO cancellation or manual override).
UPDATE facility_storage_tanks
SET
    reserved_volume_l = GREATEST(0, reserved_volume_l - $2),
    last_updated_at   = NOW()
WHERE id = $1
RETURNING *;

-- name: DeductStorageTankVolume :one
-- Deduct from current volume at loading completion (simultaneously clears reservation).
UPDATE facility_storage_tanks
SET
    current_volume_l  = current_volume_l - $2,
    reserved_volume_l = GREATEST(0, reserved_volume_l - $2),
    last_updated_at   = NOW()
WHERE id = $1
  AND active = TRUE
  AND current_volume_l >= $2
RETURNING *;

-- name: CreditStorageTankVolume :one
-- Credit volume on return-to-facility trip delivery.
UPDATE facility_storage_tanks
SET
    current_volume_l = current_volume_l + $2,
    last_updated_at  = NOW()
WHERE id = $1
  AND active = TRUE
RETURNING *;

-- name: UpdateStorageTankVolume :one
-- Manual correction by SYSTEM_ADMIN (audited separately via audit_log).
UPDATE facility_storage_tanks
SET
    current_volume_l = $2,
    last_updated_at  = NOW()
WHERE id = $1
RETURNING *;


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


-- =============================================================================
-- SECTION 10 — VEHICLE COMPARTMENTS
-- =============================================================================

-- name: GetCompartment :one
SELECT *
FROM vehicle_compartments
WHERE id = $1;

-- name: ListCompartmentsByVehicle :many
SELECT *
FROM vehicle_compartments
WHERE vehicle_id = $1
  AND is_active = TRUE
ORDER BY compartment_number;

-- name: ListAllCompartmentsByVehicle :many
-- Includes inactive compartments (admin view).
SELECT *
FROM vehicle_compartments
WHERE vehicle_id = $1
ORDER BY compartment_number;

-- name: ListCompartmentsByVehicleAndFuel :many
SELECT *
FROM vehicle_compartments
WHERE vehicle_id      = $1
  AND fuel_type_code  = $2
  AND is_active       = TRUE
ORDER BY compartment_number;

-- name: GetTotalCapacityByVehicle :one
SELECT
    vehicle_id,
    COUNT(*)                   AS compartment_count,
    SUM(capacity_l)            AS total_capacity_l
FROM vehicle_compartments
WHERE vehicle_id = $1
  AND is_active = TRUE
GROUP BY vehicle_id;

-- name: CreateCompartment :one
INSERT INTO vehicle_compartments (
    vehicle_id, compartment_number, fuel_type_code, capacity_l
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: UpdateCompartmentFuelType :one
UPDATE vehicle_compartments
SET
    fuel_type_code = $2,
    updated_at     = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateCompartment :exec
UPDATE vehicle_compartments
SET is_active = FALSE, updated_at = NOW()
WHERE id = $1;


-- =============================================================================
-- SECTION 11 — VEHICLE MAINTENANCE RECORDS
-- =============================================================================

-- name: GetMaintenanceRecord :one
SELECT *
FROM vehicle_maintenance_records
WHERE id = $1;

-- name: ListMaintenanceByVehicle :many
SELECT *
FROM vehicle_maintenance_records
WHERE vehicle_id = $1
ORDER BY started_at DESC;

-- name: ListOpenMaintenanceByVehicle :many
SELECT *
FROM vehicle_maintenance_records
WHERE vehicle_id   = $1
  AND completed_at IS NULL
ORDER BY started_at DESC;

-- name: ListAllOpenMaintenance :many
SELECT
    vmr.*,
    v.plate_number,
    vd.name AS depot_name
FROM vehicle_maintenance_records vmr
JOIN vehicles       v  ON v.id  = vmr.vehicle_id
JOIN vehicle_depots vd ON vd.id = v.current_depot_id
WHERE vmr.completed_at IS NULL
ORDER BY vmr.started_at ASC;

-- name: CreateMaintenanceRecord :one
INSERT INTO vehicle_maintenance_records (
    vehicle_id, recorded_by, maintenance_type,
    description, started_at, estimated_return_at, notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: CompleteMaintenanceRecord :one
UPDATE vehicle_maintenance_records
SET
    completed_at = NOW(),
    notes        = COALESCE($2, notes)
WHERE id = $1
RETURNING *;


-- =============================================================================
-- SECTION 12 — USERS
-- =============================================================================

-- name: GetUser :one
SELECT id, username, full_name, telegram_user_id,
       telegram_linked_at, force_password_change, active,
       last_login_at, created_at, updated_at
FROM users
WHERE id = $1;

-- name: GetUserByUsername :one
-- Used during login — includes password_hash for bcrypt comparison.
SELECT *
FROM users
WHERE username = $1;

-- name: GetUserByTelegramID :one
SELECT id, username, full_name, telegram_user_id,
       telegram_linked_at, force_password_change, active,
       last_login_at, created_at, updated_at
FROM users
WHERE telegram_user_id = $1
  AND active = TRUE;

-- name: ListUsers :many
SELECT id, username, full_name, telegram_user_id,
       telegram_linked_at, force_password_change, active,
       last_login_at, created_at, updated_at
FROM users
ORDER BY full_name;

-- name: ListActiveUsers :many
SELECT id, username, full_name, telegram_user_id,
       telegram_linked_at, force_password_change, active,
       last_login_at, created_at, updated_at
FROM users
WHERE active = TRUE
ORDER BY full_name;

-- name: CreateUser :one
INSERT INTO users (username, password_hash, full_name, force_password_change)
VALUES ($1, $2, $3, $4)
RETURNING id, username, full_name, telegram_user_id,
          telegram_linked_at, force_password_change, active,
          last_login_at, created_at, updated_at;

-- name: UpdateUserPassword :exec
UPDATE users
SET
    password_hash          = $2,
    force_password_change  = FALSE,
    updated_at             = NOW()
WHERE id = $1;

-- name: SetForcePasswordChange :exec
-- Called by admin password reset flow before sending temp password via Telegram.
UPDATE users
SET force_password_change = TRUE, updated_at = NOW()
WHERE id = $1;

-- name: LinkTelegramAccount :exec
UPDATE users
SET
    telegram_user_id   = $2,
    telegram_linked_at = NOW(),
    updated_at         = NOW()
WHERE id = $1;

-- name: UnlinkTelegramAccount :exec
UPDATE users
SET
    telegram_user_id   = NULL,
    telegram_linked_at = NULL,
    updated_at         = NOW()
WHERE id = $1;

-- name: RecordUserLogin :exec
UPDATE users
SET last_login_at = NOW(), updated_at = NOW()
WHERE id = $1;

-- name: DeactivateUser :exec
UPDATE users
SET active = FALSE, updated_at = NOW()
WHERE id = $1;


-- =============================================================================
-- SECTION 13 — USER ROLE GRANTS
-- =============================================================================

-- name: GetActiveRolesForUser :many
SELECT *
FROM user_role_grants
WHERE user_id    = $1
  AND revoked_at IS NULL
ORDER BY role, scope_type;

-- name: GetActiveRoleForUserAndScope :one
SELECT *
FROM user_role_grants
WHERE user_id    = $1
  AND role       = $2
  AND scope_type = $3
  AND scope_id   = $4
  AND revoked_at IS NULL;

-- name: CheckUserHasRoleInScope :one
-- Returns TRUE if the user has the given role active for the given scope.
SELECT EXISTS (
    SELECT 1
    FROM user_role_grants
    WHERE user_id    = $1
      AND role       = $2
      AND scope_type = $3
      AND scope_id   = $4
      AND revoked_at IS NULL
) AS has_role;

-- name: CheckUserHasCompanyRole :one
SELECT EXISTS (
    SELECT 1
    FROM user_role_grants
    WHERE user_id    = $1
      AND role       = $2
      AND scope_type = 'COMPANY'
      AND revoked_at IS NULL
) AS has_role;

-- name: ListUsersWithRoleInScope :many
SELECT
    u.id,
    u.username,
    u.full_name,
    u.telegram_user_id,
    u.active,
    urg.granted_at
FROM user_role_grants urg
JOIN users u ON u.id = urg.user_id
WHERE urg.role       = $1
  AND urg.scope_type = $2
  AND urg.scope_id   = $3
  AND urg.revoked_at IS NULL
  AND u.active       = TRUE
ORDER BY u.full_name;

-- name: GrantRole :one
INSERT INTO user_role_grants (user_id, role, scope_type, scope_id, granted_by)
VALUES ($1, $2, $3, $4, $5)
ON CONFLICT (user_id, role, scope_type, scope_id)
DO UPDATE SET
    revoked_at = NULL,
    granted_by = EXCLUDED.granted_by,
    granted_at = NOW()
RETURNING *;

-- name: RevokeRole :exec
UPDATE user_role_grants
SET revoked_at = NOW()
WHERE user_id    = $1
  AND role       = $2
  AND scope_type = $3
  AND scope_id   = $4
  AND revoked_at IS NULL;

-- name: RevokeAllRolesForUser :exec
UPDATE user_role_grants
SET revoked_at = NOW()
WHERE user_id    = $1
  AND revoked_at IS NULL;


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


-- =============================================================================
-- SECTION 15 — GAS STATIONS
-- =============================================================================

-- name: GetStation :one
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
WHERE gs.id = $1;

-- name: GetStationByCode :one
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
WHERE gs.code = $1;

-- name: ListStationsByRegion :many
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
WHERE gs.region_code = $1
  AND gs.active = TRUE
ORDER BY gs.name;

-- name: ListStationsByFacility :many
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
WHERE gs.primary_facility_id = $1
  AND gs.active = TRUE
ORDER BY gs.name;

-- name: ListStationsServedByFacility :many
-- Returns all stations where the facility is in the supply whitelist.
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
JOIN station_facility_whitelist sfw ON sfw.station_id = gs.id
WHERE sfw.facility_id = $1
  AND gs.active = TRUE
ORDER BY gs.name;

-- name: ListAllActiveStations :many
SELECT
    gs.*,
    ST_X(gs.location) AS longitude,
    ST_Y(gs.location) AS latitude
FROM gas_stations gs
WHERE gs.active = TRUE
ORDER BY gs.region_code, gs.name;

-- name: CreateStation :one
INSERT INTO gas_stations (
    code, name, spbu_license_number, region_code, primary_facility_id,
    location, address, operating_hours_start, operating_hours_end,
    contact_name, contact_phone
) VALUES (
    $1, $2, $3, $4, $5,
    ST_SetSRID(ST_MakePoint($6, $7), 4326),
    $8, $9, $10, $11, $12
)
RETURNING *;

-- name: UpdateStation :one
UPDATE gas_stations
SET
    name                  = $2,
    contact_name          = $3,
    contact_phone         = $4,
    operating_hours_start = $5,
    operating_hours_end   = $6,
    address               = $7,
    updated_at            = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateStation :exec
UPDATE gas_stations
SET active = FALSE, updated_at = NOW()
WHERE id = $1;


-- =============================================================================
-- SECTION 16 — STATION FACILITY WHITELIST
-- =============================================================================

-- name: ListFacilitiesForStation :many
SELECT
    sfw.facility_id,
    rf.code AS facility_code,
    rf.name AS facility_name,
    r.name  AS refinery_name,
    (gs.primary_facility_id = sfw.facility_id) AS is_primary
FROM station_facility_whitelist sfw
JOIN refinery_facilities rf ON rf.id = sfw.facility_id
JOIN refineries           r  ON r.id  = rf.refinery_id
JOIN gas_stations         gs ON gs.id = sfw.station_id
WHERE sfw.station_id = $1;

-- name: CheckFacilityCanServeStation :one
SELECT EXISTS (
    SELECT 1
    FROM station_facility_whitelist
    WHERE station_id  = $1
      AND facility_id = $2
) AS can_serve;

-- name: AddFacilityToStationWhitelist :exec
INSERT INTO station_facility_whitelist (station_id, facility_id)
VALUES ($1, $2)
ON CONFLICT (station_id, facility_id) DO NOTHING;

-- name: RemoveFacilityFromStationWhitelist :exec
DELETE FROM station_facility_whitelist
WHERE station_id  = $1
  AND facility_id = $2;


-- =============================================================================
-- SECTION 17 — STATION QR CODES
-- =============================================================================

-- name: GetStationQRCode :one
SELECT *
FROM station_qr_codes
WHERE id = $1;

-- name: GetStationByQRPayload :one
-- Validates QR payload belongs to the expected station and is active.
SELECT
    sqr.id       AS qr_id,
    sqr.label,
    gs.id        AS station_id,
    gs.code      AS station_code,
    gs.name      AS station_name,
    gs.primary_facility_id
FROM station_qr_codes sqr
JOIN gas_stations      gs ON gs.id = sqr.station_id
WHERE sqr.qr_payload = $1
  AND sqr.active     = TRUE
  AND gs.active      = TRUE;

-- name: ListQRCodesByStation :many
SELECT *
FROM station_qr_codes
WHERE station_id = $1
ORDER BY label;

-- name: CreateStationQRCode :one
INSERT INTO station_qr_codes (station_id, qr_payload, label)
VALUES ($1, $2, $3)
RETURNING *;

-- name: DeactivateStationQRCode :exec
UPDATE station_qr_codes
SET active = FALSE
WHERE id = $1;

-- name: DeactivateAllStationQRCodes :exec
UPDATE station_qr_codes
SET active = FALSE
WHERE station_id = $1;


-- =============================================================================
-- SECTION 18 — STATION TANKS
-- =============================================================================

-- name: GetStationTank :one
SELECT *
FROM station_tanks
WHERE id = $1;

-- name: GetStationTankByFuel :one
SELECT *
FROM station_tanks
WHERE station_id    = $1
  AND fuel_type_code = $2
  AND active = TRUE;

-- name: ListStationTanksByStation :many
SELECT *
FROM station_tanks
WHERE station_id = $1
  AND active = TRUE
ORDER BY fuel_type_code;

-- name: ListStationTanksBelowReorderThreshold :many
-- Phase 3: auto-DO trigger source. Returns all under-threshold active tanks.
SELECT
    st.*,
    gs.name                AS station_name,
    gs.code                AS station_code,
    gs.primary_facility_id,
    gs.region_code,
    ROUND((st.current_volume_l / NULLIF(st.reorder_threshold_l, 0) * 100)::NUMERIC, 1) AS fill_pct
FROM station_tanks st
JOIN gas_stations  gs ON gs.id = st.station_id
WHERE st.active           = TRUE
  AND gs.active           = TRUE
  AND st.current_volume_l <= st.reorder_threshold_l
ORDER BY fill_pct ASC;

-- name: CreateStationTank :one
INSERT INTO station_tanks (
    station_id, tank_code, fuel_type_code,
    capacity_l, current_volume_l, reorder_threshold_l
) VALUES (
    $1, $2, $3, $4, $5, $6
)
RETURNING *;

-- name: UpdateStationTankVolumeAfterDelivery :one
-- Called by reconciliation engine after delivery is confirmed.
UPDATE station_tanks
SET
    current_volume_l = current_volume_l + $2,
    last_updated_at  = NOW()
WHERE id = $1
  AND active = TRUE
RETURNING *;

-- name: UpdateDipReading :one
UPDATE station_tanks
SET
    last_dip_reading_l  = $2,
    last_dip_at         = NOW(),
    last_updated_at     = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateStationTankReorderThreshold :one
UPDATE station_tanks
SET
    reorder_threshold_l = $2,
    last_updated_at     = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateStationTankVolume :one
-- Manual correction. Caller must write to audit_log.
UPDATE station_tanks
SET
    current_volume_l = $2,
    last_updated_at  = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateStationTank :exec
UPDATE station_tanks
SET active = FALSE, last_updated_at = NOW()
WHERE id = $1;


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
    do.*,
    v.plate_number  AS vehicle_plate,
    u.full_name     AS raised_by_name
FROM delivery_orders do
LEFT JOIN vehicles   v ON v.id = do.assigned_vehicle_id
LEFT JOIN users      u ON u.id = do.raised_by
WHERE do.status = $1
ORDER BY do.scheduled_date ASC, do.created_at ASC;

-- name: ListDOsForDispatchQueue :many
-- Approved DOs awaiting or having vehicle assignment, scoped to a facility.
SELECT
    do.*,
    gs.name  AS destination_name,
    gs.code  AS destination_code
FROM delivery_orders do
LEFT JOIN gas_stations gs ON gs.id = do.destination_station_id
WHERE do.origin_facility_id = $1
  AND do.status IN ('APPROVED', 'ASSIGNED')
ORDER BY do.scheduled_date ASC, do.created_at ASC;

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


-- =============================================================================
-- SECTION 20 — DELIVERY ORDER ITEMS
-- =============================================================================

-- name: GetDeliveryOrderItem :one
SELECT *
FROM delivery_order_items
WHERE id = $1;

-- name: ListDOItemsByDO :many
SELECT
    doi.*,
    ft.name     AS fuel_type_name,
    ft.category AS fuel_category
FROM delivery_order_items doi
JOIN fuel_types ft ON ft.code = doi.fuel_type_code
WHERE doi.do_id = $1
ORDER BY doi.id;

-- name: CreateDeliveryOrderItem :one
INSERT INTO delivery_order_items (
    do_id, fuel_type_code, requested_volume_l
) VALUES (
    $1, $2, $3
)
RETURNING *;

-- name: AssignCompartmentToDOItem :one
UPDATE delivery_order_items
SET
    compartment_id = $2,
    updated_at     = NOW()
WHERE id = $1
RETURNING *;

-- name: UpdateDOItemAllocatedVolume :one
UPDATE delivery_order_items
SET
    allocated_volume_l = $2,
    updated_at         = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteDOItem :exec
DELETE FROM delivery_order_items
WHERE id = $1;


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


-- =============================================================================
-- SECTION 22 — TRIPS
-- =============================================================================

-- name: GetTrip :one
SELECT
    t.*,
    ST_AsGeoJSON(t.route_polyline) AS route_geojson
FROM trips t
WHERE t.id = $1;

-- name: GetTripByDO :one
SELECT
    t.*,
    ST_AsGeoJSON(t.route_polyline) AS route_geojson
FROM trips t
WHERE t.do_id = $1;

-- name: GetTripWithDetails :one
SELECT
    t.*,
    v.plate_number,
    u.full_name     AS driver_name,
    u.telegram_user_id AS driver_telegram_id,
    gs.name         AS destination_station_name,
    gs.code         AS destination_station_code,
    rf.name         AS origin_facility_name,
    ST_AsGeoJSON(t.route_polyline) AS route_geojson
FROM trips t
JOIN vehicles             v   ON v.id  = t.vehicle_id
JOIN drivers              d   ON d.id  = t.driver_id
JOIN users                u   ON u.id  = d.user_id
JOIN refinery_facilities  rf  ON rf.id = t.origin_facility_id
LEFT JOIN gas_stations    gs  ON gs.id = t.destination_station_id
WHERE t.id = $1;

-- name: ListActiveTrips :many
-- All in-flight trips across all facilities (for dashboard map feed).
SELECT
    t.id,
    t.status,
    t.vehicle_id,
    t.driver_id,
    t.origin_facility_id,
    t.destination_station_id,
    t.departed_at,
    v.plate_number,
    u.full_name        AS driver_name,
    u.telegram_user_id AS driver_telegram_id,
    gs.name            AS destination_name
FROM trips t
JOIN vehicles          v  ON v.id  = t.vehicle_id
JOIN drivers           d  ON d.id  = t.driver_id
JOIN users             u  ON u.id  = d.user_id
LEFT JOIN gas_stations gs ON gs.id = t.destination_station_id
WHERE t.status IN ('LOADING','LOADED','IN_TRANSIT','ARRIVED','UNLOADING')
ORDER BY t.departed_at ASC NULLS LAST;

-- name: ListActiveTripsByFacility :many
SELECT t.*
FROM trips t
WHERE t.origin_facility_id = $1
  AND t.status NOT IN ('CLOSED','CANCELLED','RECONCILED')
ORDER BY t.created_at DESC;

-- name: ListTripsByDriver :many
SELECT t.*
FROM trips t
WHERE t.driver_id = $1
ORDER BY t.created_at DESC
LIMIT $2;

-- name: ListTripsByVehicle :many
SELECT t.*
FROM trips t
WHERE t.vehicle_id = $1
ORDER BY t.created_at DESC
LIMIT $2;

-- name: ListTripsByStatus :many
SELECT t.*
FROM trips t
WHERE t.status = $1
ORDER BY t.created_at DESC;

-- name: ListReturnTrips :many
-- All auto-created return-to-facility trips (have parent_trip_id set).
SELECT t.*
FROM trips t
WHERE t.parent_trip_id IS NOT NULL
ORDER BY t.created_at DESC;

-- Real-time dashboard map: latest GPS position per active trip via LATERAL join.
-- name: ListActiveTripsWithLatestGPS :many
SELECT
    t.id,
    t.status,
    t.vehicle_id,
    t.driver_id,
    t.origin_facility_id,
    t.destination_station_id,
    t.departed_at,
    v.plate_number,
    u.full_name         AS driver_name,
    gs.name             AS destination_name,
    ge.latitude         AS last_lat,
    ge.longitude        AS last_lng,
    ge.speed_kmh        AS last_speed_kmh,
    ge.event_timestamp  AS last_gps_at
FROM trips t
JOIN vehicles          v  ON v.id  = t.vehicle_id
JOIN drivers           d  ON d.id  = t.driver_id
JOIN users             u  ON u.id  = d.user_id
LEFT JOIN gas_stations gs ON gs.id = t.destination_station_id
LEFT JOIN LATERAL (
    SELECT latitude, longitude, speed_kmh, event_timestamp
    FROM gps_events
    WHERE trip_id = t.id
    ORDER BY event_timestamp DESC
    LIMIT 1
) ge ON TRUE
WHERE t.status IN ('IN_TRANSIT', 'ARRIVED', 'UNLOADING');

-- name: CreateTrip :one
INSERT INTO trips (
    do_id, vehicle_id, driver_id, status,
    destination_type, origin_facility_id,
    destination_station_id, destination_facility_id,
    parent_trip_id
) VALUES (
    $1, $2, $3, 'CREATED',
    $4, $5, $6, $7, $8
)
RETURNING *;

-- name: UpdateTripStatus :one
UPDATE trips
SET status = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SetTripDeparted :one
UPDATE trips
SET
    status     = 'IN_TRANSIT',
    departed_at = NOW(),
    updated_at  = NOW()
WHERE id = $1
RETURNING *;

-- name: SetTripArrived :one
UPDATE trips
SET
    status    = 'ARRIVED',
    arrived_at = NOW(),
    updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: SetTripCompleted :one
UPDATE trips
SET
    status       = 'DELIVERED',
    completed_at = NOW(),
    updated_at   = NOW()
WHERE id = $1
RETURNING *;

-- name: AppendTripRoutePoint :exec
-- Extends route polyline with the latest GPS coordinate (called by background worker).
UPDATE trips
SET route_polyline = ST_MakeLine(
        COALESCE(route_polyline, ST_MakePoint($2, $3)::GEOMETRY),
        ST_SetSRID(ST_MakePoint($2, $3), 4326)
    ),
    updated_at = NOW()
WHERE id = $1;


-- =============================================================================
-- SECTION 23 — TRIP EVENTS (APPEND-ONLY)
-- =============================================================================

-- name: GetTripEventByUUID :one
-- Used by idempotency check before inserting from offline queue.
SELECT *
FROM trip_events
WHERE event_uuid = $1;

-- name: ListTripEventsByTrip :many
SELECT *
FROM trip_events
WHERE trip_id = $1
ORDER BY event_timestamp ASC;

-- name: ListTripEventsByTripAndType :many
SELECT *
FROM trip_events
WHERE trip_id    = $1
  AND event_type = $2
ORDER BY event_timestamp ASC;

-- name: GetLatestTripEventByType :one
SELECT *
FROM trip_events
WHERE trip_id    = $1
  AND event_type = $2
ORDER BY event_timestamp DESC
LIMIT 1;

-- name: GetLatestTripEvent :one
SELECT *
FROM trip_events
WHERE trip_id = $1
ORDER BY event_timestamp DESC
LIMIT 1;

-- name: InsertTripEvent :one
-- Append-only. event_uuid checked for duplicate before calling this.
INSERT INTO trip_events (
    trip_id, event_uuid, event_type, event_timestamp,
    actor_user_id, location, payload
) VALUES (
    $1, $2, $3, $4, $5,
    CASE WHEN $6::FLOAT8 IS NOT NULL AND $7::FLOAT8 IS NOT NULL
         THEN ST_SetSRID(ST_MakePoint($6, $7), 4326)
         ELSE NULL
    END,
    $8
)
RETURNING *;


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


-- =============================================================================
-- SECTION 25 — COMPARTMENT SEALS
-- =============================================================================

-- name: GetSealByTripAndCompartment :one
SELECT *
FROM compartment_seals
WHERE trip_id      = $1
  AND compartment_id = $2;

-- name: ListSealsByTrip :many
SELECT
    cs.*,
    vc.compartment_number,
    ui.full_name AS issued_by_name,
    uv.full_name AS verified_by_name
FROM compartment_seals    cs
JOIN vehicle_compartments vc ON vc.id = cs.compartment_id
JOIN users                ui ON ui.id = cs.issued_by
LEFT JOIN users           uv ON uv.id = cs.verified_by
WHERE cs.trip_id = $1
ORDER BY vc.compartment_number;

-- name: ListMismatchedSealsByTrip :many
SELECT *
FROM compartment_seals
WHERE trip_id             = $1
  AND verification_status IN ('MISMATCHED', 'BROKEN', 'MISSING');

-- name: IssueSeal :one
INSERT INTO compartment_seals (
    trip_id, compartment_id, seal_number_issued, issued_by
) VALUES (
    $1, $2, $3, $4
)
RETURNING *;

-- name: VerifySeal :one
UPDATE compartment_seals
SET
    seal_number_verified = $2,
    verified_by          = $3,
    verified_at          = NOW(),
    verification_status  = CASE
        WHEN seal_number_issued = $2 THEN 'INTACT'::seal_status_t
        ELSE 'MISMATCHED'::seal_status_t
    END,
    notes = $4
WHERE id = $1
RETURNING *;

-- name: RecordSealBreak :one
UPDATE compartment_seals
SET
    verified_by         = $2,
    verified_at         = NOW(),
    verification_status = $3,
    notes               = $4
WHERE id = $1
RETURNING *;

-- name: CountSealMismatchesByTrip :one
SELECT COUNT(*)::INT AS mismatch_count
FROM compartment_seals
WHERE trip_id = $1
  AND verification_status IN ('MISMATCHED', 'BROKEN', 'MISSING');


-- =============================================================================
-- SECTION 26 — GPS EVENTS (APPEND-ONLY, PARTITIONED)
-- =============================================================================

-- name: InsertGPSEvent :one
-- Partition routing is transparent; insert to parent table.
INSERT INTO gps_events (
    trip_id, event_uuid,
    latitude, longitude,
    location,
    speed_kmh, heading_deg, accuracy_m,
    event_timestamp
) VALUES (
    $1, $2,
    $3, $4,
    ST_SetSRID(ST_MakePoint($4, $3), 4326),
    $5, $6, $7,
    $8
)
RETURNING id, trip_id, event_uuid, latitude, longitude,
          speed_kmh, heading_deg, accuracy_m,
          event_timestamp, received_at;

-- name: GetLatestGPSEventByTrip :one
SELECT id, trip_id, latitude, longitude, speed_kmh, heading_deg, event_timestamp
FROM gps_events
WHERE trip_id = $1
ORDER BY event_timestamp DESC
LIMIT 1;

-- name: ListGPSEventsByTripAndTimeRange :many
-- Route reconstruction for a trip segment.
SELECT id, trip_id, latitude, longitude, speed_kmh, heading_deg, event_timestamp
FROM gps_events
WHERE trip_id         = $1
  AND event_timestamp >= $2
  AND event_timestamp <= $3
ORDER BY event_timestamp ASC;

-- name: ListGPSEventsByTrip :many
SELECT id, trip_id, latitude, longitude, speed_kmh, heading_deg, event_timestamp
FROM gps_events
WHERE trip_id = $1
ORDER BY event_timestamp ASC;

-- name: CheckGPSEventUUIDExists :one
SELECT EXISTS (
    SELECT 1 FROM gps_events WHERE event_uuid = $1
) AS exists;


-- =============================================================================
-- SECTION 27 — TRIP PHOTOS
-- =============================================================================

-- name: GetTripPhoto :one
SELECT *
FROM trip_photos
WHERE id = $1;

-- name: ListPhotosByTrip :many
SELECT *
FROM trip_photos
WHERE trip_id = $1
ORDER BY taken_at;

-- name: ListPhotosByTripAndEvent :many
SELECT *
FROM trip_photos
WHERE trip_id    = $1
  AND event_type = $2
ORDER BY taken_at;

-- name: ListPhotosByTripAndCompartment :many
SELECT *
FROM trip_photos
WHERE trip_id      = $1
  AND compartment_id = $2
ORDER BY taken_at;

-- name: CreateTripPhoto :one
INSERT INTO trip_photos (
    trip_id, compartment_id, event_type,
    garage_object_key, file_size_bytes, mime_type,
    uploaded_by, taken_at, notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9
)
RETURNING *;

-- name: GetMandatoryPhotoCheckByTrip :one
-- Verifies all mandatory photo events are present for a trip before allowing progression.
-- Returns a bitmask-style row showing which required events have at least one photo.
-- IMPORTANT: has_compartment_sealed_photo returns TRUE if >= 1 sealed photo exists.
-- The service layer must separately verify the sealed photo count matches the trip's
-- compartment count (ListCompartmentsByVehicle) before allowing LOADING → LOADED.
SELECT
    BOOL_OR(event_type = 'WEIGHT_BRIDGE_TARE')  AS has_tare_photo,
    BOOL_OR(event_type = 'WEIGHT_BRIDGE_GROSS') AS has_gross_photo,
    BOOL_OR(event_type = 'COMPARTMENT_SEALED')   AS has_compartment_sealed_photo,
    BOOL_OR(event_type = 'STATION_TANK_BEFORE') AS has_before_photo,
    BOOL_OR(event_type = 'PUMP_METER_READING')  AS has_pump_photo,
    BOOL_OR(event_type = 'STATION_TANK_AFTER')  AS has_after_photo
FROM trip_photos
WHERE trip_id = $1;


-- =============================================================================
-- SECTION 28 — TRIP DOCUMENTS
-- =============================================================================

-- name: GetTripDocument :one
SELECT *
FROM trip_documents
WHERE id = $1;

-- name: GetTripDocumentByType :one
SELECT *
FROM trip_documents
WHERE trip_id      = $1
  AND document_type = $2;

-- name: ListDocumentsByTrip :many
SELECT *
FROM trip_documents
WHERE trip_id = $1
ORDER BY generated_at;

-- name: CreateTripDocument :one
INSERT INTO trip_documents (
    trip_id, document_type, document_number,
    garage_object_key, generated_by
) VALUES (
    $1, $2, $3, $4, $5
)
RETURNING *;

-- name: UpdateTripDocumentKey :one
-- Called if a document is regenerated (e.g., after variance resolution).
UPDATE trip_documents
SET
    garage_object_key = $2,
    generated_at      = NOW()
WHERE id = $1
RETURNING *;


-- =============================================================================
-- SECTION 29 — ROUTE DEVIATION EVENTS
-- =============================================================================

-- name: ListDeviationsByTrip :many
SELECT *
FROM route_deviation_events
WHERE trip_id = $1
ORDER BY detected_at ASC;

-- name: GetOpenDeviationByTrip :one
-- Returns the most recent unresolved deviation (for cooldown state machine).
SELECT *
FROM route_deviation_events
WHERE trip_id    = $1
  AND resolved_at IS NULL
ORDER BY detected_at DESC
LIMIT 1;

-- name: CountTripDeviations :one
-- Determines escalation tier (1 = log, 2 = warn, >2 = Telegram alert).
SELECT COUNT(*)::INT AS deviation_count
FROM route_deviation_events
WHERE trip_id = $1;

-- name: CreateDeviationEvent :one
INSERT INTO route_deviation_events (
    trip_id, detected_at, deviation_meters, occurrence_count
) VALUES (
    $1, NOW(), $2, $3
)
RETURNING *;

-- name: UpdateDeviationDuration :one
-- Called when a deviation is resolved to record total duration.
UPDATE route_deviation_events
SET
    duration_seconds = EXTRACT(EPOCH FROM (NOW() - detected_at))::INT,
    resolved_at      = NOW()
WHERE id = $1
RETURNING *;

-- name: MarkDeviationTelegramNotified :exec
UPDATE route_deviation_events
SET
    telegram_notified    = TRUE,
    telegram_notified_at = NOW()
WHERE id = $1;

-- name: ListUnnotifiedDeviationsAboveThreshold :many
-- Background worker: fetch deviations that have exceeded alert threshold and not yet notified.
SELECT rde.*, t.driver_id, t.vehicle_id
FROM route_deviation_events rde
JOIN trips t ON t.id = rde.trip_id
WHERE rde.telegram_notified = FALSE
  AND rde.resolved_at IS NULL
  AND rde.detected_at <= (NOW() - ($1 || ' minutes')::INTERVAL)
ORDER BY rde.detected_at ASC;


-- =============================================================================
-- SECTION 30 — NOTIFICATION LOG (APPEND-ONLY)
-- =============================================================================

-- name: InsertNotification :one
INSERT INTO notification_log (
    trip_id, do_id, recipient_telegram_id, recipient_user_id,
    notification_type, message_text, delivery_status, telegram_message_id
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: ListNotificationsByTrip :many
SELECT *
FROM notification_log
WHERE trip_id = $1
ORDER BY sent_at DESC;

-- name: ListNotificationsByRecipient :many
SELECT *
FROM notification_log
WHERE recipient_user_id = $1
ORDER BY sent_at DESC
LIMIT $2;

-- name: CountNotificationsByTypeAndTrip :one
SELECT COUNT(*)::INT AS count
FROM notification_log
WHERE trip_id           = $1
  AND notification_type = $2;


-- =============================================================================
-- SECTION 31 — AUDIT LOG (APPEND-ONLY)
-- =============================================================================

-- name: InsertAuditLog :one
INSERT INTO audit_log (
    user_id, action, entity_type, entity_id,
    before_state, after_state, ip_address, user_agent
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING id, created_at;

-- name: ListAuditLogByEntity :many
SELECT
    al.*,
    u.username,
    u.full_name
FROM audit_log al
LEFT JOIN users u ON u.id = al.user_id
WHERE al.entity_type = $1
  AND al.entity_id   = $2
ORDER BY al.created_at DESC
LIMIT $3;

-- name: ListAuditLogByUser :many
SELECT *
FROM audit_log
WHERE user_id = $1
ORDER BY created_at DESC
LIMIT $2;

-- name: ListAuditLogByAction :many
SELECT
    al.*,
    u.username,
    u.full_name
FROM audit_log al
LEFT JOIN users u ON u.id = al.user_id
WHERE al.action = $1
ORDER BY al.created_at DESC
LIMIT $2;


-- =============================================================================
-- SECTION 32 — TELEGRAM LINK TOKENS
-- =============================================================================

-- name: CreateTelegramLinkToken :one
INSERT INTO telegram_link_tokens (user_id, token, expires_at)
VALUES ($1, $2, NOW() + INTERVAL '48 hours')
RETURNING *;

-- name: GetValidTelegramLinkToken :one
-- Returns token only if unused and not expired.
SELECT tlt.*, u.username, u.full_name, u.telegram_user_id
FROM telegram_link_tokens tlt
JOIN users u ON u.id = tlt.user_id
WHERE tlt.token      = $1
  AND tlt.used_at    IS NULL
  AND tlt.expires_at >  NOW();

-- name: UseTelegramLinkToken :one
UPDATE telegram_link_tokens
SET used_at = NOW()
WHERE token   = $1
  AND used_at IS NULL
RETURNING *;

-- name: DeleteExpiredTelegramLinkTokens :execrows
-- Called by cron worker nightly.
DELETE FROM telegram_link_tokens
WHERE expires_at < NOW()
  AND used_at IS NOT NULL;

-- name: ListActiveTokensForUser :many
SELECT *
FROM telegram_link_tokens
WHERE user_id    = $1
  AND used_at    IS NULL
  AND expires_at > NOW()
ORDER BY created_at DESC;


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
    do.do_number,
    CASE
        WHEN wbr.approval_status = 'PENDING'   THEN 'FACILITY_MANAGER'
        WHEN wbr.approval_status = 'ESCALATED' THEN 'REFINERY_ADMIN'
        ELSE wbr.approval_status::TEXT
    END            AS required_approver_role
FROM weight_bridge_readings wbr
JOIN vehicles v  ON v.id  = wbr.vehicle_id
JOIN users    uro ON uro.id = wbr.recorded_by
LEFT JOIN trips t  ON t.id = wbr.trip_id
LEFT JOIN delivery_orders do ON do.id = t.do_id
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
