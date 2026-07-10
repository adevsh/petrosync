-- +migrate Up
CREATE TABLE trip_events (
    id              BIGSERIAL               PRIMARY KEY,
    trip_id         BIGINT                  NOT NULL REFERENCES trips(id),
    event_uuid      UUID                    NOT NULL UNIQUE DEFAULT gen_random_uuid(),
    event_type      trip_event_type_t       NOT NULL,
    event_timestamp TIMESTAMPTZ             NOT NULL,
    received_at     TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    actor_user_id   BIGINT                  REFERENCES users(id),
    location        GEOMETRY(POINT, 4326),
    payload         JSONB,
    created_at      TIMESTAMPTZ             NOT NULL DEFAULT NOW()
);

CREATE TABLE trip_compartment_deliveries (
    id                  BIGSERIAL                       PRIMARY KEY,
    trip_id             BIGINT                          NOT NULL REFERENCES trips(id),
    compartment_id      BIGINT                          NOT NULL REFERENCES vehicle_compartments(id),
    fuel_type_code      VARCHAR(30)                     NOT NULL REFERENCES fuel_types(code),
    loaded_volume_l     NUMERIC(12, 2),
    loaded_weight_kg    NUMERIC(12, 2),
    delivered_volume_l  NUMERIC(12, 2),
    delivered_weight_kg NUMERIC(12, 2),
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

CREATE TABLE compartment_seals (
    id                      BIGSERIAL       PRIMARY KEY,
    trip_id                 BIGINT          NOT NULL REFERENCES trips(id),
    compartment_id          BIGINT          NOT NULL REFERENCES vehicle_compartments(id),
    seal_number_issued      VARCHAR(100)    NOT NULL,
    issued_by               BIGINT          NOT NULL REFERENCES users(id),
    issued_at               TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    seal_number_verified    VARCHAR(100),
    verified_by             BIGINT          REFERENCES users(id),
    verified_at             TIMESTAMPTZ,
    verification_status     seal_status_t,
    notes                   TEXT,
    UNIQUE (trip_id, compartment_id)
);
