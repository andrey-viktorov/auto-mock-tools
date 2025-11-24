package proxy

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

// Recorder writes HTTP request/response pairs to JSON files organized by mock_id.
type Recorder struct {
	baseDir string
	mutex   sync.Mutex
}

// NewRecorder creates a new recorder that writes to the specified directory.
func NewRecorder(baseDir string) (*Recorder, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	return &Recorder{
		baseDir: baseDir,
	}, nil
}

// Close is kept for API compatibility but does nothing now.
func (r *Recorder) Close() error {
	return nil
}

// generateRequestID generates a unique request ID.
func (r *Recorder) generateRequestID() string {
	// Use timestamp + nanoseconds for uniqueness
	return time.Now().Format("20060102150405.999999999")
}

// generateRandomHex generates random hex string for filename uniqueness
func generateRandomHex(n int) string {
	bytes := make([]byte, n)
	rand.Read(bytes)
	return hex.EncodeToString(bytes)
}

// sanitizeContentType converts content-type to safe filename component
func sanitizeContentType(contentType string) string {
	// Remove charset and other params
	if idx := strings.IndexByte(contentType, ';'); idx >= 0 {
		contentType = contentType[:idx]
	}
	contentType = strings.TrimSpace(contentType)

	// Replace slashes and special chars
	contentType = strings.ReplaceAll(contentType, "/", "_")
	contentType = strings.ReplaceAll(contentType, "+", "_")
	contentType = strings.ReplaceAll(contentType, ".", "_")

	if contentType == "" {
		contentType = "unknown"
	}

	return contentType
}

// RequestData holds request information for later writing
type RequestData struct {
	RequestID string
	Timestamp string
	Method    string
	URL       string
	Headers   map[string]string
	Body      interface{}
	MockID    string
}

// parseSSEEvents parses SSE body into array of JSON objects
func parseSSEEvents(body string) ([]interface{}, bool) {
	events := []interface{}{}
	lines := strings.Split(body, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)
		// SSE data lines start with "data: "
		if strings.HasPrefix(line, "data: ") {
			dataStr := strings.TrimPrefix(line, "data: ")
			// Try to parse as JSON
			var jsonData interface{}
			if err := json.Unmarshal([]byte(dataStr), &jsonData); err == nil {
				events = append(events, jsonData)
			} else {
				// If not JSON, store as string
				events = append(events, dataStr)
			}
		}
	}

	// Return true if we found any SSE events
	return events, len(events) > 0
}

// RecordPair records both HTTP request and response to a single JSON file
func (r *Recorder) RecordPair(reqData *RequestData, resp *fasthttp.Response, delay float64) error {
	// Build response headers
	respHeaders := make(map[string]string)
	resp.Header.VisitAll(func(key, value []byte) {
		keyLower := strings.ToLower(string(key))
		// Skip x-mock-id from upstream (will be added from request if provided)
		if keyLower != "x-mock-id" {
			respHeaders[string(key)] = string(value)
		}
	})

	// Add x-mock-id to response headers if provided
	if reqData.MockID != "" {
		respHeaders["x-mock-id"] = reqData.MockID
	}

	// Get content type for filename and processing
	contentType := string(resp.Header.Peek("Content-Type"))
	if contentType == "" {
		contentType = "application/octet-stream"
	}

	// Process body based on content type
	body := resp.Body()
	var bodyData interface{}

	isSSE := contentType == "text/event-stream"
	contentEncoding := string(resp.Header.Peek("Content-Encoding"))

	if contentEncoding == "gzip" {
		bodyData = base64.StdEncoding.EncodeToString(body)
	} else if isSSE {
		events, hasEvents := parseSSEEvents(string(body))
		if hasEvents {
			bodyData = events
		} else {
			bodyData = string(body)
		}
	} else {
		var jsonBody interface{}
		if err := json.Unmarshal(body, &jsonBody); err == nil {
			bodyData = jsonBody
		} else {
			bodyData = string(body)
		}
	}

	// Build complete record
	record := map[string]interface{}{
		"request": map[string]interface{}{
			"request_id": reqData.RequestID,
			"timestamp":  reqData.Timestamp,
			"method":     reqData.Method,
			"url":        reqData.URL,
			"headers":    reqData.Headers,
			"body":       reqData.Body,
		},
		"response": map[string]interface{}{
			"request_id":  reqData.RequestID,
			"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
			"status_code": resp.StatusCode(),
			"headers":     respHeaders,
			"body":        bodyData,
			"delay":       delay,
		},
	}

	// Determine mock_id (default if not set)
	mockID := reqData.MockID
	if mockID == "" {
		mockID = "default"
	}

	// Create directory for mock_id
	mockDir := filepath.Join(r.baseDir, mockID)
	if err := os.MkdirAll(mockDir, 0755); err != nil {
		return err
	}

	// Generate filename: <content-type>_<timestamp>_<random>.json
	timestamp := time.Now().Format("20060102_150405")
	randomHex := generateRandomHex(4)
	safeContentType := sanitizeContentType(contentType)
	filename := fmt.Sprintf("%s_%s_%s.json", safeContentType, timestamp, randomHex)
	filepath := filepath.Join(mockDir, filename)

	// Write JSON file
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}

// RecordSSEPair records SSE request/response with events and timestamps to a single JSON file
func (r *Recorder) RecordSSEPair(reqData *RequestData, resp *fasthttp.Response, events []interface{}, delay float64, savedHeaders map[string]string) error {
	// Use saved headers
	respHeaders := savedHeaders
	if reqData.MockID != "" {
		respHeaders["x-mock-id"] = reqData.MockID
	}

	// Build complete record
	record := map[string]interface{}{
		"request": map[string]interface{}{
			"request_id": reqData.RequestID,
			"timestamp":  reqData.Timestamp,
			"method":     reqData.Method,
			"url":        reqData.URL,
			"headers":    reqData.Headers,
			"body":       reqData.Body,
		},
		"response": map[string]interface{}{
			"request_id":  reqData.RequestID,
			"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
			"status_code": resp.StatusCode(),
			"headers":     respHeaders,
			"body":        events,
			"delay":       delay,
		},
	}

	// Determine mock_id
	mockID := reqData.MockID
	if mockID == "" {
		mockID = "default"
	}

	// Create directory for mock_id
	mockDir := filepath.Join(r.baseDir, mockID)
	if err := os.MkdirAll(mockDir, 0755); err != nil {
		return err
	}

	// Generate filename for SSE
	timestamp := time.Now().Format("20060102_150405")
	randomHex := generateRandomHex(4)
	filename := fmt.Sprintf("text_event-stream_%s_%s.json", timestamp, randomHex)
	filepath := filepath.Join(mockDir, filename)

	// Write JSON file
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filepath, data, 0644)
}
