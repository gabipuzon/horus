CREATE TABLE incidents (
    id UUID PRIMARY KEY,
    monitor_id UUID NOT NULL REFERENCES monitors(id) ON DELETE CASCADE,
    started_at TIMESTAMPTZ NOT NULL,
    resolved_at TIMESTAMPTZ,
    failure_type TEXT NOT NULL,
    status_code INTEGER NOT NULL DEFAULT 0,
    failure_message TEXT NOT NULL DEFAULT ''
);

CREATE UNIQUE INDEX idx_incidents_one_open_per_monitor
ON incidents (monitor_id)
WHERE resolved_at IS NULL;

CREATE INDEX idx_incidents_monitor_id_started_at
ON incidents (monitor_id, started_at DESC);
