package petfind

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("item not found")

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

	CreateUser(*User) error
	GetUserByGithubID(githubID int64) (*User, error)
	GetUserBySessionID(sessionID string) (*User, error)

	CreateUserSession(*Session) error
	DeleteUserSession(sessionID string) error

	MakeSchema() error
	DropSchema() error
}

type User struct {
	ID       int64
	GithubID int64
	Login    string
	Name     string
	Email    string
	Added    time.Time
}

type Session struct {
	ID      string
	UserID  int64
	Added   time.Time
	Expires time.Time
}

// TODO(psimika): Useful article in case a custom type needs to be stored in
// the database:
//
// https://husobee.github.io/golang/database/2015/06/12/scanner-valuer.html
