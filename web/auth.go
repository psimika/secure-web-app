package web

import (
	"context"
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"golang.org/x/oauth2"

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
		sessionCookie, err := r.Cookie(sessionCookieName)
		if err != nil || sessionCookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}
		sessionID := sessionCookie.Value
		user, err := s.store.GetUserBySessionID(sessionID)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusFound)
			return nil
		}

		ctx := newContextWithUser(r.Context(), user)
		ctx = newContextWithSessionID(ctx, sessionID)
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
	sessionCookieName   = "petfind_session"
	oauthStateTokenSize = 32
	// OWASP (2017b) recommends a session ID length of at least 16 bytes to
	// prevent brute force attacks:
	//
	// https://www.owasp.org/index.php/Session_Management_Cheat_Sheet#Session_ID_Length
	sessionIDSize = 32
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

	user, ok := fromContextGetUser(r.Context())
	if !ok || user == nil {
		return E(nil, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}
	sessionID, ok := fromContextGetSessionID(r.Context())
	if !ok || sessionID == "" {
		return E(nil, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
	}

	if err := s.store.DeleteUserSession(sessionID); err != nil {
		return E(err, "error deleting user session", http.StatusInternalServerError)
	}
	cookie := &http.Cookie{Name: sessionCookieName, Value: "", Path: "/", MaxAge: -1, Secure: true, HttpOnly: true}
	http.SetCookie(w, cookie)
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
	key := generateRandomKey(oauthStateTokenSize)
	if key == nil {
		return E(nil, "error generating random key", http.StatusInternalServerError)
	}
	state := base64.URLEncoding.EncodeToString(key)

	// Put state token in our whitelist to be able to track it and make sure it
	// will only be received once and thus mitigate replay attacks.
	s.stateWhitelist.Put(state)

	// Store state token in a cookie to be able to receive it in the callback
	// handler.
	oauthStateCookie := &http.Cookie{
		Name:     "petfind_oauth_state",
		Value:    state,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		Expires:  time.Now().Add(2 * time.Minute),
	}
	http.SetCookie(w, oauthStateCookie)

	url := s.github.AuthCodeURL(state, oauth2.AccessTypeOnline)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

// handleLoginGitHubCallback receives the callback request returned by GitHub
// after the user has given consent to access their information (Caserta 2015):
//
// http://pierrecaserta.com/go-oauth-facebook-github-twitter-google-plus/
func (s *server) handleLoginGitHubCallback(w http.ResponseWriter, r *http.Request) *Error {
	// Get the state token that we stored in the cookie when it was generated
	// by the handleLoginGitHub login handler.
	//
	// The application must be serving HTTPS for this to work.
	//
	// If the application is serving HTTP then the cookie will not exist after
	// the roundtrip because of the change of protocols since GitHub is using
	// HTTPS. Therefore login with GitHub will only work when the application
	// is serving HTTPS.
	oauthStateCookie, err := r.Cookie("petfind_oauth_state")
	if err == http.ErrNoCookie {
		guess := "serving HTTP? oauth state cannot be accessed when changing protocols HTTPS(GitHub)->HTTP"
		return E(err, "no oauth state: "+guess, http.StatusForbidden)
	}
	if err != nil {
		return E(err, "error reading oauth state cookie", http.StatusInternalServerError)
	}
	oauthState := oauthStateCookie.Value

	// Make sure the state token exists in our whitelist and if it does, delete
	// it from the whitelist so it cannot be used again and prevent replay
	// attacks.
	if !s.stateWhitelist.Get(oauthState) {
		return E(nil, "unknown oauth state token", http.StatusForbidden)
	}
	s.stateWhitelist.Delete(oauthState)

	// Check that the state token returned from GitHub is the same as the one
	// we generated.
	state := r.FormValue("state")
	// Constant time compare to mitigate timing attacks.
	if subtle.ConstantTimeCompare([]byte(state), []byte(oauthState)) != 1 {
		// TODO(psimika): Redirect instead of returning an error?
		e := fmt.Errorf("invalid oauth state, expected %q, got %q\n", oauthState, state)
		return E(e, "invalid oauth state", http.StatusForbidden)
	}

	// Exchange authorization code for a GitHub API token.
	code := r.FormValue("code")
	token, err := s.github.Exchange(oauth2.NoContext, code)
	if err != nil {
		return E(err, "error exchanging authorization code for token", http.StatusInternalServerError)
	}

	// Create an HTTP client that uses the GitHub API token.
	c := s.github.Client(oauth2.NoContext, token)

	// Use the client to get the consented user's info from the GitHub API.
	githubUser, err := getGitHubUser(c)
	if err != nil {
		return E(err, "could not get user from GitHub API", http.StatusInternalServerError)
	}

	return s.loginGithubUser(w, r, githubUser)
}

func (s *server) loginGithubUser(w http.ResponseWriter, r *http.Request, githubUser *GitHubUser) *Error {
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
	key := generateRandomKey(sessionIDSize)
	if key == nil {
		return E(err, "could not generate session ID", http.StatusInternalServerError)
	}
	now := time.Now()
	session := &petfind.Session{
		ID:      base64.URLEncoding.EncodeToString(key),
		UserID:  user.ID,
		Added:   now,
		Expires: now.Add(s.sessionDuration),
	}
	if err := s.store.CreateUserSession(session); err != nil {
		return E(err, "could not create session", http.StatusInternalServerError)
	}

	sessionCookie := &http.Cookie{
		Name:     sessionCookieName,
		Value:    session.ID,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
		Expires:  now.Add(s.sessionDuration),
	}
	http.SetCookie(w, sessionCookie)
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

// generateRandomKey creates a random key with the given length in bytes.
// On failure, returns nil.
//
// Callers should explicitly check for the possibility of a nil return, treat
// it as a failure of the system random number generator, and not continue.
func generateRandomKey(length int) []byte {
	k := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return nil
	}
	return k
}
