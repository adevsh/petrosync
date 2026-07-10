-- +migrate Up
CREATE TABLE gas_stations (
    id                      BIGSERIAL               PRIMARY KEY,
    code                    VARCHAR(20)             NOT NULL UNIQUE,
    name                    VARCHAR(200)            NOT NULL,
    spbu_license_number     VARCHAR(50)             NOT NULL UNIQUE,
    region_code             VARCHAR(10)             NOT NULL REFERENCES regions(code),
    primary_facility_id     BIGINT                  NOT NULL REFERENCES refinery_facilities(id),
    location                GEOMETRY(POINT, 4326)   NOT NULL,
    address                 TEXT,
    operating_hours_start   TIME,
    operating_hours_end     TIME,
    contact_name            VARCHAR(200),
    contact_phone           VARCHAR(30),
    active                  BOOLEAN                 NOT NULL DEFAULT TRUE,
    created_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

CREATE TABLE station_facility_whitelist (
    station_id      BIGINT      NOT NULL REFERENCES gas_stations(id)         ON DELETE CASCADE,
    facility_id     BIGINT      NOT NULL REFERENCES refinery_facilities(id),
    PRIMARY KEY (station_id, facility_id)
);

CREATE TABLE station_qr_codes (
    id          BIGSERIAL       PRIMARY KEY,
    station_id  BIGINT          NOT NULL REFERENCES gas_stations(id),
    qr_payload  VARCHAR(255)    NOT NULL UNIQUE,
    label       VARCHAR(100),
    active      BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE station_tanks (
    id                  BIGSERIAL       PRIMARY KEY,
    station_id          BIGINT          NOT NULL REFERENCES gas_stations(id),
    tank_code           VARCHAR(20)     NOT NULL,
    fuel_type_code      VARCHAR(30)     NOT NULL REFERENCES fuel_types(code),
    capacity_l          NUMERIC(12, 2)  NOT NULL,
    current_volume_l    NUMERIC(12, 2)  NOT NULL DEFAULT 0,
    reorder_threshold_l NUMERIC(12, 2)  NOT NULL,
    last_dip_reading_l  NUMERIC(12, 2),
    last_dip_at         TIMESTAMPTZ,
    active              BOOLEAN         NOT NULL DEFAULT TRUE,
    last_updated_at     TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    UNIQUE (station_id, tank_code),
    CONSTRAINT chk_station_volume_non_negative CHECK (current_volume_l >= 0),
    CONSTRAINT chk_station_not_over_capacity   CHECK (current_volume_l <= capacity_l)
);
