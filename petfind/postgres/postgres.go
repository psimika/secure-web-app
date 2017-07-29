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
	const pets = `CREATE TABLE IF NOT EXISTS pets (
		id bigserial PRIMARY KEY,
		name varchar(70),
		added timestamp
	)`
	if _, err := db.Exec(pets); err != nil {
		return fmt.Errorf("error creating table pets: %v", err)
	}
	const users = `CREATE TABLE IF NOT EXISTS users (
		id bigserial PRIMARY KEY,
		github_id bigint,
		name varchar(70),
		login varchar(70),
		email varchar(70),
		added timestamp
	)`
	if _, err := db.Exec(users); err != nil {
		return fmt.Errorf("error creating table users: %v", err)
	}
	const sessions = `CREATE TABLE IF NOT EXISTS sessions (
		id varchar(50) PRIMARY KEY,
		user_id int REFERENCES users (id),
		added timestamp,
		expires timestamp
	)`
	if _, err := db.Exec(sessions); err != nil {
		return fmt.Errorf("error creating table sessions: %v", err)
	}
	return nil
}

func (db *store) DropSchema() error {
	if _, err := db.Exec("DROP TABLE sessions"); err != nil {
		return fmt.Errorf("error dropping table sessions: %v", err)
	}
	if _, err := db.Exec("DROP TABLE users"); err != nil {
		return fmt.Errorf("error dropping table users: %v", err)
	}
	if _, err := db.Exec("DROP TABLE pets"); err != nil {
		return fmt.Errorf("error dropping table pets: %v", err)
	}
	return nil
}
