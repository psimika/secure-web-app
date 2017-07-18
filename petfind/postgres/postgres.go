package postgres

import (
	"database/sql"
	"fmt"

	"github.com/psimika/secure-web-app/petfind"
)

type store struct {
	*sql.DB
}

// NewStore opens connection with Postgres for a given datasource and returns a
// petfind.Store implementation.
func NewStore(datasource string) (petfind.Store, error) {
	db, err := sql.Open("postgres", datasource)
	if err != nil {
		return nil, fmt.Errorf("error connecting to postgres: %v", err)
	}

	s := &store{db}
	if err := s.MakeSchema(); err != nil {
		return nil, fmt.Errorf("error making schema: %v", err)
	}
	return s, nil
}

func (db *store) MakeSchema() error {
	if _, err := db.Exec("CREATE TABLE IF NOT EXISTS pets (id serial PRIMARY KEY, name varchar(50), added timestamp)"); err != nil {
		return fmt.Errorf("error creating table pets: %v", err)
	}
	return nil
}

func (db *store) DropSchema() error {
	if _, err := db.Exec("DROP TABLE pets"); err != nil {
		return fmt.Errorf("error dropping table pets: %v", err)
	}
	return nil
}
