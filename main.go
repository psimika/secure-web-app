package main

import (
	"html/template"
	"net/http"
)

var (
	homeTmpl        = template.Must(template.New("homeTmpl").Parse(baseTemplate + formTemplate))
	searchReplyTmpl = template.Must(template.New("searchReplyTmpl").Parse(baseTemplate + searchReplyTemplate))
)

// Pet is the pet of the app.
type Pet struct {
	Name string
	Age  int
}

var pets = []Pet{
	{Name: "Blackie", Age: 5},
	{Name: "Rocky", Age: 6},
	{Name: "Lasie", Age: 7},
}

func main() {
	http.Handle("/", http.HandlerFunc(homeHandler))
	http.Handle("/form", http.HandlerFunc(searchReplyHandler))

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

const (
	baseTemplate = `
<!DOCTYPE html>
<html>
<head>
<meta charset="UTF-8">
<title>Secure web app</title>
</head>

<body>
	{{.}}
</body>

</html>`

	formTemplate = `
<form action="/form" method="GET">
	<input name="name">
	<input type="submit" value="Search for pet">
 </form>
`
	searchReplyTemplate = `
<span>Name: {{.Name}}</span>
<br>
<span>Age: {{.Age}}</span>
`
)
