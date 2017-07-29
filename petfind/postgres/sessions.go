package postgres

import (
	"github.com/psimika/secure-web-app/petfind"
)

func (db *store) CreateUserSession(s *petfind.Session) error {
	const sessionInsertStmt = `
	INSERT INTO sessions(id, user_id, added, expires)
	VALUES ($1, $2, $3, $4)
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
	_, err = stmt.Exec(s.ID, s.UserID, s.Added, s.Expires)
	if err != nil {
		return err
	}
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
