// Package https provides helpers for starting an HTTPS server and serving an
// application's handlers.
package https

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"os"
	"time"

	"golang.org/x/crypto/acme/autocert"
)

// ListenAndServeAutocert automatically gets Let's Encrypt certificates for the
// provided hosts and launches an HTTPS server to serve the application's
// handlers.
func ListenAndServeAutocert(httpsAddr, cachePath string, hosts []string, appHandlers http.Handler) error {
	var m autocert.Manager
	m.Prompt = autocert.AcceptTOS
	if cachePath != "" {
		if err := os.MkdirAll(cachePath, 0700); err != nil {
			return fmt.Errorf("could not create or read Let's Encrypt cache directory: %v", err)
		}
		m.Cache = autocert.DirCache(cachePath)
	}
	if len(hosts) > 0 {
		m.HostPolicy = autocert.HostWhitelist(hosts...)
	}
	tlsConfig := &tls.Config{GetCertificate: m.GetCertificate}

	server := newServer(tlsConfig, appHandlers)

	ln, err := net.Listen("tcp", httpsAddr)
	if err != nil {
		return fmt.Errorf("https: %v", err)
	}
	ln = tls.NewListener(ln, tlsConfig)
	err = server.Serve(ln)
	return fmt.Errorf("https: %v", err)
}

// ListenAndServeTLS launches an HTTPS server to serve the application's
// handlers using the provided TLS certificaces.
func ListenAndServeTLS(addr, certFile, keyFile string, handler http.Handler) error {
	tlsConfig, err := newDefaultTLSConfig(certFile, keyFile)
	if err != nil {
		return fmt.Errorf("https: setting up TLS config: %v", err)
	}

	server := newServer(tlsConfig, handler)

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("https: %v", err)
	}
	ln = tls.NewListener(ln, tlsConfig)
	err = server.Serve(ln)
	return fmt.Errorf("https: %v", err)
}

func newServer(config *tls.Config, handler http.Handler) *http.Server {
	return &http.Server{
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:       60 * time.Second,
		TLSConfig:         config,
		Handler:           handler,
	}
}

func newDefaultTLSConfig(certFile, keyFile string) (*tls.Config, error) {
	cert, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("loading certificate key pair failed: %v", err)
	}
	// TLS configuration meant to be used by a Go server that is going to be
	// exposed on the internet directly (Valsorda 2016):
	// https://blog.cloudflare.com/exposing-go-on-the-internet/
	tlsConfig := &tls.Config{
		// Causes servers to use Go's default ciphersuite preferences,
		// which are tuned to avoid attacks. Does nothing on clients.
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.CurveP521, tls.CurveP384, tls.CurveP256, tls.X25519},
		MinVersion:               tls.VersionTLS12,
		Certificates:             []tls.Certificate{cert},
		CipherSuites: []uint16{
			tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_ECDSA_WITH_CHACHA20_POLY1305,
			tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
			tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
			tls.TLS_ECDHE_RSA_WITH_CHACHA20_POLY1305,

			// Vulnerable to the Lucky13 attack.
			// tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
			// tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		},
	}
	tlsConfig.BuildNameToCertificate()
	return tlsConfig, nil
}
