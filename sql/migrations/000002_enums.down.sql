-- +migrate Down
DROP TYPE IF EXISTS notification_type_t CASCADE;
DROP TYPE IF EXISTS compartment_delivery_status_t CASCADE;
DROP TYPE IF EXISTS seal_status_t CASCADE;
DROP TYPE IF EXISTS document_type_t CASCADE;
DROP TYPE IF EXISTS photo_event_t CASCADE;
DROP TYPE IF EXISTS trip_event_type_t CASCADE;
DROP TYPE IF EXISTS trip_status_t CASCADE;
DROP TYPE IF EXISTS destination_type_t CASCADE;
DROP TYPE IF EXISTS do_status_t CASCADE;
DROP TYPE IF EXISTS approval_status_t CASCADE;
DROP TYPE IF EXISTS measurement_method_t CASCADE;
DROP TYPE IF EXISTS vehicle_status_t CASCADE;
DROP TYPE IF EXISTS role_scope_t CASCADE;
DROP TYPE IF EXISTS user_role_t CASCADE;
DROP TYPE IF EXISTS fuel_category_t CASCADE;
