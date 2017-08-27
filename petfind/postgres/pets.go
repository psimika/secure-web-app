package postgres

import (
	"github.com/psimika/secure-web-app/petfind"
)

func (db *store) AddPet(p *petfind.Pet) error {
	const petInsertStmt = `
	INSERT INTO pets(name, age, size, type, gender, notes, owner_id, photo_id, created, updated)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, now(), now())
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
	err = stmt.QueryRow(p.Name, p.Age, p.Size, p.Type, p.Gender, p.Notes, p.OwnerID, p.PhotoID).Scan(&p.ID, &p.Created, &p.Updated)
	if err != nil {
		return err
	}
	return nil
}

func (db *store) GetAllPets() ([]petfind.Pet, error) {
	const petGetAllQuery = `
	SELECT
	  p.id,
	  p.name,
	  p.age,
	  p.type,
	  p.size,
	  p.gender,
	  p.notes,
	  p.created,
	  p.updated,
	  p.owner_id,
	  p.photo_id,
	  u.id,
	  u.github_id,
	  u.login,
	  u.name,
	  u.email,
	  u.created
	FROM pets p
	  JOIN users u ON p.owner_id = u.id
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
		var u petfind.User
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Age,
			&p.Type,
			&p.Size,
			&p.Gender,
			&p.Notes,
			&p.Created,
			&p.Updated,
			&p.OwnerID,
			&p.PhotoID,
			&u.ID,
			&u.GithubID,
			&u.Login,
			&u.Name,
			&u.Email,
			&u.Created,
		); err != nil {
			return nil, err
		}
		p.Owner = &u
		pets = append(pets, p)
	}
	return pets, nil
}
