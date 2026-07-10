-- +migrate Down
DROP ROLE IF EXISTS petrosync_readonly;
DROP ROLE IF EXISTS petrosync_app;
