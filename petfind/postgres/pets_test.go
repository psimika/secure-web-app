// +build db

package postgres_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/psimika/secure-web-app/petfind"
)

func TestAddPet(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	// Create pet's owner.
	githubID := int64(5)
	owner := &petfind.User{Name: "Jane Doe", GithubID: githubID}
	if err := s.CreateUser(owner); err != nil {
		t.Fatalf("CreateUser failed: %v", err)
	}
	// Create pet's photo.
	photo := &petfind.Photo{}
	if err := s.AddPhoto(photo); err != nil {
		t.Fatalf("AddPhoto failed: %v", err)
	}
	// Create group.
	group := &petfind.PlaceGroup{Name: "group"}
	if err := s.AddPlaceGroup(group); err != nil {
		t.Fatalf("AddPlaceGroup failed: %v", err)
	}
	place := &petfind.Place{Name: "place", Key: "key", GroupID: group.ID}
	if err := s.AddPlace(place); err != nil {
		t.Fatalf("AddPlace failed: %v", err)
	}

	p := &petfind.Pet{
		Name:    "blinky",
		Age:     petfind.Adult,
		Size:    petfind.Small,
		Type:    petfind.Dog,
		OwnerID: owner.ID,
		PhotoID: photo.ID,
		PlaceID: place.ID,
	}
	if err := s.AddPet(p); err != nil {
		t.Fatalf("AddPet failed: %v", err)
	}

	pets, err := s.GetAllPets()
	if err != nil {
		t.Fatalf("GetAllPets failed: %v", err)
	}

	// Ignore time from results.
	for i := range pets {
		pets[i].Created = time.Time{}
		pets[i].Updated = time.Time{}
	}
	want := []petfind.Pet{
		{
			ID:      1,
			Name:    "blinky",
			Age:     petfind.Adult,
			Size:    petfind.Small,
			Type:    petfind.Dog,
			OwnerID: 1,
			PhotoID: 1,
			PlaceID: 1,
			Owner:   owner,
		},
	}
	if got := pets; !reflect.DeepEqual(got, want) {
		t.Fatalf("GetAllPets \nhave: %#v\nwant: %#v", got, want)
	}
}
