package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"

	"golang.org/x/oauth2"
	"golang.org/x/oauth2/github"

	gorillactx "github.com/gorilla/context"
	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
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
	handlers      http.Handler
	mux           *http.ServeMux
	store         petfind.Store
	tmpl          *tmpl
	github        *oauth2.Config
	sessions      sessions.Store
	sessionTTL    int
	sessionMaxTTL int
	favicons      map[string]string
}

// tmpl contains the server's templates required to render its pages.
type tmpl struct {
	home        *template.Template
	addPet      *template.Template
	search      *template.Template
	searchReply *template.Template
	showPets    *template.Template
	login       *template.Template
}

// NewServer initializes and returns a new HTTP server.
//
// sessionTTL is used to extend the session's idle timeout.
//
// sessionMaxTTL is used to check if a session has expired by surpassing its
// absolute timeout.
func NewServer(
	store petfind.Store,
	sessionStore sessions.Store,
	sessionTTL int,
	sessionMaxTTL int,
	CSRF func(http.Handler) http.Handler,
	templatePath string,
	githubID string,
	githubSecret string,
) (http.Handler, error) {
	t, err := parseTemplates(filepath.Join(templatePath, "templates"))
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %v", err)
	}

	s := &server{
		mux:           http.NewServeMux(),
		store:         store,
		tmpl:          t,
		github:        newGitHubOAuthConfig(githubID, githubSecret),
		sessions:      sessionStore,
		sessionTTL:    sessionTTL,
		sessionMaxTTL: sessionMaxTTL,
	}
	s.handlers = gorillactx.ClearHandler(CSRF(s.mux))
	s.mux.Handle("/", s.guest(s.serveHome))
	s.mux.Handle("/form", handler(s.searchReplyHandler))
	s.mux.Handle("/pets", handler(s.servePets))
	s.mux.Handle("/pets/add", s.auth(s.serveAddPet))
	s.mux.Handle("/pets/add/submit", s.auth(s.handleAddPet))
	s.mux.Handle("/login", handler(s.serveLogin))
	s.mux.Handle("/login/github", handler(s.handleLoginGitHub))
	s.mux.Handle("/login/github/cb", handler(s.handleLoginGitHubCallback))
	s.mux.Handle("/logout", s.auth(s.handleLogout))

	fs := http.FileServer(http.Dir(filepath.Join(templatePath, "assets")))
	s.mux.Handle("/assets/", http.StripPrefix("/assets", cacheAssets(fs)))
	s.favicons = prepareFavicons(templatePath)
	return s, nil
}

func cacheAssets(h http.Handler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Cache-Control", "max-age=31536000")
		h.ServeHTTP(w, r)
	}
}

func prepareFavicons(assetsPath string) map[string]string {
	// Different favicon versions for diffent devices as suggested by Bernard (2015).
	var f = [...]string{
		"android-chrome-192x192.png",
		"android-chrome-512x512.png",
		"apple-touch-icon.png",
		"browserconfig.xml",
		"favicon.ico",
		"favicon-16x16.png",
		"favicon-32x32.png",
		"manifest.json",
		"mstile-150x150.png",
		"safari-pinned-tab.svg",
	}
	favicons := make(map[string]string)
	for i := 0; i < len(f); i++ {
		favicons["/"+f[i]] = filepath.Join(assetsPath, "assets", "favicon", f[i])
	}
	return favicons
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
	homeTmpl, err := template.ParseFiles(filepath.Join(dir, "base.tmpl"), filepath.Join(dir, "navbar.tmpl"), filepath.Join(dir, "home.tmpl"))
	if err != nil {
		return nil, err
	}
	addPetTmpl, err := template.ParseFiles(filepath.Join(dir, "base.tmpl"), filepath.Join(dir, "addpet.tmpl"))
	if err != nil {
		return nil, err
	}
	searchTmpl, err := template.ParseFiles(filepath.Join(dir, "base.tmpl"), filepath.Join(dir, "search.tmpl"))
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
	loginTmpl, err := template.ParseFiles(filepath.Join(dir, "base.tmpl"), filepath.Join(dir, "navbar.tmpl"), filepath.Join(dir, "login.tmpl"))
	t := &tmpl{
		home:        homeTmpl,
		addPet:      addPetTmpl,
		search:      searchTmpl,
		searchReply: searchReplyTmpl,
		showPets:    showPetsTmpl,
		login:       loginTmpl,
	}
	return t, err
}

// ServeHTTP satisfies the http.Handler interface for a server.
func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.TLS != nil || r.Header.Get("X-Forwarded-Proto") == "https" {
		// HSTS header suggested by OWASP (2017a) to address certain threats:
		// https://www.owasp.org/index.php/HTTP_Strict_Transport_Security_Cheat_Sheet
		w.Header().Set("Strict-Transport-Security", "max-age=86400; includeSubDomains")
	}
	s.handlers.ServeHTTP(w, r)
}

func (s *server) serveHome(w http.ResponseWriter, r *http.Request) *Error {
	// Serve favicons.
	if fname, ok := s.favicons[r.URL.Path]; ok {
		w.Header().Add("Cache-Control", "max-age=31536000")
		http.ServeFile(w, r, fname)
		return nil
	}

	return s.render(w, r, s.tmpl.home, nil)
}

func (s *server) render(w http.ResponseWriter, r *http.Request, tmpl *template.Template, data interface{}) *Error {
	m := map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
	}

	if data != nil {
		m["data"] = data
	}

	user, _ := fromContextGetUser(r.Context())
	if user != nil {
		m["user"] = user
	}

	if err := tmpl.Execute(w, m); err != nil {
		return E(err, fmt.Sprintf("could not serve %s", tmpl.Name()), http.StatusInternalServerError)
	}
	return nil
}

func (s *server) guest(fn handler) handler {
	return func(w http.ResponseWriter, r *http.Request) *Error {
		session, err := s.sessions.Get(r, sessionName)
		if err != nil {
			return E(err, "error getting guest session", http.StatusInternalServerError)
		}

		var user *petfind.User
		// Get the user's ID stored in the session.
		userID, err := fromSessionGetUserID(session)
		if err == nil {
			// Get the user from the database based on the session's user ID.
			user, err = s.store.GetUser(userID)
			if err != nil {
				return E(err, "error getting user from guest session", http.StatusInternalServerError)
			}
		}

		// Extend session's idle timeout.
		session.Options.MaxAge = s.sessionTTL
		if err = sessions.Save(r, w); err != nil {
			return E(err, "error extending guest session", http.StatusInternalServerError)
		}

		if user != nil {
			// Put user in the context so that the next handler can access it.
			ctx := newContextWithUser(r.Context(), user)
			fn(w, r.WithContext(ctx))
		} else {
			fn(w, r)
		}
		return nil
	}
}

func (s *server) serveAddPet(w http.ResponseWriter, r *http.Request) *Error {
	user, ok := fromContextGetUser(r.Context())
	if !ok || user == nil {
		return E(nil, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}

	err := s.tmpl.addPet.Execute(w,
		map[string]interface{}{
			"user":           user,
			csrf.TemplateTag: csrf.TemplateField(r),
		},
	)
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
