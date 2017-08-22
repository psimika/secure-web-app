package postgres

import (
	"github.com/psimika/secure-web-app/petfind"
)

func (db *store) AddPet(p *petfind.Pet) error {
	const petInsertStmt = `
	INSERT INTO pets(name, age, size, type, created, updated)
	VALUES ($1, $2, $3, $4, now(), now())
	RETURNING id, created, updated
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
	err = stmt.QueryRow(p.Name, p.Age, p.Size, p.Type).Scan(&p.ID, &p.Created, &p.Updated)
	if err != nil {
		return err
	}
	return nil
}

func (db *store) GetAllPets() ([]petfind.Pet, error) {
	const petGetAllQuery = `
	SELECT
	  id,
	  name,
	  age,
	  type,
	  size,
	  created,
	  updated
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
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Age,
			&p.Type,
			&p.Size,
			&p.Created,
			&p.Updated,
		); err != nil {
			return nil, err
		}
		pets = append(pets, p)
	}
	return pets, nil
}
