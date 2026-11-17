package testutil

import (
	"encoding/json"
	"net/http/httptest"
	"testing"
)

func AssertJSONError(t *testing.T, recorder *httptest.ResponseRecorder, status int, message string) {
	t.Helper()
	if recorder.Code != status {
		t.Fatalf("expected status %d, got %d: %s", status, recorder.Code, recorder.Body.String())
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected JSON error, got %q: %v", recorder.Body.String(), err)
	}
	if len(body) != 1 || body["error"] != message {
		t.Fatalf("expected only error %q, got %v", message, body)
	}
}
