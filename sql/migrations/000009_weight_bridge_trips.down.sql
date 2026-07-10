-- +migrate Down
ALTER TABLE IF EXISTS weight_bridge_readings DROP CONSTRAINT IF EXISTS fk_wbr_trip;
DROP TABLE IF EXISTS trips CASCADE;
DROP TABLE IF EXISTS weight_bridge_readings CASCADE;
