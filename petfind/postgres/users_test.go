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
	user.Created = time.Time{}

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

func TestPutGithubUser(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	// We Put the GitHub user for the first time. The user does not exist so we
	// expect Put to create the user.
	githubID := int64(5)
	got, err := s.PutGithubUser(githubID, "janedoe", "Jane Doe", "jane@doe.com")
	if err != nil {
		t.Fatal("PutGithubUser for non existent user returned err:", err)
	}

	// Save created time to check it was the same when we Put for a second time
	// below.
	created := got.Created
	// Ignore time values.
	got.Created = time.Time{}
	got.Updated = time.Time{}

	want := &petfind.User{
		ID:       1, // A newly created user should get ID 1 from Postgres.
		GithubID: githubID,
		Login:    "janedoe",
		Name:     "Jane Doe",
		Email:    "jane@doe.com",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PutGithubUser first run \nhave: %#v\nwant: %#v", got, want)
	}

	// Attempt to Put again the github User. The GitHub user should have been
	// already been created from the previous run and we now expect the values
	// to be updated.
	got, err = s.PutGithubUser(githubID, "jane", "Jane", "jane@doe.com")
	if err != nil {
		t.Fatal("PutGithubUser for existing user returned err:", err)
	}

	// Ignore updated.
	got.Updated = time.Time{}

	want = &petfind.User{
		ID:       1, // ID stays the same as we are doing an update.
		GithubID: githubID,
		Login:    "jane",
		Name:     "Jane",
		Email:    "jane@doe.com",
		Created:  created,
	}
	// This time we expect the values to be updated but the created time should
	// be the same as the first run.
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PutGithubUser second run \nhave: %#v\nwant: %#v", got, want)
	}
}
