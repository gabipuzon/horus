CREATE TABLE checks (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    status_code INTEGER,
    latency_ms BIGINT NOT NULL,
    success BOOLEAN NOT NULL,
    failure_type TEXT NOT NULL,
    error TEXT,
    checked_at TIMESTAMPTZ NOT NULL
);

CREATE INDEX idx_checks_monitor_id_checked_at
ON checks (monitor_id, checked_at DESC);