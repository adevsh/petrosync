-- +migrate Up
CREATE TABLE users (
    id                      BIGSERIAL       PRIMARY KEY,
    username                VARCHAR(100)    NOT NULL UNIQUE,
    password_hash           TEXT            NOT NULL,
    full_name               VARCHAR(200)    NOT NULL,
    telegram_user_id        BIGINT          UNIQUE,
    telegram_linked_at      TIMESTAMPTZ,
    force_password_change   BOOLEAN         NOT NULL DEFAULT TRUE,
    active                  BOOLEAN         NOT NULL DEFAULT TRUE,
    last_login_at           TIMESTAMPTZ,
    created_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at              TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

CREATE TABLE user_role_grants (
    id          BIGSERIAL       PRIMARY KEY,
    user_id     BIGINT          NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        user_role_t     NOT NULL,
    scope_type  role_scope_t    NOT NULL,
    scope_id    BIGINT,
    granted_by  BIGINT          REFERENCES users(id),
    granted_at  TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    revoked_at  TIMESTAMPTZ,
    UNIQUE (user_id, role, scope_type, scope_id)
);

CREATE TABLE drivers (
    id                  BIGSERIAL       PRIMARY KEY,
    user_id             BIGINT          NOT NULL UNIQUE REFERENCES users(id),
    employee_number     VARCHAR(50)     UNIQUE,
    sim_b2_number       VARCHAR(50)     NOT NULL,
    sim_b2_expiry       DATE            NOT NULL,
    home_depot_id       BIGINT          REFERENCES vehicle_depots(id),
    current_shift_start TIMESTAMPTZ,
    current_shift_end   TIMESTAMPTZ,
    is_on_shift         BOOLEAN         NOT NULL DEFAULT FALSE,
    created_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW(),
    updated_at          TIMESTAMPTZ     NOT NULL DEFAULT NOW()
);

ALTER TABLE vehicle_maintenance_records
    ADD CONSTRAINT fk_maintenance_recorded_by
    FOREIGN KEY (recorded_by) REFERENCES users(id);

ALTER TABLE system_settings
    ADD CONSTRAINT fk_system_settings_updated_by
    FOREIGN KEY (updated_by) REFERENCES users(id);
