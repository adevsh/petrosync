-- +migrate Down
DROP TABLE IF EXISTS vehicle_maintenance_records CASCADE;
DROP TABLE IF EXISTS vehicle_compartments CASCADE;
DROP TABLE IF EXISTS vehicles CASCADE;
