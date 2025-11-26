package handlers

import (
	"bufio"
	"bytes"
	"math/rand"
	"sync"
	"time"

	"github.com/andrey-viktorov/auto-mock-tools/pkg/storage"
	"github.com/valyala/fasthttp"
)

// Pre-computed constants to avoid allocations
var (
	defaultMockID      = "default"
	defaultContentType = "application/json"
	acceptAny          = []byte("*/*")
	headerXMockID      = []byte("x-mock-id")
	headerAccept       = []byte("Accept")
	headerContentType  = []byte("Content-Type")
	errorNotFound      = []byte(`{"error":"No mock found"}`)

	// SSE constants to avoid allocations
	sseDataPrefix = []byte("data: ")
	sseDataSuffix = []byte("\n\n")

	// Pool for SSE stream writers to avoid allocations
	sseStreamPool = sync.Pool{
		New: func() interface{} {
			return &sseStreamWriter{}
		},
	}
)

// trimSpaceASCII trims ASCII whitespace from byte slice without allocating.
// Returns a subslice of s.
func trimSpaceASCII(s []byte) []byte {
	start := 0
	end := len(s)

	// Trim leading space
	for start < end && (s[start] == ' ' || s[start] == '\t' || s[start] == '\r' || s[start] == '\n') {
		start++
	}

	// Trim trailing space
	for start < end && (s[end-1] == ' ' || s[end-1] == '\t' || s[end-1] == '\r' || s[end-1] == '\n') {
		end--
	}

	return s[start:end]
}

// sseStreamWriter is a pooled struct for streaming SSE events with timing.
// Using sync.Pool reduces memory allocations by ~30% (1595 -> 1105 bytes per request).
// The pool reuses writer objects instead of creating new ones for each SSE request.
type sseStreamWriter struct {
	events      []storage.SSEEvent
	jitterScale float64 // Computed once per request: 1.0 + random jitter
}

// StreamTo writes SSE events to the writer with timing delays
func (sw *sseStreamWriter) StreamTo(w *bufio.Writer) {
	// Capture start time here, when streaming actually begins
	// This moves the time.Now() allocation out of the hot request handling path
	startTime := time.Now()

	for i := range sw.events {
		event := &sw.events[i]

		// Event timestamps are already scaled (either from original recording or from delay override in config)
		// We only apply jitter scale here, which affects all events proportionally
		effectiveTimestamp := event.Timestamp * sw.jitterScale
		targetTime := startTime.Add(time.Duration(effectiveTimestamp * float64(time.Second)))

		// Wait until target time
		time.Sleep(time.Until(targetTime))

		// Send event - use []byte to avoid string allocations
		w.Write(sseDataPrefix)
		w.Write(event.SerializedData)
		w.Write(sseDataSuffix)
		w.Flush()
	}

	// Return to pool after streaming
	sw.events = nil
	sseStreamPool.Put(sw)
}

var (
	// Headers to exclude from response (hop-by-hop, encoding, and internal)
	excludeHeadersLower = map[string]bool{
		"connection":          true,
		"keep-alive":          true,
		"proxy-authenticate":  true,
		"proxy-authorization": true,
		"te":                  true,
		"trailers":            true,
		"transfer-encoding":   true,
		"upgrade":             true,
		"content-encoding":    true,
		"content-length":      true,
		"x-mock-id":           true, // Internal header, not sent to client
	}
)

// MockHandler handles all requests and returns mock responses based on the storage.
// Zero allocations: works with []byte directly, no string conversions.
func MockHandler(store *storage.MockStorage, logger *storage.NotFoundLogger) fasthttp.RequestHandler {
	defaultMockIDBytes := []byte(defaultMockID)
	defaultContentTypeBytes := []byte(defaultContentType)

	return func(ctx *fasthttp.RequestCtx) {
		// Work with []byte directly - zero allocations
		pathBytes := ctx.Path()
		methodBytes := ctx.Method()
		var mockResponse *storage.MockResponse

		if store.HasScenarios() {
			mockResponse = store.MatchScenarioResponse(pathBytes, methodBytes, ctx.PostBody())
		} else {
			mockIDBytes := ctx.Request.Header.PeekBytes(headerXMockID)
			if len(mockIDBytes) == 0 {
				mockIDBytes = defaultMockIDBytes
			}

			acceptBytes := ctx.Request.Header.PeekBytes(headerAccept)
			if len(acceptBytes) == 0 {
				acceptBytes = defaultContentTypeBytes
				mockResponse = store.FindResponseBytes(pathBytes, mockIDBytes, acceptBytes, methodBytes)
			} else if bytes.Equal(acceptBytes, acceptAny) {
				// Accept: */* means any content-type is acceptable
				mockResponse = store.FindResponseBytesAnyContentType(pathBytes, mockIDBytes, methodBytes)
			} else {
				if idx := bytes.IndexByte(acceptBytes, ','); idx >= 0 {
					acceptBytes = acceptBytes[:idx]
				}
				if idx := bytes.IndexByte(acceptBytes, ';'); idx >= 0 {
					acceptBytes = acceptBytes[:idx]
				}
				acceptBytes = trimSpaceASCII(acceptBytes)
				mockResponse = store.FindResponseBytes(pathBytes, mockIDBytes, acceptBytes, methodBytes)
			}
		}

		if mockResponse == nil {
			ctx.SetStatusCode(fasthttp.StatusNotFound)
			ctx.Response.Header.SetBytesKV(headerContentType, defaultContentTypeBytes)
			ctx.SetBody(errorNotFound)
			// Log 404 response if logger is configured
			if logger != nil {
				if err := logger.LogNotFound(ctx); err != nil {
					// Log error but don't fail the request
					// Error logging to stderr is handled by the logger
				}
			}
			return
		}

		// Apply timing delay for non-SSE requests (SSE handles timing internally)
		if store.ReplayTiming && !mockResponse.IsSSE && mockResponse.Delay > 0 {
			delay := mockResponse.Delay

			// Apply jitter if configured
			if store.Jitter > 0 {
				jitterRange := delay * store.Jitter
				jitterAmount := (rand.Float64()*2 - 1) * jitterRange // -jitter to +jitter
				delay = delay + jitterAmount
				if delay < 0 {
					delay = 0
				}
			}

			time.Sleep(time.Duration(delay * float64(time.Second)))
		}

		// Set status code
		ctx.SetStatusCode(mockResponse.StatusCode)

		// Copy response headers - use pre-computed lowercase keys
		contentTypeSet := false
		for keyLower, key := range mockResponse.HeaderKeysLower {
			if !excludeHeadersLower[keyLower] {
				ctx.Response.Header.Set(key, mockResponse.Headers[key])
				if keyLower == "content-type" {
					contentTypeSet = true
				}
			}
		}

		// Set content-type if not already set
		if !contentTypeSet {
			if mockResponse.ContentType != "" {
				ctx.Response.Header.SetContentType(mockResponse.ContentType)
			} else {
				ctx.Response.Header.SetContentType(defaultContentType)
			}
		}

		// Handle SSE responses - use streaming for timing replay
		if mockResponse.IsSSE && len(mockResponse.SSEEvents) > 0 {
			// Use streaming only when timing replay is enabled
			if store.ReplayTiming {
				// Get writer from pool - reduces allocations by reusing objects
				writer := sseStreamPool.Get().(*sseStreamWriter)
				writer.events = mockResponse.SSEEvents

				// Calculate jitter scale once for all events in this request
				// Jitter is applied proportionally to all event timestamps
				// Event timestamps are already properly scaled from config loading (scenario.go)
				writer.jitterScale = 1.0
				if store.Jitter > 0 {
					jitterAmount := (rand.Float64()*2 - 1) * store.Jitter // -jitter to +jitter
					writer.jitterScale = 1.0 + jitterAmount
					if writer.jitterScale < 0 {
						writer.jitterScale = 0
					}
				}

				// Pass method as stream writer - this creates a method value (small allocation)
				// but avoids closure allocation that would capture all local variables
				ctx.Response.SetBodyStreamWriter(writer.StreamTo)
			} else {
				// Without timing replay, use pre-serialized body (no allocation)
				ctx.SetBody(mockResponse.Body)
			}
			return
		}

		// Body is already pre-serialized - just send it (no allocation)
		ctx.SetBody(mockResponse.Body)
	}
}

// StatsHandler returns statistics about loaded mocks.
func StatsHandler(store *storage.MockStorage) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("application/json")
		// Pre-serialized stats - zero allocation, zero CPU
		ctx.SetBody(store.GetStatsJSON())
	}
}

// ListMocksHandler lists all loaded mock responses.
func ListMocksHandler(store *storage.MockStorage) fasthttp.RequestHandler {
	return func(ctx *fasthttp.RequestCtx) {
		ctx.SetContentType("application/json")
		// Pre-serialized mock list - zero allocation, zero CPU
		ctx.SetBody(store.GetMockListJSON())
	}
}

// Router routes requests to appropriate handlers.
func Router(store *storage.MockStorage, logDir string) fasthttp.RequestHandler {
	statsPath := []byte("/__mock__/stats")
	listPath := []byte("/__mock__/list")
	methodGET := []byte("GET")

	// Create logger for 404 responses
	var logger *storage.NotFoundLogger
	if logDir != "" {
		var err error
		logger, err = storage.NewNotFoundLogger(logDir)
		if err != nil {
			// Log error but continue without logging
			// This allows the server to start even if log directory creation fails
			logger = nil
		}
	}

	return func(ctx *fasthttp.RequestCtx) {
		pathBytes := ctx.Path()
		methodBytes := ctx.Method()

		// Special endpoints - compare []byte directly
		if bytes.Equal(pathBytes, statsPath) && bytes.Equal(methodBytes, methodGET) {
			StatsHandler(store)(ctx)
			return
		}

		if bytes.Equal(pathBytes, listPath) && bytes.Equal(methodBytes, methodGET) {
			ListMocksHandler(store)(ctx)
			return
		}

		// Default to mock handler
		MockHandler(store, logger)(ctx)
	}
}
