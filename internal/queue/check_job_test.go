package queue

import (
	"encoding/json"
	"testing"
	"time"
)

func TestCheckJobJSON(t *testing.T) {
	job := CheckJob{
		MonitorID: "monitor-123",
	}

	data, err := json.Marshal(job)
	if err != nil {
		t.Fatalf("failed to marshal check job: %v", err)
	}

	expected := `{"monitor_id":"monitor-123"}`

	if string(data) != expected {
		t.Fatalf("expected %s, got %s", expected, string(data))
	}
}

func TestCheckJobPreservesScheduledTime(t *testing.T) {
	want := CheckJob{MonitorID: "monitor-123", ScheduledAt: time.Date(2026, 10, 5, 10, 0, 0, 0, time.UTC)}
	data, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	var got CheckJob
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("job timing changed in Redis payload: got %+v, want %+v", got, want)
	}
}
