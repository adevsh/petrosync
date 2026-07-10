-- +migrate Up
INSERT INTO regions (code, name) VALUES
('RIAU', 'Riau'),
('SUMSEL', 'Sumatera Selatan'),
('JATENG', 'Jawa Tengah'),
('KALTIM', 'Kalimantan Timur'),
('JABAR', 'Jawa Barat');

INSERT INTO fuel_types (
    code,
    name,
    category,
    ron_cn,
    density_kg_per_l_at_15c,
    evaporation_factor_pct,
    is_subsidized
) VALUES
('PERTALITE', 'Pertalite', 'GASOLINE', 90, 0.7150, 0.100, TRUE),
('PERTAMAX', 'Pertamax', 'GASOLINE', 92, 0.7200, 0.100, FALSE),
('PERTAMAX_TURBO', 'Pertamax Turbo', 'GASOLINE', 98, 0.7400, 0.100, FALSE),
('BIOSOLAR', 'Biosolar B35', 'DIESEL', 48, 0.8450, 0.050, TRUE),
('DEXLITE', 'Dexlite', 'DIESEL', 51, 0.8200, 0.050, FALSE),
('PERTAMINA_DEX', 'Pertamina Dex', 'DIESEL', 53, 0.8300, 0.050, FALSE);

INSERT INTO system_settings (facility_id, key, value, description) VALUES
(
    NULL,
    'approval_escalation_hours',
    '2',
    'Hours before manual weight bridge approval auto-escalates'
),
(
    NULL,
    'variance_tolerance_pct',
    '0.3',
    'Variance % threshold for DISPUTED status'
),
(
    NULL,
    'gps_ping_interval_seconds',
    '30',
    'GPS ping frequency from Android while trip is IN_TRANSIT'
),
(
    NULL,
    'route_deviation_warn_count',
    '2',
    'Deviation count before dashboard warning'
),
(
    NULL,
    'route_deviation_alert_minutes',
    '15',
    'Sustained deviation minutes before Telegram escalation'
),
(
    NULL,
    'dispatch_candidate_limit',
    '5',
    'Max candidate trucks shown on DO assignment screen'
);

INSERT INTO refineries (code, name, region_code, commissioned_year) VALUES
('RU-II', 'Refinery Unit II Dumai', 'RIAU', 1971),
('RU-III', 'Refinery Unit III Plaju', 'SUMSEL', 1926),
('RU-IV', 'Refinery Unit IV Cilacap', 'JATENG', 1974),
('RU-V', 'Refinery Unit V Balikpapan', 'KALTIM', 1922),
('RU-VI', 'Refinery Unit VI Balongan', 'JABAR', 1994);

INSERT INTO refinery_facilities (
    code, refinery_id, name, location, is_primary, max_assignment_radius_km
) VALUES
(
    'FAC-DUM', (
        SELECT id FROM refineries
        WHERE code = 'RU-II'
    ), 'Dumai', ST_SETSRID(ST_MAKEPOINT(101.4264, 1.6573), 4326
    ), TRUE, 250
),
(
    'FAC-SKP', (
        SELECT id FROM refineries
        WHERE code = 'RU-II'
    ), 'Sungai Pakning', ST_SETSRID(ST_MAKEPOINT(102.1276, 1.3560), 4326
    ), FALSE, 200
),
(
    'FAC-PLJ', (
        SELECT id FROM refineries
        WHERE code = 'RU-III'
    ), 'Plaju', ST_SETSRID(ST_MAKEPOINT(104.8087, -2.9823), 4326
    ), TRUE, 300
),
(
    'FAC-SGR', (
        SELECT id FROM refineries
        WHERE code = 'RU-III'
    ), 'Sungai Gerong', ST_SETSRID(ST_MAKEPOINT(104.8215, -2.9667), 4326
    ), FALSE, 300
),
(
    'FAC-CLP', (
        SELECT id FROM refineries
        WHERE code = 'RU-IV'
    ), 'Cilacap', ST_SETSRID(ST_MAKEPOINT(108.9916, -7.7250), 4326
    ), TRUE, 350
),
(
    'FAC-BPP', (
        SELECT id FROM refineries
        WHERE code = 'RU-V'
    ), 'Balikpapan', ST_SETSRID(ST_MAKEPOINT(116.8301, -1.2675), 4326
    ), TRUE, 500
),
(
    'FAC-BLG', (
        SELECT id FROM refineries
        WHERE code = 'RU-VI'
    ), 'Balongan', ST_SETSRID(ST_MAKEPOINT(108.2667, -6.3700), 4326
    ), TRUE, 250
);

INSERT INTO vehicle_depots (
    code, name, primary_facility_id, location, default_truck_capacity_l
) VALUES
(
    'DEPOT-DUM', 'Depot Dumai', (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-DUM'
    ), ST_SETSRID(ST_MAKEPOINT(101.4264, 1.6573), 4326
    ), 24000
),
(
    'DEPOT-SKP', 'Depot Sungai Pakning', (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-SKP'
    ), ST_SETSRID(ST_MAKEPOINT(102.1276, 1.3560), 4326
    ), 24000
),
(
    'DEPOT-PLJ', 'Depot Plaju', (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-PLJ'
    ), ST_SETSRID(ST_MAKEPOINT(104.8000, -2.9750), 4326
    ), 24000
),
(
    'DEPOT-CLP', 'Depot Cilacap', (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-CLP'
    ), ST_SETSRID(ST_MAKEPOINT(108.9916, -7.7250), 4326
    ), 24000
),
(
    'DEPOT-BPP', 'Depot Balikpapan', (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-BPP'
    ), ST_SETSRID(ST_MAKEPOINT(116.8301, -1.2675), 4326
    ), 24000
),
(
    'DEPOT-BLG', 'Depot Balongan', (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-BLG'
    ), ST_SETSRID(ST_MAKEPOINT(108.2667, -6.3700), 4326
    ), 24000
);

INSERT INTO facility_loading_bays (facility_id, bay_code, qr_payload)
SELECT
    f.id,
    'BAY-' || LPAD(n::TEXT, 2, '0'),
    'LB-'
    || f.code
    || '-BAY'
    || LPAD(n::TEXT, 2, '0')
    || '-'
    || GEN_RANDOM_UUID()
FROM refinery_facilities AS f
CROSS JOIN GENERATE_SERIES(1, 2) AS n
WHERE f.is_primary = TRUE;

INSERT INTO facility_storage_tanks (
    facility_id,
    tank_code,
    fuel_type_code,
    capacity_l,
    current_volume_l,
    min_operational_l
)
SELECT
    f.id,
    'STK-' || ft.code,
    ft.code,
    CASE ft.category WHEN 'GASOLINE' THEN 5000000 ELSE 3000000 END,
    CASE ft.category WHEN 'GASOLINE' THEN 3000000 ELSE 1800000 END,
    CASE ft.category WHEN 'GASOLINE' THEN 500000 ELSE 300000 END
FROM refinery_facilities AS f CROSS JOIN fuel_types AS ft
WHERE f.is_primary = TRUE AND ft.active = TRUE;

INSERT INTO users (
    username, password_hash, full_name, force_password_change
) VALUES
(
    'superadmin',
    '$2b$12$xOtGv7.yp1R8AMf.Bq4zFOMfZ/n6Kb5WpmDs7n4MugIPln1ZebVmK',
    'Super Administrator',
    FALSE
),
(
    'operator.ru2',
    '$2b$12$7sM8LftNd2an1AvNkKICYuepjoOLhVbtsw16YOqoFvt7kiFE1.gC2',
    'Operator RU II Dumai',
    TRUE
),
(
    'operator.ru3',
    '$2b$12$giRyqUS7SXvfYWNh.rrO8uz71.MTPpkTbLMJma/SG2IdwnfneoBTe',
    'Operator RU III Plaju',
    TRUE
),
(
    'operator.ru4',
    '$2b$12$siXE2X2id7Ge21HkrheHUOdr22slb8VswdJoyPS9gERVqNw8ZiGuK',
    'Operator RU IV Cilacap',
    TRUE
),
(
    'operator.ru5',
    '$2b$12$kW1opWxv7XcvwmJ4Jpoh8.J6b.pmNPpLkM4.dN.nsLPcpxj9AjAqO',
    'Operator RU V Balikpapan',
    TRUE
),
(
    'operator.ru6',
    '$2b$12$LExZ9bYFGVadUq8Zsn1WKuKYUUwgh.Lf9BVMtAFotJUhhDodDCOCW',
    'Operator RU VI Balongan',
    TRUE
),
(
    'driver.01',
    '$2b$12$pmwTDOw57E/EVeuFukv8o./l.IRDc4RN873dKTXaVGLfS0Ya2txbi',
    'Budi Santoso',
    TRUE
),
(
    'operator.spbu01',
    '$2b$12$Hbl2RW8fCzaXJy73q3Otjer2.Z.D2cmRsXdPve04Wq35iDdvBwPcK',
    'Operator SPBU Palembang',
    TRUE
),
(
    'operator.spbu02',
    '$2b$12$8yHlD/069rTsFm2LfIGY6e1y034CbvUC.9/iv4hbcmzRmOeF3UNEK',
    'Operator SPBU Pekanbaru',
    TRUE
),
(
    'operator.spbu03',
    '$2b$12$4zZYi28sFfG9.K0g9uw1X.RULC40YQ4/MiVludwrwSWzCKzAISnZK',
    'Operator SPBU Jambi',
    TRUE
),
(
    'operator.spbu04',
    '$2b$12$0HtMr2cZ3ZjTBTqVuhiYiOpLPJMxWIsu/1CVvpiAZWHFAb.nNReEi',
    'Operator SPBU Balikpapan',
    TRUE
),
(
    'operator.spbu05',
    '$2b$12$pOIfjdF36GMhUnFVZuke7Oswqt5L73wV6PSqDn9q8tpYz8kvVBGZG',
    'Operator SPBU Samarinda',
    TRUE
),
(
    'operator.spbu06',
    '$2b$12$MZq8cZLTMpxksoANesPK9OQKvuSNnq3jHmKPgdT2N.nRCSXKKvkNW',
    'Operator SPBU Bontang',
    TRUE
),
(
    'operator.spbu07',
    '$2b$12$lvw2MYBkftxpbCdCdTpVMeuWNuX/8tmNTTyqERaQy5..lk7VNvAnG',
    'Operator SPBU Semarang',
    TRUE
),
(
    'operator.spbu08',
    '$2b$12$FSvRBhIQTb0jPka9nvmC0eMhE.ZIc7UXAtEjlzdqQFPRjcL1EQTRm',
    'Operator SPBU Yogyakarta',
    TRUE
),
(
    'operator.spbu09',
    '$2b$12$rlDRXvaBvduNIrd4qe4bwey3RkU71iPpEUWLO324gpbo9RmKt7DTi',
    'Operator SPBU Solo',
    TRUE
),
(
    'operator.spbu10',
    '$2b$12$O0GNskKmmzWA7dYnEfCCNO/nuDEmfQeSlUvzRbRi4OlecvTOYYxo.',
    'Operator SPBU Bandung',
    TRUE
),
(
    'operator.spbu11',
    '$2b$12$aakZ9gybKF3SWcrUE9BU..YFOq7YT77EiKhOvKakcMTuz8sLXB.RS',
    'Operator SPBU Cirebon',
    TRUE
),
(
    'operator.spbu12',
    '$2b$12$X0Mhk2BkBtKg.SJiRnNJDO97do2eE/.BY4OF.iV0XRXI1jlKVy2Ti',
    'Operator SPBU Subang',
    TRUE
);

INSERT INTO user_role_grants (user_id, role, scope_type, scope_id)
VALUES (
    (
        SELECT id FROM users
        WHERE username = 'superadmin'
    ), 'SYSTEM_ADMIN', 'COMPANY', NULL
);

INSERT INTO user_role_grants (user_id, role, scope_type, scope_id)
SELECT
    u.id,
    'FACILITY_OPERATOR',
    'FACILITY',
    f.id
FROM (
    VALUES
    ('operator.ru2', 'FAC-DUM'), ('operator.ru3', 'FAC-PLJ'),
    ('operator.ru4', 'FAC-CLP'), ('operator.ru5', 'FAC-BPP'),
    ('operator.ru6', 'FAC-BLG')
) AS m (username, fac_code)
INNER JOIN users AS u ON m.username = u.username
INNER JOIN refinery_facilities AS f ON m.fac_code = f.code;

INSERT INTO user_role_grants (user_id, role, scope_type, scope_id)
VALUES (
    (
        SELECT id FROM users
        WHERE username = 'driver.01'
    ), 'DRIVER', 'COMPANY', NULL
);

INSERT INTO drivers (
    user_id, employee_number, sim_b2_number, sim_b2_expiry, home_depot_id
)
VALUES (
    (
        SELECT id FROM users
        WHERE username = 'driver.01'
    ), 'EMP-DRV-001',
    'SIM-B2-JTG-2024-00001', '2027-12-31',
    (
        SELECT id FROM vehicle_depots
        WHERE code = 'DEPOT-CLP'
    )
);

INSERT INTO gas_stations (
    code,
    name,
    spbu_license_number,
    region_code,
    primary_facility_id,
    location,
    address,
    operating_hours_start,
    operating_hours_end
) VALUES
(
    'SPBU-01', 'SPBU Palembang Ilir Timur', 'SPBU-ID-SUM-2024-001', 'SUMSEL',
    (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-PLJ'
    ),
    ST_SETSRID(ST_MAKEPOINT(104.7619, -2.9909), 4326),
    'Jl. Jenderal Sudirman No.1, Ilir Timur I, Palembang, Sumatera Selatan',
    '05:00',
    '23:00'
),
(
    'SPBU-02', 'SPBU Pekanbaru Sail', 'SPBU-ID-SUM-2024-002', 'RIAU',
    (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-DUM'
    ),
    ST_SETSRID(ST_MAKEPOINT(101.4478, 0.5071), 4326),
    'Jl. Tuanku Tambusai, Sail, Pekanbaru, Riau', '06:00', '22:00'
),
(
    'SPBU-03', 'SPBU Jambi Telanaipura', 'SPBU-ID-SUM-2024-003', 'SUMSEL',
    (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-PLJ'
    ),
    ST_SETSRID(ST_MAKEPOINT(103.6131, -1.6101), 4326),
    'Jl. Gatot Subroto, Telanaipura, Jambi', '06:00', '22:00'
),
(
    'SPBU-04', 'SPBU Balikpapan Klandasan', 'SPBU-ID-KAL-2024-001', 'KALTIM',
    (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-BPP'
    ),
    ST_SETSRID(ST_MAKEPOINT(116.8250, -1.2659), 4326),
    'Jl. Jenderal Sudirman, Klandasan Ulu, Balikpapan, Kalimantan Timur',
    '00:00',
    '23:59'
),
(
    'SPBU-05',
    'SPBU Samarinda Sungai Kunjang',
    'SPBU-ID-KAL-2024-002',
    'KALTIM',
    (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-BPP'
    ),
    ST_SETSRID(ST_MAKEPOINT(117.1253, -0.4948), 4326),
    'Jl. MT Haryono, Sungai Kunjang, Samarinda, Kalimantan Timur',
    '05:00',
    '23:00'
),
(
    'SPBU-06', 'SPBU Bontang Bontang Baru', 'SPBU-ID-KAL-2024-003', 'KALTIM',
    (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-BPP'
    ),
    ST_SETSRID(ST_MAKEPOINT(117.5000, 0.1333), 4326),
    'Jl. Awang Long, Bontang Baru, Bontang, Kalimantan Timur', '06:00', '22:00'
),
(
    'SPBU-07', 'SPBU Semarang Gajahmungkur', 'SPBU-ID-JAV-2024-001', 'JATENG',
    (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-CLP'
    ),
    ST_SETSRID(ST_MAKEPOINT(110.4083, -6.9854), 4326),
    'Jl. Mgr Soegiyopranoto, Gajahmungkur, Semarang, Jawa Tengah',
    '00:00',
    '23:59'
),
(
    'SPBU-08', 'SPBU Yogyakarta Gondomanan', 'SPBU-ID-JAV-2024-002', 'JATENG',
    (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-CLP'
    ),
    ST_SETSRID(ST_MAKEPOINT(110.3643, -7.7956), 4326),
    'Jl. Jenderal Sudirman, Gondomanan, Yogyakarta, Daerah Istimewa Yogyakarta',
    '00:00',
    '23:59'
),
(
    'SPBU-09', 'SPBU Solo Laweyan', 'SPBU-ID-JAV-2024-003', 'JATENG',
    (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-CLP'
    ),
    ST_SETSRID(ST_MAKEPOINT(110.8003, -7.5563), 4326),
    'Jl. Slamet Riyadi, Laweyan, Surakarta, Jawa Tengah', '05:00', '23:00'
),
(
    'SPBU-10', 'SPBU Bandung Coblong', 'SPBU-ID-JAV-2024-004', 'JABAR',
    (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-BLG'
    ),
    ST_SETSRID(ST_MAKEPOINT(107.6191, -6.8905), 4326),
    'Jl. Ir. H. Juanda, Coblong, Bandung, Jawa Barat', '00:00', '23:59'
),
(
    'SPBU-11', 'SPBU Cirebon Kejaksan', 'SPBU-ID-JAV-2024-005', 'JABAR',
    (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-BLG'
    ),
    ST_SETSRID(ST_MAKEPOINT(108.5522, -6.7321), 4326),
    'Jl. Siliwangi, Kejaksan, Cirebon, Jawa Barat', '05:00', '23:00'
),
(
    'SPBU-12', 'SPBU Subang Kota', 'SPBU-ID-JAV-2024-006', 'JABAR',
    (
        SELECT id FROM refinery_facilities
        WHERE code = 'FAC-BLG'
    ),
    ST_SETSRID(ST_MAKEPOINT(107.7589, -6.5701), 4326),
    'Jl. Otto Iskandar Dinata, Subang, Jawa Barat', '06:00', '22:00'
);

INSERT INTO station_facility_whitelist (station_id, facility_id)
SELECT
    s.id,
    f.id
FROM (
    VALUES
    ('SPBU-01', 'FAC-PLJ'), ('SPBU-01', 'FAC-SGR'),
    ('SPBU-02', 'FAC-DUM'), ('SPBU-02', 'FAC-SKP'),
    ('SPBU-03', 'FAC-PLJ'), ('SPBU-03', 'FAC-SGR'),
    ('SPBU-04', 'FAC-BPP'), ('SPBU-05', 'FAC-BPP'), ('SPBU-06', 'FAC-BPP'),
    ('SPBU-07', 'FAC-CLP'), ('SPBU-08', 'FAC-CLP'), ('SPBU-09', 'FAC-CLP'),
    ('SPBU-10', 'FAC-BLG'), ('SPBU-11', 'FAC-BLG'), ('SPBU-12', 'FAC-BLG')
) AS m (station_code, fac_code)
INNER JOIN gas_stations AS s ON m.station_code = s.code
INNER JOIN refinery_facilities AS f ON m.fac_code = f.code;

INSERT INTO station_qr_codes (station_id, qr_payload, label)
SELECT
    id,
    'STA-QR-' || code || '-' || GEN_RANDOM_UUID(),
    'Delivery Point A'
FROM gas_stations;

INSERT INTO station_tanks (
    station_id,
    tank_code,
    fuel_type_code,
    capacity_l,
    current_volume_l,
    reorder_threshold_l
)
SELECT
    s.id,
    'TK-' || ft.code,
    ft.code,
    CASE ft.category WHEN 'GASOLINE' THEN 32000 ELSE 24000 END,
    CASE ft.category WHEN 'GASOLINE' THEN 16000 ELSE 12000 END,
    CASE ft.category WHEN 'GASOLINE' THEN 8000 ELSE 6000 END
FROM gas_stations AS s
CROSS JOIN fuel_types AS ft
WHERE ft.code IN ('PERTALITE', 'BIOSOLAR');

INSERT INTO station_tanks (
    station_id,
    tank_code,
    fuel_type_code,
    capacity_l,
    current_volume_l,
    reorder_threshold_l
)
SELECT
    s.id,
    'TK-' || ft.code,
    ft.code,
    16000,
    8000,
    4000
FROM gas_stations AS s CROSS JOIN fuel_types AS ft
WHERE s.code BETWEEN 'SPBU-07' AND 'SPBU-12' AND ft.code = 'PERTAMAX';

INSERT INTO user_role_grants (user_id, role, scope_type, scope_id)
SELECT
    u.id,
    'STATION_MANAGER',
    'STATION',
    s.id
FROM (
    VALUES
    ('operator.spbu01', 'SPBU-01'),
    ('operator.spbu02', 'SPBU-02'),
    ('operator.spbu03', 'SPBU-03'),
    ('operator.spbu04', 'SPBU-04'),
    ('operator.spbu05', 'SPBU-05'),
    ('operator.spbu06', 'SPBU-06'),
    ('operator.spbu07', 'SPBU-07'),
    ('operator.spbu08', 'SPBU-08'),
    ('operator.spbu09', 'SPBU-09'),
    ('operator.spbu10', 'SPBU-10'),
    ('operator.spbu11', 'SPBU-11'),
    ('operator.spbu12', 'SPBU-12')
) AS m (username, station_code)
INNER JOIN users AS u ON m.username = u.username
INNER JOIN gas_stations AS s ON m.station_code = s.code;
