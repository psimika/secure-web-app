package petfind

import (
	"errors"
	"time"
)

type PetType int64

const (
	UnknownType PetType = iota
	Cat
	Dog
)

var types = [...]string{
	"Unknown",
	"Cat",
	"Dog",
}

// String returns the English name of the pet's type ("Cat", "Dog", ...).
func (p PetType) String() string { return types[p] }

type PetSize int64

const (
	UnknownSize PetSize = iota
	Small
	Medium
	Large
	Huge
)

var sizes = [...]string{
	"Unknown",
	"Small",
	"Medium",
	"Large",
	"Huge",
}

// String returns the English name of the pet's size ("Small", "Medium", ...).
func (p PetSize) String() string { return sizes[p] }

type PetAge int64

const (
	UnknownAge PetAge = iota
	Baby
	Young
	Adult
	Senior
)

var ages = [...]string{
	"Unknown",
	"Baby",
	"Young",
	"Adult",
	"Senior",
}

// String returns the English name of the pet's age ("Baby", "Young", ...).
func (p PetAge) String() string { return ages[p] }

type PetGender int64

const (
	UnknownGender PetGender = iota
	Male
	Female
)

var genders = [...]string{
	"Unknown",
	"Male",
	"Female",
}

func (p PetGender) String() string { return genders[p] }

// Pet holds information about each pet of the application.
type Pet struct {
	ID      int64
	Name    string
	Age     PetAge
	Type    PetType
	Size    PetSize
	Gender  PetGender
	Created time.Time
	Updated time.Time
	Notes   string
	PhotoID int64
	OwnerID int64
	PlaceID int64
	Owner   *User
	Place   *Place
}

// ErrNotFound is returned whenever an item does not exist in the Store.
var ErrNotFound = errors.New("item not found")

// Store describes the operations the application needs for persisting and
// retrieving data.
type Store interface {
	AddPet(*Pet) error
	GetAllPets() ([]Pet, error)
	SearchPets(Search) ([]*Pet, error)

	CreateUser(*User) error
	GetUser(userID int64) (*User, error)
	PutGithubUser(*GithubUser) (*User, error)
	GetUserByGithubID(githubID int64) (*User, error)

	AddPhoto(*Photo) error
	GetPhoto(photoID int64) (*Photo, error)

	AddPlaceGroups([]PlaceGroup) error
	GetPlaceGroups() ([]PlaceGroup, error)
	AddPlaceGroup(*PlaceGroup) error
	AddPlace(*Place) error
	GetPlace(int64) (*Place, error)
	GetPlaceByKey(string) (*Place, error)
	CountPlaces() (int64, error)

	MakeSchema() error
	DropSchema() error
}

type Search struct {
	PlaceKey  string
	Type      int64
	Age       int64
	Gender    int64
	Size      int64
	UseType   bool
	UseAge    bool
	UseGender bool
	UseSize   bool
}

// User holds information about a user that is signed in the application.
type User struct {
	ID       int64
	GithubID int64
	Login    string
	Name     string
	Email    string
	Created  time.Time
	Updated  time.Time
}

// GithubUser holds the data that we need to retrieve from a user's GitHub
// account with their permission.
type GithubUser struct {
	ID    int64  `json:"id"`
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// TODO(psimika): Useful article in case a custom type needs to be stored in
// the database:
//
// https://husobee.github.io/golang/database/2015/06/12/scanner-valuer.html
