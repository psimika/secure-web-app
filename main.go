package main

import (
	"database/sql"
	"flag"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"time"

	_ "github.com/lib/pq"
)

var (
	homeTmpl        = template.Must(template.New("homeTmpl").Parse(baseTemplate + searchTemplate))
	searchReplyTmpl = template.Must(template.New("searchReplyTmpl").Parse(baseTemplate + searchReplyTemplate))
	addPetTmpl      = template.Must(template.New("addPetTmpl").Parse(baseTemplate + addPetTemplate))
	showPetsTmpl    = template.Must(template.New("showPetsTmpl").Parse(baseTemplate + showPetsTemplate))
)

// Pet is the pet of the app.
type Pet struct {
	Name    string
	Age     int
	Created time.Time
}

var pets = []Pet{
	{Name: "Blackie", Age: 5},
	{Name: "Rocky", Age: 6},
	{Name: "Lasie", Age: 7},
}

type App struct {
	db *sql.DB
}

func main() {
	var datasource = flag.String("datasource", "", "the database URL")
	flag.Parse()
	if *datasource == "" {
		log.Fatal("no database datasource provided")
	}

	db, err := sql.Open("postgres", *datasource)
	if err != nil {
		log.Fatalf("Error opening database: %q", err)
	}
	defer db.Close()
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS pets (id serial PRIMARY KEY, name varchar(50), created timestamp)"); err != nil {
		log.Printf("Error creating table pets %q", err)
		return
	}

	app := &App{db: db}

	http.Handle("/", http.HandlerFunc(homeHandler))
	http.Handle("/form", http.HandlerFunc(searchReplyHandler))
	http.Handle("/pets/add", http.HandlerFunc(app.serveAddPet))
	http.Handle("/pets/add/submit", http.HandlerFunc(app.handleAddPet))
	http.Handle("/pets", http.HandlerFunc(app.servePets))

	http.ListenAndServe(":8080", nil)
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	err := homeTmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "internal server error", 500)
		return
	}
}

func searchReplyHandler(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")

	for _, p := range pets {
		if p.Name == name {
			err := searchReplyTmpl.Execute(w, p)
			if err != nil {
				http.Error(w, "internal server error", 500)
				return
			}
			return
		}
	}

	err := searchReplyTmpl.Execute(w, "No pet found")
	if err != nil {
		http.Error(w, "internal server error", 500)
		return
	}

}

func (app *App) serveAddPet(w http.ResponseWriter, r *http.Request) {
	err := addPetTmpl.Execute(w, nil)
	if err != nil {
		http.Error(w, "internal server error", 500)
		return
	}
}

func (app *App) handleAddPet(w http.ResponseWriter, r *http.Request) {
	name := r.FormValue("name")
	if _, err := app.db.Exec("INSERT INTO pets(name, created) VALUES ($1, now())", name); err != nil {
		http.Error(w, fmt.Sprintf("Error adding product: %q", err), http.StatusInternalServerError)
		return
	}
	w.Write([]byte("pet added!"))
}

func (app *App) servePets(w http.ResponseWriter, r *http.Request) {
	rows, err := app.db.Query("SELECT name, created FROM pets")
	if err != nil {
		http.Error(w, fmt.Sprintf("Error selecting pets: %q", err), http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	pets := make([]*Pet, 0)
	for rows.Next() {
		var p Pet
		if err := rows.Scan(&p.Name, &p.Created); err != nil {
			http.Error(w, fmt.Sprintf("Error scanning pets: %q", err), http.StatusInternalServerError)
			return
		}
		pets = append(pets, &p)
	}

	err = showPetsTmpl.Execute(w, pets)
	if err != nil {
		http.Error(w, "internal server error", 500)
		return
	}
}

const (
	baseTemplate = `
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Secure web app</title>
</head>

<body>
  {{block "content" .}}{{end}}
</body>

</html>`

	searchTemplate = `
{{define "content"}}
  <form action="/form" method="GET">
	<input name="name">
	<input type="submit" value="Search for pet">
  </form>
{{end}}
`
	searchReplyTemplate = `
{{define "content"}}
  <span>Name: {{.Name}}</span>
  <br>
  <span>Age: {{.Age}}</span>
{{end}}
`

	addPetTemplate = `
{{define "content"}}
  <form action="/pets/add/submit" method="POST">
	<input name="name">
	<input type="submit" value="Add new pet">
  </form>
{{end}}
`
	showPetsTemplate = `
{{define "content"}}
  {{range .}}
    <li>
      <span>Name: {{.Name}}</span>
      <br>
	  <span>Created: {{.Created}}</span>
	</li>
  {{end}}
{{end}}
`
)
