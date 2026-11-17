package queue

import "time"

type CheckJob struct {
	MonitorID   string    `json:"monitor_id"`
	ScheduledAt time.Time `json:"scheduled_at,omitzero"`
}
