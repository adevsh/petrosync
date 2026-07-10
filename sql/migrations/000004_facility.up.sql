-- +migrate Up
CREATE TABLE facility_loading_bays (
    id              BIGSERIAL       PRIMARY KEY,
    facility_id     BIGINT          NOT NULL REFERENCES refinery_facilities(id),
    bay_code        VARCHAR(20)     NOT NULL,
    qr_payload      VARCHAR(255)    NOT NULL UNIQUE,
    fuel_type_code  VARCHAR(30)     REFERENCES fuel_types(code),
    active          BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (facility_id, bay_code)
);

CREATE TABLE facility_storage_tanks (
    id                  BIGSERIAL       PRIMARY KEY,
    facility_id         BIGINT          NOT NULL REFERENCES refinery_facilities(id),
    tank_code           VARCHAR(20)     NOT NULL,
    fuel_type_code      VARCHAR(30)     NOT NULL REFERENCES fuel_types(code),
    capacity_l          NUMERIC(14, 2)  NOT NULL,
    current_volume_l    NUMERIC(14, 2)  NOT NULL DEFAULT 0,
    reserved_volume_l   NUMERIC(14, 2)  NOT NULL DEFAULT 0,
    min_operational_l   NUMERIC(14, 2)  NOT NULL DEFAULT 0,
    active              BOOLEAN         NOT NULL DEFAULT TRUE,
    last_updated_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (facility_id, tank_code),
    CONSTRAINT chk_storage_volume_non_negative CHECK (current_volume_l >= 0 AND reserved_volume_l >= 0),
    CONSTRAINT chk_storage_not_over_capacity   CHECK (current_volume_l <= capacity_l)
);
