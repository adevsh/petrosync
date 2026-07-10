-- =============================================================================
-- PetroSync — Full Database Schema
-- PostgreSQL 16+ with PostGIS
--
-- Architecture decisions baked in:
--   • Multi-refinery, single-org (no tenant isolation / RLS)
--   • RBAC via user_role_grants junction table (scope-aware)
--   • Multi-compartment vehicles from day one
--   • Single-destination trips (one DO → one truck → one station)
--   • Weight bridge as source-of-truth; manual entry requires approval chain
--   • Append-only: trip_events, gps_events, audit_log, notification_log
--   • gps_events range-partitioned by month (event_timestamp)
--   • No FK on gps_events.trip_id — unsupported on partitioned tables in PG;
--     enforced at application layer
--   • Variance generated columns (loaded - delivered) stored on compartment deliveries
--   • Static QR codes at loading bays and station delivery points
--   • Photo storage via Garage (S3-compatible); object keys stored here
--   • All monetary/volumetric values in NUMERIC — no FLOAT
--
-- Run order: extensions → enums → tables → indexes → roles → seed
-- =============================================================================

-- =============================================================================
-- EXTENSIONS
-- =============================================================================

CREATE EXTENSION IF NOT EXISTS postgis;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS pgcrypto;

-- =============================================================================
-- ENUM TYPES
-- =============================================================================

CREATE TYPE fuel_category_t AS ENUM (
    'GASOLINE',
    'DIESEL'
);

CREATE TYPE user_role_t AS ENUM (
    'SYSTEM_ADMIN',
    'REFINERY_ADMIN',
    'FACILITY_MANAGER',
    'FACILITY_OPERATOR',
    'DEPOT_STAFF',
    'STATION_MANAGER',
    'DRIVER'
);

CREATE TYPE role_scope_t AS ENUM (
    'COMPANY',
    'REGION',
    'REFINERY',
    'FACILITY',
    'DEPOT',
    'STATION'
);

CREATE TYPE vehicle_status_t AS ENUM (
    'AVAILABLE',
    'ASSIGNED',
    'IN_TRANSIT',
    'UNDER_MAINTENANCE',
    'DECOMMISSIONED'
);

CREATE TYPE measurement_method_t AS ENUM (
    'WEIGHT_BRIDGE',
    'MANUAL_APPROVED'
);

CREATE TYPE approval_status_t AS ENUM (
    'PENDING',
    'APPROVED',
    'REJECTED',
    'ESCALATED'
);

CREATE TYPE do_status_t AS ENUM (
    'DRAFT',
    'PENDING_APPROVAL',
    'APPROVED',
    'ASSIGNED',
    'IN_PROGRESS',
    'DELIVERED',
    'RECONCILED',
    'CLOSED',
    'DISPUTED',
    'CANCELLED'
);

CREATE TYPE destination_type_t AS ENUM (
    'STATION',
    'REFINERY_FACILITY'         -- return-to-facility and inter-refinery transfers
);

CREATE TYPE trip_status_t AS ENUM (
    'CREATED',
    'DRIVER_ACKNOWLEDGED',
    'PRE_TRIP_INSPECTION',
    'LOADING',
    'LOADED',
    'IN_TRANSIT',
    'ARRIVED',
    'UNLOADING',
    'DELIVERED',
    'RECONCILED',
    'CLOSED',
    'DISPUTED',
    'CANCELLED'
);

CREATE TYPE trip_event_type_t AS ENUM (
    'DRIVER_ACKNOWLEDGED',
    'PRE_TRIP_INSPECTION_STARTED',
    'PRE_TRIP_INSPECTION_COMPLETED',
    'ARRIVED_AT_FACILITY',
    'LOADING_STARTED',
    'COMPARTMENT_FILLED',
    'COMPARTMENT_SEALED',
    'WEIGHT_BRIDGE_TARE_RECORDED',
    'WEIGHT_BRIDGE_GROSS_RECORDED',
    'WEIGHT_BRIDGE_APPROVED',
    'LOADING_COMPLETED',
    'DEPARTED_FACILITY',
    'ROUTE_DEVIATION_DETECTED',
    'ARRIVED_AT_DESTINATION',
    'UNLOADING_STARTED',
    'COMPARTMENT_DELIVERED',
    'SEAL_VERIFIED',
    'SEAL_MISMATCH_FLAGGED',
    'DELIVERY_COMPLETED',
    'VARIANCE_FLAGGED',
    'TRIP_CANCELLED',
    'RETURN_INITIATED'
);

CREATE TYPE photo_event_t AS ENUM (
    'WEIGHT_BRIDGE_TARE',
    'WEIGHT_BRIDGE_GROSS',
    'COMPARTMENT_SEALED',
    'STATION_TANK_BEFORE',
    'PUMP_METER_READING',
    'STATION_TANK_AFTER',
    'VARIANCE_EVIDENCE'
);

CREATE TYPE document_type_t AS ENUM (
    'DELIVERY_ORDER',
    'BILL_OF_LADING',
    'DELIVERY_RECEIPT'
);

CREATE TYPE seal_status_t AS ENUM (
    'INTACT',
    'MISMATCHED',
    'BROKEN',
    'MISSING'
);

CREATE TYPE compartment_delivery_status_t AS ENUM (
    'PENDING',
    'DELIVERED',
    'DISPUTED'
);

CREATE TYPE notification_type_t AS ENUM (
    'DO_RAISED',
    'DO_APPROVED',
    'TRIP_ASSIGNED',
    'LOADING_COMPLETE',
    'TRIP_DEPARTED',
    'DELIVERY_COMPLETE',
    'VARIANCE_FLAGGED',
    'SEAL_MISMATCH',
    'ROUTE_DEVIATION_WARN',
    'ROUTE_DEVIATION_ESCALATE',
    'MANUAL_MEASUREMENT_PENDING',
    'MANUAL_MEASUREMENT_ESCALATED',
    'DRIVER_LICENSE_EXPIRING',
    'VEHICLE_KEUR_EXPIRING',
    'RETURN_TRIP_CREATED'
);

-- =============================================================================
-- SECTION 1 — REFERENCE / CONFIG
-- =============================================================================

CREATE TABLE regions (
    code        VARCHAR(10)     PRIMARY KEY,
    name        VARCHAR(100)    NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE regions IS 'Indonesian administrative regions scoped to PetroSync coverage.';

-- ---------------------------------------------------------------------------

CREATE TABLE fuel_types (
    code                        VARCHAR(30)         PRIMARY KEY,
    name                        VARCHAR(100)        NOT NULL,
    category                    fuel_category_t     NOT NULL,
    ron_cn                      SMALLINT,           -- RON for gasoline, CN for diesel
    density_kg_per_l_at_15c     NUMERIC(6, 4)       NOT NULL,
    evaporation_factor_pct      NUMERIC(5, 3)       NOT NULL DEFAULT 0.100,
    is_subsidized               BOOLEAN             NOT NULL DEFAULT FALSE,
    active                      BOOLEAN             NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  fuel_types                        IS 'Master fuel grade catalog. Density used for weight-to-volume conversion in variance engine.';
COMMENT ON COLUMN fuel_types.density_kg_per_l_at_15c IS 'Reference density at 15°C per API standard. Adjust to Pertamina lab specs.';
COMMENT ON COLUMN fuel_types.evaporation_factor_pct  IS 'Accepted loss allowance per trip. Gasoline ~0.10%, diesel ~0.05%.';

-- ---------------------------------------------------------------------------

-- Global and per-facility configurable settings.
-- NULL facility_id = global default; facility-specific row overrides global for that facility.
-- UNIQUE enforced via two partial indexes below (NULLs break standard UNIQUE on nullable column).
CREATE TABLE system_settings (
    id              BIGSERIAL       PRIMARY KEY,
    facility_id     BIGINT,         -- FK added after refinery_facilities is created
    key             VARCHAR(100)    NOT NULL,
    value           TEXT            NOT NULL,
    description     TEXT,
    updated_by      BIGINT,         -- FK added after users is created
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  system_settings             IS 'Key-value config. NULL facility_id = global; facility row overrides global.';
COMMENT ON COLUMN system_settings.facility_id IS 'NULL = global default. Facility-specific row overrides.';

-- =============================================================================
-- SECTION 2 — ORG STRUCTURE
-- =============================================================================

CREATE TABLE refineries (
    id                  BIGSERIAL       PRIMARY KEY,
    code                VARCHAR(20)     NOT NULL UNIQUE,    -- e.g. RU-IV
    name                VARCHAR(150)    NOT NULL,
    region_code         VARCHAR(10)     NOT NULL REFERENCES regions(code),
    commissioned_year   SMALLINT,
    active              BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------

CREATE TABLE refinery_facilities (
    id                          BIGSERIAL               PRIMARY KEY,
    code                        VARCHAR(20)             NOT NULL UNIQUE,    -- e.g. FAC-CLP
    refinery_id                 BIGINT                  NOT NULL REFERENCES refineries(id),
    name                        VARCHAR(150)            NOT NULL,
    location                    GEOMETRY(POINT, 4326)   NOT NULL,
    address                     TEXT,
    is_primary                  BOOLEAN                 NOT NULL DEFAULT FALSE,
    -- Trucks dispatched from this facility must be within this radius.
    -- Island geography enforced by this value — Kaltim trucks won't cross to Java.
    max_assignment_radius_km    NUMERIC(6, 2)           NOT NULL DEFAULT 300,
    active                      BOOLEAN                 NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

-- Back-fill FK on system_settings.facility_id
ALTER TABLE system_settings
    ADD CONSTRAINT fk_system_settings_facility
    FOREIGN KEY (facility_id) REFERENCES refinery_facilities(id);

-- ---------------------------------------------------------------------------

CREATE TABLE vehicle_depots (
    id                          BIGSERIAL               PRIMARY KEY,
    code                        VARCHAR(20)             NOT NULL UNIQUE,    -- e.g. DEPOT-CLP
    name                        VARCHAR(150)            NOT NULL,
    primary_facility_id         BIGINT                  NOT NULL REFERENCES refinery_facilities(id),
    location                    GEOMETRY(POINT, 4326)   NOT NULL,
    default_truck_capacity_l    INTEGER                 NOT NULL DEFAULT 24000,
    active                      BOOLEAN                 NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN vehicle_depots.default_truck_capacity_l IS 'Default displayed in dispatch UI. Actual compartment capacity is on vehicle_compartments.';

-- ---------------------------------------------------------------------------

-- Each bay has a static QR payload (UUID-anchored string).
-- Scanning the QR during a trip event identifies the physical location.
CREATE TABLE facility_loading_bays (
    id              BIGSERIAL       PRIMARY KEY,
    facility_id     BIGINT          NOT NULL REFERENCES refinery_facilities(id),
    bay_code        VARCHAR(20)     NOT NULL,
    qr_payload      VARCHAR(255)    NOT NULL UNIQUE,
    fuel_type_code  VARCHAR(30)     REFERENCES fuel_types(code),   -- NULL = multi-grade bay
    active          BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (facility_id, bay_code)
);

-- ---------------------------------------------------------------------------

-- Source inventory: volume allocated (reserved) at DO assignment, decremented at loading.
CREATE TABLE facility_storage_tanks (
    id                  BIGSERIAL       PRIMARY KEY,
    facility_id         BIGINT          NOT NULL REFERENCES refinery_facilities(id),
    tank_code           VARCHAR(20)     NOT NULL,
    fuel_type_code      VARCHAR(30)     NOT NULL REFERENCES fuel_types(code),
    capacity_l          NUMERIC(14, 2)  NOT NULL,
    current_volume_l    NUMERIC(14, 2)  NOT NULL DEFAULT 0,
    reserved_volume_l   NUMERIC(14, 2)  NOT NULL DEFAULT 0,  -- allocated to pending DOs, not yet loaded
    min_operational_l   NUMERIC(14, 2)  NOT NULL DEFAULT 0,  -- minimum safe operating level
    active              BOOLEAN         NOT NULL DEFAULT TRUE,
    last_updated_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (facility_id, tank_code),
    CONSTRAINT chk_storage_volume_non_negative CHECK (current_volume_l >= 0 AND reserved_volume_l >= 0),
    CONSTRAINT chk_storage_not_over_capacity   CHECK (current_volume_l <= capacity_l)
);

COMMENT ON COLUMN facility_storage_tanks.reserved_volume_l IS 'Allocated to approved DOs not yet loaded. Available = current_volume_l - reserved_volume_l.';

-- =============================================================================
-- SECTION 3 — FLEET
-- =============================================================================

CREATE TABLE vehicles (
    id                      BIGSERIAL               PRIMARY KEY,
    plate_number            VARCHAR(20)             NOT NULL UNIQUE,
    chassis_number          VARCHAR(50)             NOT NULL UNIQUE,
    model                   VARCHAR(100),
    manufacture_year        SMALLINT,
    total_capacity_l        NUMERIC(10, 2)          NOT NULL,
    tare_weight_kg          NUMERIC(10, 2)          NOT NULL,   -- empty vehicle weight for weight bridge baseline
    current_depot_id        BIGINT                  REFERENCES vehicle_depots(id),
    current_location        GEOMETRY(POINT, 4326),
    status                  vehicle_status_t        NOT NULL DEFAULT 'AVAILABLE',
    keur_number             VARCHAR(50),            -- calibration certificate number
    keur_expiry             DATE,                   -- dispatch blocked if past this date
    last_inspection_date    DATE,
    next_inspection_due     DATE,
    last_assigned_at        TIMESTAMPTZ,
    notes                   TEXT,
    active                  BOOLEAN                 NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN vehicles.tare_weight_kg IS 'Used as baseline for weight bridge delta calculation. Updated after each keur calibration.';
COMMENT ON COLUMN vehicles.keur_expiry    IS 'Keur = tanker calibration certificate. Dispatch query filters out expired vehicles.';

-- ---------------------------------------------------------------------------

-- One row per physical compartment. Multi-grade deliveries use multiple compartments.
CREATE TABLE vehicle_compartments (
    id                  BIGSERIAL       PRIMARY KEY,
    vehicle_id          BIGINT          NOT NULL REFERENCES vehicles(id),
    compartment_number  SMALLINT        NOT NULL,       -- 1-based, sequential
    fuel_type_code      VARCHAR(30)     REFERENCES fuel_types(code),   -- NULL = configurable at DO assignment
    capacity_l          NUMERIC(10, 2)  NOT NULL,
    is_active           BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (vehicle_id, compartment_number)
);

-- ---------------------------------------------------------------------------

CREATE TABLE vehicle_maintenance_records (
    id                      BIGSERIAL       PRIMARY KEY,
    vehicle_id              BIGINT          NOT NULL REFERENCES vehicles(id),
    recorded_by             BIGINT,         -- FK added after users table
    maintenance_type        VARCHAR(100)    NOT NULL,
    description             TEXT,
    started_at              TIMESTAMPTZ     NOT NULL,
    estimated_return_at     TIMESTAMPTZ,
    completed_at            TIMESTAMPTZ,
    notes                   TEXT,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- SECTION 4 — PERSONNEL
-- =============================================================================

CREATE TABLE users (
    id                      BIGSERIAL       PRIMARY KEY,
    username                VARCHAR(100)    NOT NULL UNIQUE,
    password_hash           TEXT            NOT NULL,           -- bcrypt cost 12
    full_name               VARCHAR(200)    NOT NULL,
    telegram_user_id        BIGINT          UNIQUE,             -- linked after /link flow
    telegram_linked_at      TIMESTAMPTZ,
    force_password_change   BOOLEAN         NOT NULL DEFAULT TRUE,
    active                  BOOLEAN         NOT NULL DEFAULT TRUE,
    last_login_at           TIMESTAMPTZ,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN users.force_password_change IS 'Set TRUE on creation and after admin password reset. Forces change on next dashboard login.';
COMMENT ON COLUMN users.telegram_user_id      IS 'Populated after user completes /link flow with bot. Required for Telegram notifications.';

-- ---------------------------------------------------------------------------

-- One row per role+scope combination. Revoked rows kept for audit trail.
CREATE TABLE user_role_grants (
    id          BIGSERIAL       PRIMARY KEY,
    user_id     BIGINT          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        user_role_t     NOT NULL,
    scope_type  role_scope_t    NOT NULL,
    scope_id    BIGINT,                         -- NULL when scope_type = 'COMPANY'
    granted_by  BIGINT          REFERENCES users(id),
    granted_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ,
    UNIQUE (user_id, role, scope_type, scope_id)
);

COMMENT ON TABLE  user_role_grants          IS 'Active grants: WHERE revoked_at IS NULL. Multiple grants per user supported.';
COMMENT ON COLUMN user_role_grants.scope_id IS 'FK into the scoped entity table (refinery, facility, depot, or station id). NULL for COMPANY scope.';

-- ---------------------------------------------------------------------------

-- Driver-specific profile — extends users for field personnel.
CREATE TABLE drivers (
    id                  BIGSERIAL       PRIMARY KEY,
    user_id             BIGINT          NOT NULL UNIQUE REFERENCES users(id),
    employee_number     VARCHAR(50)     UNIQUE,
    sim_b2_number       VARCHAR(50)     NOT NULL,    -- hazmat tanker driving license
    sim_b2_expiry       DATE            NOT NULL,    -- dispatch blocked if expired
    home_depot_id       BIGINT          REFERENCES vehicle_depots(id),
    current_shift_start TIMESTAMPTZ,
    current_shift_end   TIMESTAMPTZ,
    is_on_shift         BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN drivers.sim_b2_expiry IS 'SIM B2 = Indonesian hazmat tanker license. Dispatch query filters out expired drivers.';
COMMENT ON COLUMN drivers.is_on_shift   IS 'Tracking only — no dispatch enforcement. Ops can see active/inactive drivers.';

-- Back-fill deferred FKs
ALTER TABLE vehicle_maintenance_records
    ADD CONSTRAINT fk_maintenance_recorded_by
    FOREIGN KEY (recorded_by) REFERENCES users(id);

ALTER TABLE system_settings
    ADD CONSTRAINT fk_system_settings_updated_by
    FOREIGN KEY (updated_by) REFERENCES users(id);

-- =============================================================================
-- SECTION 5 — GAS STATIONS
-- =============================================================================

CREATE TABLE gas_stations (
    id                      BIGSERIAL               PRIMARY KEY,
    code                    VARCHAR(20)             NOT NULL UNIQUE,    -- e.g. SPBU-07
    name                    VARCHAR(200)            NOT NULL,
    spbu_license_number     VARCHAR(50)             NOT NULL UNIQUE,    -- printed on delivery receipts
    region_code             VARCHAR(10)             NOT NULL REFERENCES regions(code),
    primary_facility_id     BIGINT                  NOT NULL REFERENCES refinery_facilities(id),
    location                GEOMETRY(POINT, 4326)   NOT NULL,
    address                 TEXT,
    operating_hours_start   TIME,                   -- tracking only; no dispatch enforcement in Phase 1
    operating_hours_end     TIME,
    contact_name            VARCHAR(200),
    contact_phone           VARCHAR(30),
    active                  BOOLEAN                 NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN gas_stations.primary_facility_id  IS 'Default supply source. Authoritative source list is station_facility_whitelist.';
COMMENT ON COLUMN gas_stations.operating_hours_start IS 'Internal tracking only — Phase 1. No dispatch block enforced until Phase 3.';

-- ---------------------------------------------------------------------------

-- Authoritative list of which facilities can serve each station.
-- primary_facility_id on gas_stations is a denormalized default for display.
CREATE TABLE station_facility_whitelist (
    station_id      BIGINT      NOT NULL REFERENCES gas_stations(id)         ON DELETE CASCADE,
    facility_id     BIGINT      NOT NULL REFERENCES refinery_facilities(id),
    PRIMARY KEY (station_id, facility_id)
);

-- ---------------------------------------------------------------------------

-- One static QR per delivery point. Scanning identifies station + validates trip state.
CREATE TABLE station_qr_codes (
    id          BIGSERIAL       PRIMARY KEY,
    station_id  BIGINT          NOT NULL REFERENCES gas_stations(id),
    qr_payload  VARCHAR(255)    NOT NULL UNIQUE,
    label       VARCHAR(100),   -- e.g. "Delivery Point A"
    active      BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------

-- Underground tanks at each station. Volume updated after each confirmed delivery.
CREATE TABLE station_tanks (
    id                  BIGSERIAL       PRIMARY KEY,
    station_id          BIGINT          NOT NULL REFERENCES gas_stations(id),
    tank_code           VARCHAR(20)     NOT NULL,
    fuel_type_code      VARCHAR(30)     NOT NULL REFERENCES fuel_types(code),
    capacity_l          NUMERIC(12, 2)  NOT NULL,
    current_volume_l    NUMERIC(12, 2)  NOT NULL DEFAULT 0,
    reorder_threshold_l NUMERIC(12, 2)  NOT NULL,       -- auto-DO trigger threshold (Phase 3)
    last_dip_reading_l  NUMERIC(12, 2),                  -- manual ATG/dip stick reading
    last_dip_at         TIMESTAMPTZ,
    active              BOOLEAN         NOT NULL DEFAULT TRUE,
    last_updated_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (station_id, tank_code),
    CONSTRAINT chk_station_volume_non_negative CHECK (current_volume_l >= 0),
    CONSTRAINT chk_station_not_over_capacity   CHECK (current_volume_l <= capacity_l)
);

-- =============================================================================
-- SECTION 6 — DELIVERY ORDERS
-- =============================================================================

CREATE TABLE delivery_orders (
    id                      BIGSERIAL           PRIMARY KEY,
    do_number               VARCHAR(30)         NOT NULL UNIQUE,    -- app-generated: DO-RU4-2026-00001
    status                  do_status_t         NOT NULL DEFAULT 'DRAFT',
    origin_facility_id      BIGINT              NOT NULL REFERENCES refinery_facilities(id),
    destination_type        destination_type_t  NOT NULL DEFAULT 'STATION',
    destination_station_id  BIGINT              REFERENCES gas_stations(id),
    destination_facility_id BIGINT              REFERENCES refinery_facilities(id),     -- inter-refinery / return
    scheduled_date          DATE                NOT NULL,
    notes                   TEXT,
    raised_by               BIGINT              NOT NULL REFERENCES users(id),
    approved_by             BIGINT              REFERENCES users(id),
    approved_at             TIMESTAMPTZ,
    assigned_vehicle_id     BIGINT              REFERENCES vehicles(id),
    assigned_driver_id      BIGINT              REFERENCES drivers(id),
    assigned_at             TIMESTAMPTZ,
    created_at              TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    -- Exactly one destination must be set, matching destination_type
    CONSTRAINT chk_do_destination CHECK (
        (destination_type = 'STATION'           AND destination_station_id IS NOT NULL AND destination_facility_id IS NULL)
        OR
        (destination_type = 'REFINERY_FACILITY' AND destination_facility_id IS NOT NULL AND destination_station_id IS NULL)
    )
);

COMMENT ON COLUMN delivery_orders.do_number IS 'Application generates: DO-{RU_CODE}-{YYYY}-{5-digit-seq}. Not DB-generated.';

-- ---------------------------------------------------------------------------

-- Line items: one per fuel grade requested in this DO.
-- compartment_id is NULL until a vehicle is assigned.
CREATE TABLE delivery_order_items (
    id                  BIGSERIAL       PRIMARY KEY,
    do_id               BIGINT          NOT NULL REFERENCES delivery_orders(id),
    fuel_type_code      VARCHAR(30)     NOT NULL REFERENCES fuel_types(code),
    compartment_id      BIGINT          REFERENCES vehicle_compartments(id),    -- assigned at dispatch
    requested_volume_l  NUMERIC(12, 2)  NOT NULL,
    allocated_volume_l  NUMERIC(12, 2),                                          -- confirmed at weight bridge
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

-- =============================================================================
-- SECTION 7 — WEIGHT BRIDGE
-- =============================================================================

-- Source-of-truth for loaded and delivered volumes.
-- TARE = truck weight before loading. GROSS = truck weight after loading.
-- Net = GROSS - TARE. Volume = Net / fuel density at ambient temp.
-- MANUAL_APPROVED readings require approval from facility_manager or refinery_admin.
CREATE TABLE weight_bridge_readings (
    id                      BIGSERIAL               PRIMARY KEY,
    trip_id                 BIGINT,                 -- FK added after trips table
    vehicle_id              BIGINT                  NOT NULL REFERENCES vehicles(id),
    reading_type            VARCHAR(5)              NOT NULL CHECK (reading_type IN ('TARE', 'GROSS')),
    weight_kg               NUMERIC(12, 2)          NOT NULL,
    method                  measurement_method_t    NOT NULL,
    ambient_temp_celsius    NUMERIC(5, 2),           -- used for temp-corrected volume calculation
    recorded_by             BIGINT                  NOT NULL REFERENCES users(id),
    -- Approval chain for MANUAL_APPROVED readings only
    approval_status         approval_status_t       NOT NULL DEFAULT 'PENDING',
    approved_by             BIGINT                  REFERENCES users(id),
    approved_at             TIMESTAMPTZ,
    escalated_at            TIMESTAMPTZ,            -- set when Facility Manager window expires
    escalated_to            BIGINT                  REFERENCES users(id),       -- Refinery Admin
    notes                   TEXT,
    created_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_wbr_weight_positive CHECK (weight_kg > 0)
);

COMMENT ON TABLE  weight_bridge_readings              IS 'WEIGHT_BRIDGE readings are auto-approved. MANUAL_APPROVED requires approval chain: Facility Manager → Refinery Admin (escalation window configurable in system_settings).';
COMMENT ON COLUMN weight_bridge_readings.reading_type IS 'TARE = empty truck before loading. GROSS = loaded truck after loading.';

-- =============================================================================
-- SECTION 8 — TRIPS
-- =============================================================================

-- One trip per DO. Return-to-facility trips reference parent_trip_id.
CREATE TABLE trips (
    id                      BIGSERIAL               PRIMARY KEY,
    do_id                   BIGINT                  NOT NULL UNIQUE REFERENCES delivery_orders(id),
    vehicle_id              BIGINT                  NOT NULL REFERENCES vehicles(id),
    driver_id               BIGINT                  NOT NULL REFERENCES drivers(id),
    status                  trip_status_t           NOT NULL DEFAULT 'CREATED',
    destination_type        destination_type_t      NOT NULL,
    origin_facility_id      BIGINT                  NOT NULL REFERENCES refinery_facilities(id),
    destination_station_id  BIGINT                  REFERENCES gas_stations(id),
    destination_facility_id BIGINT                  REFERENCES refinery_facilities(id),
    route_polyline          GEOMETRY(LINESTRING, 4326),     -- accumulated from GPS stream
    departed_at             TIMESTAMPTZ,
    arrived_at              TIMESTAMPTZ,
    completed_at            TIMESTAMPTZ,
    parent_trip_id          BIGINT                  REFERENCES trips(id),       -- set on auto-created return trips
    created_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

COMMENT ON COLUMN trips.parent_trip_id IS 'Non-null on auto-created RETURN_TO_FACILITY trips. Links back to cancelled origin trip.';
COMMENT ON COLUMN trips.route_polyline IS 'Updated incrementally by background worker from gps_events stream. Used for map display.';

-- Back-fill FK on weight_bridge_readings.trip_id
ALTER TABLE weight_bridge_readings
    ADD CONSTRAINT fk_wbr_trip
    FOREIGN KEY (trip_id) REFERENCES trips(id);

-- ---------------------------------------------------------------------------

-- Append-only audit trail of every state transition and field action.
-- petrosync_app role has no UPDATE or DELETE on this table.
CREATE TABLE trip_events (
    id              BIGSERIAL               PRIMARY KEY,
    trip_id         BIGINT                  NOT NULL REFERENCES trips(id),
    -- event_uuid: client-generated UUID v4 for idempotency. Duplicate UUIDs silently discarded.
    event_uuid      UUID                    NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    event_type      trip_event_type_t       NOT NULL,
    -- event_timestamp: client device time (Android). May arrive out of order.
    event_timestamp TIMESTAMPTZ             NOT NULL,
    received_at     TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    actor_user_id   BIGINT                  REFERENCES users(id),
    location        GEOMETRY(POINT, 4326),
    -- Flexible per-event payload. Schema documented per event_type in application code.
    payload         JSONB,
    created_at      TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE  trip_events            IS 'Append-only. petrosync_app has INSERT only — no UPDATE/DELETE. Enforced at DB role level.';
COMMENT ON COLUMN trip_events.event_uuid IS 'Client-generated UUID v4. Server discards duplicate UUIDs for idempotent offline sync.';
COMMENT ON COLUMN trip_events.event_timestamp IS 'Device time from Android. Processed ordered by this column, not received_at.';

-- ---------------------------------------------------------------------------

-- Per-compartment delivery record. Variance computed as stored generated columns.
CREATE TABLE trip_compartment_deliveries (
    id                  BIGSERIAL                       PRIMARY KEY,
    trip_id             BIGINT                          NOT NULL REFERENCES trips(id),
    compartment_id      BIGINT                          NOT NULL REFERENCES vehicle_compartments(id),
    fuel_type_code      VARCHAR(30)                     NOT NULL REFERENCES fuel_types(code),
    loaded_volume_l     NUMERIC(12, 2),                 -- from weight bridge net weight ÷ density
    loaded_weight_kg    NUMERIC(12, 2),                 -- net weight from weight bridge
    delivered_volume_l  NUMERIC(12, 2),                 -- confirmed at station
    delivered_weight_kg NUMERIC(12, 2),
    -- Variance: positive = loss (loaded > delivered), negative = over-delivery
    variance_l          NUMERIC(12, 2)  GENERATED ALWAYS AS (
                            CASE WHEN loaded_volume_l IS NOT NULL AND delivered_volume_l IS NOT NULL
                            THEN loaded_volume_l - delivered_volume_l
                            ELSE NULL END
                        ) STORED,
    variance_pct        NUMERIC(8, 4)   GENERATED ALWAYS AS (
                            CASE WHEN loaded_volume_l IS NOT NULL
                                 AND loaded_volume_l > 0
                                 AND delivered_volume_l IS NOT NULL
                            THEN (loaded_volume_l - delivered_volume_l) / loaded_volume_l * 100
                            ELSE NULL END
                        ) STORED,
    measurement_method  measurement_method_t,
    delivery_status     compartment_delivery_status_t   NOT NULL DEFAULT 'PENDING',
    created_at          TIMESTAMPTZ                     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ                     NOT NULL DEFAULT NOW(),
    UNIQUE (trip_id, compartment_id)
);

COMMENT ON COLUMN trip_compartment_deliveries.variance_l   IS 'Generated: loaded_volume_l - delivered_volume_l. Positive = loss. Compared against system_settings variance_tolerance_pct.';
COMMENT ON COLUMN trip_compartment_deliveries.variance_pct IS 'Generated: percentage loss relative to loaded volume. Dispute threshold set in system_settings.';

-- =============================================================================
-- SECTION 9 — SEALS
-- =============================================================================

-- Physical tamper-evident seal applied to each compartment hatch after loading.
-- Mismatch at delivery triggers SEAL_MISMATCH_FLAGGED trip event and Telegram alert.
CREATE TABLE compartment_seals (
    id                      BIGSERIAL       PRIMARY KEY,
    trip_id                 BIGINT          NOT NULL REFERENCES trips(id),
    compartment_id          BIGINT          NOT NULL REFERENCES vehicle_compartments(id),
    seal_number_issued      VARCHAR(100)    NOT NULL,
    issued_by               BIGINT          NOT NULL REFERENCES users(id),
    issued_at               TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    seal_number_verified    VARCHAR(100),           -- entered by driver at station
    verified_by             BIGINT          REFERENCES users(id),
    verified_at             TIMESTAMPTZ,
    verification_status     seal_status_t,
    notes                   TEXT,
    UNIQUE (trip_id, compartment_id)
);

-- =============================================================================
-- SECTION 10 — GPS EVENTS (RANGE-PARTITIONED BY MONTH)
-- =============================================================================

-- High-volume table. Range-partitioned by event_timestamp (monthly).
-- IMPORTANT: No FK on trip_id — FK constraints are unsupported on partitioned tables
--            in PostgreSQL. trip_id referential integrity enforced at application layer.
-- IMPORTANT: Use pg_partman in production for automatic monthly partition creation.
--            Manual partitions created here cover 2025-01 through 2027-12 (36 months).
CREATE TABLE gps_events (
    id              BIGINT          NOT NULL GENERATED ALWAYS AS IDENTITY,
    trip_id         BIGINT          NOT NULL,   -- No FK — partitioned table limitation
    event_uuid      UUID            NOT NULL DEFAULT gen_random_uuid(),
    latitude        NUMERIC(10, 7)  NOT NULL,
    longitude       NUMERIC(10, 7)  NOT NULL,
    location        GEOMETRY(POINT, 4326) NOT NULL,
    speed_kmh       NUMERIC(6, 2),
    heading_deg     NUMERIC(5, 2),
    accuracy_m      NUMERIC(8, 2),
    event_timestamp TIMESTAMPTZ     NOT NULL,   -- partition key; client device time
    received_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, event_timestamp)           -- partition key must be in PK
) PARTITION BY RANGE (event_timestamp);

COMMENT ON TABLE  gps_events            IS 'Append-only. Partitioned by event_timestamp monthly. No FK on trip_id (PG limitation). Use pg_partman for auto-partition in production.';
COMMENT ON COLUMN gps_events.event_uuid IS 'Client-generated UUID v4. Idempotency key — duplicate UUIDs discarded at application layer.';

-- 2025 partitions
CREATE TABLE gps_events_2025_01 PARTITION OF gps_events FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
CREATE TABLE gps_events_2025_02 PARTITION OF gps_events FOR VALUES FROM ('2025-02-01') TO ('2025-03-01');
CREATE TABLE gps_events_2025_03 PARTITION OF gps_events FOR VALUES FROM ('2025-03-01') TO ('2025-04-01');
CREATE TABLE gps_events_2025_04 PARTITION OF gps_events FOR VALUES FROM ('2025-04-01') TO ('2025-05-01');
CREATE TABLE gps_events_2025_05 PARTITION OF gps_events FOR VALUES FROM ('2025-05-01') TO ('2025-06-01');
CREATE TABLE gps_events_2025_06 PARTITION OF gps_events FOR VALUES FROM ('2025-06-01') TO ('2025-07-01');
CREATE TABLE gps_events_2025_07 PARTITION OF gps_events FOR VALUES FROM ('2025-07-01') TO ('2025-08-01');
CREATE TABLE gps_events_2025_08 PARTITION OF gps_events FOR VALUES FROM ('2025-08-01') TO ('2025-09-01');
CREATE TABLE gps_events_2025_09 PARTITION OF gps_events FOR VALUES FROM ('2025-09-01') TO ('2025-10-01');
CREATE TABLE gps_events_2025_10 PARTITION OF gps_events FOR VALUES FROM ('2025-10-01') TO ('2025-11-01');
CREATE TABLE gps_events_2025_11 PARTITION OF gps_events FOR VALUES FROM ('2025-11-01') TO ('2025-12-01');
CREATE TABLE gps_events_2025_12 PARTITION OF gps_events FOR VALUES FROM ('2025-12-01') TO ('2026-01-01');

-- 2026 partitions
CREATE TABLE gps_events_2026_01 PARTITION OF gps_events FOR VALUES FROM ('2026-01-01') TO ('2026-02-01');
CREATE TABLE gps_events_2026_02 PARTITION OF gps_events FOR VALUES FROM ('2026-02-01') TO ('2026-03-01');
CREATE TABLE gps_events_2026_03 PARTITION OF gps_events FOR VALUES FROM ('2026-03-01') TO ('2026-04-01');
CREATE TABLE gps_events_2026_04 PARTITION OF gps_events FOR VALUES FROM ('2026-04-01') TO ('2026-05-01');
CREATE TABLE gps_events_2026_05 PARTITION OF gps_events FOR VALUES FROM ('2026-05-01') TO ('2026-06-01');
CREATE TABLE gps_events_2026_06 PARTITION OF gps_events FOR VALUES FROM ('2026-06-01') TO ('2026-07-01');
CREATE TABLE gps_events_2026_07 PARTITION OF gps_events FOR VALUES FROM ('2026-07-01') TO ('2026-08-01');
CREATE TABLE gps_events_2026_08 PARTITION OF gps_events FOR VALUES FROM ('2026-08-01') TO ('2026-09-01');
CREATE TABLE gps_events_2026_09 PARTITION OF gps_events FOR VALUES FROM ('2026-09-01') TO ('2026-10-01');
CREATE TABLE gps_events_2026_10 PARTITION OF gps_events FOR VALUES FROM ('2026-10-01') TO ('2026-11-01');
CREATE TABLE gps_events_2026_11 PARTITION OF gps_events FOR VALUES FROM ('2026-11-01') TO ('2026-12-01');
CREATE TABLE gps_events_2026_12 PARTITION OF gps_events FOR VALUES FROM ('2026-12-01') TO ('2027-01-01');

-- 2027 partitions
CREATE TABLE gps_events_2027_01 PARTITION OF gps_events FOR VALUES FROM ('2027-01-01') TO ('2027-02-01');
CREATE TABLE gps_events_2027_02 PARTITION OF gps_events FOR VALUES FROM ('2027-02-01') TO ('2027-03-01');
CREATE TABLE gps_events_2027_03 PARTITION OF gps_events FOR VALUES FROM ('2027-03-01') TO ('2027-04-01');
CREATE TABLE gps_events_2027_04 PARTITION OF gps_events FOR VALUES FROM ('2027-04-01') TO ('2027-05-01');
CREATE TABLE gps_events_2027_05 PARTITION OF gps_events FOR VALUES FROM ('2027-05-01') TO ('2027-06-01');
CREATE TABLE gps_events_2027_06 PARTITION OF gps_events FOR VALUES FROM ('2027-06-01') TO ('2027-07-01');
CREATE TABLE gps_events_2027_07 PARTITION OF gps_events FOR VALUES FROM ('2027-07-01') TO ('2027-08-01');
CREATE TABLE gps_events_2027_08 PARTITION OF gps_events FOR VALUES FROM ('2027-08-01') TO ('2027-09-01');
CREATE TABLE gps_events_2027_09 PARTITION OF gps_events FOR VALUES FROM ('2027-09-01') TO ('2027-10-01');
CREATE TABLE gps_events_2027_10 PARTITION OF gps_events FOR VALUES FROM ('2027-10-01') TO ('2027-11-01');
CREATE TABLE gps_events_2027_11 PARTITION OF gps_events FOR VALUES FROM ('2027-11-01') TO ('2027-12-01');
CREATE TABLE gps_events_2027_12 PARTITION OF gps_events FOR VALUES FROM ('2027-12-01') TO ('2028-01-01');

-- =============================================================================
-- SECTION 11 — PHOTOS
-- =============================================================================

-- Photos taken on Android at mandatory scan events. Stored in Garage (S3-compatible).
-- Mandatory events: WEIGHT_BRIDGE_TARE, WEIGHT_BRIDGE_GROSS, COMPARTMENT_SEALED,
--   STATION_TANK_BEFORE, PUMP_METER_READING, STATION_TANK_AFTER, VARIANCE_EVIDENCE.
CREATE TABLE trip_photos (
    id                  BIGSERIAL           PRIMARY KEY,
    trip_id             BIGINT              NOT NULL REFERENCES trips(id),
    compartment_id      BIGINT              REFERENCES vehicle_compartments(id),    -- NULL for station/global photos
    event_type          photo_event_t       NOT NULL,
    garage_object_key   VARCHAR(500)        NOT NULL,   -- e.g. trips/{trip_id}/{event_type}/{uuid}.jpg
    file_size_bytes     BIGINT,
    mime_type           VARCHAR(50)         NOT NULL DEFAULT 'image/jpeg',
    uploaded_by         BIGINT              NOT NULL REFERENCES users(id),
    taken_at            TIMESTAMPTZ         NOT NULL,   -- EXIF timestamp from device
    uploaded_at         TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    notes               TEXT
);

COMMENT ON COLUMN trip_photos.garage_object_key IS 'Garage S3-compatible object key. Format: trips/{trip_id}/{event_type}/{uuid}.jpg';
COMMENT ON COLUMN trip_photos.taken_at          IS 'From device EXIF/metadata. May differ from uploaded_at for offline queued uploads.';

-- =============================================================================
-- SECTION 12 — DOCUMENTS
-- =============================================================================

-- Generated PDFs stored in Garage. One document per type per trip.
-- Types: DELIVERY_ORDER (at dispatch), BILL_OF_LADING (at loading complete),
--        DELIVERY_RECEIPT (at delivery confirmed).
CREATE TABLE trip_documents (
    id                  BIGSERIAL           PRIMARY KEY,
    trip_id             BIGINT              NOT NULL REFERENCES trips(id),
    document_type       document_type_t     NOT NULL,
    document_number     VARCHAR(50)         UNIQUE,     -- e.g. BOL-2026-00001
    garage_object_key   VARCHAR(500)        NOT NULL,
    generated_by        BIGINT              REFERENCES users(id),    -- NULL = system-generated
    generated_at        TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    UNIQUE (trip_id, document_type)
);

-- =============================================================================
-- SECTION 13 — ROUTE MONITORING
-- =============================================================================

-- Route deviation events per trip.
-- Policy (from system_settings):
--   occurrence_count = 1 → log only
--   occurrence_count = 2 → dashboard warning (no Telegram)
--   sustained > route_deviation_alert_minutes → Telegram escalation to supervisor
CREATE TABLE route_deviation_events (
    id                  BIGSERIAL       PRIMARY KEY,
    trip_id             BIGINT          NOT NULL REFERENCES trips(id),
    detected_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    duration_seconds    INTEGER,
    deviation_meters    NUMERIC(10, 2),
    -- Cumulative count of deviations for this trip (used for escalation policy)
    occurrence_count    SMALLINT        NOT NULL DEFAULT 1,
    telegram_notified   BOOLEAN         NOT NULL DEFAULT FALSE,
    telegram_notified_at TIMESTAMPTZ,
    resolved_at         TIMESTAMPTZ,
    notes               TEXT
);

-- =============================================================================
-- SECTION 14 — NOTIFICATIONS
-- =============================================================================

-- Append-only log of all Telegram messages sent.
-- petrosync_app has INSERT only — no UPDATE/DELETE.
CREATE TABLE notification_log (
    id                      BIGSERIAL               PRIMARY KEY,
    trip_id                 BIGINT                  REFERENCES trips(id),
    do_id                   BIGINT                  REFERENCES delivery_orders(id),
    recipient_telegram_id   BIGINT                  NOT NULL,
    recipient_user_id       BIGINT                  REFERENCES users(id),
    notification_type       notification_type_t     NOT NULL,
    message_text            TEXT                    NOT NULL,
    sent_at                 TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    delivery_status         VARCHAR(20)             NOT NULL DEFAULT 'SENT',  -- SENT | FAILED | DELIVERED
    telegram_message_id     BIGINT,                 -- from Telegram API response
    error_message           TEXT
);

COMMENT ON TABLE notification_log IS 'Append-only. petrosync_app has INSERT only — no UPDATE/DELETE.';

-- =============================================================================
-- SECTION 15 — AUDIT LOG
-- =============================================================================

-- Append-only record of all state-changing actions in the system.
-- petrosync_app has INSERT only — no UPDATE/DELETE.
CREATE TABLE audit_log (
    id              BIGSERIAL       PRIMARY KEY,
    user_id         BIGINT          REFERENCES users(id),
    action          VARCHAR(100)    NOT NULL,        -- e.g. DO_APPROVED, TRIP_CANCELLED
    entity_type     VARCHAR(100)    NOT NULL,        -- e.g. delivery_orders, trips
    entity_id       BIGINT,
    before_state    JSONB,
    after_state     JSONB,
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE audit_log IS 'Append-only. petrosync_app has INSERT only — no UPDATE/DELETE.';

-- =============================================================================
-- SECTION 16 — TELEGRAM LINKING
-- =============================================================================

-- One-time tokens for linking a user's Telegram account to their PetroSync account.
-- Flow: admin creates user → token generated → admin sends token to user via Telegram →
--       user sends /link {token} to bot → bot records telegram_user_id on users table.
CREATE TABLE telegram_link_tokens (
    id          BIGSERIAL       PRIMARY KEY,
    user_id     BIGINT          NOT NULL REFERENCES users(id),
    token       VARCHAR(64)     NOT NULL UNIQUE,         -- random hex, 32 bytes
    expires_at  TIMESTAMPTZ     NOT NULL,                -- 48-hour validity
    used_at     TIMESTAMPTZ,                             -- NULL = unused
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

COMMENT ON TABLE telegram_link_tokens IS '48-hour one-time tokens. Used once: set used_at, copy telegram_user_id to users. Expired/used tokens ignored by bot.';

-- =============================================================================
-- SECTION 17 — INDEXES
-- =============================================================================

-- ---- fuel_types ----
CREATE INDEX idx_fuel_types_category   ON fuel_types(category);
CREATE INDEX idx_fuel_types_active     ON fuel_types(active) WHERE active = TRUE;

-- ---- system_settings — partial uniques to handle NULL facility_id correctly ----
CREATE UNIQUE INDEX idx_system_settings_global   ON system_settings(key)               WHERE facility_id IS NULL;
CREATE UNIQUE INDEX idx_system_settings_facility ON system_settings(facility_id, key)  WHERE facility_id IS NOT NULL;
CREATE INDEX        idx_system_settings_facility_id ON system_settings(facility_id);

-- ---- refineries ----
CREATE INDEX idx_refineries_region ON refineries(region_code);
CREATE INDEX idx_refineries_active ON refineries(active) WHERE active = TRUE;

-- ---- refinery_facilities ----
CREATE INDEX idx_facilities_refinery ON refinery_facilities(refinery_id);
CREATE INDEX idx_facilities_location ON refinery_facilities USING GIST(location);
CREATE INDEX idx_facilities_primary  ON refinery_facilities(refinery_id, is_primary);
CREATE INDEX idx_facilities_active   ON refinery_facilities(active) WHERE active = TRUE;

-- ---- vehicle_depots ----
CREATE INDEX idx_depots_facility ON vehicle_depots(primary_facility_id);
CREATE INDEX idx_depots_location ON vehicle_depots USING GIST(location);

-- ---- facility_loading_bays ----
CREATE INDEX idx_loading_bays_facility ON facility_loading_bays(facility_id);
CREATE INDEX idx_loading_bays_qr       ON facility_loading_bays(qr_payload);
CREATE INDEX idx_loading_bays_active   ON facility_loading_bays(facility_id, active) WHERE active = TRUE;

-- ---- facility_storage_tanks ----
CREATE INDEX idx_storage_tanks_facility ON facility_storage_tanks(facility_id);
CREATE INDEX idx_storage_tanks_fuel     ON facility_storage_tanks(fuel_type_code);
CREATE INDEX idx_storage_tanks_active   ON facility_storage_tanks(facility_id, fuel_type_code) WHERE active = TRUE;

-- ---- vehicles ----
CREATE INDEX idx_vehicles_status      ON vehicles(status);
CREATE INDEX idx_vehicles_depot       ON vehicles(current_depot_id);
CREATE INDEX idx_vehicles_location    ON vehicles USING GIST(current_location);
CREATE INDEX idx_vehicles_keur_expiry ON vehicles(keur_expiry);
-- Dispatch query: available vehicles near a facility, with valid keur
CREATE INDEX idx_vehicles_dispatch    ON vehicles(status, keur_expiry)
    WHERE status = 'AVAILABLE' AND active = TRUE;

-- ---- vehicle_compartments ----
CREATE INDEX idx_compartments_vehicle ON vehicle_compartments(vehicle_id);
CREATE INDEX idx_compartments_fuel    ON vehicle_compartments(fuel_type_code);
CREATE INDEX idx_compartments_active  ON vehicle_compartments(vehicle_id, is_active) WHERE is_active = TRUE;

-- ---- vehicle_maintenance_records ----
CREATE INDEX idx_maintenance_vehicle   ON vehicle_maintenance_records(vehicle_id);
CREATE INDEX idx_maintenance_open      ON vehicle_maintenance_records(vehicle_id, completed_at)
    WHERE completed_at IS NULL;

-- ---- users ----
CREATE INDEX idx_users_username  ON users(username);
CREATE INDEX idx_users_telegram  ON users(telegram_user_id) WHERE telegram_user_id IS NOT NULL;
CREATE INDEX idx_users_active    ON users(active) WHERE active = TRUE;

-- ---- user_role_grants ----
CREATE INDEX idx_role_grants_user        ON user_role_grants(user_id);
CREATE INDEX idx_role_grants_scope       ON user_role_grants(scope_type, scope_id);
CREATE INDEX idx_role_grants_role        ON user_role_grants(role);
-- Active grants only (most frequent query pattern)
CREATE INDEX idx_role_grants_active      ON user_role_grants(user_id, role, scope_type, scope_id)
    WHERE revoked_at IS NULL;

-- ---- drivers ----
CREATE INDEX idx_drivers_user       ON drivers(user_id);
CREATE INDEX idx_drivers_depot      ON drivers(home_depot_id);
CREATE INDEX idx_drivers_sim_expiry ON drivers(sim_b2_expiry);
-- Dispatch query: on-shift drivers with valid SIM B2
CREATE INDEX idx_drivers_dispatch   ON drivers(is_on_shift, sim_b2_expiry)
    WHERE is_on_shift = TRUE;

-- ---- gas_stations ----
CREATE INDEX idx_stations_region   ON gas_stations(region_code);
CREATE INDEX idx_stations_facility ON gas_stations(primary_facility_id);
CREATE INDEX idx_stations_location ON gas_stations USING GIST(location);
CREATE INDEX idx_stations_active   ON gas_stations(active) WHERE active = TRUE;

-- ---- station_qr_codes ----
CREATE INDEX idx_station_qr_station ON station_qr_codes(station_id);
CREATE INDEX idx_station_qr_payload ON station_qr_codes(qr_payload);
CREATE INDEX idx_station_qr_active  ON station_qr_codes(qr_payload, active) WHERE active = TRUE;

-- ---- station_tanks ----
CREATE INDEX idx_station_tanks_station ON station_tanks(station_id);
CREATE INDEX idx_station_tanks_fuel    ON station_tanks(fuel_type_code);
-- Phase 3: auto-DO trigger scan (tanks below reorder threshold)
CREATE INDEX idx_station_tanks_reorder ON station_tanks(station_id, current_volume_l, reorder_threshold_l)
    WHERE active = TRUE;

-- ---- delivery_orders ----
CREATE INDEX idx_do_status           ON delivery_orders(status);
CREATE INDEX idx_do_origin           ON delivery_orders(origin_facility_id);
CREATE INDEX idx_do_destination_sta  ON delivery_orders(destination_station_id);
CREATE INDEX idx_do_scheduled        ON delivery_orders(scheduled_date);
CREATE INDEX idx_do_vehicle          ON delivery_orders(assigned_vehicle_id);
CREATE INDEX idx_do_driver           ON delivery_orders(assigned_driver_id);
CREATE INDEX idx_do_raised_by        ON delivery_orders(raised_by);
-- Dispatch queue: approved DOs pending vehicle assignment
CREATE INDEX idx_do_dispatch_queue   ON delivery_orders(origin_facility_id, status, scheduled_date)
    WHERE status IN ('APPROVED', 'ASSIGNED');

-- ---- delivery_order_items ----
CREATE INDEX idx_do_items_do          ON delivery_order_items(do_id);
CREATE INDEX idx_do_items_compartment ON delivery_order_items(compartment_id);
CREATE INDEX idx_do_items_fuel        ON delivery_order_items(fuel_type_code);

-- ---- weight_bridge_readings ----
CREATE INDEX idx_wbr_trip     ON weight_bridge_readings(trip_id);
CREATE INDEX idx_wbr_vehicle  ON weight_bridge_readings(vehicle_id);
-- Pending approvals dashboard (ops manager view)
CREATE INDEX idx_wbr_pending  ON weight_bridge_readings(approval_status, created_at)
    WHERE approval_status IN ('PENDING', 'ESCALATED');

-- ---- trips — heavily queried table ----
CREATE INDEX idx_trips_do              ON trips(do_id);
CREATE INDEX idx_trips_vehicle         ON trips(vehicle_id);
CREATE INDEX idx_trips_driver          ON trips(driver_id);
CREATE INDEX idx_trips_status          ON trips(status);
CREATE INDEX idx_trips_origin          ON trips(origin_facility_id);
CREATE INDEX idx_trips_destination_sta ON trips(destination_station_id);
CREATE INDEX idx_trips_parent          ON trips(parent_trip_id) WHERE parent_trip_id IS NOT NULL;
CREATE INDEX idx_trips_departed        ON trips(departed_at);
-- Real-time map query: active trips only (WebSocket feed + dashboard map)
CREATE INDEX idx_trips_active          ON trips(status, vehicle_id)
    WHERE status IN ('LOADING','LOADED','IN_TRANSIT','ARRIVED','UNLOADING');

-- ---- trip_events — append-only, high-write ----
CREATE INDEX idx_trip_events_trip      ON trip_events(trip_id);
CREATE INDEX idx_trip_events_uuid      ON trip_events(event_uuid);
CREATE INDEX idx_trip_events_type      ON trip_events(event_type);
CREATE INDEX idx_trip_events_time      ON trip_events(event_timestamp);
-- Primary access pattern: all events for a trip in chronological order
CREATE INDEX idx_trip_events_trip_time ON trip_events(trip_id, event_timestamp);
-- State machine queries: latest event of a given type for a trip
CREATE INDEX idx_trip_events_trip_type ON trip_events(trip_id, event_type);

-- ---- trip_compartment_deliveries ----
CREATE INDEX idx_tcd_trip        ON trip_compartment_deliveries(trip_id);
CREATE INDEX idx_tcd_compartment ON trip_compartment_deliveries(compartment_id);
CREATE INDEX idx_tcd_status      ON trip_compartment_deliveries(delivery_status);
-- Variance report: disputed deliveries
CREATE INDEX idx_tcd_disputed    ON trip_compartment_deliveries(trip_id, delivery_status)
    WHERE delivery_status = 'DISPUTED';

-- ---- compartment_seals ----
CREATE INDEX idx_seals_trip        ON compartment_seals(trip_id);
CREATE INDEX idx_seals_compartment ON compartment_seals(compartment_id);
-- Mismatch alert query
CREATE INDEX idx_seals_mismatch    ON compartment_seals(trip_id, verification_status)
    WHERE verification_status IN ('MISMATCHED','BROKEN','MISSING');

-- ---- gps_events — partitioned; index on parent propagates to all partitions ----
CREATE INDEX idx_gps_trip      ON gps_events(trip_id);
CREATE INDEX idx_gps_uuid      ON gps_events(event_uuid);
-- Most common: reconstruct route for a trip (trip_id + time range)
CREATE INDEX idx_gps_trip_time ON gps_events(trip_id, event_timestamp DESC);
-- Spatial query (geofencing checks)
CREATE INDEX idx_gps_location  ON gps_events USING GIST(location);

-- ---- trip_photos ----
CREATE INDEX idx_photos_trip      ON trip_photos(trip_id);
CREATE INDEX idx_photos_event     ON trip_photos(event_type);
CREATE INDEX idx_photos_trip_event ON trip_photos(trip_id, event_type);

-- ---- trip_documents ----
CREATE INDEX idx_docs_trip ON trip_documents(trip_id);
CREATE INDEX idx_docs_type ON trip_documents(document_type);

-- ---- route_deviation_events ----
CREATE INDEX idx_deviations_trip      ON route_deviation_events(trip_id);
CREATE INDEX idx_deviations_unresolved ON route_deviation_events(trip_id, resolved_at)
    WHERE resolved_at IS NULL;

-- ---- notification_log ----
CREATE INDEX idx_notif_trip      ON notification_log(trip_id);
CREATE INDEX idx_notif_recipient ON notification_log(recipient_telegram_id);
CREATE INDEX idx_notif_type      ON notification_log(notification_type);
CREATE INDEX idx_notif_sent      ON notification_log(sent_at);

-- ---- audit_log ----
CREATE INDEX idx_audit_user    ON audit_log(user_id);
CREATE INDEX idx_audit_entity  ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_action  ON audit_log(action);
CREATE INDEX idx_audit_created ON audit_log(created_at);

-- ---- telegram_link_tokens ----
CREATE INDEX idx_tg_tokens_user    ON telegram_link_tokens(user_id);
CREATE INDEX idx_tg_tokens_valid   ON telegram_link_tokens(token, expires_at)
    WHERE used_at IS NULL;

-- =============================================================================
-- SECTION 18 — DATABASE ROLES & APPEND-ONLY ENFORCEMENT
-- =============================================================================

-- Application role: used by Go API (Gin)
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'petrosync_app') THEN
        CREATE ROLE petrosync_app LOGIN PASSWORD 'change_me_in_production';
    END IF;
END $$;

-- Read-only role: analytics, reporting tools, Metabase, etc.
DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'petrosync_readonly') THEN
        CREATE ROLE petrosync_readonly LOGIN PASSWORD 'change_me_in_production';
    END IF;
END $$;

GRANT USAGE ON SCHEMA public TO petrosync_app;
GRANT USAGE ON SCHEMA public TO petrosync_readonly;

-- Grant full DML to application role
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES    IN SCHEMA public TO petrosync_app;
GRANT USAGE, SELECT                  ON ALL SEQUENCES IN SCHEMA public TO petrosync_app;

-- Read-only access
GRANT SELECT ON ALL TABLES IN SCHEMA public TO petrosync_readonly;

-- Enforce append-only on audit-trail tables
-- (INSERT is still permitted; only UPDATE/DELETE are revoked)
REVOKE UPDATE, DELETE ON trip_events      FROM petrosync_app;
REVOKE UPDATE, DELETE ON audit_log        FROM petrosync_app;
REVOKE UPDATE, DELETE ON notification_log FROM petrosync_app;

-- GPS partitioned table: revoke on all existing partitions
DO $$
DECLARE
    r RECORD;
BEGIN
    FOR r IN
        SELECT c.relname
        FROM pg_class c
        JOIN pg_inherits i ON i.inhrelid = c.oid
        JOIN pg_class p    ON i.inhparent = p.oid
        WHERE p.relname = 'gps_events'
          AND c.relkind = 'r'
    LOOP
        EXECUTE format('REVOKE UPDATE, DELETE ON %I FROM petrosync_app', r.relname);
    END LOOP;
END $$;

-- Ensure future objects created by superuser are accessible
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES    TO petrosync_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT USAGE, SELECT                  ON SEQUENCES TO petrosync_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public
    GRANT SELECT ON TABLES TO petrosync_readonly;

-- =============================================================================
-- SECTION 19 — SEED DATA
-- =============================================================================

-- ---- Regions ----
INSERT INTO regions (code, name) VALUES
    ('RIAU',    'Riau'),
    ('SUMSEL',  'Sumatera Selatan'),
    ('JATENG',  'Jawa Tengah'),
    ('KALTIM',  'Kalimantan Timur'),
    ('JABAR',   'Jawa Barat');

-- ---- Fuel Types ----
INSERT INTO fuel_types (code, name, category, ron_cn, density_kg_per_l_at_15c, evaporation_factor_pct, is_subsidized) VALUES
    ('PERTALITE',       'Pertalite',       'GASOLINE', 90, 0.7150, 0.100, TRUE),
    ('PERTAMAX',        'Pertamax',        'GASOLINE', 92, 0.7200, 0.100, FALSE),
    ('PERTAMAX_TURBO',  'Pertamax Turbo',  'GASOLINE', 98, 0.7400, 0.100, FALSE),
    ('BIOSOLAR',        'Biosolar B35',    'DIESEL',   48, 0.8450, 0.050, TRUE),
    ('DEXLITE',         'Dexlite',         'DIESEL',   51, 0.8200, 0.050, FALSE),
    ('PERTAMINA_DEX',   'Pertamina Dex',   'DIESEL',   53, 0.8300, 0.050, FALSE);

-- ---- Global System Settings ----
INSERT INTO system_settings (facility_id, key, value, description) VALUES
    (NULL, 'approval_escalation_hours',     '2',    'Hours before manual weight bridge approval auto-escalates from Facility Manager to Refinery Admin'),
    (NULL, 'variance_tolerance_pct',        '0.3',  'Variance % threshold; above this value compartment_delivery_status set to DISPUTED'),
    (NULL, 'gps_ping_interval_seconds',     '30',   'GPS ping frequency (seconds) from Android while a trip is IN_TRANSIT'),
    (NULL, 'route_deviation_warn_count',    '2',    'Deviation occurrence count within one trip before dashboard warning is raised'),
    (NULL, 'route_deviation_alert_minutes', '15',   'Sustained deviation duration (minutes) before Telegram escalation to supervisor'),
    (NULL, 'dispatch_candidate_limit',      '5',    'Max candidate trucks shown to dispatcher on DO assignment screen');

-- ---- Refineries ----
INSERT INTO refineries (code, name, region_code, commissioned_year) VALUES
    ('RU-II',  'Refinery Unit II Dumai',     'RIAU',   1971),
    ('RU-III', 'Refinery Unit III Plaju',    'SUMSEL', 1926),
    ('RU-IV',  'Refinery Unit IV Cilacap',   'JATENG', 1974),
    ('RU-V',   'Refinery Unit V Balikpapan', 'KALTIM', 1922),
    ('RU-VI',  'Refinery Unit VI Balongan',  'JABAR',  1994);

-- ---- Refinery Facilities ----
-- ST_MakePoint(longitude, latitude) — note coordinate order
INSERT INTO refinery_facilities (code, refinery_id, name, location, is_primary, max_assignment_radius_km) VALUES
    ('FAC-DUM', (SELECT id FROM refineries WHERE code = 'RU-II'),  'Dumai',           ST_SetSRID(ST_MakePoint(101.4264,  1.6573), 4326), TRUE,  250),
    ('FAC-SKP', (SELECT id FROM refineries WHERE code = 'RU-II'),  'Sungai Pakning',  ST_SetSRID(ST_MakePoint(102.1276,  1.3560), 4326), FALSE, 200),
    ('FAC-PLJ', (SELECT id FROM refineries WHERE code = 'RU-III'), 'Plaju',           ST_SetSRID(ST_MakePoint(104.8087, -2.9823), 4326), TRUE,  300),
    ('FAC-SGR', (SELECT id FROM refineries WHERE code = 'RU-III'), 'Sungai Gerong',   ST_SetSRID(ST_MakePoint(104.8215, -2.9667), 4326), FALSE, 300),
    ('FAC-CLP', (SELECT id FROM refineries WHERE code = 'RU-IV'),  'Cilacap',         ST_SetSRID(ST_MakePoint(108.9916, -7.7250), 4326), TRUE,  350),
    ('FAC-BPP', (SELECT id FROM refineries WHERE code = 'RU-V'),   'Balikpapan',      ST_SetSRID(ST_MakePoint(116.8301, -1.2675), 4326), TRUE,  500),
    ('FAC-BLG', (SELECT id FROM refineries WHERE code = 'RU-VI'),  'Balongan',        ST_SetSRID(ST_MakePoint(108.2667, -6.3700), 4326), TRUE,  250);

-- ---- Vehicle Depots ----
-- RU-III: Plaju and Sungai Gerong share DEPOT-PLJ (~2km apart)
INSERT INTO vehicle_depots (code, name, primary_facility_id, location, default_truck_capacity_l) VALUES
    ('DEPOT-DUM', 'Depot Dumai',          (SELECT id FROM refinery_facilities WHERE code = 'FAC-DUM'), ST_SetSRID(ST_MakePoint(101.4264,  1.6573), 4326), 24000),
    ('DEPOT-SKP', 'Depot Sungai Pakning', (SELECT id FROM refinery_facilities WHERE code = 'FAC-SKP'), ST_SetSRID(ST_MakePoint(102.1276,  1.3560), 4326), 24000),
    ('DEPOT-PLJ', 'Depot Plaju',          (SELECT id FROM refinery_facilities WHERE code = 'FAC-PLJ'), ST_SetSRID(ST_MakePoint(104.8000, -2.9750), 4326), 24000),
    ('DEPOT-CLP', 'Depot Cilacap',        (SELECT id FROM refinery_facilities WHERE code = 'FAC-CLP'), ST_SetSRID(ST_MakePoint(108.9916, -7.7250), 4326), 24000),
    ('DEPOT-BPP', 'Depot Balikpapan',     (SELECT id FROM refinery_facilities WHERE code = 'FAC-BPP'), ST_SetSRID(ST_MakePoint(116.8301, -1.2675), 4326), 24000),
    ('DEPOT-BLG', 'Depot Balongan',       (SELECT id FROM refinery_facilities WHERE code = 'FAC-BLG'), ST_SetSRID(ST_MakePoint(108.2667, -6.3700), 4326), 24000);

-- ---- Facility Loading Bays (2 per primary facility) ----
INSERT INTO facility_loading_bays (facility_id, bay_code, qr_payload)
SELECT
    f.id,
    'BAY-' || LPAD(n::TEXT, 2, '0'),
    'LB-' || f.code || '-BAY' || LPAD(n::TEXT, 2, '0') || '-' || gen_random_uuid()
FROM refinery_facilities f
CROSS JOIN generate_series(1, 2) n
WHERE f.is_primary = TRUE;

-- ---- Facility Storage Tanks (primary facilities, all 6 fuel grades) ----
-- Capacities are illustrative; update to actual tank specs before go-live.
INSERT INTO facility_storage_tanks (facility_id, tank_code, fuel_type_code, capacity_l, current_volume_l, min_operational_l)
SELECT
    f.id,
    'STK-' || ft.code,
    ft.code,
    CASE ft.category WHEN 'GASOLINE' THEN 5000000 ELSE 3000000 END,
    CASE ft.category WHEN 'GASOLINE' THEN 3000000 ELSE 1800000 END,
    CASE ft.category WHEN 'GASOLINE' THEN  500000 ELSE  300000 END
FROM refinery_facilities f
CROSS JOIN fuel_types ft
WHERE f.is_primary = TRUE
  AND ft.active = TRUE;

-- ---- Users (all passwords: "password", bcrypt cost 12, individual salts) ----
-- superadmin: force_password_change = FALSE (primary admin account)
-- all others: force_password_change = TRUE (must change on first login)
INSERT INTO users (username, password_hash, full_name, force_password_change) VALUES
    ('superadmin',      '$2b$12$xOtGv7.yp1R8AMf.Bq4zFOMfZ/n6Kb5WpmDs7n4MugIPln1ZebVmK', 'Super Administrator',         FALSE),
    ('operator.ru2',    '$2b$12$7sM8LftNd2an1AvNkKICYuepjoOLhVbtsw16YOqoFvt7kiFE1.gC2', 'Operator RU II Dumai',        TRUE),
    ('operator.ru3',    '$2b$12$giRyqUS7SXvfYWNh.rrO8uz71.MTPpkTbLMJma/SG2IdwnfneoBTe', 'Operator RU III Plaju',       TRUE),
    ('operator.ru4',    '$2b$12$siXE2X2id7Ge21HkrheHUOdr22slb8VswdJoyPS9gERVqNw8ZiGuK', 'Operator RU IV Cilacap',      TRUE),
    ('operator.ru5',    '$2b$12$kW1opWxv7XcvwmJ4Jpoh8.J6b.pmNPpLkM4.dN.nsLPcpxj9AjAqO', 'Operator RU V Balikpapan',   TRUE),
    ('operator.ru6',    '$2b$12$LExZ9bYFGVadUq8Zsn1WKuKYUUwgh.Lf9BVMtAFotJUhhDodDCOCW', 'Operator RU VI Balongan',    TRUE),
    ('driver.01',       '$2b$12$pmwTDOw57E/EVeuFukv8o./l.IRDc4RN873dKTXaVGLfS0Ya2txbi', 'Budi Santoso',                TRUE),
    ('operator.spbu01', '$2b$12$Hbl2RW8fCzaXJy73q3Otjer2.Z.D2cmRsXdPve04Wq35iDdvBwPcK', 'Operator SPBU Palembang',    TRUE),
    ('operator.spbu02', '$2b$12$8yHlD/069rTsFm2LfIGY6e1y034CbvUC.9/iv4hbcmzRmOeF3UNEK', 'Operator SPBU Pekanbaru',    TRUE),
    ('operator.spbu03', '$2b$12$4zZYi28sFfG9.K0g9uw1X.RULC40YQ4/MiVludwrwSWzCKzAISnZK', 'Operator SPBU Jambi',        TRUE),
    ('operator.spbu04', '$2b$12$0HtMr2cZ3ZjTBTqVuhiYiOpLPJMxWIsu/1CVvpiAZWHFAb.nNReEi', 'Operator SPBU Balikpapan',   TRUE),
    ('operator.spbu05', '$2b$12$pOIfjdF36GMhUnFVZuke7Oswqt5L73wV6PSqDn9q8tpYz8kvVBGZG', 'Operator SPBU Samarinda',    TRUE),
    ('operator.spbu06', '$2b$12$MZq8cZLTMpxksoANesPK9OQKvuSNnq3jHmKPgdT2N.nRCSXKKvkNW', 'Operator SPBU Bontang',      TRUE),
    ('operator.spbu07', '$2b$12$lvw2MYBkftxpbCdCdTpVMeuWNuX/8tmNTTyqERaQy5..lk7VNvAnG', 'Operator SPBU Semarang',     TRUE),
    ('operator.spbu08', '$2b$12$FSvRBhIQTb0jPka9nvmC0eMhE.ZIc7UXAtEjlzdqQFPRjcL1EQTRm', 'Operator SPBU Yogyakarta',   TRUE),
    ('operator.spbu09', '$2b$12$rlDRXvaBvduNIrd4qe4bwey3RkU71iPpEUWLO324gpbo9RmKt7DTi', 'Operator SPBU Solo',          TRUE),
    ('operator.spbu10', '$2b$12$O0GNskKmmzWA7dYnEfCCNO/nuDEmfQeSlUvzRbRi4OlecvTOYYxo.', 'Operator SPBU Bandung',      TRUE),
    ('operator.spbu11', '$2b$12$aakZ9gybKF3SWcrUE9BU..YFOq7YT77EiKhOvKakcMTuz8sLXB.RS', 'Operator SPBU Cirebon',      TRUE),
    ('operator.spbu12', '$2b$12$X0Mhk2BkBtKg.SJiRnNJDO97do2eE/.BY4OF.iV0XRXI1jlKVy2Ti', 'Operator SPBU Subang',       TRUE);

-- ---- User Role Grants ----
-- superadmin → SYSTEM_ADMIN / COMPANY
INSERT INTO user_role_grants (user_id, role, scope_type, scope_id)
VALUES ((SELECT id FROM users WHERE username = 'superadmin'), 'SYSTEM_ADMIN', 'COMPANY', NULL);

-- Refinery operators → FACILITY_OPERATOR scoped to primary facility of their RU
INSERT INTO user_role_grants (user_id, role, scope_type, scope_id)
SELECT u.id, 'FACILITY_OPERATOR', 'FACILITY', f.id
FROM (VALUES
    ('operator.ru2', 'FAC-DUM'),
    ('operator.ru3', 'FAC-PLJ'),
    ('operator.ru4', 'FAC-CLP'),
    ('operator.ru5', 'FAC-BPP'),
    ('operator.ru6', 'FAC-BLG')
) AS m(username, fac_code)
JOIN users              u ON u.username = m.username
JOIN refinery_facilities f ON f.code    = m.fac_code;

-- Driver → DRIVER / COMPANY (can be assigned to any depot)
INSERT INTO user_role_grants (user_id, role, scope_type, scope_id)
VALUES ((SELECT id FROM users WHERE username = 'driver.01'), 'DRIVER', 'COMPANY', NULL);

-- ---- Driver Profile ----
INSERT INTO drivers (user_id, employee_number, sim_b2_number, sim_b2_expiry, home_depot_id)
VALUES (
    (SELECT id FROM users  WHERE username = 'driver.01'),
    'EMP-DRV-001',
    'SIM-B2-JTG-2024-00001',
    '2027-12-31',
    (SELECT id FROM vehicle_depots WHERE code = 'DEPOT-CLP')
);

-- ---- Gas Stations ----
INSERT INTO gas_stations (code, name, spbu_license_number, region_code, primary_facility_id, location, address, operating_hours_start, operating_hours_end) VALUES
    -- Sumatra (3 stations)
    ('SPBU-01', 'SPBU Palembang Ilir Timur',   'SPBU-ID-SUM-2024-001', 'SUMSEL',
        (SELECT id FROM refinery_facilities WHERE code = 'FAC-PLJ'),
        ST_SetSRID(ST_MakePoint(104.7619, -2.9909), 4326),
        'Jl. Jenderal Sudirman No.1, Ilir Timur I, Palembang, Sumatera Selatan',
        '05:00', '23:00'),
    ('SPBU-02', 'SPBU Pekanbaru Sail',          'SPBU-ID-SUM-2024-002', 'RIAU',
        (SELECT id FROM refinery_facilities WHERE code = 'FAC-DUM'),
        ST_SetSRID(ST_MakePoint(101.4478, 0.5071), 4326),
        'Jl. Tuanku Tambusai, Sail, Pekanbaru, Riau',
        '06:00', '22:00'),
    ('SPBU-03', 'SPBU Jambi Telanaipura',        'SPBU-ID-SUM-2024-003', 'SUMSEL',
        (SELECT id FROM refinery_facilities WHERE code = 'FAC-PLJ'),
        ST_SetSRID(ST_MakePoint(103.6131, -1.6101), 4326),
        'Jl. Gatot Subroto, Telanaipura, Jambi',
        '06:00', '22:00'),
    -- Kalimantan (3 stations)
    ('SPBU-04', 'SPBU Balikpapan Klandasan',     'SPBU-ID-KAL-2024-001', 'KALTIM',
        (SELECT id FROM refinery_facilities WHERE code = 'FAC-BPP'),
        ST_SetSRID(ST_MakePoint(116.8250, -1.2659), 4326),
        'Jl. Jenderal Sudirman, Klandasan Ulu, Balikpapan, Kalimantan Timur',
        '00:00', '23:59'),
    ('SPBU-05', 'SPBU Samarinda Sungai Kunjang', 'SPBU-ID-KAL-2024-002', 'KALTIM',
        (SELECT id FROM refinery_facilities WHERE code = 'FAC-BPP'),
        ST_SetSRID(ST_MakePoint(117.1253, -0.4948), 4326),
        'Jl. MT Haryono, Sungai Kunjang, Samarinda, Kalimantan Timur',
        '05:00', '23:00'),
    ('SPBU-06', 'SPBU Bontang Bontang Baru',     'SPBU-ID-KAL-2024-003', 'KALTIM',
        (SELECT id FROM refinery_facilities WHERE code = 'FAC-BPP'),
        ST_SetSRID(ST_MakePoint(117.5000, 0.1333), 4326),
        'Jl. Awang Long, Bontang Baru, Bontang, Kalimantan Timur',
        '06:00', '22:00'),
    -- Java — RU-IV area (3 stations)
    ('SPBU-07', 'SPBU Semarang Gajahmungkur',    'SPBU-ID-JAV-2024-001', 'JATENG',
        (SELECT id FROM refinery_facilities WHERE code = 'FAC-CLP'),
        ST_SetSRID(ST_MakePoint(110.4083, -6.9854), 4326),
        'Jl. Mgr Soegiyopranoto, Gajahmungkur, Semarang, Jawa Tengah',
        '00:00', '23:59'),
    ('SPBU-08', 'SPBU Yogyakarta Gondomanan',    'SPBU-ID-JAV-2024-002', 'JATENG',
        (SELECT id FROM refinery_facilities WHERE code = 'FAC-CLP'),
        ST_SetSRID(ST_MakePoint(110.3643, -7.7956), 4326),
        'Jl. Jenderal Sudirman, Gondomanan, Yogyakarta, Daerah Istimewa Yogyakarta',
        '00:00', '23:59'),
    ('SPBU-09', 'SPBU Solo Laweyan',             'SPBU-ID-JAV-2024-003', 'JATENG',
        (SELECT id FROM refinery_facilities WHERE code = 'FAC-CLP'),
        ST_SetSRID(ST_MakePoint(110.8003, -7.5563), 4326),
        'Jl. Slamet Riyadi, Laweyan, Surakarta, Jawa Tengah',
        '05:00', '23:00'),
    -- Java — RU-VI area (3 stations)
    ('SPBU-10', 'SPBU Bandung Coblong',          'SPBU-ID-JAV-2024-004', 'JABAR',
        (SELECT id FROM refinery_facilities WHERE code = 'FAC-BLG'),
        ST_SetSRID(ST_MakePoint(107.6191, -6.8905), 4326),
        'Jl. Ir. H. Juanda, Coblong, Bandung, Jawa Barat',
        '00:00', '23:59'),
    ('SPBU-11', 'SPBU Cirebon Kejaksan',         'SPBU-ID-JAV-2024-005', 'JABAR',
        (SELECT id FROM refinery_facilities WHERE code = 'FAC-BLG'),
        ST_SetSRID(ST_MakePoint(108.5522, -6.7321), 4326),
        'Jl. Siliwangi, Kejaksan, Cirebon, Jawa Barat',
        '05:00', '23:00'),
    ('SPBU-12', 'SPBU Subang Kota',              'SPBU-ID-JAV-2024-006', 'JABAR',
        (SELECT id FROM refinery_facilities WHERE code = 'FAC-BLG'),
        ST_SetSRID(ST_MakePoint(107.7589, -6.5701), 4326),
        'Jl. Otto Iskandar Dinata, Subang, Jawa Barat',
        '06:00', '22:00');

-- ---- Station Facility Whitelist ----
INSERT INTO station_facility_whitelist (station_id, facility_id)
SELECT s.id, f.id
FROM (VALUES
    -- Sumatra: served by both RU-II and RU-III facilities
    ('SPBU-01', 'FAC-PLJ'), ('SPBU-01', 'FAC-SGR'),
    ('SPBU-02', 'FAC-DUM'), ('SPBU-02', 'FAC-SKP'),
    ('SPBU-03', 'FAC-PLJ'), ('SPBU-03', 'FAC-SGR'),
    -- Kalimantan: single facility
    ('SPBU-04', 'FAC-BPP'),
    ('SPBU-05', 'FAC-BPP'),
    ('SPBU-06', 'FAC-BPP'),
    -- Java RU-IV
    ('SPBU-07', 'FAC-CLP'),
    ('SPBU-08', 'FAC-CLP'),
    ('SPBU-09', 'FAC-CLP'),
    -- Java RU-VI
    ('SPBU-10', 'FAC-BLG'),
    ('SPBU-11', 'FAC-BLG'),
    ('SPBU-12', 'FAC-BLG')
) AS m(station_code, fac_code)
JOIN gas_stations        s ON s.code = m.station_code
JOIN refinery_facilities f ON f.code = m.fac_code;

-- ---- Station QR Codes (one per station) ----
INSERT INTO station_qr_codes (station_id, qr_payload, label)
SELECT id, 'STA-QR-' || code || '-' || gen_random_uuid(), 'Delivery Point A'
FROM gas_stations;

-- ---- Station Tanks (Pertalite + Biosolar for all stations; Pertamax for Java stations) ----
-- Base: all stations get Pertalite + Biosolar underground tanks
INSERT INTO station_tanks (station_id, tank_code, fuel_type_code, capacity_l, current_volume_l, reorder_threshold_l)
SELECT
    s.id,
    'TK-' || ft.code,
    ft.code,
    CASE ft.category WHEN 'GASOLINE' THEN 32000 ELSE 24000 END,
    CASE ft.category WHEN 'GASOLINE' THEN 16000 ELSE 12000 END,
    CASE ft.category WHEN 'GASOLINE' THEN  8000 ELSE  6000 END
FROM gas_stations s
CROSS JOIN fuel_types ft
WHERE ft.code IN ('PERTALITE', 'BIOSOLAR');

-- Additional: Pertamax for Java stations (SPBU-07 through SPBU-12)
INSERT INTO station_tanks (station_id, tank_code, fuel_type_code, capacity_l, current_volume_l, reorder_threshold_l)
SELECT
    s.id,
    'TK-' || ft.code,
    ft.code,
    16000,
    8000,
    4000
FROM gas_stations s
CROSS JOIN fuel_types ft
WHERE s.code BETWEEN 'SPBU-07' AND 'SPBU-12'
  AND ft.code = 'PERTAMAX';

-- ---- Station Manager Role Grants ----
INSERT INTO user_role_grants (user_id, role, scope_type, scope_id)
SELECT u.id, 'STATION_MANAGER', 'STATION', s.id
FROM (VALUES
    ('operator.spbu01', 'SPBU-01'),
    ('operator.spbu02', 'SPBU-02'),
    ('operator.spbu03', 'SPBU-03'),
    ('operator.spbu04', 'SPBU-04'),
    ('operator.spbu05', 'SPBU-05'),
    ('operator.spbu06', 'SPBU-06'),
    ('operator.spbu07', 'SPBU-07'),
    ('operator.spbu08', 'SPBU-08'),
    ('operator.spbu09', 'SPBU-09'),
    ('operator.spbu10', 'SPBU-10'),
    ('operator.spbu11', 'SPBU-11'),
    ('operator.spbu12', 'SPBU-12')
) AS m(username, station_code)
JOIN users       u ON u.username  = m.username
JOIN gas_stations s ON s.code     = m.station_code;

-- =============================================================================
-- END OF SCHEMA
-- =============================================================================
