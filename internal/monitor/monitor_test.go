package monitor

import (
	"testing"
	"time"
)

func TestNewMonitor(t *testing.T) {
	m, err := New(
		"monitor-1",
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
		"monitor-1",
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
