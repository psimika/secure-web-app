package postgres

import (
	"time"

	"github.com/psimika/secure-web-app/petfind"
)

func (db *store) AddPet(p *petfind.Pet) error {
	const petInsertStmt = `
	INSERT INTO pets(name, added)
	VALUES ($1, now())
	RETURNING id, added
	`
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

func (db *store) GetAllPets() ([]petfind.Pet, error) {
	const petGetAllQuery = `
	SELECT *
	FROM pets
	`
	rows, err := db.Query(petGetAllQuery)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); err == nil {
			err = cerr
			return
		}
	}()

	pets := make([]petfind.Pet, 0)
	for rows.Next() {
		var p petfind.Pet
		if err := rows.Scan(&p.ID, &p.Name, &p.Added); err != nil {
			return nil, err
		}
		pets = append(pets, p)
	}
	return pets, nil
}
