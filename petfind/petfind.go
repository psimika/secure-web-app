package petfind

import "time"

// Pet holds information about each pet of the application.
type Pet struct {
	ID    int64
	Name  string
	Age   int
	Added time.Time
}

// Store describes the operations the application needs for persisting and
// retrieving data.
type Store interface {
	AddPet(*Pet) error
	GetAllPets() ([]Pet, error)

	MakeSchema() error
	DropSchema() error
}
