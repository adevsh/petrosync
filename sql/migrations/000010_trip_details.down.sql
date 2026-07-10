-- +migrate Down
DROP TABLE IF EXISTS compartment_seals CASCADE;
DROP TABLE IF EXISTS trip_compartment_deliveries CASCADE;
DROP TABLE IF EXISTS trip_events CASCADE;
