package postgres

import (
	"database/sql"
	"fmt"
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

func (db *store) PutGithubUser(githubID int64, login, name, email string) (user *petfind.User, err error) {
	// A GitHub user might not have provided their name in their profile but
	// every GitHub user has a login. So in the case they haven't provided a
	// name we will use their login as a name instead.
	if name == "" {
		name = login
	}
	user = &petfind.User{
		GithubID: githubID,
		Login:    login,
		Name:     name,
		Email:    email,
	}
	const (
		userUpdateStmt = `
	UPDATE users SET
	  login = $2,
	  name = $3,
	  email = $4,
	  updated = now()
	WHERE github_id = $1
	RETURNING id, created, updated
	`
		userInsertStmt = `
	INSERT INTO users(github_id, login, name, email, created, updated)
	VALUES ($1, $2, $3, $4, now(), now())
	RETURNING id, created, updated
	`
	)

	tx, err := db.Begin()
	if err != nil {
		return nil, err
	}
	defer func() {
		if err != nil {
			if rerr := tx.Rollback(); rerr != nil {
				err = fmt.Errorf("rollback failed: %v: %v", rerr, err)
			}
			return
		}
		err = tx.Commit()
	}()

	err = tx.QueryRow(userUpdateStmt, githubID, login, name, email).Scan(&user.ID, &user.Created, &user.Updated)
	if err == sql.ErrNoRows {
		err = tx.QueryRow(userInsertStmt, githubID, login, name, email).Scan(&user.ID, &user.Created, &user.Updated)
		if err != nil {
			return nil, err
		}
		return user, nil
	}
	if err != nil {
		return nil, err
	}
	return user, nil
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
