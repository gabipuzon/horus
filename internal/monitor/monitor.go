package monitor

import (
	"errors"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

var ErrNotFound = errors.New("monitor not found")

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
	rawURL string,
	interval time.Duration,
	timeout time.Duration,
	expectedStatus int,
) (*Monitor, error) {
	if strings.TrimSpace(name) == "" {
		return nil, errors.New("monitor name is required")
	}

	if rawURL == "" {
		return nil, errors.New("monitor URL is required")
	}
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil || (parsedURL.Scheme != "http" && parsedURL.Scheme != "https") || parsedURL.Hostname() == "" || parsedURL.User != nil {
		return nil, errors.New("monitor URL must be an absolute HTTP or HTTPS URL without credentials")
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
		URL:            rawURL,
		Interval:       interval,
		Timeout:        timeout,
		ExpectedStatus: expectedStatus,
		Enabled:        true,
		NextCheckAt:    now,
		CreatedAt:      now,
		UpdatedAt:      now,
	}, nil
}
