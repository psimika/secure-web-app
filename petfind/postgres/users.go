package postgres

import (
	"database/sql"
	"time"

	"github.com/psimika/secure-web-app/petfind"
)

func (db *store) CreateUser(u *petfind.User) error {
	const userInsertStmt = `
	INSERT INTO users(github_id, login, name, email, created)
	VALUES ($1, $2, $3, $4, now())
	RETURNING id, created
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
	var created time.Time
	err = stmt.QueryRow(u.GithubID, u.Login, u.Name, u.Email).Scan(&id, &created)
	if err != nil {
		return err
	}
	u.ID = id
	u.Created = created
	return nil
}

func (db *store) GetUser(userID int64) (*petfind.User, error) {
	const userGetQuery = `
	SELECT
	  id,
	  github_id,
	  login,
	  name,
	  email,
	  created
	FROM users
	WHERE id = $1
	`
	u := new(petfind.User)
	err := db.QueryRow(userGetQuery, userID).Scan(
		&u.ID,
		&u.GithubID,
		&u.Login,
		&u.Name,
		&u.Email,
		&u.Created,
	)
	if err == sql.ErrNoRows {
		return nil, petfind.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *store) GetUserByGithubID(userID int64) (*petfind.User, error) {
	const userGetByGithubIDQuery = `
	SELECT
	  id,
	  github_id,
	  login,
	  name,
	  email,
	  created
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
		&u.Created,
	)
	if err == sql.ErrNoRows {
		return nil, petfind.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}
