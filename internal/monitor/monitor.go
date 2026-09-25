package monitor

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

type Monitor struct {
	ID             string
	Name           string
	URL            string
	Interval       time.Duration
	Timeout        time.Duration
	ExpectedStatus int
	Enabled        bool
	NextCheckAt    time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func New(
	name string,
	url string,
	interval time.Duration,
	timeout time.Duration,
	expectedStatus int,
) (*Monitor, error) {
	if name == "" {
		return nil, errors.New("monitor name is required")
	}

	if url == "" {
		return nil, errors.New("monitor URL is required")
	}

	if interval <= 0 {
		return nil, errors.New("monitor interval must be greater than zero")
	}

	if timeout <= 0 {
		return nil, errors.New("monitor timeout must be greater than zero")
	}

	if expectedStatus < 100 || expectedStatus > 599 {
		return nil, errors.New("invalid expected status code")
	}

	now := time.Now()

	return &Monitor{
		ID:             uuid.NewString(),
		Name:           name,
		URL:            url,
		Interval:       interval,
		Timeout:        timeout,
		ExpectedStatus: expectedStatus,
		Enabled:        true,
		NextCheckAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}
