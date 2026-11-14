package main

import (
	"fmt"
	"net/http"
	"strconv"
	"time"
)

func main() {
	http.HandleFunc("/check", func(w http.ResponseWriter, r *http.Request) {
		id, err := strconv.Atoi(r.URL.Query().Get("id"))
		if err != nil || id < 0 {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		var delay time.Duration
		switch {
		case id%100 == 0:
			delay = 10 * time.Second // Exceeds the benchmark's five-second check timeout.
		case id%20 == 0:
			delay = 3 * time.Second
		default:
			delay = time.Duration(50+(id*37)%451) * time.Millisecond
		}
		select {
		case <-time.After(delay):
		case <-r.Context().Done():
			return
		}
		if id%50 == 1 {
			http.Error(w, "simulated failure", http.StatusInternalServerError)
			return
		}
		fmt.Fprintln(w, "ok")
	})
	http.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(w, "ok")
	})
	if err := http.ListenAndServe(":8081", nil); err != nil {
		panic(err)
	}
}
