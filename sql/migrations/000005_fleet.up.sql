-- +migrate Up
CREATE TABLE vehicles (
    id                      BIGSERIAL               PRIMARY KEY,
    plate_number            VARCHAR(20)             NOT NULL UNIQUE,
    chassis_number          VARCHAR(50)             NOT NULL UNIQUE,
    model                   VARCHAR(100),
    manufacture_year        SMALLINT,
    total_capacity_l        NUMERIC(10, 2)          NOT NULL,
    tare_weight_kg          NUMERIC(10, 2)          NOT NULL,
    current_depot_id        BIGINT                  REFERENCES vehicle_depots(id),
    current_location        GEOMETRY(POINT, 4326),
    status                  vehicle_status_t        NOT NULL DEFAULT 'AVAILABLE',
    keur_number             VARCHAR(50),
    keur_expiry             DATE,
    last_inspection_date    DATE,
    next_inspection_due     DATE,
    last_assigned_at        TIMESTAMPTZ,
    notes                   TEXT,
    active                  BOOLEAN                 NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

CREATE TABLE vehicle_compartments (
    id                  BIGSERIAL       PRIMARY KEY,
    vehicle_id          BIGINT          NOT NULL REFERENCES vehicles(id),
    compartment_number  SMALLINT        NOT NULL,
    fuel_type_code      VARCHAR(30)     REFERENCES fuel_types(code),
    capacity_l          NUMERIC(10, 2)  NOT NULL,
    is_active           BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (vehicle_id, compartment_number)
);

CREATE TABLE vehicle_maintenance_records (
    id                      BIGSERIAL       PRIMARY KEY,
    vehicle_id              BIGINT          NOT NULL REFERENCES vehicles(id),
    recorded_by             BIGINT,
    maintenance_type        VARCHAR(100)    NOT NULL,
    description             TEXT,
    started_at              TIMESTAMPTZ     NOT NULL,
    estimated_return_at     TIMESTAMPTZ,
    completed_at            TIMESTAMPTZ,
    notes                   TEXT,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
