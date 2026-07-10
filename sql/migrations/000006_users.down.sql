-- +migrate Down
ALTER TABLE IF EXISTS system_settings DROP CONSTRAINT IF EXISTS fk_system_settings_updated_by;
ALTER TABLE IF EXISTS vehicle_maintenance_records DROP CONSTRAINT IF EXISTS fk_maintenance_recorded_by;
DROP TABLE IF EXISTS drivers CASCADE;
DROP TABLE IF EXISTS user_role_grants CASCADE;
DROP TABLE IF EXISTS users CASCADE;
