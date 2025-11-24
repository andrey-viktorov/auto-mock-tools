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

	if err := store.LoadScenarioConfig("../../mock-example.yml"); err != nil {
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
