// +build db

package postgres_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/psimika/secure-web-app/petfind"
)

func TestCreateUser(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	githubID := int64(5)
	u := &petfind.User{Name: "Jane Doe", GithubID: githubID}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	user, err := s.GetUserByGithubID(githubID)
	if err != nil {
		t.Fatalf("GetUserByGithubID failed: %v", err)
	}

	// Ignore time field.
	user.Added = time.Time{}

	want := &petfind.User{ID: 1, Name: "Jane Doe", GithubID: githubID}
	if got := user; !reflect.DeepEqual(got, want) {
		t.Fatalf("GetUserByGithubID \nhave: %#v\nwant: %#v", got, want)
	}
}

func TestGetUserByGithubID_notFound(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	_, err := s.GetUserByGithubID(0)
	if err != petfind.ErrNotFound {
		t.Fatalf("GetUserByGithubID for unknown githubID returned %v, expected: %q", err, petfind.ErrNotFound)
	}
}

func TestGetUserBySessionID(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	githubID := int64(5)
	u := &petfind.User{Name: "Jane Doe", GithubID: githubID}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	session := &petfind.Session{ID: "foo", UserID: 1, Added: time.Now(), Expires: time.Now().Add(time.Duration(30) * time.Minute)}
	if err := s.CreateUserSession(session); err != nil {
		t.Fatalf("CreateUserSession failed: %v", err)
	}

	user, err := s.GetUserBySessionID("foo")
	if err != nil {
		t.Fatalf("GetUserBySessionID failed: %v", err)
	}

	// Ignore time field.
	user.Added = time.Time{}

	want := &petfind.User{ID: 1, Name: "Jane Doe", GithubID: githubID}
	if got := user; !reflect.DeepEqual(got, want) {
		t.Fatalf("GetUserBySessionID \nhave: %#v\nwant: %#v", got, want)
	}
}

func TestGetUserBySessionID_expired(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	githubID := int64(5)
	u := &petfind.User{Name: "Jane Doe", GithubID: githubID}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	session := &petfind.Session{ID: "foo", UserID: 1, Added: time.Now(), Expires: time.Now()}
	if err := s.CreateUserSession(session); err != nil {
		t.Fatalf("CreateUserSession failed: %v", err)
	}

	_, err := s.GetUserBySessionID("foo")
	if err != petfind.ErrNotFound {
		t.Fatalf("GetUserBySessionID for expired session returned %v, expected: %q", err, petfind.ErrNotFound)
	}
}

func TestGetUserBySessionID_notFound(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	_, err := s.GetUserBySessionID("foo")
	if err != petfind.ErrNotFound {
		t.Fatalf("GetUserBySessionID for unknown sessionID returned %v, expected: %q", err, petfind.ErrNotFound)
	}
}
