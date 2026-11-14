TRUNCATE monitors CASCADE;
INSERT INTO monitors (
    id, name, url, interval_seconds, timeout_seconds, expected_status,
    enabled, created_at, updated_at, next_check_at
)
SELECT gen_random_uuid(), 'bench-' || g,
       'http://target:8081/check?id=' || g,
       30, 5, 200, true, clock_timestamp(), clock_timestamp(),
       clock_timestamp() + interval '10 seconds' + ((g - 1) % 30) * interval '1 second'
FROM generate_series(1, :'n'::integer) AS g;
