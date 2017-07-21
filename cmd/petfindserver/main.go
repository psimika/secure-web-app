// +build !heroku

package main

import (
	"crypto/tls"
	"flag"
	"go/build"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"

	"github.com/psimika/secure-web-app/petfind/postgres"
	"github.com/psimika/secure-web-app/web"
)

func main() {
	var (
		dataSource   = flag.String("datasource", "", "the database URL")
		httpAddr     = flag.String("http", ":8080", "HTTP address for the server to listen on")
		httpsAddr    = flag.String("https", ":8443", "HTTPS address for the server to listen on")
		tmplPath     = flag.String("tmpl", defaultTmplPath(), "path containing the application's templates")
		insecureHTTP = flag.Bool("insecure", false, "whether to serve insecure HTTP instead of HTTPS")
	)
	flag.Parse()

	if *dataSource == "" {
		log.Fatal("No database datasource provided, exiting...")
	}

	store, err := postgres.NewStore(*dataSource)
	if err != nil {
		log.Println("NewStore failed:", err)
		return
	}

	handlers, err := web.NewServer(*tmplPath, store, false)
	if err != nil {
		log.Println("NewServer failed:", err)
		return
	}

	if *insecureHTTP {
		log.Fatal(http.ListenAndServe(*httpAddr, handlers))
	} else {
		go func() {
			log.Printf("Serving HTTP->HTTPS redirect on %q", *httpAddr)
			redirectServer := newRedirectServer(*httpAddr, http.HandlerFunc(redirectHTTP))
			log.Fatal(redirectServer.ListenAndServe())
		}()
		// TODO: Serve TLS
		// log.Fatal(http.ListenAndServeTLS(*httpsAddr, handlers))
		log.Fatal(http.ListenAndServe(*httpsAddr, handlers))
	}

}

func newRedirectServer(httpAddr string, redirectHandler http.Handler) *http.Server {
	return &http.Server{
		Addr:         httpAddr,
		ReadTimeout:  5 * time.Second,
		WriteTimeout: 5 * time.Second,
		Handler:      redirectHandler,
	}
}

func defaultTmplPath() string {
	p, err := build.Import("github.com/psimika/secure-web-app/web", "", build.FindOnly)
	if err != nil {
		return ""
	}
	return p.Dir
}

func redirectHTTP(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Connection", "close")
	u := r.URL
	u.Host = r.Host
	u.Scheme = "https"
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}

func newDefaultTLSConfig() *tls.Config {
	// TLS configuration meant to be used by a Go server that is going to be
	// exposed on the internet directly (Valsorda 2016):
	// https://blog.cloudflare.com/exposing-go-on-the-internet/
	tlsConfig := &tls.Config{
		// Causes servers to use Go's default ciphersuite preferences,
		// which are tuned to avoid attacks. Does nothing on clients.
		PreferServerCipherSuites: true,
		CurvePreferences:         []tls.CurveID{tls.CurveP256, tls.X25519},
		MinVersion:               tls.VersionTLS12,
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
		// TODO: TLS certs
		//Certificates: []tls.Certificate{cert},
	}
	return tlsConfig
}
