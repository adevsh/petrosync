-- +migrate Up
-- Reference tables + org structure

CREATE TABLE regions (
    code        VARCHAR(10)     PRIMARY KEY,
    name        VARCHAR(100)    NOT NULL,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE fuel_types (
    code                        VARCHAR(30)         PRIMARY KEY,
    name                        VARCHAR(100)        NOT NULL,
    category                    fuel_category_t     NOT NULL,
    ron_cn                      SMALLINT,
    density_kg_per_l_at_15c     NUMERIC(6, 4)       NOT NULL,
    evaporation_factor_pct      NUMERIC(5, 3)       NOT NULL DEFAULT 0.100,
    is_subsidized               BOOLEAN             NOT NULL DEFAULT FALSE,
    active                      BOOLEAN             NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ         NOT NULL DEFAULT NOW()
);

CREATE TABLE system_settings (
    id              BIGSERIAL       PRIMARY KEY,
    facility_id     BIGINT,
    key             VARCHAR(100)    NOT NULL,
    value           TEXT            NOT NULL,
    description     TEXT,
    updated_by      BIGINT,
    updated_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE refineries (
    id                  BIGSERIAL       PRIMARY KEY,
    code                VARCHAR(20)     NOT NULL UNIQUE,
    name                VARCHAR(150)    NOT NULL,
    region_code         VARCHAR(10)     NOT NULL REFERENCES regions(code),
    commissioned_year   SMALLINT,
    active              BOOLEAN         NOT NULL DEFAULT TRUE,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE refinery_facilities (
    id                          BIGSERIAL               PRIMARY KEY,
    code                        VARCHAR(20)             NOT NULL UNIQUE,
    refinery_id                 BIGINT                  NOT NULL REFERENCES refineries(id),
    name                        VARCHAR(150)            NOT NULL,
    location                    GEOMETRY(POINT, 4326)   NOT NULL,
    address                     TEXT,
    is_primary                  BOOLEAN                 NOT NULL DEFAULT FALSE,
    max_assignment_radius_km    NUMERIC(6, 2)           NOT NULL DEFAULT 300,
    active                      BOOLEAN                 NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

ALTER TABLE system_settings
    ADD CONSTRAINT fk_system_settings_facility
    FOREIGN KEY (facility_id) REFERENCES refinery_facilities(id);

CREATE TABLE vehicle_depots (
    id                          BIGSERIAL               PRIMARY KEY,
    code                        VARCHAR(20)             NOT NULL UNIQUE,
    name                        VARCHAR(150)            NOT NULL,
    primary_facility_id         BIGINT                  NOT NULL REFERENCES refinery_facilities(id),
    location                    GEOMETRY(POINT, 4326)   NOT NULL,
    default_truck_capacity_l    INTEGER                 NOT NULL DEFAULT 24000,
    active                      BOOLEAN                 NOT NULL DEFAULT TRUE,
    created_at                  TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at                  TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);
