package proxy

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/valyala/fasthttp"
)

// ProxyHandler creates a proxy handler that forwards requests and records them.
type ProxyHandler struct {
	recorder      *Recorder
	client        *fasthttp.Client
	targetURL     string // Target URL to proxy to
	headerXMockID []byte
	tlsConfig     *tls.Config // TLS configuration for client certs and SSE
}

// NewProxyHandler creates a new proxy handler.
func NewProxyHandler(recorder *Recorder, targetURL string) *ProxyHandler {
	// Default TLS config
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true, // Skip verification for self-signed certs in testing
	}

	return &ProxyHandler{
		recorder:  recorder,
		targetURL: targetURL,
		client: &fasthttp.Client{
			MaxConnsPerHost:               1000,
			ReadTimeout:                   30 * time.Second,
			WriteTimeout:                  30 * time.Second,
			MaxIdleConnDuration:           90 * time.Second,
			DisableHeaderNamesNormalizing: true,
			DisablePathNormalizing:        true,
			TLSConfig:                     tlsConfig,
		},
		headerXMockID: []byte("x-mock-id"),
		tlsConfig:     tlsConfig,
	}
}

// LoadClientCertificate loads a client certificate and key for mTLS authentication
func (p *ProxyHandler) LoadClientCertificate(certFile, keyFile string) error {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("failed to load client certificate: %w", err)
	}

	// Update TLS config with client certificate
	p.tlsConfig.Certificates = []tls.Certificate{cert}

	// Update fasthttp client's TLS config
	p.client.TLSConfig = p.tlsConfig

	return nil
}

// Handle handles an incoming proxy request.
func (p *ProxyHandler) Handle(ctx *fasthttp.RequestCtx) {
	// Generate request ID
	requestID := p.recorder.generateRequestID()

	// Extract x-mock-id from headers
	mockID := string(ctx.Request.Header.PeekBytes(p.headerXMockID))

	// Log incoming request
	logMockID := mockID
	if logMockID == "" {
		logMockID = "default"
	}
	log.Printf("[%s] %s %s (mock-id: %s)", requestID, string(ctx.Method()), string(ctx.URI().FullURI()), logMockID)

	// Prepare request data for later recording
	reqHeaders := make(map[string]string)
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		reqHeaders[string(key)] = string(value)
	})
	if mockID != "" {
		reqHeaders["x-mock-id"] = mockID
	}

	// Parse request body as JSON if possible
	var reqBody interface{}
	requestBodyBytes := ctx.Request.Body()
	if len(requestBodyBytes) > 0 {
		var jsonBody interface{}
		if err := json.Unmarshal(requestBodyBytes, &jsonBody); err == nil {
			reqBody = jsonBody
		} else {
			reqBody = string(requestBodyBytes)
		}
	} else {
		reqBody = ""
	}

	reqData := &RequestData{
		RequestID: requestID,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
		Method:    string(ctx.Method()),
		URL:       string(ctx.URI().FullURI()),
		Headers:   reqHeaders,
		Body:      reqBody,
		MockID:    mockID,
	}

	// Prepare the proxied request
	req := fasthttp.AcquireRequest()
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseRequest(req)
	defer fasthttp.ReleaseResponse(resp)

	// Build target URL: targetURL + request path + query
	path := string(ctx.Path())
	queryString := ctx.URI().QueryString()
	targetURL := p.targetURL + path
	if len(queryString) > 0 {
		targetURL += "?" + string(queryString)
	}

	// Set up the request
	req.SetRequestURI(targetURL)
	req.Header.SetMethod(string(ctx.Method()))

	// Copy headers (except Host and x-mock-id)
	ctx.Request.Header.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		keyLower := strings.ToLower(keyStr)
		if keyLower != "host" && keyLower != "x-mock-id" {
			req.Header.SetBytesKV(key, value)
		}
	})

	// Copy body
	req.SetBody(ctx.Request.Body())

	// Remove proxy-specific headers
	req.Header.Del("Proxy-Connection")
	req.Header.Del("Proxy-Authenticate")
	req.Header.Del("Proxy-Authorization")

	// Check Accept header to detect SSE request
	acceptHeader := string(ctx.Request.Header.Peek("Accept"))
	expectSSE := strings.Contains(acceptHeader, "text/event-stream")

	if expectSSE {
		// Handle SSE with streaming
		p.handleSSEStreaming(ctx, req, reqData)
		return
	}

	// Forward the request (non-SSE)
	startTime := time.Now()
	err := p.client.Do(req, resp)
	elapsedSeconds := time.Since(startTime).Seconds()

	if err != nil {
		log.Printf("[%s] ❌ Proxy error: %v", requestID, err)
		ctx.SetStatusCode(fasthttp.StatusBadGateway)
		ctx.SetBodyString("Proxy error: " + err.Error())
		return
	}

	// Record the request/response pair
	if err := p.recorder.RecordPair(reqData, resp, elapsedSeconds); err != nil {
		log.Printf("[%s] ⚠️  Failed to record: %v", requestID, err)
	}

	log.Printf("[%s] ✓ %d %s (%.3fs)", requestID, resp.StatusCode(), http.StatusText(resp.StatusCode()), elapsedSeconds)

	// Copy response to client
	ctx.SetStatusCode(resp.StatusCode())

	// Copy headers
	resp.Header.VisitAll(func(key, value []byte) {
		// Skip hop-by-hop headers and x-mock-id
		keyStr := string(key)
		keyLower := strings.ToLower(keyStr)
		switch keyLower {
		case "connection", "keep-alive", "proxy-authenticate",
			"proxy-authorization", "te", "trailers", "transfer-encoding", "upgrade",
			"x-mock-id":
			return
		}
		ctx.Response.Header.SetBytesKV(key, value)
	})

	// Copy body
	ctx.SetBody(resp.Body())
}

// handleSSEStreaming handles SSE requests with true streaming and event recording
func (p *ProxyHandler) handleSSEStreaming(ctx *fasthttp.RequestCtx, req *fasthttp.Request, reqData *RequestData) {
	log.Printf("[%s] 📡 SSE streaming started", reqData.RequestID)
	startTime := time.Now()

	// Determine if target is HTTPS
	isHTTPS := strings.HasPrefix(p.targetURL, "https://")

	// Extract host for connection
	targetHost := strings.TrimPrefix(p.targetURL, "http://")
	targetHost = strings.TrimPrefix(targetHost, "https://")

	// If no port specified, add default port
	if !strings.Contains(targetHost, ":") {
		if isHTTPS {
			targetHost += ":443"
		} else {
			targetHost += ":80"
		}
	}

	log.Printf("[%s] SSE connecting to %s (HTTPS: %v)", reqData.RequestID, targetHost, isHTTPS)

	// Connect to upstream
	var conn net.Conn
	var err error

	if isHTTPS {
		// For HTTPS, use TLS connection with configured TLS config (includes client certs if loaded)
		conn, err = tls.DialWithDialer(
			&net.Dialer{Timeout: 10 * time.Second},
			"tcp",
			targetHost,
			p.tlsConfig,
		)
	} else {
		// For HTTP, use plain TCP
		conn, err = net.DialTimeout("tcp", targetHost, 10*time.Second)
	}

	if err != nil {
		log.Printf("[%s] ❌ SSE connection error: %v", reqData.RequestID, err)
		ctx.SetStatusCode(fasthttp.StatusBadGateway)
		ctx.SetBodyString("Failed to connect to upstream")
		return
	}
	// Don't defer close - will close after streaming completes

	// Send request to upstream
	bw := bufio.NewWriter(conn)
	if err := req.Write(bw); err != nil {
		log.Printf("[%s] ❌ SSE write error: %v", reqData.RequestID, err)
		conn.Close()
		ctx.SetStatusCode(fasthttp.StatusBadGateway)
		ctx.SetBodyString("Failed to write request to upstream")
		return
	}
	if err := bw.Flush(); err != nil {
		log.Printf("[%s] ❌ SSE flush error: %v", reqData.RequestID, err)
		conn.Close()
		ctx.SetStatusCode(fasthttp.StatusBadGateway)
		ctx.SetBodyString("Failed to flush request to upstream")
		return
	}

	// Read response headers only
	br := bufio.NewReader(conn)
	resp := fasthttp.AcquireResponse()
	defer fasthttp.ReleaseResponse(resp)

	if err := resp.Header.Read(br); err != nil {
		log.Printf("[%s] ❌ SSE header read error: %v", reqData.RequestID, err)
		conn.Close()
		ctx.SetStatusCode(fasthttp.StatusBadGateway)
		ctx.SetBodyString("Failed to read response headers from upstream")
		return
	}

	// Copy headers to client
	log.Printf("[%s] SSE response status: %d", reqData.RequestID, resp.StatusCode())
	ctx.SetStatusCode(resp.StatusCode())
	resp.Header.VisitAll(func(key, value []byte) {
		keyStr := string(key)
		keyLower := strings.ToLower(keyStr)
		if keyLower != "connection" && keyLower != "keep-alive" && keyLower != "transfer-encoding" && keyLower != "content-length" && keyLower != "x-mock-id" {
			ctx.Response.Header.SetBytesKV(key, value)
		}
	})

	// Save headers for recording BEFORE SetBodyStreamWriter (which may modify them)
	savedHeaders := make(map[string]string)
	resp.Header.VisitAll(func(key, value []byte) {
		keyLower := strings.ToLower(string(key))
		// Skip x-mock-id from upstream (will be added from request if provided)
		if keyLower != "x-mock-id" {
			savedHeaders[string(key)] = string(value)
		}
	})

	// Check if response is chunked
	isChunked := string(resp.Header.Peek("Transfer-Encoding")) == "chunked"

	// Prepare for streaming
	events := []interface{}{}
	currentEvent := &bytes.Buffer{}

	// Stream body: read line → send to client → accumulate for log
	ctx.Response.SetBodyStreamWriter(func(w *bufio.Writer) {
		lineNum := 0

		if isChunked {
			// Read chunked encoding manually
			for {
				// Read chunk size line
				chunkSizeLine, err := br.ReadString('\n')
				if err != nil {

					break
				}
				chunkSizeLine = strings.TrimSpace(chunkSizeLine)

				// Parse chunk size (hex)
				var chunkSize int
				if _, err := fmt.Sscanf(chunkSizeLine, "%x", &chunkSize); err != nil {

					break
				}

				if chunkSize == 0 {

					break
				}

				// Read chunk data
				chunkData := make([]byte, chunkSize)
				if _, err := io.ReadFull(br, chunkData); err != nil {

					break
				}

				// Read trailing \r\n after chunk data
				br.ReadString('\n')

				// Process chunk line by line
				lines := strings.Split(string(chunkData), "\n")
				for _, line := range lines {
					line = strings.TrimRight(line, "\r")
					if line == "" && len(lines) == 1 {
						continue // Skip empty chunks
					}

					lineNum++
					elapsed := time.Since(startTime).Seconds()

					// Send line to client
					w.WriteString(line + "\n")
					w.Flush()

					// Accumulate for recording
					currentEvent.WriteString(line + "\n")

					// Empty line = end of SSE event
					if line == "" && currentEvent.Len() > 1 {
						eventStr := currentEvent.String()
						eventLines := strings.Split(strings.TrimSpace(eventStr), "\n")

						// Parse data lines
						for _, l := range eventLines {
							if strings.HasPrefix(l, "data: ") {
								dataStr := strings.TrimPrefix(l, "data: ")

								// Try parse as JSON
								var jsonData interface{}
								if err := json.Unmarshal([]byte(dataStr), &jsonData); err == nil {
									events = append(events, map[string]interface{}{
										"data":      jsonData,
										"timestamp": elapsed,
									})
								} else {
									events = append(events, map[string]interface{}{
										"data":      dataStr,
										"timestamp": elapsed,
									})
								}
							}
						}

						currentEvent.Reset()
					}
				}
			}
		} else {
			// Non-chunked - read line by line
			scanner := bufio.NewScanner(br)
			for scanner.Scan() {
				line := scanner.Text()
				lineNum++
				elapsed := time.Since(startTime).Seconds()

				// Send line to client
				w.WriteString(line + "\n")
				w.Flush()

				// Accumulate for recording
				currentEvent.WriteString(line + "\n")

				// Empty line = end of SSE event
				if line == "" && currentEvent.Len() > 1 {
					eventStr := currentEvent.String()
					eventLines := strings.Split(strings.TrimSpace(eventStr), "\n")

					// Parse data lines
					for _, l := range eventLines {
						if strings.HasPrefix(l, "data: ") {
							dataStr := strings.TrimPrefix(l, "data: ")

							// Try parse as JSON
							var jsonData interface{}
							if err := json.Unmarshal([]byte(dataStr), &jsonData); err == nil {
								events = append(events, map[string]interface{}{
									"data":      jsonData,
									"timestamp": elapsed,
								})
							} else {
								events = append(events, map[string]interface{}{
									"data":      dataStr,
									"timestamp": elapsed,
								})
							}
						}
					}

					currentEvent.Reset()
				}
			}

		}

		// Close upstream connection
		conn.Close()

		// Streaming finished - save to log
		elapsedSeconds := time.Since(startTime).Seconds()
		if err := p.recorder.RecordSSEPair(reqData, resp, events, elapsedSeconds, savedHeaders); err != nil {
			log.Printf("[%s] ⚠️  Failed to record SSE: %v", reqData.RequestID, err)
		} else {
			log.Printf("[%s] ✓ SSE completed: %d events recorded (%.3fs)", reqData.RequestID, len(events), elapsedSeconds)
		}
	})
}

// HandleConnect handles CONNECT requests for HTTPS tunneling.
func (p *ProxyHandler) HandleConnect(ctx *fasthttp.RequestCtx) {
	// For now, reject CONNECT requests
	// Full HTTPS proxy support requires more complex tunneling
	ctx.SetStatusCode(fasthttp.StatusMethodNotAllowed)
	ctx.SetBodyString("CONNECT method not supported. Use HTTP proxy mode only.")
}
