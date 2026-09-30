package monitor

import (
	"testing"
	"time"
)

func TestNewMonitor(t *testing.T) {
	m, err := New(
		"My Website",
		"https://example.com",
		5*time.Minute,
		10*time.Second,
		200,
	)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if m.Name != "My Website" {
		t.Errorf("expected name %q, got %q", "My Website", m.Name)
	}

	if m.URL != "https://example.com" {
		t.Errorf("expected URL %q, got %q", "https://example.com", m.URL)
	}

	if !m.Enabled {
		t.Error("expected monitor to be enabled")
	}
}

func TestNewMonitorRejectsInvalidInterval(t *testing.T) {
	_, err := New(
		"My Website",
		"https://example.com",
		0,
		10*time.Second,
		200,
	)

	if err == nil {
		t.Fatal("expected an error")
	}
}

func TestNewMonitorRejectsInvalidFields(t *testing.T) {
	for _, test := range []struct {
		name           string
		monitorName    string
		url            string
		interval       time.Duration
		timeout        time.Duration
		expectedStatus int
	}{
		{name: "blank name", monitorName: "  ", url: "https://example.com", interval: time.Minute, timeout: time.Second, expectedStatus: 200},
		{name: "empty URL", monitorName: "Example", interval: time.Minute, timeout: time.Second, expectedStatus: 200},
		{name: "relative URL", monitorName: "Example", url: "not-a-url", interval: time.Minute, timeout: time.Second, expectedStatus: 200},
		{name: "unsupported scheme", monitorName: "Example", url: "ftp://example.com", interval: time.Minute, timeout: time.Second, expectedStatus: 200},
		{name: "URL credentials", monitorName: "Example", url: "https://user:password@example.com", interval: time.Minute, timeout: time.Second, expectedStatus: 200},
		{name: "zero timeout", monitorName: "Example", url: "https://example.com", interval: time.Minute, expectedStatus: 200},
		{name: "invalid status", monitorName: "Example", url: "https://example.com", interval: time.Minute, timeout: time.Second, expectedStatus: 600},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, err := New(test.monitorName, test.url, test.interval, test.timeout, test.expectedStatus); err == nil {
				t.Fatal("expected invalid monitor to be rejected")
			}
		})
	}
}
