package main

import (
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"os"

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
		} else {
			clientDN = "No client certificate"
		}

		ctx.SetContentType("application/json")
		fmt.Fprintf(ctx, `{"message": "mTLS success", "client": "%s", "path": "%s"}`, clientDN, ctx.Path())
	}

	server := &fasthttp.Server{
		Handler:   handler,
		TLSConfig: tlsConfig,
	}

	log.Println("Starting mTLS test server on :5558")
	log.Println("Requires client certificate signed by ca-cert.pem")

	if err := server.ListenAndServeTLS(":5558", "server-cert.pem", "server-key.pem"); err != nil {
		log.Fatalf("Error in server: %s", err)
	}
}
