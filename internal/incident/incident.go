package incident

import "time"

// Incident represents one continuous period of monitor failure. A nil
// ResolvedAt means the incident is still open.
type Incident struct {
	ID             string
	MonitorID      string
	StartedAt      time.Time
	ResolvedAt     *time.Time
	FailureType    string
	StatusCode     int
	FailureMessage string
}
