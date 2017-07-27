package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

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
		log.SetFlags(log.LstdFlags | log.Lshortfile)
		log.Println(e)
		http.Error(w, e.Message, e.Code)
	}
}

// server is the application's HTTP server.
type server struct {
	handlers        http.Handler
	mux             *http.ServeMux
	store           petfind.Store
	tmpl            *tmpl
	github          *oauth2.Config
	stateWhitelist  *stateWhitelist
	sessionDuration time.Duration
}

// tmpl contains the server's templates required to render its pages.
type tmpl struct {
	home        *template.Template
	addPet      *template.Template
	searchReply *template.Template
	showPets    *template.Template
	login       *template.Template
}

// NewServer initializes and returns a new HTTP server.
func NewServer(store petfind.Store, templatePath, githubID, githubSecret string, sessionDuration time.Duration) (http.Handler, error) {
	t, err := parseTemplates(filepath.Join(templatePath, "templates"))
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %v", err)
	}
	if sessionDuration == 0 {
		sessionDuration = time.Duration(30 * time.Minute)
	}
	s := &server{
		mux:             http.NewServeMux(),
		store:           store,
		tmpl:            t,
		github:          newGitHubOAuthConfig(githubID, githubSecret),
		stateWhitelist:  newStateWhitelist(),
		sessionDuration: sessionDuration,
	}
	s.handlers = s.mux
	s.mux.Handle("/", handler(s.homeHandler))
	s.mux.Handle("/form", handler(s.searchReplyHandler))
	s.mux.Handle("/pets", handler(s.servePets))
	s.mux.Handle("/pets/add", s.auth(s.serveAddPet))
	s.mux.Handle("/pets/add/submit", s.auth(s.handleAddPet))
	s.mux.Handle("/login", handler(s.serveLogin))
	s.mux.Handle("/login/github", handler(s.handleLoginGitHub))
	s.mux.Handle("/login/github/cb", handler(s.handleLoginGitHubCallback))
	s.mux.Handle("/logout", s.auth(s.handleLogout))
	return s, nil
}

func newGitHubOAuthConfig(clientID, clientSecret string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		// Only requiring the user's public info and their email.
		//
		// Full list of scopes:
		// https://developer.github.com/apps/building-integrations/setting-up-and-registering-oauth-apps/about-scopes-for-oauth-apps/
		Scopes:   []string{"user:email"},
		Endpoint: github.Endpoint,
	}
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
	if err != nil {
		return nil, err
	}
	loginTmpl, err := template.ParseFiles(filepath.Join(dir, "base.tmpl"), filepath.Join(dir, "login.tmpl"))
	t := &tmpl{
		home:        homeTmpl,
		addPet:      addPetTmpl,
		searchReply: searchReplyTmpl,
		showPets:    showPetsTmpl,
		login:       loginTmpl,
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
	s.handlers.ServeHTTP(w, r)
}

type stateWhitelist struct {
	mu sync.Mutex
	m  map[string]bool
}

func newStateWhitelist() *stateWhitelist {
	return &stateWhitelist{m: make(map[string]bool)}
}

func (w *stateWhitelist) Put(key string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.m[key] = true
}

func (w *stateWhitelist) Get(key string) bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.m[key]
}

func (w *stateWhitelist) Delete(key string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.m, key)
}

func (s *server) homeHandler(w http.ResponseWriter, r *http.Request) *Error {

	user, ok := fromContextGetUser(r.Context())
	log.Println("context user:", user, ok)
	sessionID, ok := fromContextGetSessionID(r.Context())
	log.Println("context sessionID:", sessionID, ok)

	err := s.tmpl.home.Execute(w, nil)
	if err != nil {
		return E(err, "could not serve home", http.StatusInternalServerError)
	}
	return nil
}

func (s *server) serveAddPet(w http.ResponseWriter, r *http.Request) *Error {
	user, ok := fromContextGetUser(r.Context())
	if !ok || user == nil {
		return E(nil, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}

	err := s.tmpl.addPet.Execute(w, user)
	if err != nil {
		return E(err, "could not serve addPet", http.StatusInternalServerError)
	}
	return nil
}

func (s *server) handleAddPet(w http.ResponseWriter, r *http.Request) *Error {
	user, ok := fromContextGetUser(r.Context())
	if !ok || user == nil {
		return E(nil, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}

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
