-- +migrate Down
DROP TABLE IF EXISTS telegram_link_tokens CASCADE;
DROP TABLE IF EXISTS audit_log CASCADE;
DROP TABLE IF EXISTS notification_log CASCADE;
DROP TABLE IF EXISTS route_deviation_events CASCADE;
DROP TABLE IF EXISTS trip_documents CASCADE;
DROP TABLE IF EXISTS trip_photos CASCADE;
