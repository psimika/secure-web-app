// +build !heroku

package main

import (
	"flag"
	"go/build"
	"log"
	"net/http"
	"strings"
	"time"

	_ "github.com/lib/pq"

	"github.com/psimika/secure-web-app/https"
	"github.com/psimika/secure-web-app/petfind/postgres"
	"github.com/psimika/secure-web-app/web"
)

const defaultSessionMinutes = 30

func main() {
	var (
		dataSource     = flag.String("datasource", "", "the database URL")
		httpAddr       = flag.String("http", ":8080", "HTTP address for the server to listen on")
		httpsAddr      = flag.String("https", ":8443", "HTTPS address for the server to listen on")
		tmplPath       = flag.String("tmpl", defaultTmplPath(), "path containing the application's templates")
		insecureHTTP   = flag.Bool("insecure", false, "whether to serve insecure HTTP instead of HTTPS")
		certFile       = flag.String("tlscert", "", "TLS public key in PEM format used together with -tlskey")
		keyFile        = flag.String("tlskey", "", "TLS private key in PEM format used together with -tlscert")
		autocertHosts  = flag.String("autocert", "", "one or more host names separated by space to get Let's Encrypt certificates automatically")
		autocertCache  = flag.String("autocertdir", "", "directory to cache the Let's Encrypt certificates")
		githubID       = flag.String("githubid", "", "GitHub Client ID used for Login with GitHub")
		githubSecret   = flag.String("githubsecret", "", "GitHub Client Secret used for Login with GitHub")
		sessionMinutes = flag.Int("sessionmins", defaultSessionMinutes, "`minutes` that the server's sessions will remain valid")
	)
	flag.Parse()
	if !*insecureHTTP && *autocertHosts == "" && (*certFile == "" || *keyFile == "") {
		log.Println("Not enough flags set to start server, exiting...")
		log.Println("This application serves HTTPS by default.")
		log.Println("Use -autocert=example.com to specify a domain for an automatic Let's encrypt certificate.")
		log.Println("Or use -tlscert=<public key file> -tlskey=<private key file> to provide your own certificate.")
		log.Fatal("Or use the -insecure flag to serve insecure HTTP instead of HTTPS.")
	}

	if *dataSource == "" {
		log.Fatal("No database datasource provided, exiting...")
	}

	store, err := postgres.NewStore(*dataSource)
	if err != nil {
		log.Println("NewStore failed:", err)
		return
	}

	appHandlers, err := web.NewServer(store, *tmplPath, *githubID, *githubSecret, time.Duration(*sessionMinutes)*time.Minute)
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
	// OWASP says: "Web applications should avoid the extremely common HTTP to
	// HTTPS redirection on the home page (using a 30x HTTP response), as this
	// single unprotected HTTP request/response exchange can be used by an
	// attacker to gather (or fix) a valid session ID."
	//
	// https://www.owasp.org/index.php/Session_Management_Cheat_Sheet#Transport_Layer_Security
	w.Header().Set("Connection", "close")
	u := r.URL
	u.Host = r.Host
	u.Scheme = "https"
	http.Redirect(w, r, u.String(), http.StatusMovedPermanently)
}
