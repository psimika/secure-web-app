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
		age integer,
		type integer,
		size integer,
		gender integer,
		notes text,
		created timestamptz,
		updated timestamptz,
		owner_id bigint references users,
		photo_id bigint references photos
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
		created timestamptz,
		updated timestamptz
	)`
	if _, err := db.Exec(users); err != nil {
		return fmt.Errorf("error creating table users: %v", err)
	}
	const photos = `CREATE TABLE IF NOT EXISTS photos (
		id bigserial PRIMARY KEY,
		key varchar(70),
		original_filename varchar(70),
		content_type varchar(70),
		created timestamptz
	)`
	if _, err := db.Exec(photos); err != nil {
		return fmt.Errorf("error creating table photos: %v", err)
	}
	return nil
}

func (db *store) DropSchema() error {
	if _, err := db.Exec("DROP TABLE photos"); err != nil {
		return fmt.Errorf("error dropping table photos: %v", err)
	}
	if _, err := db.Exec("DROP TABLE users"); err != nil {
		return fmt.Errorf("error dropping table users: %v", err)
	}
	if _, err := db.Exec("DROP TABLE pets"); err != nil {
		return fmt.Errorf("error dropping table pets: %v", err)
	}
	return nil
}
