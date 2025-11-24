package main

import (
	"bufio"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/valyala/fasthttp"
)

func main() {
	// Load CA certificate for client verification
	caCert, err := os.ReadFile("../certs/ca-cert.pem")
	if err != nil {
		log.Fatalf("Failed to read CA certificate: %v", err)
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Configure TLS with client certificate requirement
	tlsConfig := &tls.Config{
		ClientAuth: tls.RequireAndVerifyClientCert,
		ClientCAs:  caCertPool,
	}

	handler := func(ctx *fasthttp.RequestCtx) {
		// Get client certificate info
		tlsConn := ctx.Conn().(interface{ ConnectionState() tls.ConnectionState })
		state := tlsConn.ConnectionState()

		var clientDN string
		if len(state.PeerCertificates) > 0 {
			cert := state.PeerCertificates[0]
			clientDN = cert.Subject.String()
		}

		// Set SSE headers
		ctx.Response.Header.Set("Content-Type", "text/event-stream; charset=utf-8")
		ctx.Response.Header.Set("Cache-Control", "no-cache")
		ctx.Response.Header.Set("Connection", "keep-alive")

		// Stream 3 events with client info
		ctx.Response.SetBodyStreamWriter(func(w *bufio.Writer) {
			for i := 0; i < 3; i++ {
				if i > 0 {
					time.Sleep(100 * time.Millisecond)
				}
				fmt.Fprintf(w, "data: {\"event\": \"message_%d\", \"client\": \"%s\"}\n\n", i, clientDN)
				w.Flush()
			}
		})
	}

	server := &fasthttp.Server{
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	log.Println("Starting mTLS SSE server on :5559")
	if err := server.ListenAndServeTLS(":5559", "../certs/server-cert.pem", "../certs/server-key.pem"); err != nil {
		log.Fatalf("Error: %s", err)
	}
}
