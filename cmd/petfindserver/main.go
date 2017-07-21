// +build !heroku

package main

import (
	"flag"
	"go/build"
	"log"
	"net/http"

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

	if !insecureHTTP {
		log.Fatal(http.ListenAndServe(*httpAddr, handlers))
	} else {
		go func() {
			log.Printf("Serving HTTP->HTTPS redirect on %q", *httpAddr)
			log.Fatal(http.ListenAndServe(*httpAddr, http.HandlerFunc(redirectHTTP)))
		}()
		// TODO: Serve TLS
		// log.Fatal(http.ListenAndServeTLS(*httpsAddr, handlers))
		log.Fatal(http.ListenAndServe(*httpsAddr, handlers))
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
