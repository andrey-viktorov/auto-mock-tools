package storage

import (
	"testing"
)

func BenchmarkFindResponse(b *testing.B) {
	store, err := NewMockStorage("../../test_mocks")
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp := store.FindResponse("/users/1", "default", "application/json", "GET")
		if resp == nil {
			b.Fatal("Expected response, got nil")
		}
	}
}

func BenchmarkFindResponseBytes(b *testing.B) {
	store, err := NewMockStorage("../../test_mocks")
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	pathBytes := []byte("/users/1")
	mockIDBytes := []byte("default")
	contentTypeBytes := []byte("application/json")
	methodBytes := []byte("GET")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp := store.FindResponseBytes(pathBytes, mockIDBytes, contentTypeBytes, methodBytes)
		if resp == nil {
			b.Fatal("Expected response, got nil")
		}
	}
}

func BenchmarkStorageLoad(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := NewMockStorage("../../test_mocks")
		if err != nil {
			b.Fatalf("Failed to load storage: %v", err)
		}
	}
}

func BenchmarkSSEFindResponse(b *testing.B) {
	store, err := NewMockStorage("../../test_mocks")
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp := store.FindResponse("/events", "default", "text/event-stream", "GET")
		if resp == nil {
			b.Fatal("Expected SSE response, got nil")
		}
		if !resp.IsSSE {
			b.Fatal("Expected IsSSE=true")
		}
	}
}

func TestNewMockStorage(t *testing.T) {
	store, err := NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	if store == nil {
		t.Fatal("Expected storage, got nil")
	}

	if len(store.Responses) == 0 {
		t.Fatal("Expected some responses loaded")
	}
}

func TestFindResponse(t *testing.T) {
	store, err := NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Test finding a response
	resp := store.FindResponse("/users/1", "default", "application/json", "GET")
	if resp == nil {
		t.Fatal("Expected to find response")
	}

	// Test not finding a response
	resp = store.FindResponse("/nonexistent", "default", "application/json", "GET")
	if resp != nil {
		t.Fatal("Expected nil for nonexistent path")
	}
}

func TestGetStats(t *testing.T) {
	store, err := NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	stats := store.GetStats()
	if stats == nil {
		t.Fatal("Expected stats, got nil")
	}

	if _, ok := stats["total_responses"]; !ok {
		t.Fatal("Expected total_responses in stats")
	}
}

func TestSetTimingConfig(t *testing.T) {
	store, err := NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	store.SetTimingConfig(true, 0.1)

	if !store.ReplayTiming {
		t.Fatal("Expected ReplayTiming to be true")
	}

	if store.Jitter != 0.1 {
		t.Fatalf("Expected Jitter to be 0.1, got %f", store.Jitter)
	}
}

func TestScenarioConfigMatching(t *testing.T) {
	store, err := NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	if err := store.LoadScenarioConfig("../../tests/fixtures/mock-example.yml"); err != nil {
		t.Fatalf("Failed to load scenarios: %v", err)
	}

	if !store.HasScenarios() {
		t.Fatal("Expected scenarios to be enabled")
	}

	matchBody := []byte(`{"processing":{"state":"done"},"payload":{"id":"ABC-1234"}}`)
	resp := store.MatchScenarioResponse([]byte("/api/v1/status"), []byte("POST"), matchBody)
	if resp == nil {
		t.Fatal("Expected scenario match for valid payload")
	}
	if resp.MockID != "Status Ready With Valid ID" {
		t.Fatalf("Expected first scenario, got %s", resp.MockID)
	}

	defaultBody := []byte(`{"processing":{"state":"pending"}}`)
	fallback := store.MatchScenarioResponse([]byte("/api/v1/status"), []byte("POST"), defaultBody)
	if fallback == nil {
		t.Fatal("Expected fallback scenario match")
	}
	if fallback.MockID != "Status Fallback Default" {
		t.Fatalf("Expected fallback scenario, got %s", fallback.MockID)
	}
}

func TestSSEDelayOverride(t *testing.T) {
	store, err := NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	if err := store.LoadScenarioConfig("../../tests/fixtures/test-sse-delay-override.yml"); err != nil {
		t.Fatalf("Failed to load scenario config: %v", err)
	}

	resp := store.MatchScenarioResponse([]byte("/stream"), []byte("GET"), []byte(""))
	if resp == nil {
		t.Fatal("Expected SSE scenario match")
	}

	if !resp.IsSSE {
		t.Fatal("Expected SSE response")
	}

	// Check that delay was overridden
	if resp.Delay != 1.0 {
		t.Fatalf("Expected delay to be 1.0, got %f", resp.Delay)
	}

	// Check that event timestamps were scaled proportionally
	// Original: 0.1, 0.2, 0.3, 0.4, 0.5 with total delay 5.0
	// After override to 1.0: should be scaled by 1.0/5.0 = 0.2
	expectedTimestamps := []float64{0.02, 0.04, 0.06, 0.08, 0.10}

	if len(resp.SSEEvents) != len(expectedTimestamps) {
		t.Fatalf("Expected %d events, got %d", len(expectedTimestamps), len(resp.SSEEvents))
	}

	for i, expected := range expectedTimestamps {
		actual := resp.SSEEvents[i].Timestamp
		// Allow small floating point tolerance
		if actual < expected-0.001 || actual > expected+0.001 {
			t.Fatalf("Event %d: expected timestamp %f, got %f", i+1, expected, actual)
		}
	}
}

func TestScenarioWithoutFilter(t *testing.T) {
	store, err := NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	if err := store.LoadScenarioConfig("../../tests/fixtures/test-scenario-no-filter.yml"); err != nil {
		t.Fatalf("Failed to load scenarios: %v", err)
	}

	// Test 1: Body matches filter - should return first scenario
	matchingBody := []byte(`{"status":"active"}`)
	resp := store.MatchScenarioResponse([]byte("/api/test"), []byte("GET"), matchingBody)
	if resp == nil {
		t.Fatal("Expected scenario match for matching filter")
	}
	if resp.MockID != "Filtered Scenario" {
		t.Fatalf("Expected 'Filtered Scenario', got %s", resp.MockID)
	}

	// Test 2: Body doesn't match filter - should fall back to no-filter scenario
	nonMatchingBody := []byte(`{"status":"inactive"}`)
	resp = store.MatchScenarioResponse([]byte("/api/test"), []byte("GET"), nonMatchingBody)
	if resp == nil {
		t.Fatal("Expected fallback to no-filter scenario")
	}
	if resp.MockID != "No Filter Scenario" {
		t.Fatalf("Expected 'No Filter Scenario', got %s", resp.MockID)
	}

	// Test 3: Empty body should also match no-filter scenario
	emptyBody := []byte(`{}`)
	resp = store.MatchScenarioResponse([]byte("/api/test"), []byte("GET"), emptyBody)
	if resp == nil {
		t.Fatal("Expected no-filter scenario to match empty body")
	}
	if resp.MockID != "No Filter Scenario" {
		t.Fatalf("Expected 'No Filter Scenario' for empty body, got %s", resp.MockID)
	}

	// Test 4: Different path with no filter
	anyBody := []byte(`{"any":"data"}`)
	resp = store.MatchScenarioResponse([]byte("/api/other"), []byte("POST"), anyBody)
	if resp == nil {
		t.Fatal("Expected scenario match for /api/other")
	}
	if resp.MockID != "Another No Filter" {
		t.Fatalf("Expected 'Another No Filter', got %s", resp.MockID)
	}

	// Test 5: Wrong method should not match
	resp = store.MatchScenarioResponse([]byte("/api/other"), []byte("GET"), anyBody)
	if resp != nil {
		t.Fatal("Expected nil for wrong method")
	}

	// Test 6: Non-existent path should return nil
	resp = store.MatchScenarioResponse([]byte("/api/nonexistent"), []byte("GET"), anyBody)
	if resp != nil {
		t.Fatal("Expected nil for non-existent path")
	}
}

func TestFindResponseBytesAnyContentType(t *testing.T) {
	store, err := NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Test finding response with any content-type
	resp := store.FindResponseBytesAnyContentType([]byte("/users/1"), []byte("default"), []byte("GET"))
	if resp == nil {
		t.Fatal("Expected to find response with any content-type")
	}

	// Should return a response regardless of content-type
	if resp.Path != "/users/1" {
		t.Fatalf("Expected path /users/1, got %s", resp.Path)
	}
	if resp.MockID != "default" {
		t.Fatalf("Expected mock_id default, got %s", resp.MockID)
	}

	// Test not finding a response
	resp = store.FindResponseBytesAnyContentType([]byte("/nonexistent"), []byte("default"), []byte("GET"))
	if resp != nil {
		t.Fatal("Expected nil for nonexistent path")
	}

	// Test with different mock_id
	resp = store.FindResponseBytesAnyContentType([]byte("/data/2"), []byte("api-v1"), []byte("GET"))
	if resp == nil {
		t.Fatal("Expected to find response for api-v1 mock_id")
	}
	if resp.MockID != "api-v1" {
		t.Fatalf("Expected mock_id api-v1, got %s", resp.MockID)
	}
}

func BenchmarkFindResponseBytesAnyContentType(b *testing.B) {
	store, err := NewMockStorage("../../test_mocks")
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	pathBytes := []byte("/users/1")
	mockIDBytes := []byte("default")
	methodBytes := []byte("GET")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		resp := store.FindResponseBytesAnyContentType(pathBytes, mockIDBytes, methodBytes)
		if resp == nil {
			b.Fatal("Expected response, got nil")
		}
	}
}
