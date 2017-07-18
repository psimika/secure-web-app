package petfind

import "time"

// Pet holds information about each pet of the application.
type Pet struct {
	ID    int64
	Name  string
	Age   int
	Added time.Time
}

type Store interface {
	AddPet(*Pet) error
}
