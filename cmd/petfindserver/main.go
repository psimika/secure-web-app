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
		dataSource = flag.String("datasource", "", "the database URL")
		httpAddr   = flag.String("http", ":8080", "HTTP address for the server to listen on")
		tmplPath   = flag.String("tmpl", defaultTmplPath(), "path containing the application's templates")
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

	handlers, err := web.NewServer(*tmplPath, store)
	if err != nil {
		log.Println("NewServer failed:", err)
		return
	}

	log.Fatal(http.ListenAndServe(*httpAddr, handlers))
}

func defaultTmplPath() string {
	p, err := build.Import("github.com/psimika/secure-web-app/web", "", build.FindOnly)
	if err != nil {
		return ""
	}
	return p.Dir
}
