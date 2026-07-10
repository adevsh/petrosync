-- +migrate Down
DROP TABLE IF EXISTS station_tanks CASCADE;
DROP TABLE IF EXISTS station_qr_codes CASCADE;
DROP TABLE IF EXISTS station_facility_whitelist CASCADE;
DROP TABLE IF EXISTS gas_stations CASCADE;
