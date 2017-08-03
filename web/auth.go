package web

import (
	"context"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"golang.org/x/oauth2"

	"github.com/gorilla/securecookie"
	"github.com/gorilla/sessions"
	"github.com/psimika/secure-web-app/petfind"
)

type contextKey int

const (
	userContextKey contextKey = iota
	sessionIDContextKey
)

// auth protects other handlers letting only logged in users access them. If
// the session ID is not found in the context, the handler redirects to /login.
func (s *server) auth(fn handler) handler {
	return func(w http.ResponseWriter, r *http.Request) *Error {
		log.Println("r.Referer():", r.Referer())
		log.Println("r.RemoteAddr:", r.RemoteAddr)
		log.Println("r.Header.Get(\"X-Forwarded-For\"):", r.Header.Get("X-Forwarded-For"))

		session, err := s.sessions.Get(r, sessionName)
		if err != nil {
			log.Printf("error getting session: %v", err)
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}

		// Get the session's created value to find when it was created.
		t, ok := session.Values["created"]
		if !ok {
			log.Println("session has no created value")
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}
		created, ok := t.(int64)
		if !ok {
			log.Println("unexpected created type")
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}

		// If the session has expired then we delete it.
		expirationTime := time.Unix(created, 0).Add(time.Duration(s.sessionMaxTTL) * time.Second)
		if expirationTime.Before(time.Now()) {
			log.Println("session expired")
			session.Options.MaxAge = -1
			if err = sessions.Save(r, w); err != nil {
				return E(err, "error deleting expired session", http.StatusInternalServerError)
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}

		// Get the user's ID stored in the session.
		v, ok := session.Values["userID"]
		if !ok {
			log.Println("session has no userID")
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}
		userID, ok := v.(int64)
		if !ok {
			log.Println("unexpected userID type")
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}

		// Get the user from the database based on the session's user ID.
		user, err := s.store.GetUser(userID)
		if err != nil {
			return E(err, "error getting user", http.StatusInternalServerError)
		}

		// Extend session's idle timeout.
		session.Options.MaxAge = s.sessionTTL
		if err = sessions.Save(r, w); err != nil {
			return E(err, "error saving session", http.StatusInternalServerError)
		}

		// Put user in the context so that the next handler can access it.
		ctx := newContextWithUser(r.Context(), user)
		fn(w, r.WithContext(ctx))
		return nil
	}
}

// ---
// Helper functions to add and retrieve details of the logged in user from the
// context (Ajmani 2014).
//
// https://blog.golang.org/context

// fromContextGetUser retrieves User from the context.
func fromContextGetUser(ctx context.Context) (*petfind.User, bool) {
	user, ok := ctx.Value(userContextKey).(*petfind.User)
	return user, ok
}

// newContextWithUser adds User to the context.
func newContextWithUser(ctx context.Context, user *petfind.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// fromContextGetSessionID retrieves sessionID from the context.
func fromContextGetSessionID(ctx context.Context) (string, bool) {
	sessionID, ok := ctx.Value(sessionIDContextKey).(string)
	return sessionID, ok
}

// newContextWithSessionID adds sessionID to the context.
func newContextWithSessionID(ctx context.Context, sessionID string) context.Context {
	return context.WithValue(ctx, sessionIDContextKey, sessionID)
}

// ---

const (
	// OWASP (2017b) recommends using a generic name such as "id" for the
	// session ID name to avoid exposing implementation details:
	//
	// https://www.owasp.org/index.php/Session_Management_Cheat_Sheet#Session_ID_Name_Fingerprinting
	sessionName         = "id"
	oauthStateTokenSize = 32
)

func (s *server) serveLogin(w http.ResponseWriter, r *http.Request) *Error {
	if err := s.tmpl.login.Execute(w, nil); err != nil {
		return E(err, "error rendering login", http.StatusInternalServerError)
	}
	return nil
}

func (s *server) handleLogout(w http.ResponseWriter, r *http.Request) *Error {
	if r.Method != "POST" {
		return E(nil, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
	}

	session, err := s.sessions.Get(r, sessionName)
	if err != nil {
		return E(err, "error getting session for logout", http.StatusInternalServerError)
	}
	session.Options.MaxAge = -1
	if err = sessions.Save(r, w); err != nil {
		return E(err, "error deleting session for logout", http.StatusInternalServerError)
	}

	http.Redirect(w, r, "/", http.StatusFound)
	return nil
}

type GitHubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// handleLoginGitHub sends an oauth request to login with GitHub (Caserta
// 2015).
//
// http://pierrecaserta.com/go-oauth-facebook-github-twitter-google-plus/
func (s *server) handleLoginGitHub(w http.ResponseWriter, r *http.Request) *Error {
	// Create oauth state token.
	key := securecookie.GenerateRandomKey(oauthStateTokenSize)
	if key == nil {
		return E(nil, "error generating random oauth state token", http.StatusInternalServerError)
	}
	state := base64.URLEncoding.EncodeToString(key)

	session, err := s.sessions.Get(r, sessionName)
	if err != nil {
		return E(err, "error getting session", http.StatusInternalServerError)
	}
	session.Values["state"] = state
	session.Values["created"] = time.Now().UTC().Unix()
	if err := session.Save(r, w); err != nil {
		return E(err, "error saving session", http.StatusInternalServerError)
	}

	url := s.github.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

// handleLoginGitHubCallback receives the callback request returned by GitHub
// after the user has given consent to access their information (Caserta 2015):
//
// http://pierrecaserta.com/go-oauth-facebook-github-twitter-google-plus/
func (s *server) handleLoginGitHubCallback(w http.ResponseWriter, r *http.Request) *Error {
	session, err := s.sessions.Get(r, sessionName)
	if err != nil {
		return E(err, "error getting session", http.StatusInternalServerError)
	}
	val, ok := session.Values["state"]
	if !ok {
		return E(nil, "no state in session", http.StatusForbidden)
	}
	oauthState, ok := val.(string)
	if !ok {
		return E(nil, "unexpected state type", http.StatusForbidden)
	}

	// Check that the state token returned from GitHub is the same as the one
	// we generated.
	state := r.FormValue("state")
	// Constant time compare to mitigate timing attacks.
	if subtle.ConstantTimeCompare([]byte(state), []byte(oauthState)) != 1 {
		// TODO(psimika): Redirect instead of returning an error?
		e := fmt.Errorf("invalid oauth state, expected %q, got %q", oauthState, state)
		return E(e, "invalid oauth state", http.StatusForbidden)
	}

	// Exchange authorization code for a GitHub API token.
	code := r.FormValue("code")
	token, err := s.github.Exchange(context.Background(), code)
	if err != nil {
		return E(err, "error exchanging authorization code for token", http.StatusInternalServerError)
	}

	// Create an HTTP client that uses the GitHub API token.
	c := s.github.Client(context.Background(), token)

	// Use the client to get the consented user's info from the GitHub API.
	githubUser, err := getGitHubUser(c)
	if err != nil {
		return E(err, "could not get user from GitHub API", http.StatusInternalServerError)
	}

	user, err := s.store.GetUserByGithubID(githubUser.ID)
	// If the user is not found in our database we create them.
	if err != nil && err == petfind.ErrNotFound {
		user = &petfind.User{
			Name:     githubUser.Name,
			GithubID: githubUser.ID,
			Login:    githubUser.Login,
		}
		if cerr := s.store.CreateUser(user); cerr != nil {
			return E(cerr, "could not create user", http.StatusInternalServerError)
		}
	}
	if err != nil && err != petfind.ErrNotFound {
		return E(err, "could not retrieve user", http.StatusInternalServerError)
	}

	session.Values["userID"] = user.ID
	delete(session.Values, "state")
	if err := session.Save(r, w); err != nil {
		return E(err, "error saving session", http.StatusInternalServerError)
	}
	return nil
}

func getGitHubUser(client *http.Client) (*GitHubUser, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}
	user := new(GitHubUser)
	if err := json.NewDecoder(resp.Body).Decode(user); err != nil {
		return nil, fmt.Errorf("could not decode user: %v", err)
	}
	return user, nil
}
