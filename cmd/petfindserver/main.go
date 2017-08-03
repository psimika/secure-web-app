// +build !heroku

package main

import (
	"flag"
	"go/build"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/boj/redistore"
	"github.com/gorilla/sessions"
	_ "github.com/lib/pq"

	"github.com/psimika/secure-web-app/https"
	"github.com/psimika/secure-web-app/petfind/postgres"
	"github.com/psimika/secure-web-app/web"
)

func main() {
	var (
		dataSource    = flag.String("datasource", "", "the database URL")
		httpAddr      = flag.String("http", ":8080", "HTTP address for the server to listen on")
		httpsAddr     = flag.String("https", ":8443", "HTTPS address for the server to listen on")
		tmplPath      = flag.String("tmpl", defaultTmplPath(), "path containing the application's templates")
		insecureHTTP  = flag.Bool("insecure", false, "whether to serve insecure HTTP instead of HTTPS")
		certFile      = flag.String("tlscert", "", "TLS public key in PEM format used together with -tlskey")
		keyFile       = flag.String("tlskey", "", "TLS private key in PEM format used together with -tlscert")
		autocertHosts = flag.String("autocert", "", "one or more host names separated by space to get Let's Encrypt certificates automatically")
		autocertCache = flag.String("autocertdir", "", "directory to cache the Let's Encrypt certificates")
		githubID      = flag.String("githubid", "", "GitHub Client ID used for Login with GitHub")
		githubSecret  = flag.String("githubsecret", "", "GitHub Client Secret used for Login with GitHub")
		hashKeyStr    = flag.String("hashkey", "", "random key (32 or 64 bytes) used to sign/authenticate values using HMAC")
		blockKeyStr   = flag.String("blockkey", "", "random key (32 bytes) used to encrypt values using AES-256")
		redisAddr     = flag.String("redis", ":6379", "Redis address to connect to and store sessions")
		redisPass     = flag.String("redispass", "", "Redis password if needed")
		redisMaxIdle  = flag.Int("redismaxidle", 10, "maximum number of idle Redis connections")
		sessionTTL    = flag.Int("sessionttl", 1200, "`seconds` before a session expires due to inactivity (idle timeout)")
		sessionMaxTTL = flag.Int("sessionmaxttl", 3600, "`seconds` before a session expires regardless of activity (absolute timeout)")
	)
	flag.Parse()
	if !*insecureHTTP && *autocertHosts == "" && (*certFile == "" || *keyFile == "") {
		log.Println("Not enough flags set to start server, exiting...")
		log.Println("This application serves HTTPS by default.")
		log.Println("Use -autocert=example.com to specify a domain for an automatic Let's encrypt certificate.")
		log.Println("Or use -tlscert=<public key file> -tlskey=<private key file> to provide your own certificate.")
		log.Fatal("Or use the -insecure flag to serve insecure HTTP instead of HTTPS.")
	}
	hashKey := validHashKey(*hashKeyStr)
	blockKey := validBlockKey(*blockKeyStr)

	if *dataSource == "" {
		log.Fatal("No database datasource provided, exiting...")
	}

	store, err := postgres.NewStore(*dataSource)
	if err != nil {
		log.Println("NewStore failed:", err)
		return
	}

	sessionStore, err := redistore.NewRediStore(*redisMaxIdle, "tcp", *redisAddr, *redisPass, hashKey, blockKey)
	if err != nil {
		log.Println("NewRediStore failed:", err)
		return
	}
	defer func() {
		if err = sessionStore.Close(); err != nil {
			log.Println("Error closing session store:", err)
		}
	}()
	sessionStore.Options = &sessions.Options{
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		MaxAge:   *sessionTTL,
	}
	if *insecureHTTP {
		sessionStore.Options.Secure = false
	}

	appHandlers, err := web.NewServer(
		store,
		sessionStore,
		*sessionTTL,
		*sessionMaxTTL,
		hashKey,
		blockKey,
		*tmplPath,
		*githubID,
		*githubSecret,
	)
	if err != nil {
		log.Println("NewServer failed:", err)
		return
	}

	if *insecureHTTP {
		log.Printf("Serving insecure HTTP on %q", *httpAddr)
		log.Fatal(http.ListenAndServe(*httpAddr, appHandlers))
	}

	go func() {
		log.Printf("Serving HTTP->HTTPS redirect on %q", *httpAddr)
		redirectServer := newRedirectServer(*httpAddr, http.HandlerFunc(redirectHTTP))
		log.Fatal(redirectServer.ListenAndServe())
	}()

	if *autocertHosts != "" {
		hosts := strings.Split(*autocertHosts, " ")
		log.Printf("Serving HTTPS on %q using Let's Encrypt certificates for %v", *httpsAddr, hosts)
		if *autocertCache != "" {
			log.Printf("Caching Let's Encrypt certificates in %v", *autocertCache)
		}
		log.Fatal(https.ListenAndServeAutocert(*httpsAddr, *autocertCache, hosts, appHandlers))
	}

	log.Printf("Serving HTTPS on %q using provided certificates", *httpsAddr)
	log.Fatal(https.ListenAndServeTLS(*httpsAddr, *certFile, *keyFile, appHandlers))
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
	// TODO(psimika): Research how safe is redirect.
	//
	// According to OWASP: "Web applications should avoid the extremely common
	// HTTP to HTTPS redirection on the home page (using a 30x HTTP response),
	// as this single unprotected HTTP request/response exchange can be used by
	// an attacker to gather (or fix) a valid session ID."
	//
	// https://www.owasp.org/index.php/Session_Management_Cheat_Sheet#Transport_Layer_Security
	w.Header().Set("Connection", "close")
	u := r.URL
	u.Host = r.Host
	u.Scheme = "https"
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}
