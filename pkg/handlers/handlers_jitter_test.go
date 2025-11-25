package handlers

import (
	"bufio"
	"bytes"
	"math/rand"
	"testing"
	"time"

	"github.com/andrey-viktorov/auto-mock-tools/pkg/storage"
)

func TestSSEJitterOriginalDelay(t *testing.T) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	if err := store.LoadScenarioConfig("../../tests/fixtures/test-jitter-original.yml"); err != nil {
		t.Fatalf("Failed to load scenario config: %v", err)
	}

	// Get the response to check timestamps
	resp := store.MatchScenarioResponse([]byte("/sse-stream"), []byte("GET"), []byte(""))
	if resp == nil {
		t.Fatal("Expected to find SSE response")
	}

	if !resp.IsSSE {
		t.Fatal("Expected SSE response")
	}

	// Verify original delay
	if resp.Delay != 1.0 {
		t.Fatalf("Expected delay to be 1.0, got %f", resp.Delay)
	}

	jitter := 0.05 // 5%

	// Run 10 times to verify jitter behavior
	for i := 0; i < 10; i++ {
		// Simulate jitter calculation as in handlers.go
		jitterAmount := (rand.Float64()*2 - 1) * jitter
		jitterScale := 1.0 + jitterAmount
		if jitterScale < 0 {
			jitterScale = 0
		}

		writer := &sseStreamWriter{
			events:      resp.SSEEvents,
			jitterScale: jitterScale,
		}

		var buf bytes.Buffer
		bufWriter := bufio.NewWriter(&buf)

		start := time.Now()
		writer.StreamTo(bufWriter)
		elapsed := time.Since(start)

		// Original delay is 1.0s, with 5% jitter max should be 1.05s
		maxExpected := 1.05 * float64(time.Second)
		if elapsed > time.Duration(maxExpected) {
			t.Errorf("Run %d: Elapsed time %v exceeds max expected %v (1.0s + 5%% jitter, scale: %.3f)",
				i+1, elapsed, time.Duration(maxExpected), jitterScale)
		}

		// Should be at least 0.95s (1.0 - 5% jitter)
		minExpected := 0.94 * float64(time.Second) // Allow small tolerance
		if elapsed < time.Duration(minExpected) {
			t.Errorf("Run %d: Elapsed time %v is less than min expected %v (scale: %.3f)",
				i+1, elapsed, time.Duration(minExpected), jitterScale)
		}

		t.Logf("Run %d: Elapsed time: %v (jitter scale: %.3f)", i+1, elapsed, jitterScale)
	}
}

func TestSSEJitterWithDelayOverride(t *testing.T) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	if err := store.LoadScenarioConfig("../../tests/fixtures/test-jitter-override.yml"); err != nil {
		t.Fatalf("Failed to load scenario config: %v", err)
	}

	// Get the response to check timestamps
	resp := store.MatchScenarioResponse([]byte("/sse-stream"), []byte("GET"), []byte(""))
	if resp == nil {
		t.Fatal("Expected to find SSE response")
	}

	if !resp.IsSSE {
		t.Fatal("Expected SSE response")
	}

	// Verify overridden delay
	if resp.Delay != 0.5 {
		t.Fatalf("Expected delay to be 0.5, got %f", resp.Delay)
	}

	jitter := 0.05 // 5%

	// Run 10 times to verify jitter behavior with overridden delay
	for i := 0; i < 10; i++ {
		// Simulate jitter calculation as in handlers.go
		jitterAmount := (rand.Float64()*2 - 1) * jitter
		jitterScale := 1.0 + jitterAmount
		if jitterScale < 0 {
			jitterScale = 0
		}

		writer := &sseStreamWriter{
			events:      resp.SSEEvents,
			jitterScale: jitterScale,
		}

		var buf bytes.Buffer
		bufWriter := bufio.NewWriter(&buf)

		start := time.Now()
		writer.StreamTo(bufWriter)
		elapsed := time.Since(start)

		// Delay is overridden to 0.5s, with 5% jitter max should be 0.525s
		maxExpected := 0.525 * float64(time.Second)
		if elapsed > time.Duration(maxExpected) {
			t.Errorf("Run %d: Elapsed time %v exceeds max expected %v (0.5s + 5%% jitter, scale: %.3f)",
				i+1, elapsed, time.Duration(maxExpected), jitterScale)
		}

		// Should be at least 0.475s (0.5 - 5% jitter)
		minExpected := 0.47 * float64(time.Second) // Allow small tolerance
		if elapsed < time.Duration(minExpected) {
			t.Errorf("Run %d: Elapsed time %v is less than min expected %v (scale: %.3f)",
				i+1, elapsed, time.Duration(minExpected), jitterScale)
		}

		t.Logf("Run %d: Elapsed time: %v (jitter scale: %.3f)", i+1, elapsed, jitterScale)
	}
}

func TestSSEJitterScaling(t *testing.T) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	if err := store.LoadScenarioConfig("../../tests/fixtures/test-jitter-override.yml"); err != nil {
		t.Fatalf("Failed to load scenario config: %v", err)
	}

	// Verify that event timestamps were scaled proportionally
	resp := store.MatchScenarioResponse([]byte("/sse-stream"), []byte("GET"), []byte(""))
	if resp == nil {
		t.Fatal("Expected to find SSE response")
	}

	if !resp.IsSSE {
		t.Fatal("Expected SSE response")
	}

	if resp.Delay != 0.5 {
		t.Fatalf("Expected delay to be 0.5, got %f", resp.Delay)
	}

	// Original timestamps: 0.2, 0.4, 0.6, 0.8, 1.0 (total 1.0s)
	// With delay override to 0.5s, should be scaled by 0.5/1.0 = 0.5
	// Expected: 0.1, 0.2, 0.3, 0.4, 0.5
	expectedTimestamps := []float64{0.1, 0.2, 0.3, 0.4, 0.5}

	if len(resp.SSEEvents) != len(expectedTimestamps) {
		t.Fatalf("Expected %d events, got %d", len(expectedTimestamps), len(resp.SSEEvents))
	}

	for i, expected := range expectedTimestamps {
		actual := resp.SSEEvents[i].Timestamp
		tolerance := 0.001
		if actual < expected-tolerance || actual > expected+tolerance {
			t.Errorf("Event %d: expected timestamp %f, got %f (difference: %f)",
				i+1, expected, actual, actual-expected)
		}
	}

	t.Logf("Event timestamps correctly scaled: %v",
		[]float64{
			resp.SSEEvents[0].Timestamp,
			resp.SSEEvents[1].Timestamp,
			resp.SSEEvents[2].Timestamp,
			resp.SSEEvents[3].Timestamp,
			resp.SSEEvents[4].Timestamp,
		})
}

func TestSSEStreamWriter(t *testing.T) {
	// Create test events
	events := []storage.SSEEvent{
		{SerializedData: []byte(`{"event":1}`), Timestamp: 0.1},
		{SerializedData: []byte(`{"event":2}`), Timestamp: 0.2},
		{SerializedData: []byte(`{"event":3}`), Timestamp: 0.3},
	}

	tests := []struct {
		name           string
		jitterScale    float64
		expectedMaxDur time.Duration
		expectedMinDur time.Duration
	}{
		{
			name:           "No jitter",
			jitterScale:    1.0,
			expectedMaxDur: 350 * time.Millisecond,
			expectedMinDur: 250 * time.Millisecond,
		},
		{
			name:           "With 10% jitter up",
			jitterScale:    1.1,
			expectedMaxDur: 380 * time.Millisecond,
			expectedMinDur: 280 * time.Millisecond,
		},
		{
			name:           "With 10% jitter down",
			jitterScale:    0.9,
			expectedMaxDur: 320 * time.Millisecond,
			expectedMinDur: 220 * time.Millisecond,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			writer := &sseStreamWriter{
				events:      events,
				jitterScale: tt.jitterScale,
			}

			var buf bytes.Buffer
			bufWriter := bufio.NewWriter(&buf)

			start := time.Now()
			writer.StreamTo(bufWriter)
			elapsed := time.Since(start)

			if elapsed > tt.expectedMaxDur {
				t.Errorf("Elapsed time %v exceeds max expected %v", elapsed, tt.expectedMaxDur)
			}

			if elapsed < tt.expectedMinDur {
				t.Errorf("Elapsed time %v is less than min expected %v", elapsed, tt.expectedMinDur)
			}

			t.Logf("Elapsed time: %v (scale: %.2f)", elapsed, tt.jitterScale)
		})
	}
}
