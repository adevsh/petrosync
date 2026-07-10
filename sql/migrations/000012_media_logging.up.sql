-- +migrate Up
CREATE TABLE trip_photos (
    id                  BIGSERIAL           PRIMARY KEY,
    trip_id             BIGINT              NOT NULL REFERENCES trips(id),
    compartment_id      BIGINT              REFERENCES vehicle_compartments(id),
    event_type          photo_event_t       NOT NULL,
    garage_object_key   VARCHAR(500)        NOT NULL,
    file_size_bytes     BIGINT,
    mime_type           VARCHAR(50)         NOT NULL DEFAULT 'image/jpeg',
    uploaded_by         BIGINT              NOT NULL REFERENCES users(id),
    taken_at            TIMESTAMPTZ         NOT NULL,
    uploaded_at         TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    notes               TEXT
);

CREATE TABLE trip_documents (
    id                  BIGSERIAL           PRIMARY KEY,
    trip_id             BIGINT              NOT NULL REFERENCES trips(id),
    document_type       document_type_t     NOT NULL,
    document_number     VARCHAR(50)         UNIQUE,
    garage_object_key   VARCHAR(500)        NOT NULL,
    generated_by        BIGINT              REFERENCES users(id),
    generated_at        TIMESTAMPTZ         NOT NULL DEFAULT NOW(),
    UNIQUE (trip_id, document_type)
);

CREATE TABLE route_deviation_events (
    id                  BIGSERIAL       PRIMARY KEY,
    trip_id             BIGINT          NOT NULL REFERENCES trips(id),
    detected_at         TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    duration_seconds    INTEGER,
    deviation_meters    NUMERIC(10, 2),
    occurrence_count    SMALLINT        NOT NULL DEFAULT 1,
    telegram_notified   BOOLEAN         NOT NULL DEFAULT FALSE,
    telegram_notified_at TIMESTAMPTZ,
    resolved_at         TIMESTAMPTZ,
    notes               TEXT
);

CREATE TABLE notification_log (
    id                      BIGSERIAL               PRIMARY KEY,
    trip_id                 BIGINT                  REFERENCES trips(id),
    do_id                   BIGINT                  REFERENCES delivery_orders(id),
    recipient_telegram_id   BIGINT                  NOT NULL,
    recipient_user_id       BIGINT                  REFERENCES users(id),
    notification_type       notification_type_t     NOT NULL,
    message_text            TEXT                    NOT NULL,
    sent_at                 TIMESTAMPTZ             NOT NULL DEFAULT NOW(),
    delivery_status         VARCHAR(20)             NOT NULL DEFAULT 'SENT',
    telegram_message_id     BIGINT,
    error_message           TEXT
);

CREATE TABLE audit_log (
    id              BIGSERIAL       PRIMARY KEY,
    user_id         BIGINT          REFERENCES users(id),
    action          VARCHAR(100)    NOT NULL,
    entity_type     VARCHAR(100)    NOT NULL,
    entity_id       BIGINT,
    before_state    JSONB,
    after_state     JSONB,
    ip_address      INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE telegram_link_tokens (
    id          BIGSERIAL       PRIMARY KEY,
    user_id     BIGINT          NOT NULL REFERENCES users(id),
    token       VARCHAR(64)     NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ     NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);
