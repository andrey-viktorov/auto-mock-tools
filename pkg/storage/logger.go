package storage

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

// NotFoundLogger writes 404 request/response pairs to JSON files.
type NotFoundLogger struct {
	baseDir string
}

// NewNotFoundLogger creates a new logger that writes to the specified directory.
func NewNotFoundLogger(baseDir string) (*NotFoundLogger, error) {
	// Create base directory if it doesn't exist
	if err := os.MkdirAll(baseDir, 0755); err != nil {
		return nil, err
	}

	return &NotFoundLogger{
		baseDir: baseDir,
	}, nil
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

// LogNotFound logs a 404 request with its response to a JSON file.
func (l *NotFoundLogger) LogNotFound(ctx *fasthttp.RequestCtx) error {
	// Generate request ID and timestamp
	requestID := time.Now().Format("20060102150405.999999999")
	timestamp := time.Now().UTC().Format(time.RFC3339Nano)

	// Build request headers
	reqHeaders := make(map[string]string)
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		reqHeaders[string(key)] = string(value)
	})

	// Parse request body
	var reqBody interface{}
	bodyBytes := ctx.PostBody()
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &reqBody); err == nil {
			// Successfully parsed as JSON
		} else {
			// Not JSON, store as string
			reqBody = string(bodyBytes)
		}
	} else {
		reqBody = ""
	}

	// Build response headers
	respHeaders := make(map[string]string)
	ctx.Response.Header.VisitAll(func(key, value []byte) {
		respHeaders[string(key)] = string(value)
	})

	// Parse response body (should be JSON error message)
	var respBody interface{}
	respBodyBytes := ctx.Response.Body()
	if len(respBodyBytes) > 0 {
		if err := json.Unmarshal(respBodyBytes, &respBody); err == nil {
			// Successfully parsed as JSON
		} else {
			// Not JSON, store as string
			respBody = string(respBodyBytes)
		}
	} else {
		respBody = ""
	}

	// Build complete record
	record := map[string]interface{}{
		"request": map[string]interface{}{
			"request_id": requestID,
			"timestamp":  timestamp,
			"method":     string(ctx.Method()),
			"url":        string(ctx.RequestURI()),
			"headers":    reqHeaders,
			"body":       reqBody,
		},
		"response": map[string]interface{}{
			"request_id":  requestID,
			"timestamp":   time.Now().UTC().Format(time.RFC3339Nano),
			"status_code": ctx.Response.StatusCode(),
			"headers":     respHeaders,
			"body":        respBody,
			"delay":       0,
		},
	}

	// Get content type for filename (use Accept header from request)
	contentType := string(ctx.Request.Header.Peek("Accept"))
	if contentType == "" || contentType == "*/*" {
		contentType = "application/json"
	} else {
		// Use first content type from Accept header
		if idx := strings.IndexByte(contentType, ','); idx >= 0 {
			contentType = contentType[:idx]
		}
		if idx := strings.IndexByte(contentType, ';'); idx >= 0 {
			contentType = contentType[:idx]
		}
		contentType = strings.TrimSpace(contentType)
	}

	// Generate filename: <content-type>_<timestamp>_<random>.json
	ts := time.Now().Format("20060102_150405")
	randomHex := generateRandomHex(4)
	safeContentType := sanitizeContentType(contentType)
	filename := fmt.Sprintf("%s_%s_%s.json", safeContentType, ts, randomHex)
	filePath := filepath.Join(l.baseDir, filename)

	// Write JSON file
	data, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(filePath, data, 0644)
}
