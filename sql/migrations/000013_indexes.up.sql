-- +migrate Up
CREATE INDEX idx_fuel_types_category   ON fuel_types(category);
CREATE INDEX idx_fuel_types_active     ON fuel_types(active) WHERE active = TRUE;

CREATE UNIQUE INDEX idx_system_settings_global   ON system_settings(key)               WHERE facility_id IS NULL;
CREATE UNIQUE INDEX idx_system_settings_facility ON system_settings(facility_id, key)  WHERE facility_id IS NOT NULL;
CREATE INDEX        idx_system_settings_facility_id ON system_settings(facility_id);

CREATE INDEX idx_refineries_region ON refineries(region_code);
CREATE INDEX idx_refineries_active ON refineries(active) WHERE active = TRUE;

CREATE INDEX idx_facilities_refinery ON refinery_facilities(refinery_id);
CREATE INDEX idx_facilities_location ON refinery_facilities USING GIST(location);
CREATE INDEX idx_facilities_primary  ON refinery_facilities(refinery_id, is_primary);
CREATE INDEX idx_facilities_active   ON refinery_facilities(active) WHERE active = TRUE;

CREATE INDEX idx_depots_facility ON vehicle_depots(primary_facility_id);
CREATE INDEX idx_depots_location ON vehicle_depots USING GIST(location);

CREATE INDEX idx_loading_bays_facility ON facility_loading_bays(facility_id);
CREATE INDEX idx_loading_bays_qr       ON facility_loading_bays(qr_payload);
CREATE INDEX idx_loading_bays_active   ON facility_loading_bays(facility_id, active) WHERE active = TRUE;

CREATE INDEX idx_storage_tanks_facility ON facility_storage_tanks(facility_id);
CREATE INDEX idx_storage_tanks_fuel     ON facility_storage_tanks(fuel_type_code);
CREATE INDEX idx_storage_tanks_active   ON facility_storage_tanks(facility_id, fuel_type_code) WHERE active = TRUE;

CREATE INDEX idx_vehicles_status      ON vehicles(status);
CREATE INDEX idx_vehicles_depot       ON vehicles(current_depot_id);
CREATE INDEX idx_vehicles_location    ON vehicles USING GIST(current_location);
CREATE INDEX idx_vehicles_keur_expiry ON vehicles(keur_expiry);
CREATE INDEX idx_vehicles_dispatch    ON vehicles(status, keur_expiry) WHERE status = 'AVAILABLE' AND active = TRUE;

CREATE INDEX idx_compartments_vehicle ON vehicle_compartments(vehicle_id);
CREATE INDEX idx_compartments_fuel    ON vehicle_compartments(fuel_type_code);
CREATE INDEX idx_compartments_active  ON vehicle_compartments(vehicle_id, is_active) WHERE is_active = TRUE;

CREATE INDEX idx_maintenance_vehicle   ON vehicle_maintenance_records(vehicle_id);
CREATE INDEX idx_maintenance_open      ON vehicle_maintenance_records(vehicle_id, completed_at) WHERE completed_at IS NULL;

CREATE INDEX idx_users_username  ON users(username);
CREATE INDEX idx_users_telegram  ON users(telegram_user_id) WHERE telegram_user_id IS NOT NULL;
CREATE INDEX idx_users_active    ON users(active) WHERE active = TRUE;

CREATE INDEX idx_role_grants_user        ON user_role_grants(user_id);
CREATE INDEX idx_role_grants_scope       ON user_role_grants(scope_type, scope_id);
CREATE INDEX idx_role_grants_role        ON user_role_grants(role);
CREATE INDEX idx_role_grants_active      ON user_role_grants(user_id, role, scope_type, scope_id) WHERE revoked_at IS NULL;

CREATE INDEX idx_drivers_user       ON drivers(user_id);
CREATE INDEX idx_drivers_depot      ON drivers(home_depot_id);
CREATE INDEX idx_drivers_sim_expiry ON drivers(sim_b2_expiry);
CREATE INDEX idx_drivers_dispatch   ON drivers(is_on_shift, sim_b2_expiry) WHERE is_on_shift = TRUE;

CREATE INDEX idx_stations_region   ON gas_stations(region_code);
CREATE INDEX idx_stations_facility ON gas_stations(primary_facility_id);
CREATE INDEX idx_stations_location ON gas_stations USING GIST(location);
CREATE INDEX idx_stations_active   ON gas_stations(active) WHERE active = TRUE;

CREATE INDEX idx_station_qr_station ON station_qr_codes(station_id);
CREATE INDEX idx_station_qr_payload ON station_qr_codes(qr_payload);
CREATE INDEX idx_station_qr_active  ON station_qr_codes(qr_payload, active) WHERE active = TRUE;

CREATE INDEX idx_station_tanks_station ON station_tanks(station_id);
CREATE INDEX idx_station_tanks_fuel    ON station_tanks(fuel_type_code);
CREATE INDEX idx_station_tanks_reorder ON station_tanks(station_id, current_volume_l, reorder_threshold_l) WHERE active = TRUE;

CREATE INDEX idx_do_status           ON delivery_orders(status);
CREATE INDEX idx_do_origin           ON delivery_orders(origin_facility_id);
CREATE INDEX idx_do_destination_sta  ON delivery_orders(destination_station_id);
CREATE INDEX idx_do_scheduled        ON delivery_orders(scheduled_date);
CREATE INDEX idx_do_vehicle          ON delivery_orders(assigned_vehicle_id);
CREATE INDEX idx_do_driver           ON delivery_orders(assigned_driver_id);
CREATE INDEX idx_do_raised_by        ON delivery_orders(raised_by);
CREATE INDEX idx_do_dispatch_queue   ON delivery_orders(origin_facility_id, status, scheduled_date) WHERE status IN ('APPROVED', 'ASSIGNED');

CREATE INDEX idx_do_items_do          ON delivery_order_items(do_id);
CREATE INDEX idx_do_items_compartment ON delivery_order_items(compartment_id);
CREATE INDEX idx_do_items_fuel        ON delivery_order_items(fuel_type_code);

CREATE INDEX idx_wbr_trip     ON weight_bridge_readings(trip_id);
CREATE INDEX idx_wbr_vehicle  ON weight_bridge_readings(vehicle_id);
CREATE INDEX idx_wbr_pending  ON weight_bridge_readings(approval_status, created_at) WHERE approval_status IN ('PENDING', 'ESCALATED');

CREATE INDEX idx_trips_do              ON trips(do_id);
CREATE INDEX idx_trips_vehicle         ON trips(vehicle_id);
CREATE INDEX idx_trips_driver          ON trips(driver_id);
CREATE INDEX idx_trips_status          ON trips(status);
CREATE INDEX idx_trips_origin          ON trips(origin_facility_id);
CREATE INDEX idx_trips_destination_sta ON trips(destination_station_id);
CREATE INDEX idx_trips_parent          ON trips(parent_trip_id) WHERE parent_trip_id IS NOT NULL;
CREATE INDEX idx_trips_departed        ON trips(departed_at);
CREATE INDEX idx_trips_active          ON trips(status, vehicle_id) WHERE status IN ('LOADING','LOADED','IN_TRANSIT','ARRIVED','UNLOADING');

CREATE INDEX idx_trip_events_trip      ON trip_events(trip_id);
CREATE INDEX idx_trip_events_uuid      ON trip_events(event_uuid);
CREATE INDEX idx_trip_events_type      ON trip_events(event_type);
CREATE INDEX idx_trip_events_time      ON trip_events(event_timestamp);
CREATE INDEX idx_trip_events_trip_time ON trip_events(trip_id, event_timestamp);
CREATE INDEX idx_trip_events_trip_type ON trip_events(trip_id, event_type);

CREATE INDEX idx_tcd_trip        ON trip_compartment_deliveries(trip_id);
CREATE INDEX idx_tcd_compartment ON trip_compartment_deliveries(compartment_id);
CREATE INDEX idx_tcd_status      ON trip_compartment_deliveries(delivery_status);
CREATE INDEX idx_tcd_disputed    ON trip_compartment_deliveries(trip_id, delivery_status) WHERE delivery_status = 'DISPUTED';

CREATE INDEX idx_seals_trip        ON compartment_seals(trip_id);
CREATE INDEX idx_seals_compartment ON compartment_seals(compartment_id);
CREATE INDEX idx_seals_mismatch    ON compartment_seals(trip_id, verification_status) WHERE verification_status IN ('MISMATCHED','BROKEN','MISSING');

CREATE INDEX idx_gps_trip      ON gps_events(trip_id);
CREATE INDEX idx_gps_uuid      ON gps_events(event_uuid);
CREATE INDEX idx_gps_trip_time ON gps_events(trip_id, event_timestamp DESC);
CREATE INDEX idx_gps_location  ON gps_events USING GIST(location);

CREATE INDEX idx_photos_trip       ON trip_photos(trip_id);
CREATE INDEX idx_photos_event      ON trip_photos(event_type);
CREATE INDEX idx_photos_trip_event ON trip_photos(trip_id, event_type);

CREATE INDEX idx_docs_trip ON trip_documents(trip_id);
CREATE INDEX idx_docs_type ON trip_documents(document_type);

CREATE INDEX idx_deviations_trip       ON route_deviation_events(trip_id);
CREATE INDEX idx_deviations_unresolved ON route_deviation_events(trip_id, resolved_at) WHERE resolved_at IS NULL;

CREATE INDEX idx_notif_trip      ON notification_log(trip_id);
CREATE INDEX idx_notif_recipient ON notification_log(recipient_telegram_id);
CREATE INDEX idx_notif_type      ON notification_log(notification_type);
CREATE INDEX idx_notif_sent      ON notification_log(sent_at);

CREATE INDEX idx_audit_user    ON audit_log(user_id);
CREATE INDEX idx_audit_entity  ON audit_log(entity_type, entity_id);
CREATE INDEX idx_audit_action  ON audit_log(action);
CREATE INDEX idx_audit_created ON audit_log(created_at);

CREATE INDEX idx_tg_tokens_user  ON telegram_link_tokens(user_id);
CREATE INDEX idx_tg_tokens_valid ON telegram_link_tokens(token, expires_at) WHERE used_at IS NULL;
