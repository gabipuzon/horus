package monitor

import (
	"errors"
	"time"
)

type Monitor struct {
	ID             string
	Name           string
	URL            string
	Interval       time.Duration
	Timeout        time.Duration
	ExpectedStatus int
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func New(
	id string,
	name string,
	url string,
	interval time.Duration,
	timeout time.Duration,
	expectedStatus int,
) (*Monitor, error) {
	if id == "" {
		return nil, errors.New("monitor id is required")
	}

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
		ID:             id,
		Name:           name,
		URL:            url,
		Interval:       interval,
		Timeout:        timeout,
		ExpectedStatus: expectedStatus,
		Enabled:        true,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}
