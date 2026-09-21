package monitor

import (
	"context"
	"errors"
	"net/http"
	"time"
)

type FailureType string

const (
	FailureNone    FailureType = ""
	FailureHTTP    FailureType = "http"
	FailureNetwork FailureType = "network"
	FailureTimeout FailureType = "timeout"
)

type CheckResult struct {
	StatusCode  int
	Latency     time.Duration
	Success     bool
	FailureType FailureType
	Error       error
}

type Checker struct {
	client *http.Client
}

func NewChecker(client *http.Client) *Checker {
	return &Checker{
		client: client,
	}
}

func (c *Checker) Check(ctx context.Context, m *Monitor) CheckResult {
	start := time.Now()

	ctx, cancel := context.WithTimeout(ctx, m.Timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(
		ctx,
		http.MethodGet,
		m.URL,
		nil,
	)
	if err != nil {
		return CheckResult{
			Latency:     time.Since(start),
			FailureType: FailureNetwork,
			Error:       err,
		}
	}

	response, err := c.client.Do(req)
	if err != nil {
		failureType := FailureNetwork

		if errors.Is(err, context.DeadlineExceeded) {
			failureType = FailureTimeout
		}

		return CheckResult{
			Latency:     time.Since(start),
			FailureType: failureType,
			Error:       err,
		}
	}

	defer response.Body.Close()

	success := response.StatusCode == m.ExpectedStatus

	result := CheckResult{
		StatusCode: response.StatusCode,
		Latency:    time.Since(start),
		Success:    success,
	}

	if !success {
		result.FailureType = FailureHTTP
	}

	return result
}
