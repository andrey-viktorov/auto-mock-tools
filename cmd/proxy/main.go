package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/andrey-viktorov/auto-mock-tools/pkg/proxy"
	"github.com/valyala/fasthttp"
)

func main() {
	// Define CLI flags
	logDir := flag.String("log-dir", "mocks", "Directory to store recorded mock files")
	host := flag.String("host", "127.0.0.1", "Host to bind the proxy to")
	port := flag.Int("port", 8080, "Port to bind the proxy to")
	targetURL := flag.String("target", "", "Target URL to proxy requests to (e.g., http://localhost:3000)")
	clientCert := flag.String("client-cert", "", "Path to client certificate file for mTLS (optional)")
	clientKey := flag.String("client-key", "", "Path to client key file for mTLS (optional)")
	flag.Parse()

	if *targetURL == "" {
		log.Fatal("Error: -target flag is required. Specify the target URL to proxy to.")
	}

	// Create recorder
	fmt.Println("🚀 Starting HTTP recording proxy...")
	fmt.Printf("📁 Recording to directory: %s\n", *logDir)

	recorder, err := proxy.NewRecorder(*logDir)
	if err != nil {
		log.Fatalf("Failed to create recorder: %v", err)
	}
	defer recorder.Close()

	// Create proxy handler
	proxyHandler := proxy.NewProxyHandler(recorder, *targetURL)

	// Load client certificate if provided
	if *clientCert != "" && *clientKey != "" {
		if err := proxyHandler.LoadClientCertificate(*clientCert, *clientKey); err != nil {
			log.Fatalf("Failed to load client certificate: %v", err)
		}
		fmt.Printf("🔐 Client certificate loaded: %s\n", *clientCert)
	}

	// Create request handler
	handler := func(ctx *fasthttp.RequestCtx) {
		method := string(ctx.Method())

		// Handle CONNECT for HTTPS (currently not supported)
		if method == "CONNECT" {
			proxyHandler.HandleConnect(ctx)
			return
		}

		// Handle regular HTTP proxy requests
		proxyHandler.Handle(ctx)
	}

	addr := fmt.Sprintf("%s:%d", *host, *port)
	fmt.Printf("\n🌐 Reverse proxy running at http://%s\n", addr)
	fmt.Printf("🎯 Proxying to: %s\n", *targetURL)
	fmt.Println("📝 All requests will be recorded with x-mock-id header support")
	fmt.Println("\nUsage examples:")
	fmt.Printf("  curl http://%s/get\n", addr)
	fmt.Printf("  curl -H \"x-mock-id: test-1\" http://%s/get\n", addr)
	fmt.Println("\nPress Ctrl+C to stop")

	// Create server
	server := &fasthttp.Server{
		Handler: handler,
		Name:    "AutoRecordingProxy",
	}

	// Handle graceful shutdown
	go func() {
		sigint := make(chan os.Signal, 1)
		signal.Notify(sigint, os.Interrupt, syscall.SIGTERM)
		<-sigint

		fmt.Println("\n👋 Shutting down proxy...")
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
