package handlers

import (
	"testing"

	"github.com/andrey-viktorov/auto-mock-tools/pkg/storage"
	"github.com/valyala/fasthttp"
)

func BenchmarkMockHandler(b *testing.B) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	handler := MockHandler(store)

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/users/1")
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.Header.Set("Accept", "application/json")
	ctx.Request.Header.Set("x-mock-id", "default")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler(ctx)
		ctx.Response.Reset()
	}
}

func BenchmarkSSEHandlerNoTiming(b *testing.B) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	// Test without timing replay (production-like for instant mode)
	store.SetTimingConfig(false, 0.0)

	handler := MockHandler(store)

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/events")
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.Header.Set("Accept", "text/event-stream")
	ctx.Request.Header.Set("x-mock-id", "default")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler(ctx)
		ctx.Response.Reset()
	}
}

func BenchmarkSSEHandlerWithTiming(b *testing.B) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	// Enable timing replay to see allocation cost
	store.SetTimingConfig(true, 0.0)

	handler := MockHandler(store)

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/events")
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.Header.Set("Accept", "text/event-stream")
	ctx.Request.Header.Set("x-mock-id", "default")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler(ctx)
		ctx.Response.Reset()
	}
}

func BenchmarkRouter(b *testing.B) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	handler := Router(store)

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/users/1")
	ctx.Request.Header.SetMethod("GET")
	ctx.Request.Header.Set("Accept", "application/json")
	ctx.Request.Header.Set("x-mock-id", "default")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler(ctx)
		ctx.Response.Reset()
	}
}

func BenchmarkStatsHandler(b *testing.B) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		b.Fatalf("Failed to create storage: %v", err)
	}

	handler := StatsHandler(store)

	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/__mock__/stats")
	ctx.Request.Header.SetMethod("GET")

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		handler(ctx)
		ctx.Response.Reset()
	}
}

func TestMockHandlerScenarioMode(t *testing.T) {
	store, err := storage.NewMockStorage("../../test_mocks")
	if err != nil {
		t.Fatalf("Failed to create storage: %v", err)
	}

	if err := store.LoadScenarioConfig("../../mock-example.yml"); err != nil {
		t.Fatalf("Failed to load scenarios: %v", err)
	}

	handler := MockHandler(store)
	ctx := &fasthttp.RequestCtx{}
	ctx.Request.SetRequestURI("/api/v1/status")
	ctx.Request.Header.SetMethod("POST")
	ctx.Request.SetBody([]byte(`{"processing":{"state":"done"},"payload":{"id":"ABC-1234"}}`))

	handler(ctx)
	if ctx.Response.StatusCode() != fasthttp.StatusOK {
		t.Fatalf("Expected 200, got %d", ctx.Response.StatusCode())
	}
	if string(ctx.Response.Body()) != `{"data":8,"version":1}` {
		t.Fatalf("Unexpected scenario body: %s", ctx.Response.Body())
	}

	ctx.Response.Reset()
	ctx.Request.SetBody([]byte(`{"processing":{"state":"pending"}}`))
	handler(ctx)
	if string(ctx.Response.Body()) != `{"id":17,"name":"User 17"}` {
		t.Fatalf("Expected fallback scenario body, got %s", ctx.Response.Body())
	}
}
