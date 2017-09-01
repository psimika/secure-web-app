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

	// Test get user by their GitHub ID.
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

	// Test get user by their ID.
	user, err = s.GetUser(1)
	if err != nil {
		t.Fatalf("GetUser failed: %v", err)
	}

	// Ignore time field.
	user.Created = time.Time{}
	user.Updated = time.Time{}
	want = &petfind.User{ID: 1, Name: "Jane Doe", GithubID: githubID}
	if got := user; !reflect.DeepEqual(got, want) {
		t.Fatalf("GetUser \nhave: %#v\nwant: %#v", got, want)
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

func TestGetUser_notFound(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	_, err := s.GetUser(0)
	if err != petfind.ErrNotFound {
		t.Fatalf("GetUser for unknown ID returned %v, expected: %q", err, petfind.ErrNotFound)
	}
}

func TestPutGithubUser(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	// We Put the GitHub user for the first time. The user does not exist so we
	// expect Put to create the user.
	ghu := &petfind.GithubUser{
		ID:    5,
		Login: "janedoe",
		Name:  "Jane Doe",
		Email: "jane@doe.com",
	}
	got, err := s.PutGithubUser(ghu)
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
		GithubID: 5,
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
	ghu = &petfind.GithubUser{
		ID:    5,
		Login: "jane", // changed
		Name:  "Jane", // changed
		Email: "jane@doe.com",
	}
	got, err = s.PutGithubUser(ghu)
	if err != nil {
		t.Fatal("PutGithubUser for existing user returned err:", err)
	}

	// Ignore updated.
	got.Updated = time.Time{}

	want = &petfind.User{
		ID:       1, // ID stays the same as we are doing an update.
		GithubID: 5,
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

// When a github user doesn't have their name filled in their profile, we
// use login instead.
func TestPutGithubUser_emptyName(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	githubUser := &petfind.GithubUser{
		ID:    5,
		Login: "janedoe",
		Name:  "",
		Email: "",
	}
	u, err := s.PutGithubUser(githubUser)
	if err != nil {
		t.Fatal("PutGithubUser with empty name returned err:", err)
	}

	// Check that login was used as a Name.
	if got, want := u.Name, "janedoe"; got != want {
		t.Errorf("PutGithubUser with empty name -> user.Name=%q, expected %q", got, want)
	}
}
func TestPutLinkedinbUser(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	// We Put the LinkedIn user for the first time. The user does not exist so we
	// expect Put to create the user.
	llu := &petfind.LinkedinUser{
		ID:        "JANEDOE",
		FirstName: "Jane",
		LastName:  "Doe",
	}
	got, err := s.PutLinkedinUser(llu)
	if err != nil {
		t.Fatal("PutLinkedinUser for non existent user returned err:", err)
	}

	// Save created time to check it was the same when we Put for a second time
	// below.
	created := got.Created
	// Ignore time values.
	got.Created = time.Time{}
	got.Updated = time.Time{}

	want := &petfind.User{
		ID:         1, // A newly created user should get ID 1 from Postgres.
		LinkedinID: "JANEDOE",
		Name:       "Jane Doe",
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PutLinkedinUser first run \nhave: %#v\nwant: %#v", got, want)
	}

	// Attempt to Put again the github User. The LinkedIn user should have been
	// already been created from the previous run and we now expect the values
	// to be updated.
	llu = &petfind.LinkedinUser{
		ID:        "JANEDOE",
		FirstName: "Michaella", // changed
		LastName:  "Neirou",    // changed
	}
	got, err = s.PutLinkedinUser(llu)
	if err != nil {
		t.Fatal("PutLinkedinUser for existing user returned err:", err)
	}

	// Ignore updated.
	got.Updated = time.Time{}

	want = &petfind.User{
		ID:         1, // ID stays the same as we are doing an update.
		LinkedinID: "JANEDOE",
		Name:       "Michaella Neirou",
		Created:    created,
	}
	// This time we expect the values to be updated but the created time should
	// be the same as the first run.
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("PutLinkedinUser second run \nhave: %#v\nwant: %#v", got, want)
	}
}
