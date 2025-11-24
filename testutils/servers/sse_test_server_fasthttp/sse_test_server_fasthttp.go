package main

import (
	"bufio"
	"fmt"
	"log"
	"time"

	"github.com/valyala/fasthttp"
)

func sseHandler(ctx *fasthttp.RequestCtx) {
	// Set SSE headers
	ctx.Response.Header.Set("Content-Type", "text/event-stream; charset=utf-8")
	ctx.Response.Header.Set("Cache-Control", "no-cache")
	ctx.Response.Header.Set("Connection", "keep-alive")
	ctx.SetStatusCode(fasthttp.StatusOK)

	// Stream events
	ctx.Response.SetBodyStreamWriter(func(w *bufio.Writer) {
		for i := 0; i < 3; i++ {
			data := fmt.Sprintf(`{"event": "message_%d", "value": %d}`, i, i*10)
			fmt.Fprintf(w, "data: %s\n\n", data)
			w.Flush()
			time.Sleep(100 * time.Millisecond)
		}
	})
}

func main() {
	server := &fasthttp.Server{
		Handler: sseHandler,
		Name:    "SSE-Test-Server",
	}

	addr := ":5555"
	log.Printf("SSE test server (fasthttp) listening on %s", addr)
	if err := server.ListenAndServe(addr); err != nil {
		log.Fatalf("Error in ListenAndServe: %v", err)
	}
}
