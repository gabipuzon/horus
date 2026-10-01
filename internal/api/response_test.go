package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func assertJSONError(t *testing.T, recorder *httptest.ResponseRecorder, status int, message string) {
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

func TestWriteError(t *testing.T) {
	for _, test := range []struct {
		status  int
		message string
	}{
		{http.StatusBadRequest, "invalid request body"},
		{http.StatusNotFound, "monitor not found"},
		{http.StatusInternalServerError, "internal server error"},
	} {
		recorder := httptest.NewRecorder()
		writeError(recorder, test.status, test.message)
		assertJSONError(t, recorder, test.status, test.message)
	}
}

func TestWriteInternalErrorLogsContextWithoutSecrets(t *testing.T) {
	var output bytes.Buffer
	previous := log.Writer()
	log.SetOutput(&output)
	t.Cleanup(func() { log.SetOutput(previous) })

	recorder := httptest.NewRecorder()
	writeInternalError(recorder, "list checks", errors.New("postgres password=secret"))
	assertJSONError(t, recorder, http.StatusInternalServerError, "internal server error")
	if !strings.Contains(output.String(), "list checks failed") || strings.Contains(output.String(), "secret") {
		t.Fatalf("expected operation without credentials in log, got %q", output.String())
	}
}
