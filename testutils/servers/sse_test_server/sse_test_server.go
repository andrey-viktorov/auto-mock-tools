package main

import (
	"fmt"
	"log"
	"net/http"
	"time"
)

func sseHandler(w http.ResponseWriter, r *http.Request) {
	// CRITICAL: Delete Content-Type first to prevent Go from setting it automatically
	w.Header().Del("Content-Type")

	// Set SSE headers
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	// Send 3 events
	for i := 0; i < 3; i++ {
		data := fmt.Sprintf(`{"event": "message_%d", "value": %d}`, i, i*10)
		fmt.Fprintf(w, "data: %s\n\n", data)

		// Flush immediately
		if flusher, ok := w.(http.Flusher); ok {
			flusher.Flush()
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func main() {
	http.HandleFunc("/events", sseHandler)
	log.Println("SSE test server listening on :5555")
	log.Fatal(http.ListenAndServe(":5555", nil))
}
