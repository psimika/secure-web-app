package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"github.com/psimika/secure-web-app/petfind"
)

// Error can be returned by the handlers of application's HTTP server.
type Error struct {
	Err     error
	Message string
	Code    int
}

// E constructs an *Error and can be used as a shorthand when a handler returns
// an *Error.
func E(err error, message string, code int) *Error {
	return &Error{Err: err, Message: message, Code: code}
}

// handler is a custom HTTP handler that can return an *Error. It is used
// instead of the standard http.Handler in order to simplify repetitive error
// handling as proposed by Gerrand (2011a):
// https://blog.golang.org/error-handling-and-go
type handler func(http.ResponseWriter, *http.Request) *Error

func (fn handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if e := fn(w, r); e != nil { // e is *web.Error, not error.
		log.Println(e)
		http.Error(w, e.Message, e.Code)
	}
}

// server is the application's HTTP server.
type server struct {
	handlers       http.Handler
	mux            *http.ServeMux
	store          petfind.Store
	tmpl           *tmpl
	secureRedirect bool
}

// tmpl contains the server's templates required to render its pages.
type tmpl struct {
	home        *template.Template
	addPet      *template.Template
	searchReply *template.Template
	showPets    *template.Template
}

// NewServer initializes and returns a new HTTP server.
func NewServer(templatePath string, store petfind.Store, secureRedirect bool) (http.Handler, error) {
	t, err := parseTemplates(filepath.Join(templatePath, "templates"))
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %v", err)
	}
	s := &server{mux: http.NewServeMux(), store: store, tmpl: t, secureRedirect: secureRedirect}
	s.handlers = s.mux
	s.mux.Handle("/", handler(s.homeHandler))
	s.mux.Handle("/form", handler(s.searchReplyHandler))
	s.mux.Handle("/pets/add", handler(s.serveAddPet))
	s.mux.Handle("/pets/add/submit", handler(s.handleAddPet))
	s.mux.Handle("/pets", handler(s.servePets))
	return s, nil
}

func parseTemplates(dir string) (*tmpl, error) {
	homeTmpl, err := template.ParseFiles(filepath.Join(dir, "base.tmpl"), filepath.Join(dir, "search.tmpl"))
	if err != nil {
		return nil, err
	}
	addPetTmpl, err := template.ParseFiles(filepath.Join(dir, "base.tmpl"), filepath.Join(dir, "addpet.tmpl"))
	if err != nil {
		return nil, err
	}
	searchReplyTmpl, err := template.ParseFiles(filepath.Join(dir, "base.tmpl"), filepath.Join(dir, "searchreply.tmpl"))
	if err != nil {
		return nil, err
	}
	showPetsTmpl, err := template.ParseFiles(filepath.Join(dir, "base.tmpl"), filepath.Join(dir, "showpets.tmpl"))
	t := &tmpl{
		home:        homeTmpl,
		addPet:      addPetTmpl,
		searchReply: searchReplyTmpl,
		showPets:    showPetsTmpl,
	}
	return t, err
}

// ServeHTTP satisfies the http.Handler interface for a server.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		// HSTS header suggested by OWASP (2017) to address certain threats:
		// https://www.owasp.org/index.php/HTTP_Strict_Transport_Security_Cheat_Sheet
		w.Header().Set("Strict-Transport-Security", "max-age=86400; includeSubDomains")
	}
	//if r.Header.Get("X-Forwarded-Proto") == "http" && s.secureRedirect || r.TLS != nil && s.secureRedirect {
	//	w.Header().Set("Connection", "close")
	//	url := "https://" + r.Host + r.URL.String()
	//	http.Redirect(w, r, url, http.StatusMovedPermanently)
	//	return
	//}
	s.handlers.ServeHTTP(w, r)
}

func (s *server) homeHandler(w http.ResponseWriter, r *http.Request) *Error {
	err := s.tmpl.home.Execute(w, nil)
	if err != nil {
		return E(err, "could not serve home", http.StatusInternalServerError)
	}
	return nil
}

func (s *server) serveAddPet(w http.ResponseWriter, r *http.Request) *Error {
	err := s.tmpl.addPet.Execute(w, nil)
	if err != nil {
		return E(err, "could not serve addPet", http.StatusInternalServerError)
	}
	return nil
}

func (s *server) handleAddPet(w http.ResponseWriter, r *http.Request) *Error {
	name := r.FormValue("name")
	p := &petfind.Pet{Name: name}
	if err := s.store.AddPet(p); err != nil {
		return E(err, "Error adding pet", http.StatusInternalServerError)
	}

	w.Write([]byte("pet added!"))
	return nil
}

func (s *server) searchReplyHandler(w http.ResponseWriter, r *http.Request) *Error {
	name := r.FormValue("name")

	pets, err := s.store.GetAllPets()
	if err != nil {
		return E(err, "internal server error", http.StatusInternalServerError)
	}
	var p *petfind.Pet
	for i := range pets {
		if pets[i].Name == name {
			p = &pets[i]
		}
	}

	if err := s.tmpl.searchReply.Execute(w, p); err != nil {
		return E(err, "internal server error", http.StatusInternalServerError)
	}
	return nil
}

func (s *server) servePets(w http.ResponseWriter, r *http.Request) *Error {
	pets, err := s.store.GetAllPets()
	if err != nil {
		return E(err, "Error getting all pets", http.StatusInternalServerError)
	}
	err = s.tmpl.showPets.Execute(w, pets)
	if err != nil {
		return E(err, "internal server error", http.StatusInternalServerError)
	}
	return nil
}
