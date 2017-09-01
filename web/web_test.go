package web

import (
	"bytes"
	"go/build"
	"log"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"testing"

	"github.com/gorilla/csrf"
	"github.com/gorilla/sessions"
	"github.com/psimika/secure-web-app/petfind"
)

var (
	hashKey      = []byte("this-hashkey-is-exactly-32-bytes")
	blockKey     = []byte("this-blockkey-is-exactly-32bytes")
	authKey      = []byte("this-authkey-is-exactly-32-bytes")
	githubID     = ""
	githubSecret = ""
)

func defaultTmplPath() string {
	p, err := build.Import("github.com/psimika/secure-web-app/web", "", build.FindOnly)
	if err != nil {
		return ""
	}
	return p.Dir
}

func defaultPhotosPath() string {
	p, err := build.Import("github.com/psimika/secure-web-app", "", build.FindOnly)
	if err != nil {
		return "photos"
	}
	return filepath.Join(p.Dir, "photos")
}

type store struct{}

func newStore() *store { return &store{} }

func (s *store) AddPet(*petfind.Pet) error                                { return nil }
func (s *store) GetAllPets() ([]petfind.Pet, error)                       { return nil, nil }
func (s *store) CreateUser(*petfind.User) error                           { return nil }
func (s *store) GetUser(userID int64) (*petfind.User, error)              { return nil, nil }
func (s *store) PutGithubUser(*petfind.GithubUser) (*petfind.User, error) { return nil, nil }
func (s *store) GetUserByGithubID(githubID int64) (*petfind.User, error)  { return nil, nil }
func (s *store) MakeSchema() error                                        { return nil }
func (s *store) DropSchema() error                                        { return nil }

func (s *store) AddPhoto(*petfind.Photo) error                  { return nil }
func (s *store) GetPhoto(photoID int64) (*petfind.Photo, error) { return nil, nil }

func setup() (http.Handler, *templates) {
	store := newStore()
	sessionsStore := sessions.NewCookieStore(hashKey, blockKey)
	CSRF := csrf.Protect(authKey)
	photosPath := defaultPhotosPath()
	photoStore := petfind.NewPhotoStore(photosPath)
	h, err := NewServer(store, sessionsStore, 1200, 3600, CSRF, defaultTmplPath(), photoStore, githubID, githubSecret)
	if err != nil {
		log.Fatal("web test setup error:", err)
	}
	tmpl, err := parseTemplates(filepath.Join(defaultTmplPath(), "templates"))
	if err != nil {
		log.Fatal("web test setup parsing templates error:", err)
	}
	return h, tmpl
}

func TestServer_HomeHandler(t *testing.T) {
	h, tmpl := setup()
	// Create a request to pass to our handler. We don't have any query parameters for now, so we'll
	// pass 'nil' as the third parameter.
	r, err := http.NewRequest("GET", "/", nil)
	if err != nil {
		t.Fatal(err)
	}

	// We create a ResponseRecorder (which satisfies http.ResponseWriter) to record the response.
	w := httptest.NewRecorder()
	//handler := http.HandlerFunc(h)

	// Our handlers satisfy http.Handler, so we can call their ServeHTTP method
	// directly and pass in our Request and ResponseRecorder.
	h.ServeHTTP(w, r)

	// Check the status code is what we expect.
	if got, want := w.Code, http.StatusOK; got != want {
		t.Errorf("home handler status code got: %v, want: %v", got, want)
	}

	// Check the response body is what we expect.
	buf := new(bytes.Buffer)
	if err := tmpl.home.Execute(buf, nil); err != nil {
		t.Fatal(err)
	}
	if got, want := w.Body.String(), buf.String(); got != want {
		t.Errorf("home handler body got: %v want: %v", got, want)
	}
}
