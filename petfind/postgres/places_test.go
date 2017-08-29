// +build db

package postgres_test

import (
	"testing"

	"github.com/psimika/secure-web-app/petfind"
)

func TestGetPlaceGroups(t *testing.T) {
	s := setup(t)
	defer teardown(t, s)

	count, err := s.CountPlaces()
	if err != nil {
		t.Fatalf("CountPlaces failed: %v", err)
	}
	if count != 0 {
		t.Fatalf("Expected to have no places at the start of test but have: %v", count)
	}

	if err := s.AddPlaceGroups(petfind.PlaceGroups); err != nil {
		t.Fatalf("AddPlaceGroups failed: %v", err)
	}

	groups, err := s.GetPlaceGroups()
	if err != nil {
		t.Fatalf("GetPlaceGroups failed: %v", err)
	}

	a, b := groups, petfind.PlaceGroups
	for i := range a {
		if got, want := a[i].Name, b[i].Name; got != want {
			t.Fatalf("GetPlaceGroups #%d have: %#v want: %#v", i, got, want)
		}
		ap, bp := a[i].Places, b[i].Places
		for k := range ap {
			if got, want := ap[k].Name, bp[k].Name; got != want {
				t.Fatalf("GetPlaceGroups #%d.%d have: %#v want: %#v", i, k, got, want)
			}
			if got, want := ap[k].Key, bp[k].Key; got != want {
				t.Fatalf("GetPlaceGroups #%d.%d have: %#v want: %#v", i, k, got, want)
			}
		}
	}
}
