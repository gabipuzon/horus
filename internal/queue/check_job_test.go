package queue

import (
	"encoding/json"
	"testing"
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
