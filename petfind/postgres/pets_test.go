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

	p := &petfind.Pet{Name: "blinky"}
	if err := s.AddPet(p); err != nil {
		t.Fatalf("AddPet failed: %v", err)
	}

	pets, err := s.GetAllPets()
	if err != nil {
		t.Fatalf("GetAllPets failed: %v", err)
	}

	// Ignore time from results.
	for i := range pets {
		pets[i].Added = time.Time{}
	}
	want := []petfind.Pet{
		{ID: 1, Name: "blinky"},
	}
	if got := pets; !reflect.DeepEqual(got, want) {
		t.Fatalf("GetAllPets \nhave: %#v\nwant: %#v", got, want)
	}
}
