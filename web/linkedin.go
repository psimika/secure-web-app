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
	"golang.org/x/oauth2/linkedin"

	"github.com/gorilla/securecookie"
	"github.com/psimika/secure-web-app/petfind"
)

func NewLinkedInOAuthConfig(clientID, clientSecret, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURL:  redirectURL,
		Scopes:       []string{},
		Endpoint:     linkedin.Endpoint,
	}
}

// handleLoginLinkedIn sends an oauth request to login with LinkedIn (Caserta
// 2015).
//
// http://pierrecaserta.com/go-oauth-facebook-github-twitter-google-plus/
func (s *server) handleLoginLinkedIn(w http.ResponseWriter, r *http.Request) *Error {
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

	url := s.linkedin.AuthCodeURL(state)
	http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	return nil
}

// handleLoginLinkedInCallback receives the callback request returned by
// LinkedIn after the user has given consent to access their information
// (Caserta 2015):
//
// http://pierrecaserta.com/go-oauth-facebook-github-twitter-google-plus/
func (s *server) handleLoginLinkedInCallback(w http.ResponseWriter, r *http.Request) *Error {
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

	// Check that the state token returned from LinkedIn is the same as the one
	// we generated.
	state := r.FormValue("state")
	// Constant time compare to mitigate timing attacks.
	if subtle.ConstantTimeCompare([]byte(state), []byte(oauthState)) != 1 {
		// TODO(psimika): Redirect instead of returning an error?
		e := fmt.Errorf("invalid oauth state, expected %q, got %q", oauthState, state)
		return E(e, "invalid oauth state", http.StatusForbidden)
	}

	// Exchange authorization code for a LinkedIn API token.
	code := r.FormValue("code")
	token, err := s.linkedin.Exchange(context.Background(), code)
	if err != nil {
		return E(err, "error exchanging authorization code for token", http.StatusInternalServerError)
	}

	// Create an HTTP client that uses the LinkedIn API token.
	c := s.linkedin.Client(context.Background(), token)

	// Use the client to get the consented user's info from the LinkedIn API.
	linkedinUser, err := getLinkedinUser(c, token.AccessToken)
	if err != nil {
		return E(err, "could not get user from LinkedIn API", http.StatusInternalServerError)
	}
	fmt.Printf("from api: %#v\n", linkedinUser)

	user, err := s.store.PutLinkedinUser(linkedinUser)
	if err != nil {
		return E(err, "error storing linkedin user", http.StatusInternalServerError)
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

func getLinkedinUser(client *http.Client, accessToken string) (*petfind.LinkedinUser, error) {
	resp, err := client.Get("https://api.linkedin.com/v1/people/~?format=json")

	if err != nil {
		return nil, err
	}

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		linkedinErr := new(LinkedinError)
		if err := json.NewDecoder(resp.Body).Decode(linkedinErr); err != nil {
			return nil, fmt.Errorf("could not decode linkedin error: %v", err)
		}
		return nil, fmt.Errorf("%d %s", linkedinErr.Status, linkedinErr.Message)
	}

	user := new(petfind.LinkedinUser)
	if err := json.NewDecoder(resp.Body).Decode(user); err != nil {
		return nil, fmt.Errorf("could not decode linkedin user: %v", err)
	}
	return user, nil
}

type LinkedinError struct {
	ErrorCode int    `json:"errorCode"`
	Message   string `json:"message"`
	RequestID string `json:"requestId"`
	Status    int    `json:"status"`
	Timestamp int64  `json:"timestamp"`
}
