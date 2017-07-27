package postgres

import (
	"time"

	"github.com/psimika/secure-web-app/petfind"
)

func (db *store) CreateUserSession(s *petfind.Session) error {
	const sessionInsertStmt = `
	INSERT INTO sessions(id, user_id, added)
	VALUES ($1, $2, now())
	RETURNING added
	`
	stmt, err := db.Prepare(sessionInsertStmt)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := stmt.Close(); err == nil {
			err = cerr
			return
		}
	}()
	var added time.Time
	err = stmt.QueryRow(s.ID, s.UserID).Scan(&added)
	if err != nil {
		return err
	}
	s.Added = added
	return nil
}

func (db *store) DeleteUserSession(sessionID string) error {
	const sessionDeleteStmt = `DELETE FROM sessions WHERE id=$1`
	stmt, err := db.Prepare(sessionDeleteStmt)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := stmt.Close(); err == nil {
			err = cerr
			return
		}
	}()

	if _, err := stmt.Exec(sessionID); err != nil {
		return err
	}
	return nil
}
