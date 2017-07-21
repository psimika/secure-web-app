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

func main() {
	// Heroku uses the environment variables DATABASE_URL and PORT so that the
	// app knows on which database to connect and on which port to listen on.
	dataSource := os.Getenv("DATABASE_URL")
	httpAddr := ":" + os.Getenv("PORT")
	// Heroku deploys the application under /app.
	tmplPath := "/app/web"

	if dataSource == "" {
		log.Fatal("No database datasource provided, exiting...")
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
