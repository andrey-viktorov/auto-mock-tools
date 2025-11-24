package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrey-viktorov/auto-mock-tools/pkg/handlers"
	"github.com/andrey-viktorov/auto-mock-tools/pkg/storage"
	"github.com/valyala/fasthttp"
)

func main() {
	// Define CLI flags
	mockDir := flag.String("mock-dir", "mocks", "Directory containing recorded mock files")
	scenarioConfig := flag.String("mock-config", "", "YAML file describing scenario filters and responses")
	host := flag.String("host", "127.0.0.1", "Host to bind the server to")
	port := flag.Int("port", 8000, "Port to bind the server to")
	replayTiming := flag.Bool("replay-timing", false, "Replay original request/response timing (latency)")
	jitter := flag.Float64("jitter", 0.0, "Add random jitter to timing (0.0-1.0, 0.1 = ±10%)")
	flag.Parse()

	// Create storage
	fmt.Println("🚀 Starting mock server...")
	fmt.Printf("📁 Loading mocks from directory: %s\n", *mockDir)

	store, err := storage.NewMockStorage(*mockDir)
	if err != nil {
		log.Fatalf("Failed to load mocks: %v", err)
	}

	if *scenarioConfig != "" {
		fmt.Printf("🧩 Loading scenarios from: %s\n", *scenarioConfig)
		if err := store.LoadScenarioConfig(*scenarioConfig); err != nil {
			log.Fatalf("Failed to load scenarios: %v", err)
		}
	} else {
		fmt.Println("🎯 Scenario mode: disabled (using x-mock-id header)")
	}

	// Configure timing
	store.SetTimingConfig(*replayTiming, *jitter)
	if *replayTiming {
		fmt.Printf("⏱️  Timing replay: enabled (jitter: %.1f%%)\n", *jitter*100)
	} else {
		fmt.Println("⚡ Timing replay: disabled (instant responses)")
	}

	// Get stats
	stats := store.GetStats()
	fmt.Printf("📊 Loaded %d responses\n", stats["total_responses"])
	fmt.Printf("🔗 %d unique paths\n", stats["unique_paths"])
	if uniqueMockIDs, ok := stats["unique_mock_ids"].(int); ok && uniqueMockIDs > 0 {
		fmt.Printf("🏷️  %d unique mock IDs\n", uniqueMockIDs)
	}

	addr := fmt.Sprintf("%s:%d", *host, *port)
	fmt.Printf("\n🌐 Server running at http://%s\n", addr)
	fmt.Printf("📈 Stats endpoint: http://%s/__mock__/stats\n", addr)
	fmt.Printf("📋 List endpoint: http://%s/__mock__/list\n", addr)
	fmt.Println("\nPress Ctrl+C to stop")

	// Create router
	handler := handlers.Router(store)

	// Create server
	server := &fasthttp.Server{
		Handler: handler,
		Name:    "AutoMockServer",
	}

	// Handle graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		fmt.Println("\n👋 Shutting down mock server...")
		if err := server.Shutdown(); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
		os.Exit(0)
	}()

	// Start server
	if err := server.ListenAndServe(addr); err != nil {
		log.Fatalf("Error in ListenAndServe: %v", err)
	}
}
