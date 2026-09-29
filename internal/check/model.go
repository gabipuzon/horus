package check

import "time"

type Record struct {
	ID          string
	MonitorID   string
	StatusCode  int
	Latency     time.Duration
	Success     bool
	FailureType FailureType
	Error       string
	CheckedAt   time.Time
}

type Summary struct {
	TotalChecks      int
	SuccessfulChecks int
	FailedChecks     int
	AverageLatency   time.Duration
	LatestStatus     int
}
