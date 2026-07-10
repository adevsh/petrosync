-- +migrate Up
DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'petrosync_app') THEN
        CREATE ROLE petrosync_app LOGIN PASSWORD 'change_me_in_production';
    END IF;
END $$;

DO $$ BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'petrosync_readonly') THEN
        CREATE ROLE petrosync_readonly LOGIN PASSWORD 'change_me_in_production';
    END IF;
END $$;

GRANT USAGE ON SCHEMA public TO petrosync_app;
GRANT USAGE ON SCHEMA public TO petrosync_readonly;
GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA public TO petrosync_app;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA public TO petrosync_app;
GRANT SELECT ON ALL TABLES IN SCHEMA public TO petrosync_readonly;

REVOKE UPDATE, DELETE ON trip_events      FROM petrosync_app;
REVOKE UPDATE, DELETE ON audit_log        FROM petrosync_app;
REVOKE UPDATE, DELETE ON notification_log FROM petrosync_app;

DO $$
DECLARE r RECORD;
BEGIN
    FOR r IN
        SELECT c.relname FROM pg_class c
        JOIN pg_inherits i ON i.inhrelid = c.oid
        JOIN pg_class p ON i.inhparent = p.oid
        WHERE p.relname = 'gps_events' AND c.relkind = 'r'
    LOOP
        EXECUTE format('REVOKE UPDATE, DELETE ON %I FROM petrosync_app', r.relname);
    END LOOP;
END $$;

ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO petrosync_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT USAGE, SELECT ON SEQUENCES TO petrosync_app;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT SELECT ON TABLES TO petrosync_readonly;
