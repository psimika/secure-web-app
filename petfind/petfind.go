package petfind

import "time"

// Pet holds information about each pet of the application.
type Pet struct {
	Name    string
	Age     int
	Created time.Time
}
