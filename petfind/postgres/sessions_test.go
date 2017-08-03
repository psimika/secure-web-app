// +build db

package postgres_test

import (
	"testing"

	"github.com/psimika/secure-web-app/petfind"
)

func TestDeleteUserSession(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	githubID := int64(5)
	u := &petfind.User{Name: "Jane Doe", GithubID: githubID}
	if err := s.CreateUser(u); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}

	if err := s.CreateUserSession(&petfind.Session{ID: "foo", UserID: 1}); err != nil {
		t.Fatalf("CreateUserSession failed: %v", err)
	}

	if err := s.DeleteUserSession("foo"); err != nil {
		t.Fatalf("DeleteUserSession failed: %v", err)
	}

	_, err := s.GetUserBySessionID("foo")
	if err != petfind.ErrNotFound {
		t.Fatalf("GetUserBySessionID after deleting session returned %v, expected: %q", err, petfind.ErrNotFound)
	}
}

func TestCreateUserSession_forUnknownUser(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	// Test that we cannot create a session for an unknown user.
	err := s.CreateUserSession(&petfind.Session{ID: "foo", UserID: 1})
	if err == nil {
		t.Fatalf("CreateUserSession for unknown user returned %v, expected err", err)
	}
}
