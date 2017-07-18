package postgres

import (
	"time"

	"github.com/psimika/secure-web-app/petfind"
)

func (db *store) AddPet(p *petfind.Pet) error {
	const petInsertStmt = `
	INSERT INTO pets(name, added)
	VALUES ($1, now())
	RETURNING id, added`

	stmt, err := db.Prepare(petInsertStmt)
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
	err = stmt.QueryRow(p.Name).Scan(&id, &added)
	if err != nil {
		return err
	}
	p.ID = id
	p.Added = added
	return nil
}
