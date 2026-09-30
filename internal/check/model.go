package check

import (
	"math"
	"time"
)

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

// UptimePercentage is the share of persisted checks that succeeded, rounded to
// two decimal places. It is undefined when there are no checks.
func (s Summary) UptimePercentage() *float64 {
	if s.TotalChecks == 0 {
		return nil
	}
	value := math.Round(10000*float64(s.SuccessfulChecks)/float64(s.TotalChecks)) / 100
	return &value
}
