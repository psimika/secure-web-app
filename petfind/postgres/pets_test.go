// +build db

package postgres_test

import (
	"fmt"
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
		Notes:   "notes",
		Contact: "123",
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
			Place:   place,
			Notes:   "notes",
			Contact: "123",
		},
	}
	if got := pets; !reflect.DeepEqual(got, want) {
		t.Fatalf("GetAllPets \nhave: %#v\nwant: %#v", got, want)
	}
}

var searchPetsTests = []struct {
	s petfind.Search
}{
	{
		petfind.Search{
			// #0 - 0000
			PlaceKey: "key",
			Age:      petfind.Baby,
			Gender:   petfind.Female,
			Size:     petfind.Small,
			Type:     petfind.Cat,
		},
	},
	{
		petfind.Search{
			// 0001 - #1
			PlaceKey: "key",
			Age:      petfind.Baby,
			Gender:   petfind.Female,
			Size:     petfind.Small,
			Type:     petfind.Cat,
			UseType:  true,
		},
	},
	{
		petfind.Search{
			// 0010 - #2
			PlaceKey: "key",
			Age:      petfind.Baby,
			Gender:   petfind.Female,
			Size:     petfind.Small,
			Type:     petfind.Cat,
			UseSize:  true,
		},
	},
	{
		petfind.Search{
			// 0011 - #3
			PlaceKey: "key",
			Age:      petfind.Baby,
			Gender:   petfind.Female,
			Size:     petfind.Small,
			Type:     petfind.Cat,
			UseSize:  true,
			UseType:  true,
		},
	},
	{
		petfind.Search{
			// 0100 - #4
			PlaceKey:  "key",
			Age:       petfind.Baby,
			Gender:    petfind.Female,
			Size:      petfind.Small,
			Type:      petfind.Cat,
			UseGender: true,
		},
	},
	{
		petfind.Search{
			// 0101 - #5
			PlaceKey:  "key",
			Age:       petfind.Baby,
			Gender:    petfind.Female,
			Size:      petfind.Small,
			Type:      petfind.Cat,
			UseGender: true,
			UseType:   true,
		},
	},
	{
		petfind.Search{
			// 0110 - #6
			PlaceKey:  "key",
			Age:       petfind.Baby,
			Gender:    petfind.Female,
			Size:      petfind.Small,
			Type:      petfind.Cat,
			UseGender: true,
			UseSize:   true,
		},
	},
	{
		petfind.Search{
			// 0111 - #7
			PlaceKey:  "key",
			Age:       petfind.Baby,
			Gender:    petfind.Female,
			Size:      petfind.Small,
			Type:      petfind.Cat,
			UseGender: true,
			UseSize:   true,
			UseType:   true,
		},
	},
	{
		petfind.Search{
			// 1000 - #8
			PlaceKey: "key",
			Age:      petfind.Baby,
			Gender:   petfind.Female,
			Size:     petfind.Small,
			Type:     petfind.Cat,
			UseAge:   true,
		},
	},
	{
		petfind.Search{
			// 1001 - #9
			PlaceKey: "key",
			Age:      petfind.Baby,
			Gender:   petfind.Female,
			Size:     petfind.Small,
			Type:     petfind.Cat,
			UseAge:   true,
			UseType:  true,
		},
	},
	{
		petfind.Search{
			// 1010 - #10
			PlaceKey: "key",
			Age:      petfind.Baby,
			Gender:   petfind.Female,
			Size:     petfind.Small,
			Type:     petfind.Cat,
			UseAge:   true,
			UseSize:  true,
		},
	},
	{
		petfind.Search{
			// 1011 - #11
			PlaceKey: "key",
			Age:      petfind.Baby,
			Gender:   petfind.Female,
			Size:     petfind.Small,
			Type:     petfind.Cat,
			UseAge:   true,
			UseSize:  true,
			UseType:  true,
		},
	},
	{
		petfind.Search{
			// 1100 - #12
			PlaceKey:  "key",
			Age:       petfind.Baby,
			Gender:    petfind.Female,
			Size:      petfind.Small,
			Type:      petfind.Cat,
			UseAge:    true,
			UseGender: true,
		},
	},
	{
		petfind.Search{
			// 1101 - #13
			PlaceKey:  "key",
			Age:       petfind.Baby,
			Gender:    petfind.Female,
			Size:      petfind.Small,
			Type:      petfind.Cat,
			UseAge:    true,
			UseGender: true,
			UseType:   true,
		},
	},
	{
		petfind.Search{
			// 1110 - #14
			PlaceKey:  "key",
			Age:       petfind.Baby,
			Gender:    petfind.Female,
			Size:      petfind.Small,
			Type:      petfind.Cat,
			UseAge:    true,
			UseGender: true,
			UseSize:   true,
		},
	},
	{
		petfind.Search{
			// 1111 - #15
			PlaceKey:  "key",
			Age:       petfind.Baby,
			Gender:    petfind.Female,
			Size:      petfind.Small,
			Type:      petfind.Cat,
			UseAge:    true,
			UseGender: true,
			UseSize:   true,
			UseType:   true,
		},
	},
}

func TestSearchPets(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)
	p := addTestPet(t, s)

	want := []*petfind.Pet{p}
	for i, tt := range searchPetsTests {
		got, err := s.SearchPets(tt.s)
		if err != nil {
			t.Fatalf("SearchPets failed: %v", err)
		}
		if !reflect.DeepEqual(got, want) {
			fmt.Printf("have: %#v\n", got[0])
			fmt.Printf("want: %#v\n", want[0])
			t.Fatalf("GetAllPets #%d \nhave: %#v\nwant: %#v", i, got, want)
		}
	}
}

func TestCountPets(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)
	addTestPet(t, s)

	got, err := s.CountPets()
	if err != nil {
		t.Fatalf("CountPets failed: %v", err)
	}
	if want := int64(1); got != want {
		t.Fatalf("CountPets got: %d, want: %d", got, want)
	}
}

func TestGetFeaturedPets(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)
	// Add a pet here.
	addTestPet(t, s)

	// Add 3 more pets.
	petsToAdd := []*petfind.Pet{
		{Name: "zazzles1", OwnerID: 1, PhotoID: 1, PlaceID: 1},
		{Name: "zazzles2", OwnerID: 1, PhotoID: 1, PlaceID: 1},
		{Name: "zazzles3", OwnerID: 1, PhotoID: 1, PlaceID: 1},
	}
	for _, p := range petsToAdd {
		if err := s.AddPet(p); err != nil {
			t.Fatalf("AddPet failed: %v", err)
		}
	}

	pets, err := s.GetFeaturedPets()
	if err != nil {
		t.Fatalf("GetFeaturedPets failed: %v", err)
	}
	if got, want := len(pets), 3; got != want {
		t.Fatalf("GetFeaturedPets got %d results, want %d", got, want)
	}
	if got, want := pets[0].Name, "zazzles3"; got != want {
		t.Fatalf("GetFeaturedPets first result Name = %v, want %v", got, want)
	}
	if got, want := pets[1].Name, "zazzles2"; got != want {
		t.Fatalf("GetFeaturedPets second result Name = %v, want %v", got, want)
	}
	if got, want := pets[2].Name, "zazzles1"; got != want {
		t.Fatalf("GetFeaturedPets third result Name = %v, want %v", got, want)
	}
}

func addTestPet(t *testing.T, s petfind.Store) *petfind.Pet {
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
		Name:    "zazzles",
		Age:     petfind.Baby,
		Size:    petfind.Small,
		Type:    petfind.Cat,
		Gender:  petfind.Female,
		OwnerID: owner.ID,
		PhotoID: photo.ID,
		PlaceID: place.ID,
	}
	if err := s.AddPet(p); err != nil {
		t.Fatalf("AddPet failed: %v", err)
	}
	pet, err := s.GetPet(1)
	if err != nil {
		t.Fatalf("GetPet failed: %v", err)
	}
	return pet
}
