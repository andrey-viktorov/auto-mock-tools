package main

import (
	"bufio"
	"fmt"
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

func main() {
	handler := func(ctx *fasthttp.RequestCtx) {
		// Set SSE headers
		ctx.Response.Header.Set("Content-Type", "text/event-stream; charset=utf-8")
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")

		// Stream 3 events with delays
		ctx.Response.SetBodyStreamWriter(func(w *bufio.Writer) {
			events := []string{
				`{"event": "message_0", "value": 0}`,
				`{"event": "message_1", "value": 10}`,
				`{"event": "message_2", "value": 20}`,
			}

			for i, eventData := range events {
				if i > 0 {
					time.Sleep(100 * time.Millisecond)
				}
				fmt.Fprintf(w, "data: %s\n\n", eventData)
				w.Flush()
			}
		})
	}

	log.Println("Starting HTTPS SSE test server on :5557")

	// Use ListenAndServeTLS for HTTPS
	// For testing, we'll generate a self-signed certificate on the fly
	// In production, use proper certificates
	if err := fasthttp.ListenAndServeTLS(":5557", "../certs/cert.pem", "../certs/key.pem", handler); err != nil {
		log.Fatalf("Error in server: %s", err)
	}
}
