package api

import (
	"encoding/json"
	"log"
	"net/http"
)

type errorResponse struct {
	Error string `json:"error"`
}

func writeError(w http.ResponseWriter, status int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(errorResponse{Error: message}); err != nil {
		log.Printf("failed to write API error response (%T)", err)
	}
}

func writeInternalError(w http.ResponseWriter, operation string, err error) {
	// Repository errors can contain connection details or credentials. Log the
	// operation and error type without copying potentially sensitive text.
	log.Printf("%s failed (%T)", operation, err)
	writeError(w, http.StatusInternalServerError, "internal server error")
}

func logResponseWriteError(operation string, err error) {
	// A write may already have committed the response; do not append a second
	// error body or attempt to change its status.
	log.Printf("%s response write failed (%T)", operation, err)
}
