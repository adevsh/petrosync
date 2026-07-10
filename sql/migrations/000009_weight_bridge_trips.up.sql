-- +migrate Up
CREATE TABLE weight_bridge_readings (
    id                      BIGSERIAL               PRIMARY KEY,
    trip_id                 BIGINT,
    vehicle_id              BIGINT                  NOT NULL REFERENCES vehicles(id),
    reading_type            VARCHAR(5)              NOT NULL CHECK (reading_type IN ('TARE', 'GROSS')),
    weight_kg               NUMERIC(12, 2)          NOT NULL,
    method                  measurement_method_t    NOT NULL,
    ambient_temp_celsius    NUMERIC(5, 2),
    recorded_by             BIGINT                  NOT NULL REFERENCES users(id),
    approval_status         approval_status_t       NOT NULL DEFAULT 'PENDING',
    approved_by             BIGINT                  REFERENCES users(id),
    approved_at             TIMESTAMPTZ,
    escalated_at            TIMESTAMPTZ,
    escalated_to            BIGINT                  REFERENCES users(id),
    notes                   TEXT,
    created_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_wbr_weight_positive CHECK (weight_kg > 0)
);

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
    route_polyline          GEOMETRY(LINESTRING, 4326),
    departed_at             TIMESTAMPTZ,
    arrived_at              TIMESTAMPTZ,
    completed_at            TIMESTAMPTZ,
    parent_trip_id          BIGINT                  REFERENCES trips(id),
    created_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

ALTER TABLE weight_bridge_readings
    ADD CONSTRAINT fk_wbr_trip
    FOREIGN KEY (trip_id) REFERENCES trips(id);
