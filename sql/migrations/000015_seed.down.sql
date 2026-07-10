-- +migrate Down
DELETE
FROM user_role_grants
WHERE user_id IN (
    SELECT id FROM users
    WHERE username LIKE 'operator.spbu%'
);
DELETE FROM station_tanks;
DELETE FROM station_qr_codes;
DELETE FROM station_facility_whitelist;
DELETE FROM gas_stations;
DELETE FROM drivers;
DELETE
FROM user_role_grants
WHERE user_id IN (
    SELECT id FROM users
    WHERE
        username IN (
            'superadmin',
            'driver.01',
            'operator.ru2',
            'operator.ru3',
            'operator.ru4',
            'operator.ru5',
            'operator.ru6'
        )
);
DELETE FROM users;
DELETE FROM facility_storage_tanks;
DELETE FROM facility_loading_bays;
DELETE FROM vehicle_depots;
DELETE FROM refinery_facilities;
DELETE FROM refineries;
DELETE FROM system_settings;
DELETE FROM fuel_types;
DELETE FROM regions;
