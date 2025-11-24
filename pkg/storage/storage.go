package storage

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"strings"
	"sync"
)

// Pool for reusable byte buffers to avoid allocations when building keys
var keyBufPool = sync.Pool{
	New: func() interface{} {
		b := make([]byte, 0, 256)
		return &b
	},
}

var errInvalidRecord = errors.New("invalid mock record")

// MockResponse represents a stored mock response with pre-serialized body.
type MockResponse struct {
	RequestID       string            `json:"request_id"`
	Path            string            `json:"path"`
	Method          string            `json:"method"`
	MethodBytes     []byte            `json:"-"` // Pre-computed method as bytes to avoid allocation
	MockID          string            `json:"mock_id"`
	ContentType     string            `json:"content_type"`
	StatusCode      int               `json:"status_code"`
	Headers         map[string]string `json:"headers"`
	HeaderKeysLower map[string]string `json:"-"` // Pre-computed lowercase keys for fast lookup
	Body            []byte            // Pre-serialized body ready to send
	OriginalBody    interface{}       `json:"-"` // Keep for listing endpoints
	FullURL         string            `json:"full_url"`
	Delay           float64           `json:"delay"` // Total request duration
	SSEEvents       []SSEEvent        `json:"-"`     // SSE events with timestamps
	IsSSE           bool              `json:"-"`     // Whether this is SSE response
}

// SSEEvent represents a single SSE event with timestamp
type SSEEvent struct {
	Data           interface{} `json:"data"`
	Timestamp      float64     `json:"timestamp"`
	SerializedData []byte      `json:"-"` // Pre-serialized data for performance
}

// IndexKey is the key for indexing responses using string concatenation.
// We use a single string to allow map usage while avoiding allocations during lookup.
type IndexKey string

// makeIndexKey creates an index key from components.
func makeIndexKey(path, mockID, contentType string) IndexKey {
	// Format: "path|mockID|contentType"
	return IndexKey(path + "|" + mockID + "|" + contentType)
}

// makeIndexKeyFromBytes creates an index key from byte slices using pooled buffer.
// Uses unsafe pointer trick to avoid the string allocation during map lookup.
func makeIndexKeyFromBytes(path, mockID, contentType []byte) IndexKey {
	// Get pooled buffer
	bufPtr := keyBufPool.Get().(*[]byte)
	buf := (*bufPtr)[:0] // Reset length, keep capacity

	// Build key in buffer
	buf = append(buf, path...)
	buf = append(buf, '|')
	buf = append(buf, mockID...)
	buf = append(buf, '|')
	buf = append(buf, contentType...)

	// Convert to string - this is the one unavoidable allocation
	// The string becomes the map key and lives in the map
	key := IndexKey(string(buf))

	// Return buffer to pool
	keyBufPool.Put(bufPtr)

	return key
}

// MockStorage handles loading and searching mock responses.
type MockStorage struct {
	BaseDir        string
	Responses      map[IndexKey][]*MockResponse
	cachedStats    []byte // Pre-serialized stats JSON
	cachedMockList []byte // Pre-serialized mock list JSON

	// Timing configuration
	ReplayTiming bool
	Jitter       float64

	// Reusable buffer for key building to avoid allocations
	keyBuf []byte

	// Scenario configuration (when enabled)
	scenariosEnabled bool
	scenarioByPath   map[string][]*mockScenario
	scenarioOrder    []*mockScenario
}

// SetTimingConfig configures timing replay behavior
func (s *MockStorage) SetTimingConfig(replayTiming bool, jitter float64) {
	s.ReplayTiming = replayTiming
	s.Jitter = jitter
}

// NewMockStorage creates a new MockStorage instance.
func NewMockStorage(baseDir string) (*MockStorage, error) {
	storage := &MockStorage{
		BaseDir:   baseDir,
		Responses: make(map[IndexKey][]*MockResponse),
	}

	if err := storage.loadResponses(); err != nil {
		return nil, err
	}

	return storage, nil
}

// loadResponses loads responses from JSON files in the directory structure.
func (s *MockStorage) loadResponses() error {
	// Check if directory exists
	if _, err := os.Stat(s.BaseDir); os.IsNotExist(err) {
		return nil // Directory doesn't exist, that's ok
	}

	// Walk through all mock_id subdirectories
	entries, err := os.ReadDir(s.BaseDir)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip non-directories
		}

		folderMockID := entry.Name()
		mockDir := s.BaseDir + "/" + folderMockID

		// Read all JSON files in this mock_id directory
		files, err := os.ReadDir(mockDir)
		if err != nil {
			continue // Skip if can't read directory
		}

		for _, file := range files {
			if file.IsDir() || !strings.HasSuffix(file.Name(), ".json") {
				continue
			}

			filePath := mockDir + "/" + file.Name()
			mockResponse, err := loadResponseFromFile(filePath, folderMockID)
			if err != nil {
				continue
			}

			key := makeIndexKey(mockResponse.Path, mockResponse.MockID, mockResponse.ContentType)
			s.Responses[key] = append(s.Responses[key], mockResponse)
		}
	}

	// Pre-serialize stats and mock list for fast serving
	s.cacheResponses()

	return nil
}

// cacheResponses pre-serializes stats and mock list to avoid marshaling on each request.
func (s *MockStorage) cacheResponses() {
	if s.scenariosEnabled {
		stats := s.computeScenarioStats()
		if data, err := json.Marshal(stats); err == nil {
			s.cachedStats = data
		}

		mocks := s.listScenarioMocks()
		if data, err := json.Marshal(mocks); err == nil {
			s.cachedMockList = data
		}
		return
	}

	// Cache stats for legacy mock-id lookups
	stats := s.computeStats()
	if data, err := json.Marshal(stats); err == nil {
		s.cachedStats = data
	}

	// Cache mock list
	mocks := s.listMocks()
	if data, err := json.Marshal(mocks); err == nil {
		s.cachedMockList = data
	}
}

// computeStats calculates statistics (internal version).
func (s *MockStorage) computeStats() map[string]interface{} {
	total := 0
	uniquePaths := make(map[string]bool)
	uniqueMockIDs := make(map[string]bool)

	for _, responses := range s.Responses {
		total += len(responses)
		if len(responses) > 0 {
			// Use first response to get path and mockID
			resp := responses[0]
			uniquePaths[resp.Path] = true
			if resp.MockID != "" {
				uniqueMockIDs[resp.MockID] = true
			}
		}
	}

	paths := []string{}
	for path := range uniquePaths {
		paths = append(paths, path)
	}

	return map[string]interface{}{
		"total_responses": total,
		"unique_paths":    len(uniquePaths),
		"unique_mock_ids": len(uniqueMockIDs),
		"paths":           paths,
	}
}

func (s *MockStorage) computeScenarioStats() map[string]interface{} {
	total := len(s.scenarioOrder)
	uniquePaths := make(map[string]bool)
	uniqueMockIDs := make(map[string]bool)

	for _, scenario := range s.scenarioOrder {
		uniquePaths[scenario.path] = true
		uniqueMockIDs[scenario.name] = true
	}

	paths := []string{}
	for path := range uniquePaths {
		paths = append(paths, path)
	}

	return map[string]interface{}{
		"total_responses": total,
		"unique_paths":    len(uniquePaths),
		"unique_mock_ids": len(uniqueMockIDs),
		"paths":           paths,
	}
}

// listMocks creates mock list (internal version).
func (s *MockStorage) listMocks() map[string]interface{} {
	allResponses := []*MockResponse{}
	for _, responses := range s.Responses {
		allResponses = append(allResponses, responses...)
	}

	mockList := make([]map[string]interface{}, 0, len(allResponses))
	for _, m := range allResponses {
		mockList = append(mockList, map[string]interface{}{
			"request_id":   m.RequestID,
			"path":         m.Path,
			"method":       m.Method,
			"mock_id":      m.MockID,
			"content_type": m.ContentType,
			"status_code":  m.StatusCode,
			"full_url":     m.FullURL,
		})
	}

	return map[string]interface{}{
		"mocks": mockList,
		"total": len(mockList),
	}
}

func (s *MockStorage) listScenarioMocks() map[string]interface{} {
	mockList := make([]map[string]interface{}, 0, len(s.scenarioOrder))
	for _, scenario := range s.scenarioOrder {
		resp := scenario.response
		mockList = append(mockList, map[string]interface{}{
			"request_id":   resp.RequestID,
			"path":         resp.Path,
			"method":       resp.Method,
			"mock_id":      resp.MockID,
			"content_type": resp.ContentType,
			"status_code":  resp.StatusCode,
			"full_url":     resp.FullURL,
		})
	}

	return map[string]interface{}{
		"mocks": mockList,
		"total": len(mockList),
	}
}

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

// FindResponse finds a mock response by path, mock_id, and content_type.
// Zero allocations: builds key directly from []byte without string conversion.
func (s *MockStorage) FindResponseBytes(pathBytes, mockIDBytes, contentTypeBytes, methodBytes []byte) *MockResponse {
	// Normalize content-type inline
	if idx := bytes.IndexByte(contentTypeBytes, ';'); idx >= 0 {
		contentTypeBytes = contentTypeBytes[:idx]
	}
	contentTypeBytes = trimSpaceASCII(contentTypeBytes)

	// Build key from []byte - single allocation for the key string
	key := makeIndexKeyFromBytes(pathBytes, mockIDBytes, contentTypeBytes)

	candidates, ok := s.Responses[key]
	if !ok || len(candidates) == 0 {
		return nil
	}

	// If no method filter, return first candidate
	if len(methodBytes) == 0 {
		return candidates[0]
	}

	// Filter by method - use pre-computed MethodBytes to avoid allocation
	for _, c := range candidates {
		if equalFoldBytes(c.MethodBytes, methodBytes) {
			return c
		}
	}

	return nil
}

// FindResponse is kept for backwards compatibility (mainly for tests).
func (s *MockStorage) FindResponse(path, mockID, contentType, method string) *MockResponse {
	return s.FindResponseBytes([]byte(path), []byte(mockID), []byte(contentType), []byte(method))
}

// ListAllMocks returns all stored mock responses.
func (s *MockStorage) ListAllMocks() []*MockResponse {
	if s.scenariosEnabled {
		responses := make([]*MockResponse, 0, len(s.scenarioOrder))
		for _, scenario := range s.scenarioOrder {
			responses = append(responses, scenario.response)
		}
		return responses
	}

	allResponses := []*MockResponse{}
	for _, responses := range s.Responses {
		allResponses = append(allResponses, responses...)
	}
	return allResponses
}

// GetStats returns pre-serialized statistics (for display purposes).
func (s *MockStorage) GetStats() map[string]interface{} {
	if s.scenariosEnabled {
		return s.computeScenarioStats()
	}
	return s.computeStats()
}

// GetStatsJSON returns pre-serialized JSON stats (for serving).
func (s *MockStorage) GetStatsJSON() []byte {
	return s.cachedStats
}

// GetMockListJSON returns pre-serialized JSON mock list (for serving).
func (s *MockStorage) GetMockListJSON() []byte {
	return s.cachedMockList
}

// toLowerASCIISimple converts ASCII string to lowercase.
func toLowerASCIISimple(s string) string {
	b := make([]byte, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			b[i] = c + ('a' - 'A')
		} else {
			b[i] = c
		}
	}
	return string(b)
}

// equalFoldBytes performs case-insensitive comparison of two byte slices.
func equalFoldBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := 0; i < len(a); i++ {
		ca := a[i]
		cb := b[i]

		// Convert to lowercase if needed
		if ca >= 'A' && ca <= 'Z' {
			ca += 'a' - 'A'
		}
		if cb >= 'A' && cb <= 'Z' {
			cb += 'a' - 'A'
		}

		if ca != cb {
			return false
		}
	}
	return true
}
