WITH recent AS (
    SELECT c.*, m.interval_seconds,
           EXTRACT(EPOCH FROM c.started_at - c.scheduled_at) AS lag_s
    FROM checks c JOIN monitors m ON m.id = c.monitor_id
    WHERE c.started_at >= now() - (:'window_seconds'::integer * interval '1 second')
      AND c.scheduled_at IS NOT NULL
), gaps AS (
    SELECT monitor_id, started_at,
           EXTRACT(EPOCH FROM started_at - lag(started_at) OVER
               (PARTITION BY monitor_id ORDER BY started_at)) AS gap_s,
           interval_seconds
    FROM recent
)
SELECT (SELECT count(*) FROM monitors) AS monitors,
       (SELECT count(*) FROM recent) AS completed,
       round((SELECT count(*) FROM recent)::numeric / :'window_seconds'::numeric, 2) AS actual_per_s,
       round((SELECT count(*) FROM monitors)::numeric / 30, 2) AS expected_per_s,
       round(percentile_cont(0.50) WITHIN GROUP (ORDER BY lag_s)::numeric, 3) AS p50_lag_s,
       round(percentile_cont(0.95) WITHIN GROUP (ORDER BY lag_s)::numeric, 3) AS p95_lag_s,
       round(percentile_cont(0.99) WITHIN GROUP (ORDER BY lag_s)::numeric, 3) AS p99_lag_s,
       count(*) - count(DISTINCT (monitor_id, scheduled_at)) AS duplicate_due_checks,
       count(*) FILTER (WHERE failure_type = 'timeout') AS timeouts,
       count(*) FILTER (WHERE status_code = 500) AS http_500,
       (SELECT count(*) FROM gaps WHERE gap_s > interval_seconds * 1.5) AS gaps_over_1_5x
FROM recent;
WITH recent AS (
    SELECT started_at, EXTRACT(EPOCH FROM started_at - scheduled_at) AS lag_s
    FROM checks
    WHERE started_at >= now() - (:'window_seconds'::integer * interval '1 second')
      AND scheduled_at IS NOT NULL
), bucketed AS (
    SELECT floor(EXTRACT(EPOCH FROM started_at - (SELECT min(started_at) FROM recent)) / 600)::integer AS ten_minute_bucket,
           lag_s
    FROM recent
)
SELECT ten_minute_bucket, count(*) AS completed,
       round(percentile_cont(0.95) WITHIN GROUP (ORDER BY lag_s)::numeric, 3) AS p95_lag_s
FROM bucketed
GROUP BY ten_minute_bucket ORDER BY ten_minute_bucket;
SELECT pg_size_pretty(pg_total_relation_size('checks')) AS checks_total_size,
       pg_size_pretty(pg_database_size(current_database())) AS database_size;
SELECT calls, round(mean_exec_time::numeric, 3) AS mean_check_insert_ms,
       round(max_exec_time::numeric, 3) AS max_check_insert_ms,
       round(total_exec_time::numeric, 1) AS total_check_insert_ms
FROM pg_stat_statements
WHERE query ILIKE '%INSERT INTO checks%'
ORDER BY calls DESC LIMIT 1;
