// +build heroku

package main

import (
	"log"
	"net/http"
	"os"

	_ "github.com/lib/pq"

	"github.com/psimika/secure-web-app/petfind/postgres"
	"github.com/psimika/secure-web-app/web"
)

// This is an implementation of make.go that is specific to Heroku as indicated
// by the build tag. The build tag means that on Heroku the main_heroku.go will
// be built and on any other case main.go will be built instead.
//
// In order to deploy to Heroku for the first time we need these steps:
//
//   heroku login
//
//   heroku create
//
//   heroku addons:create heroku-postgresql:hobby-dev
//
// After that and each time we make a change on master branch:
//
//   git push heroku master
//
// Or when working on a different branch:
//
//   git push heroku somebranch:master

func main() {
	// Heroku uses the environment variables DATABASE_URL and PORT so that the
	// app knows on which database to connect and on which port to listen on.
	dataSource := os.Getenv("DATABASE_URL")
	httpAddr := ":" + os.Getenv("PORT")
	// Heroku deploys the application under /app.
	tmplPath := "/app/web"

	if dataSource == "" {
		log.Fatal("No database URL provided, exiting...")
	}

	store, err := postgres.NewStore(dataSource)
	if err != nil {
		log.Println("NewStore failed:", err)
		return
	}

	handlers, err := web.NewServer(tmplPath, store)
	if err != nil {
		log.Println("NewServer failed:", err)
		return
	}

	log.Fatal(http.ListenAndServe(httpAddr, handlers))
}
