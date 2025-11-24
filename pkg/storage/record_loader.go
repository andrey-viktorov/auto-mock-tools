package storage

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/url"
	"os"
	"strings"
)

// loadResponseFromFile loads a single mock response from disk using the same
// semantics as directory-based loading. The returned MockResponse is ready to
// be indexed or reused by scenario definitions.
func loadResponseFromFile(filePath string, fallbackMockID string) (*MockResponse, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}
	return parseMockRecord(data, fallbackMockID)
}

func parseMockRecord(data []byte, fallbackMockID string) (*MockResponse, error) {
	var record map[string]interface{}
	if err := json.Unmarshal(data, &record); err != nil {
		return nil, err
	}

	requestData, hasRequest := record["request"].(map[string]interface{})
	responseData, hasResponse := record["response"].(map[string]interface{})
	if !hasRequest || !hasResponse {
		return nil, errInvalidRecord
	}

	urlStr, _ := requestData["url"].(string)
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	path := parsedURL.Path
	if path == "" {
		path = "/"
	}

	mockID := fallbackMockID
	if headers, ok := requestData["headers"].(map[string]interface{}); ok {
		headersLower := make(map[string]interface{})
		for k, v := range headers {
			headersLower[strings.ToLower(k)] = v
		}
		if id, ok := headersLower["x-mock-id"].(string); ok && id != "" {
			mockID = id
		}
	}

	responseHeaders, _ := responseData["headers"].(map[string]interface{})
	responseHeadersStr := make(map[string]string)
	responseHeadersLower := make(map[string]string)
	for k, v := range responseHeaders {
		if str, ok := v.(string); ok {
			responseHeadersStr[k] = str
			responseHeadersLower[strings.ToLower(k)] = str
		}
	}

	contentType := responseHeadersLower["content-type"]
	if contentType != "" {
		contentType = strings.Split(contentType, ";")[0]
		contentType = strings.TrimSpace(contentType)
	} else {
		contentType = "application/json"
	}

	body := responseData["body"]
	if bodyStr, ok := body.(string); ok && bodyStr != "" {
		if responseHeadersLower["content-encoding"] == "gzip" {
			bodyBytes, err := base64.StdEncoding.DecodeString(bodyStr)
			if err == nil {
				gzReader, err := gzip.NewReader(bytes.NewReader(bodyBytes))
				if err == nil {
					decompressed, err := io.ReadAll(gzReader)
					gzReader.Close()
					if err == nil {
						var jsonBody interface{}
						if err := json.Unmarshal(decompressed, &jsonBody); err == nil {
							body = jsonBody
						} else {
							body = string(decompressed)
						}
					}
				}
			}
		}
	}

	method, _ := requestData["method"].(string)
	if method == "" {
		method = "GET"
	}

	statusCode := 200
	if sc, ok := responseData["status_code"].(float64); ok {
		statusCode = int(sc)
	}

	requestID, _ := requestData["request_id"].(string)

	var bodyBytes []byte
	var serErr error
	if contentType == "text/event-stream" {
		if arr, ok := body.([]interface{}); ok {
			var sseBuilder strings.Builder
			for _, event := range arr {
				// Extract data field from event object
				if eventMap, ok := event.(map[string]interface{}); ok {
					if eventData, hasData := eventMap["data"]; hasData {
						// Special handling for [DONE] - send without quotes
						if str, ok := eventData.(string); ok && str == "[DONE]" {
							sseBuilder.WriteString("data: [DONE]\n\n")
						} else {
							eventJSON, err := json.Marshal(eventData)
							if err != nil {
								continue
							}
							sseBuilder.WriteString("data: ")
							sseBuilder.Write(eventJSON)
							sseBuilder.WriteString("\n\n")
						}
					}
				} else {
					// Fallback: treat as direct data
					eventJSON, err := json.Marshal(event)
					if err != nil {
						continue
					}
					sseBuilder.WriteString("data: ")
					sseBuilder.Write(eventJSON)
					sseBuilder.WriteString("\n\n")
				}
			}
			bodyBytes = []byte(sseBuilder.String())
		} else if str, ok := body.(string); ok {
			bodyBytes = []byte(str)
		}
	} else {
		switch v := body.(type) {
		case string:
			bodyBytes = []byte(v)
		case []byte:
			bodyBytes = v
		case map[string]interface{}, []interface{}:
			bodyBytes, serErr = json.Marshal(v)
			if serErr != nil {
				return nil, serErr
			}
		default:
			bodyBytes, serErr = json.Marshal(v)
			if serErr != nil {
				return nil, serErr
			}
		}
	}

	headerKeysLower := make(map[string]string, len(responseHeadersStr))
	for k := range responseHeadersStr {
		headerKeysLower[toLowerASCIISimple(k)] = k
	}

	delay := 0.0
	if d, ok := responseData["delay"].(float64); ok {
		delay = d
	} else if elapsed, ok := responseData["elapsed_seconds"].(float64); ok {
		// Backward compatibility
		delay = elapsed
	}

	var sseEvents []SSEEvent
	isSSE := contentType == "text/event-stream"
	if isSSE {
		if arr, ok := body.([]interface{}); ok {
			for _, eventItem := range arr {
				if eventMap, ok := eventItem.(map[string]interface{}); ok {
					timestamp := 0.0
					if ts, ok := eventMap["timestamp"].(float64); ok {
						timestamp = ts
					}
					if eventData, ok := eventMap["data"]; ok {
						var serializedData []byte
						// Special handling for [DONE] - send without quotes
						if str, ok := eventData.(string); ok && str == "[DONE]" {
							serializedData = []byte("[DONE]")
						} else {
							serializedData, _ = json.Marshal(eventData)
						}
						sseEvents = append(sseEvents, SSEEvent{
							Data:           eventData,
							Timestamp:      timestamp,
							SerializedData: serializedData,
						})
					}
				}
			}
		}
	}

	mockResponse := &MockResponse{
		RequestID:       requestID,
		Path:            path,
		Method:          method,
		MethodBytes:     []byte(method),
		MockID:          mockID,
		ContentType:     contentType,
		StatusCode:      statusCode,
		Headers:         responseHeadersStr,
		HeaderKeysLower: headerKeysLower,
		Body:            bodyBytes,
		OriginalBody:    body,
		FullURL:         urlStr,
		Delay:           delay,
		SSEEvents:       sseEvents,
		IsSSE:           isSSE,
	}

	return mockResponse, nil
}
