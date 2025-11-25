package handlers

import (
	"testing"
	"time"

	"github.com/andrey-viktorov/auto-mock-tools/pkg/storage"
	"github.com/valyala/fasthttp"
)

func TestNonSSEDelayWithoutReplayTiming(t *testing.T) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Disable timing replay
	store.SetTimingConfig(false, 0.0)

	handler := MockHandler(store, nil)

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/users/17")
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.Header.Set("Accept", "application/json")
	ctx.Request.Header.Set("x-mock-id", "default")

	start := time.Now()
	handler(ctx)
	elapsed := time.Since(start)

	// Without replay timing, response should be instant (< 10ms)
	maxExpected := 10 * time.Millisecond
	if elapsed > maxExpected {
		t.Errorf("Expected instant response (< 10ms), got %v", elapsed)
	}

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("Expected 200, got %d", ctx.Response.StatusCode())
	}

	t.Logf("Response time without replay timing: %v", elapsed)
}

func TestNonSSEDelayWithReplayTiming(t *testing.T) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Enable timing replay without jitter
	store.SetTimingConfig(true, 0.0)

	handler := MockHandler(store, nil)

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/users/17")
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.Header.Set("Accept", "application/json")
	ctx.Request.Header.Set("x-mock-id", "default")

	start := time.Now()
	handler(ctx)
	elapsed := time.Since(start)

	// The mock file has elapsed_seconds: 0.1 (100ms)
	// Allow some tolerance for scheduling overhead
	minExpected := 90 * time.Millisecond
	maxExpected := 120 * time.Millisecond

	if elapsed < minExpected {
		t.Errorf("Expected delay >= %v, got %v", minExpected, elapsed)
	}

	if elapsed > maxExpected {
		t.Errorf("Expected delay <= %v, got %v", maxExpected, elapsed)
	}

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("Expected 200, got %d", ctx.Response.StatusCode())
	}

	t.Logf("Response time with replay timing: %v (expected ~100ms)", elapsed)
}

func TestNonSSEDelayWithJitter(t *testing.T) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Enable timing replay with 10% jitter
	store.SetTimingConfig(true, 0.1)

	handler := MockHandler(store, nil)

	// Run multiple times to test jitter variance
	var minElapsed, maxElapsed time.Duration
	runs := 10

	for i := 0; i < runs; i++ {
		ctx := &fasthttp.RequestCtx{}
		ctx.Request.SetRequestURI("/users/17")
		ctx.Request.Header.SetMethod("GET")
		ctx.Request.Header.Set("Accept", "application/json")
		ctx.Request.Header.Set("x-mock-id", "default")

		start := time.Now()
		handler(ctx)
		elapsed := time.Since(start)

		if i == 0 || elapsed < minElapsed {
			minElapsed = elapsed
		}
		if i == 0 || elapsed > maxElapsed {
			maxElapsed = elapsed
		}

		if ctx.Response.StatusCode() != fasthttp.StatusOK {
			t.Fatalf("Run %d: Expected 200, got %d", i+1, ctx.Response.StatusCode())
		}

		t.Logf("Run %d: Response time: %v", i+1, elapsed)
	}

	// Base delay is 100ms (0.1s), with 10% jitter: 90ms - 110ms
	// Add tolerance for CI overhead
	minExpectedOverall := 80 * time.Millisecond
	maxExpectedOverall := 130 * time.Millisecond

	if minElapsed < minExpectedOverall {
		t.Errorf("Minimum elapsed time %v is less than expected %v", minElapsed, minExpectedOverall)
	}

	if maxElapsed > maxExpectedOverall {
		t.Errorf("Maximum elapsed time %v exceeds expected %v", maxElapsed, maxExpectedOverall)
	}

	t.Logf("Elapsed time range: %v - %v (expected ~90ms - ~110ms with jitter)", minElapsed, maxElapsed)
}

func TestNonSSEDelayZeroValue(t *testing.T) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Enable timing replay
	store.SetTimingConfig(true, 0.0)

	handler := MockHandler(store, nil)

	// Use a mock that doesn't have elapsed_seconds field (or has 0)
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api/endpoint")
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.Header.Set("Accept", "application/json")
	ctx.Request.Header.Set("x-mock-id", "default")

	start := time.Now()
	handler(ctx)
	elapsed := time.Since(start)

	// With zero delay, response should be instant even with replay timing enabled
	maxExpected := 10 * time.Millisecond
	if elapsed > maxExpected {
		t.Errorf("Expected instant response for zero delay (< 10ms), got %v", elapsed)
	}

	t.Logf("Response time with zero delay: %v", elapsed)
}

func TestNonSSEDelayScenarioOverride(t *testing.T) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	// Load scenario config with delay override
	if err := store.LoadScenarioConfig("../../tests/fixtures/test-delay-override.yml"); err != nil {
		t.Fatalf("Failed to load scenario config: %v", err)
	}

	// Enable timing replay
	store.SetTimingConfig(true, 0.0)

	handler := MockHandler(store, nil)

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api/test-delay")
	ctx.Request.Header.SetMethod("GET")

	start := time.Now()
	handler(ctx)
	elapsed := time.Since(start)

	// Scenario config should override delay to 0.2s (200ms)
	minExpected := 190 * time.Millisecond
	maxExpected := 220 * time.Millisecond

	if elapsed < minExpected {
		t.Errorf("Expected delay >= %v, got %v", minExpected, elapsed)
	}

	if elapsed > maxExpected {
		t.Errorf("Expected delay <= %v, got %v", maxExpected, elapsed)
	}

	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("Expected 200, got %d", ctx.Response.StatusCode())
	}

	t.Logf("Response time with scenario delay override: %v (expected ~200ms)", elapsed)
}
