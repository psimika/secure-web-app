package web

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

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

func (e Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%d %s: %v", e.Code, e.Message, e.Err)
	}
	return fmt.Sprintf("%d %s", e.Code, e.Message)
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
	templates     *templates
	github        *oauth2.Config
	sessions      sessions.Store
	sessionTTL    int
	sessionMaxTTL int
	favicons      map[string]string
	photos        petfind.PhotoStore
	placeGroups   []petfind.PlaceGroup
}

// templates contains the server's templates required to render its pages.
type templates struct {
	home        *tmpl
	addPet      *tmpl
	search      *tmpl
	searchReply *tmpl
	showPets    *tmpl
	login       *tmpl
	demoXSS     *tmpl
}

type tmpl struct {
	*template.Template
	nav string
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
	photoStore petfind.PhotoStore,
	githubOAuth *oauth2.Config,
) (http.Handler, error) {
	t, err := parseTemplates(filepath.Join(templatePath, "templates"))
	if err != nil {
		return nil, fmt.Errorf("error parsing templates: %v", err)
	}
	groups, err := store.GetPlaceGroups()
	if err != nil {
		return nil, fmt.Errorf("error getting places and groups: %v", err)
	}

	s := &server{
		mux:           http.NewServeMux(),
		store:         store,
		templates:     t,
		github:        githubOAuth,
		sessions:      sessionStore,
		sessionTTL:    sessionTTL,
		sessionMaxTTL: sessionMaxTTL,
		photos:        photoStore,
		placeGroups:   groups,
	}
	s.handlers = gorillactx.ClearHandler(CSRF(s.mux))
	s.mux.Handle("/", s.guest(s.serveHome))
	s.mux.Handle("/search", handler(s.serveSearch))
	s.mux.Handle("/search/submit", handler(s.handleSearch))
	s.mux.Handle("/pets", handler(s.servePets))
	s.mux.Handle("/pets/add", s.auth(s.serveAddPet))
	s.mux.Handle("/pets/add/submit", s.auth(s.handleAddPet))
	s.mux.Handle("/login", handler(s.serveLogin))
	s.mux.Handle("/login/github", handler(s.handleLoginGitHub))
	s.mux.Handle("/login/github/cb", handler(s.handleLoginGitHubCallback))
	s.mux.Handle("/logout", s.auth(s.handleLogout))
	s.mux.Handle("/photos/", handler(s.servePhoto))
	s.mux.Handle("/demo/xss", handler(s.demoXSS))

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
func NewGitHubOAuthConfig(clientID, clientSecret string) *oauth2.Config {
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

func parseTemplates(dir string) (*templates, error) {
	homeTmpl, err := template.ParseFiles(
		filepath.Join(dir, "base.tmpl"),
		filepath.Join(dir, "navbar.tmpl"),
		filepath.Join(dir, "searchform.tmpl"),
		filepath.Join(dir, "home.tmpl"),
		filepath.Join(dir, "pets.tmpl"),
	)
	if err != nil {
		return nil, err
	}
	addPetTmpl, err := template.ParseFiles(
		filepath.Join(dir, "base.tmpl"),
		filepath.Join(dir, "navbar.tmpl"),
		filepath.Join(dir, "addpet.tmpl"),
	)
	if err != nil {
		return nil, err
	}
	searchTmpl, err := template.ParseFiles(
		filepath.Join(dir, "base.tmpl"),
		filepath.Join(dir, "navbar.tmpl"),
		filepath.Join(dir, "search.tmpl"),
		filepath.Join(dir, "searchform.tmpl"),
	)
	if err != nil {
		return nil, err
	}
	searchReplyTmpl, err := template.ParseFiles(
		filepath.Join(dir, "base.tmpl"),
		filepath.Join(dir, "navbar.tmpl"),
		filepath.Join(dir, "searchreply.tmpl"),
		filepath.Join(dir, "searchform.tmpl"),
		filepath.Join(dir, "pets.tmpl"),
	)
	if err != nil {
		return nil, err
	}
	showPetsTmpl, err := template.ParseFiles(
		filepath.Join(dir, "base.tmpl"),
		filepath.Join(dir, "navbar.tmpl"),
		filepath.Join(dir, "showpets.tmpl"),
		filepath.Join(dir, "pets.tmpl"),
	)
	if err != nil {
		return nil, err
	}
	demoXSSTmpl, err := template.ParseFiles(
		filepath.Join(dir, "base.tmpl"),
		filepath.Join(dir, "demo-xss.tmpl"),
	)
	if err != nil {
		return nil, err
	}
	loginTmpl, err := template.ParseFiles(
		filepath.Join(dir, "base.tmpl"),
		filepath.Join(dir, "navbar.tmpl"),
		filepath.Join(dir, "login.tmpl"),
	)
	t := &templates{
		home:        &tmpl{homeTmpl, "home"},
		addPet:      &tmpl{addPetTmpl, "add"},
		search:      &tmpl{searchTmpl, "search"},
		searchReply: &tmpl{searchReplyTmpl, "search"},
		showPets:    &tmpl{showPetsTmpl, "search"},
		login:       &tmpl{loginTmpl, ""},
		demoXSS:     &tmpl{demoXSSTmpl, ""},
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
	pets, err := s.store.GetFeaturedPets()
	if err != nil {
		return E(err, "Failed to get featured pets", http.StatusInternalServerError)
	}

	return s.render(w, r, s.templates.home, pets, searchForm{})
}

func (s *server) render(w http.ResponseWriter, r *http.Request, tmpl *tmpl, data interface{}, form interface{}) *Error {
	m := map[string]interface{}{
		csrf.TemplateTag: csrf.TemplateField(r),
		"nav":            tmpl.nav,
		"groups":         s.placeGroups,
	}

	if data != nil {
		m["data"] = data
	}
	if form != nil {
		m["form"] = form
	}

	user, _ := fromContextGetUser(r.Context())
	if user != nil {
		m["user"] = user
	}

	if err := tmpl.Execute(w, m); err != nil {
		log.Printf("could not serve %s: %v", tmpl.Name(), err)
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
			return fn(w, r.WithContext(ctx))
		}
		return fn(w, r)
	}
}

func (s *server) serveAddPet(w http.ResponseWriter, r *http.Request) *Error {
	user, ok := fromContextGetUser(r.Context())
	if !ok || user == nil {
		return E(nil, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}

	return s.render(w, r, s.templates.addPet, nil, addPetForm{})
}

func (s *server) handleAddPet(w http.ResponseWriter, r *http.Request) *Error {
	if r.Method != "POST" {
		fmt.Println("post")
		return E(nil, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	user, ok := fromContextGetUser(r.Context())
	if !ok || user == nil {
		return E(nil, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}

	pet, form, err := s.postFormPet(r)
	if err != nil {
		return E(err, "error inspecting add pet form", http.StatusInternalServerError)
	}
	if form.Invalid {
		return s.render(w, r, s.templates.addPet, nil, form)
	}

	photo, err := s.handlePetPhoto(w, r)
	if err != nil {
		return E(err, "Error uploading photo", http.StatusInternalServerError)
	}

	pet.PhotoID = photo.ID
	pet.OwnerID = user.ID
	pet.PlaceID = user.ID
	if err := s.store.AddPet(pet); err != nil {
		return E(err, "Error adding pet", http.StatusInternalServerError)
	}

	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}

func (s *server) handlePetPhoto(w http.ResponseWriter, r *http.Request) (*petfind.Photo, error) {
	file, handler, err := r.FormFile("photo")
	if err == http.ErrMissingFile {
		return nil, err
	}
	if err != nil {
		return nil, fmt.Errorf("error getting photo form file: %v", err)
	}
	defer file.Close()

	contentType := handler.Header.Get("Content-Type")
	photo, err := s.photos.Upload(file, contentType)
	if err != nil {
		return nil, fmt.Errorf("error uploading photo: %v", err)
	}

	if err := s.store.AddPhoto(photo); err != nil {
		return nil, fmt.Errorf("error adding photo to database: %v", err)
	}
	return photo, nil
}

type addPetForm struct {
	Invalid   bool
	Name      string
	NameErr   string
	Place     string
	PlaceErr  string
	Age       string
	AgeErr    string
	Size      string
	SizeErr   string
	Type      string
	TypeErr   string
	Gender    string
	GenderErr string
	Notes     string
	NotesErr  string

	PhotoErr string
}

type invalidReason string

func (ir invalidReason) String() string { return string(ir) }

func (s *server) postFormPet(r *http.Request) (*petfind.Pet, addPetForm, error) {
	form := addPetForm{}

	name := r.PostFormValue("name")
	form.Name = name
	if valid, reason := validName(name); !valid {
		form.Invalid = true
		form.NameErr = reason.String()
	}

	placeKey := r.PostFormValue("place")
	form.Place = placeKey
	place, valid, reason := s.validPlace(placeKey)
	if !valid {
		form.Invalid = true
		form.PlaceErr = reason.String()
	}

	ageStr := r.PostFormValue("age")
	form.Age = ageStr
	age, valid, reason := validAge(ageStr)
	if !valid {
		form.Invalid = true
		form.AgeErr = reason.String()
	}

	sizeStr := r.PostFormValue("size")
	form.Size = sizeStr
	size, valid, reason := validSize(sizeStr)
	if !valid {
		form.Invalid = true
		form.SizeErr = reason.String()
	}

	typeStr := r.PostFormValue("type")
	form.Type = typeStr
	t, valid, reason := validType(typeStr)
	if !valid {
		form.Invalid = true
		form.TypeErr = reason.String()
	}

	genderStr := r.PostFormValue("gender")
	form.Gender = genderStr
	gender, valid, reason := validGender(genderStr)
	if !valid {
		form.Invalid = true
		form.GenderErr = reason.String()
	}

	notes := r.PostFormValue("notes")
	form.Notes = notes
	if valid, reason := validNotes(notes); !valid {
		form.Invalid = true
		form.NotesErr = reason.String()
	}

	_, handler, err := r.FormFile("photo")
	if err == http.ErrMissingFile {
		form.Invalid = true
		form.PhotoErr = "Please choose a photo for the pet."
	}
	if err == nil {
		contentType := handler.Header.Get("Content-Type")
		if !validContentType(contentType) {
			form.Invalid = true
			form.PhotoErr = "Photo format must be jpeg, png or webp."
		}
	}
	if err != nil && err != http.ErrMissingFile {
		form.Invalid = true
		return nil, form, fmt.Errorf("error getting form file for photo validation: %v", err)
	}

	p := &petfind.Pet{Name: name, Age: age, Size: size, Type: t, Gender: gender, Notes: notes}
	if place != nil {
		p.PlaceID = place.ID
	}
	return p, form, nil
}

func validContentType(v string) bool {
	validValues := []string{"image/jpeg", "image/png", "image/webp"}
	for _, valid := range validValues {
		if v == valid {
			return true
		}
	}
	return false
}

func (s *server) validPlace(placeKey string) (*petfind.Place, bool, invalidReason) {
	if placeKey == "" {
		return nil, false, "Pet's location is required."
	}
	found := false
	var place *petfind.Place
	for _, g := range s.placeGroups {
		for _, p := range g.Places {
			if p.Key == placeKey {
				found = true
				place = &p
			}
		}
	}
	if !found {
		return nil, false, "Unrecognized location."
	}
	return place, true, ""
}

var nameRegex = regexp.MustCompile(`^[a-zA-Z]+$`)

func validName(name string) (bool, invalidReason) {
	if name == "" {
		return false, "Pet's name cannot be empty."
	}
	if len(name) > 20 {
		return false, "Pet's name cannot be longer than 20 characters."
	}
	if m := nameRegex.MatchString(name); !m {
		return false, "Pet's name can only contain letters."
	}
	return true, ""
}

func validNotes(notes string) (bool, invalidReason) {
	if notes == "" {
		return false, "Pet's notes cannot be empty."
	}
	if len(notes) > 1000 {
		return false, "Pet's notes cannot be longer than 1000 characters."
	}
	return true, ""
}

func validAge(ageStr string) (petfind.PetAge, bool, invalidReason) {
	if ageStr == "" {
		return petfind.UnknownAge, false, "Age is required."
	}
	age, err := strconv.ParseInt(ageStr, 10, 64)
	if err != nil {
		return petfind.UnknownAge, false, "Bad value for age."
	}

	petAge := petfind.PetAge(age)
	if !validPetAge(petAge) {
		return petfind.UnknownAge, false, "Invalid value for age."
	}
	return petAge, true, ""
}

func validPetAge(v petfind.PetAge) bool {
	validValues := []petfind.PetAge{petfind.UnknownAge, petfind.Baby, petfind.Young, petfind.Adult, petfind.Senior}
	for _, valid := range validValues {
		if v == valid {
			return true
		}
	}
	return false
}

func validSize(ageStr string) (petfind.PetSize, bool, invalidReason) {
	if ageStr == "" {
		return petfind.UnknownSize, false, "Pet's size is required."
	}
	size, err := strconv.ParseInt(ageStr, 10, 64)
	if err != nil {
		return petfind.UnknownSize, false, "Bad value for size."
	}

	petSize := petfind.PetSize(size)
	if !validPetSize(petSize) {
		return petfind.UnknownSize, false, "Invalid value for size."
	}
	return petSize, true, ""
}

func validPetSize(v petfind.PetSize) bool {
	validValues := []petfind.PetSize{petfind.UnknownSize, petfind.Small, petfind.Medium, petfind.Large, petfind.Huge}
	for _, valid := range validValues {
		if v == valid {
			return true
		}
	}
	return false
}

func validType(typeStr string) (petfind.PetType, bool, invalidReason) {
	if typeStr == "" {
		return petfind.UnknownType, false, "Pet's type is required."
	}
	t, err := strconv.ParseInt(typeStr, 10, 64)
	if err != nil {
		return petfind.UnknownType, false, "Bad value for pet's type."
	}

	petType := petfind.PetType(t)
	if !validPetType(petType) {
		return petfind.UnknownType, false, "Invalid value for pet's type."
	}
	return petType, true, ""
}

func validPetType(v petfind.PetType) bool {
	validValues := []petfind.PetType{petfind.UnknownType, petfind.Cat, petfind.Dog}
	for _, valid := range validValues {
		if v == valid {
			return true
		}
	}
	return false
}

func validGender(typeStr string) (petfind.PetGender, bool, invalidReason) {
	if typeStr == "" {
		return petfind.UnknownGender, false, "Pet's gender is required."
	}
	t, err := strconv.ParseInt(typeStr, 10, 64)
	if err != nil {
		return petfind.UnknownGender, false, "Bad value for pet's gender."
	}

	petGender := petfind.PetGender(t)
	if !validPetGender(petGender) {
		return petfind.UnknownGender, false, "Invalid value for pet's gender."
	}
	return petGender, true, ""
}

func validPetGender(v petfind.PetGender) bool {
	validValues := []petfind.PetGender{petfind.UnknownGender, petfind.Male, petfind.Female}
	for _, valid := range validValues {
		if v == valid {
			return true
		}
	}
	return false
}

type searchForm struct {
	Invalid   bool
	Place     string
	PlaceErr  string
	Type      string
	TypeErr   string
	Age       string
	AgeErr    string
	Size      string
	SizeErr   string
	Gender    string
	GenderErr string
}

func (s *server) handleSearch(w http.ResponseWriter, r *http.Request) *Error {
	search := petfind.Search{}
	form := searchForm{}

	placeKey := r.FormValue("place")
	form.Place = placeKey
	_, valid, reason := s.validPlace(placeKey)
	if !valid {
		form.Invalid = true
		form.PlaceErr = reason.String()
	}
	search.PlaceKey = placeKey

	typeStr := r.FormValue("type")
	form.Type = typeStr
	if typeStr != "" {
		t, valid, reason := validType(typeStr)
		if !valid {
			form.Invalid = true
			form.TypeErr = reason.String()
		} else {
			search.Type = t
			search.UseType = true
		}
	}

	ageStr := r.FormValue("age")
	form.Age = ageStr
	if ageStr != "" {
		age, valid, reason := validAge(ageStr)
		if !valid {
			form.Invalid = true
			form.AgeErr = reason.String()
		} else {
			search.Age = age
			search.UseAge = true
		}
	}

	sizeStr := r.FormValue("size")
	form.Size = sizeStr
	if sizeStr != "" {
		size, valid, reason := validSize(sizeStr)
		if !valid {
			form.Invalid = true
			form.SizeErr = reason.String()
		} else {
			search.Size = size
			search.UseSize = true
		}
	}

	genderStr := r.FormValue("gender")
	form.Gender = genderStr
	if genderStr != "" {
		gender, valid, reason := validGender(genderStr)
		if !valid {
			form.Invalid = true
			form.GenderErr = reason.String()
		} else {
			search.Gender = gender
			search.UseGender = true
		}
	}

	if form.Invalid {
		return s.render(w, r, s.templates.search, nil, form)
	}

	pets, err := s.store.SearchPets(search)
	if err != nil {
		return E(err, "internal server error", http.StatusInternalServerError)
	}

	return s.render(w, r, s.templates.searchReply, pets, form)
}

func (s *server) serveSearch(w http.ResponseWriter, r *http.Request) *Error {
	form := searchForm{}
	return s.render(w, r, s.templates.search, nil, form)
}

func (s *server) servePets(w http.ResponseWriter, r *http.Request) *Error {
	pets, err := s.store.GetAllPets()
	if err != nil {
		return E(err, "Error getting all pets", http.StatusInternalServerError)
	}
	return s.render(w, r, s.templates.showPets, pets, nil)
}

func (s *server) servePhoto(w http.ResponseWriter, r *http.Request) *Error {
	idStr := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		return E(err, "invalid photo id", http.StatusBadRequest)
	}

	photo, err := s.store.GetPhoto(id)
	if err == petfind.ErrNotFound {
		return E(nil, "Photo does not exist", http.StatusNotFound)
	}
	if err != nil {
		return E(err, "Error getting photo from database", http.StatusInternalServerError)
	}

	w.Header().Set("Content-Type", photo.ContentType)
	if err := s.photos.ServePhoto(w, photo); err != nil {
		return E(err, "error serving photo", http.StatusInternalServerError)
	}
	return nil
}
