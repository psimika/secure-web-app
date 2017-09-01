package postgres

import (
	"database/sql"
	"fmt"

	"github.com/psimika/secure-web-app/petfind"
)

func (db *store) CreateUser(u *petfind.User) error {
	const userInsertStmt = `
	INSERT INTO users(github_id, login, name, email, created, updated)
	VALUES ($1, $2, $3, $4, now(), now())
	RETURNING id, created, updated
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
	err = stmt.QueryRow(u.GithubID, u.Login, u.Name, u.Email).Scan(&u.ID, &u.Created, &u.Updated)
	if err != nil {
		return err
	}
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
	  created,
	  updated
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
		&u.Updated,
	)
	if err == sql.ErrNoRows {
		return nil, petfind.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *store) PutGithubUser(ghu *petfind.GithubUser) (u *petfind.User, err error) {
	// A GitHub user might not have provided their name in their profile but
	// every GitHub user has a login. So in the case they haven't provided a
	// name we will use their login as a name instead.
	if ghu.Name == "" {
		ghu.Name = ghu.Login
	}
	const (
		userUpdateStmt = `
	UPDATE users SET
	  login = $2,
	  name = $3,
	  email = $4,
	  updated = now()
	WHERE github_id = $1
	RETURNING id, github_id, login, name, email, created, updated
	`
		userInsertStmt = `
	INSERT INTO users(github_id, login, name, email, created, updated)
	VALUES ($1, $2, $3, $4, now(), now())
	RETURNING id, github_id, login, name, email, created, updated
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

	u = new(petfind.User)
	err = tx.QueryRow(userUpdateStmt, ghu.ID, ghu.Login, ghu.Name, ghu.Email).
		Scan(&u.ID, &u.GithubID, &u.Login, &u.Name, &u.Email, &u.Created, &u.Updated)
	if err == sql.ErrNoRows {
		err = tx.QueryRow(userInsertStmt, ghu.ID, ghu.Login, ghu.Name, ghu.Email).
			Scan(&u.ID, &u.GithubID, &u.Login, &u.Name, &u.Email, &u.Created, &u.Updated)
		if err != nil {
			return nil, err
		}
		return u, nil
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

func (db *store) PutFacebookUser(fbu *petfind.FacebookUser) (u *petfind.User, err error) {
	const (
		userUpdateStmt = `
	UPDATE users SET
	  name = $2,
	  updated = now()
	WHERE facebook_id = $1
	RETURNING id, github_id, facebook_id, login, name, email, created, updated
	`
		userInsertStmt = `
	INSERT INTO users(facebook_id, name, created, updated)
	VALUES ($1, $2, $3, $4, $5, now(), now())
	RETURNING id, github_id, facebook_id, login, name, email, created, updated
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

	u = new(petfind.User)
	err = tx.QueryRow(userUpdateStmt, fbu.ID, fbu.Name).
		Scan(&u.ID, &u.GithubID, &u.FacebookID, &u.Login, &u.Name, &u.Email, &u.Created, &u.Updated)
	if err == sql.ErrNoRows {
		err = tx.QueryRow(userInsertStmt, fbu.ID, fbu.Name).
			Scan(&u.ID, &u.GithubID, &u.FacebookID, &u.Login, &u.Name, &u.Email, &u.Created, &u.Updated)
		if err != nil {
			return nil, err
		}
		return u, nil
	}
	if err != nil {
		return nil, err
	}
	return u, nil
}

func (db *store) GetUserByFacebookID(facebookID int64) (*petfind.User, error) {
	const userGetByFacebookIDQuery = `
	SELECT
	  id,
	  github_id,
	  facebook_id,
	  login,
	  name,
	  email,
	  created
	FROM users
	WHERE facebook_id = $1
	`
	u := new(petfind.User)
	err := db.QueryRow(userGetByFacebookIDQuery, facebookID).Scan(
		&u.ID,
		&u.GithubID,
		&u.FacebookID,
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
