-- +migrate Up
CREATE TABLE delivery_orders (
    id                      BIGSERIAL           PRIMARY KEY,
    do_number               VARCHAR(30)         NOT NULL UNIQUE,
    status                  do_status_t         NOT NULL DEFAULT 'DRAFT',
    origin_facility_id      BIGINT              NOT NULL REFERENCES refinery_facilities(id),
    destination_type        destination_type_t  NOT NULL DEFAULT 'STATION',
    destination_station_id  BIGINT              REFERENCES gas_stations(id),
    destination_facility_id BIGINT              REFERENCES refinery_facilities(id),
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
    CONSTRAINT chk_do_destination CHECK (
        (destination_type = 'STATION' AND destination_station_id IS NOT NULL AND destination_facility_id IS NULL)
        OR
        (destination_type = 'REFINERY_FACILITY' AND destination_facility_id IS NOT NULL AND destination_station_id IS NULL)
    )
);

CREATE TABLE delivery_order_items (
    id                  BIGSERIAL       PRIMARY KEY,
    do_id               BIGINT          NOT NULL REFERENCES delivery_orders(id),
    fuel_type_code      VARCHAR(30)     NOT NULL REFERENCES fuel_types(code),
    compartment_id      BIGINT          REFERENCES vehicle_compartments(id),
    requested_volume_l  NUMERIC(12, 2)  NOT NULL,
    allocated_volume_l  NUMERIC(12, 2),
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
