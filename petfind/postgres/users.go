package postgres

import (
	"database/sql"
	"time"

	"github.com/psimika/secure-web-app/petfind"
)

func (db *store) CreateUser(u *petfind.User) error {
	const userInsertStmt = `
	INSERT INTO users(github_id, login, name, email, added)
	VALUES ($1, $2, $3, $4, now())
	RETURNING id, added
	`
	stmt, err := db.Prepare(userInsertStmt)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := stmt.Close(); err == nil {
			err = cerr
			return
		}
	}()
	var id int64
	var added time.Time
	err = stmt.QueryRow(u.GithubID, u.Login, u.Name, u.Email).Scan(&id, &added)
	if err != nil {
		return err
	}
	u.ID = id
	u.Added = added
	return nil
}

func (db *store) GetUserByGithubID(userID int64) (*petfind.User, error) {
	const userGetByGithubIDQuery = `
	SELECT
	  id,
	  github_id,
	  login,
	  name,
	  email,
	  added
	FROM users
	WHERE github_id = $1
	`
	u := new(petfind.User)
	err := db.QueryRow(userGetByGithubIDQuery, userID).Scan(
		&u.ID,
		&u.GithubID,
		&u.Login,
		&u.Name,
		&u.Email,
		&u.Added,
	)
	if err == sql.ErrNoRows {
		return nil, petfind.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *store) GetUserBySessionID(sessionID string) (*petfind.User, error) {
	const userGetBySessionIDQuery = `
	SELECT
	  u.id,
	  u.github_id,
	  u.login,
	  u.name,
	  u.email,
	  u.added
	FROM sessions s
	  JOIN users u ON s.user_id = u.id
	WHERE s.id = $1
	`
	u := new(petfind.User)
	err := db.QueryRow(userGetBySessionIDQuery, sessionID).Scan(
		&u.ID,
		&u.GithubID,
		&u.Login,
		&u.Name,
		&u.Email,
		&u.Added,
	)
	if err == sql.ErrNoRows {
		return nil, petfind.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}
