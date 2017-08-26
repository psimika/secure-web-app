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

const (
	// OWASP (2017b) recommends using a generic name such as "id" for the
	// session ID name to avoid exposing implementation details:
	//
	// https://www.owasp.org/index.php/Session_Management_Cheat_Sheet#Session_ID_Name_Fingerprinting
	sessionName         = "id"
	oauthStateTokenSize = 32
)

// auth protects other handlers letting only logged in users access them. If
// the session ID is not found in the context, the handler redirects to /login.
func (s *server) auth(fn handler) handler {
	return func(w http.ResponseWriter, r *http.Request) *Error {
		log.Println("r.URL.Path:", r.URL.Path)
		log.Println("r.Referer():", r.Referer())
		log.Println("r.RemoteAddr:", r.RemoteAddr)
		log.Println("r.Header.Get(\"X-Forwarded-For\"):", r.Header.Get("X-Forwarded-For"))

		session, err := s.sessions.Get(r, sessionName)
		if err != nil {
			log.Printf("error getting session: %v", err)
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}

		// Store the original URL path the user was trying to access so we can
		// redirect to that later after a successful login.
		session.Values["redirectPath"] = r.URL.Path
		if err = session.Save(r, w); err != nil {
			return E(err, "error storing redirectPath in session", http.StatusInternalServerError)
		}

		// If the session is brand new, it means that the user has not logged
		// in before.
		if session.IsNew {
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}

		// Get the user's ID stored in the session.
		userID, err := fromSessionGetUserID(session)
		if err != nil {
			log.Println(err)
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}

		// Get the user from the database based on the session's user ID.
		user, err := s.store.GetUser(userID)
		if err != nil {
			return E(err, "error getting user", http.StatusInternalServerError)
		}

		// If the session is not valid then we delete it.
		if err = s.validateSession(r, session); err != nil {
			log.Println(err)
			session.Options.MaxAge = -1
			if err = sessions.Save(r, w); err != nil {
				return E(err, "error deleting invalid session", http.StatusInternalServerError)
			}
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
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

func (s *server) validateSession(r *http.Request, session *sessions.Session) error {
	// Get the session's userAgent value and check with the current HTTP
	// request's user agent. If it's not the same we consider the session
	// as invalid for extra safety.
	userAgent, err := fromSessionGetUserAgent(session)
	if err != nil {
		return err
	}
	if subtle.ConstantTimeCompare([]byte(userAgent), []byte(r.UserAgent())) != 1 {
		return fmt.Errorf("User-Agent doesn't match")
	}

	// Get the session's created value to find when it was created.
	created, err := fromSessionGetCreated(session)
	if err != nil {
		return err
	}

	// Check if the session has expired.
	expirationTime := time.Unix(created, 0).Add(time.Duration(s.sessionMaxTTL) * time.Second)
	if expirationTime.Before(time.Now()) {
		return fmt.Errorf("session expired")
	}
	return nil
}

// ---
// Helper functions to add and retrieve details of the logged in user from the
// context (Ajmani 2014).
//
// https://blog.golang.org/context

type contextKey int

const (
	userContextKey contextKey = iota
)

// fromContextGetUser retrieves User from the context.
func fromContextGetUser(ctx context.Context) (*petfind.User, bool) {
	user, ok := ctx.Value(userContextKey).(*petfind.User)
	return user, ok
}

// newContextWithUser adds User to the context.
func newContextWithUser(ctx context.Context, user *petfind.User) context.Context {
	return context.WithValue(ctx, userContextKey, user)
}

// -- Helper functions to retrieve session values.

// fromSessionGetCreated returns the session's created value as int64. It will
// return an error if the value does not exist or if it is the wrong type.
func fromSessionGetCreated(session *sessions.Session) (int64, error) {
	v, ok := session.Values["created"]
	if !ok {
		return 0, fmt.Errorf("session has no created value")
	}
	created, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected created type")
	}
	return created, nil
}

// fromSessionGetUserID returns the session's userID value as int64. It will
// return an error if the value does not exist or if it is the wrong type.
func fromSessionGetUserID(session *sessions.Session) (int64, error) {
	v, ok := session.Values["userID"]
	if !ok {
		return 0, fmt.Errorf("session has no userID")
	}
	userID, ok := v.(int64)
	if !ok {
		return 0, fmt.Errorf("unexpected userID type")
	}
	return userID, nil
}

// fromSessionGetUserAgent returns the session's userAgent value as string. It
// will return an error if the value does not exist or if it is the wrong type.
func fromSessionGetUserAgent(session *sessions.Session) (string, error) {
	return fromSessionGetString(session, "userAgent")
}

// fromSessionGetRedirectPath returns the path the user was trying to access as
// string. It will return an error if the value does not exist or if it is the
// wrong type.
func fromSessionGetRedirectPath(session *sessions.Session) (string, error) {
	return fromSessionGetString(session, "redirectPath")
}

func fromSessionGetString(session *sessions.Session, key string) (string, error) {
	v, ok := session.Values[key]
	if !ok {
		return "", fmt.Errorf("session has no %s", key)
	}
	strVal, ok := v.(string)
	if !ok {
		return "", fmt.Errorf("unexpected %s type", key)
	}
	return strVal, nil
}

// ---

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
	if err = session.Save(r, w); err != nil {
		return E(err, "error deleting session for logout", http.StatusInternalServerError)
	}

	http.Redirect(w, r, "/", http.StatusFound)
	return nil
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
	session.Values["userAgent"] = r.UserAgent()
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
	githubUser, err := getGithubUser(c)
	if err != nil {
		return E(err, "could not get user from GitHub API", http.StatusInternalServerError)
	}

	user, err := s.store.PutGithubUser(githubUser)
	if err != nil {
		return E(err, "error storing github user", http.StatusInternalServerError)
	}

	session.Values["userID"] = user.ID
	delete(session.Values, "state")
	if err := session.Save(r, w); err != nil {
		return E(err, "error saving session", http.StatusInternalServerError)
	}

	redirectPath, err := fromSessionGetRedirectPath(session)
	if err != nil {
		log.Println("no redirectPath in session")
		http.Redirect(w, r, "/", http.StatusFound)
		return nil
	}
	http.Redirect(w, r, redirectPath, http.StatusFound)

	return nil
}

func getGithubUser(client *http.Client) (*petfind.GithubUser, error) {
	resp, err := client.Get("https://api.github.com/user")
	if err != nil {
		return nil, err
	}

	user := new(petfind.GithubUser)
	if err := json.NewDecoder(resp.Body).Decode(user); err != nil {
		return nil, fmt.Errorf("could not decode user: %v", err)
	}
	return user, nil
}
