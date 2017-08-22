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

	p := &petfind.Pet{Name: "blinky", Age: petfind.Adult, Size: petfind.Small, Type: petfind.Dog}
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
		{ID: 1, Name: "blinky", Age: petfind.Adult, Size: petfind.Small, Type: petfind.Dog},
	}
	if got := pets; !reflect.DeepEqual(got, want) {
		t.Fatalf("GetAllPets \nhave: %#v\nwant: %#v", got, want)
	}
}
