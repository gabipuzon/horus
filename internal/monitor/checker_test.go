package monitor

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestCheckerTimeout(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			time.Sleep(500 * time.Millisecond)
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	m, err := New(
		"Slow Server",
		server.URL,
		time.Minute,
		50*time.Millisecond,
		http.StatusOK,
	)

	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	checker := NewChecker(&http.Client{})

	result := checker.Check(context.Background(), m)

	if result.Error == nil {
		t.Fatal("expected timeout error")
	}

	if result.Latency > 200*time.Millisecond {
		t.Fatalf("request took too long: %v", result.Latency)
	}

	if result.FailureType != FailureTimeout {
		t.Fatalf("expected timeout failure, got %q", result.FailureType)
	}
}

func TestChecker(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)
	defer server.Close()

	m, err := New(
		"Test Server",
		server.URL,
		time.Minute,
		5*time.Second,
		http.StatusOK,
	)

	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	checker := NewChecker(&http.Client{})

	result := checker.Check(context.Background(), m)

	if result.Error != nil {
		t.Fatalf("expected no error, got %v", result.Error)
	}

	if result.StatusCode != http.StatusOK {
		t.Fatalf("expected status 200, got %d", result.StatusCode)
	}

	if !result.Success {
		t.Fatal("expected check to succeed")
	}

	if result.Latency <= 0 {
		t.Fatal("expected latency to be greater than zero")
	}

	if result.FailureType != FailureNone {
		t.Fatalf("expected no failure, got %q", result.FailureType)
	}
}

func TestCheckerHTTPFailure(t *testing.T) {
	server := httptest.NewServer(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}),
	)
	defer server.Close()

	m, err := New(
		"Broken Server",
		server.URL,
		time.Minute,
		5*time.Second,
		http.StatusOK,
	)

	if err != nil {
		t.Fatalf("failed to create monitor: %v", err)
	}

	checker := NewChecker(&http.Client{})

	result := checker.Check(context.Background(), m)

	if result.Success {
		t.Fatal("expected check to fail")
	}

	if result.StatusCode != http.StatusInternalServerError {
		t.Fatalf(
			"expected status 500, got %d",
			result.StatusCode,
		)
	}

	if result.FailureType != FailureHTTP {
		t.Fatalf(
			"expected HTTP failure, got %q",
			result.FailureType,
		)
	}
}
