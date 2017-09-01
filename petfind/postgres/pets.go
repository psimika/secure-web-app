package postgres

import (
	"database/sql"

	"github.com/psimika/secure-web-app/petfind"
)

func (db *store) CountPets() (int64, error) {
	const petCountQuery = `
	SELECT COUNT(*)
	FROM pets
	`

	var count int64
	err := db.QueryRow(petCountQuery).Scan(&count)
	if err != nil {
		return -1, err
	}
	return count, nil
}

func (db *store) AddPet(p *petfind.Pet) error {
	const petInsertStmt = `
	INSERT INTO pets(name, age, size, type, gender, contact, notes, owner_id, photo_id, place_id, created, updated)
	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, now(), now())
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
	err = stmt.QueryRow(p.Name, p.Age, p.Size, p.Type, p.Gender, p.Contact, p.Notes, p.OwnerID, p.PhotoID, p.PlaceID).Scan(&p.ID, &p.Created, &p.Updated)
	if err != nil {
		return err
	}
	return nil
}

func (db *store) GetPet(petID int64) (*petfind.Pet, error) {
	const petGetQuery = `
	SELECT *
	FROM pets p
	  JOIN users u ON p.owner_id = u.id
	  JOIN places pl ON p.place_id = pl.id
	WHERE p.id = $1
	`
	p := new(petfind.Pet)
	u := new(petfind.User)
	pl := new(petfind.Place)
	err := db.QueryRow(petGetQuery, petID).Scan(
		&p.ID,
		&p.Name,
		&p.Age,
		&p.Type,
		&p.Size,
		&p.Gender,
		&p.Contact,
		&p.Notes,
		&p.Created,
		&p.Updated,
		&p.OwnerID,
		&p.PhotoID,
		&p.PlaceID,
		&u.ID,
		&u.GithubID,
		&u.Name,
		&u.Login,
		&u.Email,
		&u.Created,
		&u.Updated,
		&pl.ID,
		&pl.Key,
		&pl.Name,
		&pl.GroupID,
	)
	if err == sql.ErrNoRows {
		return nil, petfind.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	p.Owner = u
	p.Place = pl
	return p, nil
}

func (db *store) GetFeaturedPets() ([]*petfind.Pet, error) {
	const petGetFeaturedQuery = `
	SELECT *
	FROM pets p
	  JOIN users u ON p.owner_id = u.id
	  JOIN places pl ON p.place_id = pl.id
	  ORDER by p.Created desc LIMIT 3
	`
	rows, err := db.Query(petGetFeaturedQuery)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); err == nil {
			err = cerr
			return
		}
	}()

	pets := make([]*petfind.Pet, 0, 3)
	for rows.Next() {
		p := new(petfind.Pet)
		u := new(petfind.User)
		pl := new(petfind.Place)
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Age,
			&p.Type,
			&p.Size,
			&p.Gender,
			&p.Contact,
			&p.Notes,
			&p.Created,
			&p.Updated,
			&p.OwnerID,
			&p.PhotoID,
			&p.PlaceID,
			&u.ID,
			&u.GithubID,
			&u.Name,
			&u.Login,
			&u.Email,
			&u.Created,
			&u.Updated,
			&pl.ID,
			&pl.Key,
			&pl.Name,
			&pl.GroupID,
		); err != nil {
			return nil, err
		}
		p.Owner = u
		p.Place = pl
		pets = append(pets, p)
	}
	return pets, nil
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
	  p.contact,
	  p.notes,
	  p.created,
	  p.updated,
	  p.owner_id,
	  p.photo_id,
	  p.place_id,
	  u.id,
	  u.github_id,
	  u.name,
	  u.login,
	  u.email,
	  u.created,
	  u.updated,
	  pl.id,
	  pl.key,
	  pl.name,
	  pl.group_id
	FROM pets p
	  JOIN users u ON p.owner_id = u.id
	  JOIN places pl ON p.place_id = pl.id
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
		var pl petfind.Place
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Age,
			&p.Type,
			&p.Size,
			&p.Gender,
			&p.Contact,
			&p.Notes,
			&p.Created,
			&p.Updated,
			&p.OwnerID,
			&p.PhotoID,
			&p.PlaceID,
			&u.ID,
			&u.GithubID,
			&u.Name,
			&u.Login,
			&u.Email,
			&u.Created,
			&u.Updated,
			&pl.ID,
			&pl.Key,
			&pl.Name,
			&pl.GroupID,
		); err != nil {
			return nil, err
		}
		p.Owner = &u
		p.Place = &pl
		pets = append(pets, p)
	}
	return pets, nil
}

func (db *store) SearchPets(s petfind.Search) ([]*petfind.Pet, error) {
	var q string
	var rows *sql.Rows
	var err error
	switch {
	// 0000
	default:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1`
		rows, err = db.Query(q, s.PlaceKey)
	// 0001
	case !s.UseAge && !s.UseGender && !s.UseSize && s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.type = $2`
		rows, err = db.Query(q, s.PlaceKey, s.Type)
	// 0010
	case !s.UseAge && !s.UseGender && s.UseSize && !s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.size = $2`
		rows, err = db.Query(q, s.PlaceKey, s.Size)
	// 0011
	case !s.UseAge && !s.UseGender && s.UseSize && s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.size = $2
	      AND p.type = $3`
		rows, err = db.Query(q, s.PlaceKey, s.Size, s.Type)
	// 0100
	case !s.UseAge && s.UseGender && !s.UseSize && !s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.gender = $2`
		rows, err = db.Query(q, s.PlaceKey, s.Gender)
	// 0101
	case !s.UseAge && s.UseGender && !s.UseSize && s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.gender = $2
	      AND p.type = $3`
		rows, err = db.Query(q, s.PlaceKey, s.Gender, s.Type)
	// 0110
	case !s.UseAge && s.UseGender && s.UseSize && !s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.gender = $2
	      AND p.size = $3`
		rows, err = db.Query(q, s.PlaceKey, s.Gender, s.Size)
	// 0111
	case !s.UseAge && s.UseGender && s.UseSize && s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.gender = $2
	      AND p.size = $3
		  AND p.type = $4`
		rows, err = db.Query(q, s.PlaceKey, s.Gender, s.Size, s.Type)
	// 1000
	case s.UseAge && !s.UseGender && !s.UseSize && !s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.age = $2`
		rows, err = db.Query(q, s.PlaceKey, s.Age)
	// 1001
	case s.UseAge && !s.UseGender && !s.UseSize && s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.age = $2
		  AND p.type = $3`
		rows, err = db.Query(q, s.PlaceKey, s.Age, s.Type)
	// 1010
	case s.UseAge && !s.UseGender && s.UseSize && !s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.age = $2
		  AND p.size = $3`
		rows, err = db.Query(q, s.PlaceKey, s.Age, s.Size)
	// 1011
	case s.UseAge && !s.UseGender && s.UseSize && s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.age = $2
		  AND p.size = $3
		  AND p.type = $4`
		rows, err = db.Query(q, s.PlaceKey, s.Age, s.Size, s.Type)
	// 1100
	case s.UseAge && s.UseGender && !s.UseSize && !s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.age = $2
		  AND p.gender = $3`
		rows, err = db.Query(q, s.PlaceKey, s.Age, s.Gender)
	// 1101
	case s.UseAge && s.UseGender && !s.UseSize && s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.age = $2
		  AND p.gender = $3
		  AND p.type = $4`
		rows, err = db.Query(q, s.PlaceKey, s.Age, s.Gender, s.Type)
	// 1110
	case s.UseAge && s.UseGender && s.UseSize && !s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.age = $2
		  AND p.gender = $3
		  AND p.size = $4`
		rows, err = db.Query(q, s.PlaceKey, s.Age, s.Gender, s.Size)
	// 1111
	case s.UseAge && s.UseGender && s.UseSize && s.UseType:
		q = `SELECT *
	    FROM pets p
	      JOIN users u ON p.owner_id = u.id
	      JOIN places pl ON p.place_id = pl.id
	      WHERE pl.key = $1
	      AND p.age = $2
		  AND p.gender = $3
		  AND p.size = $4
		  AND p.type = $5`
		rows, err = db.Query(q, s.PlaceKey, s.Age, s.Gender, s.Size, s.Type)
	}
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); err == nil {
			err = cerr
			return
		}
	}()

	pets := make([]*petfind.Pet, 0)
	for rows.Next() {
		var p petfind.Pet
		var u petfind.User
		var pl petfind.Place
		if err := rows.Scan(
			&p.ID,
			&p.Name,
			&p.Age,
			&p.Type,
			&p.Size,
			&p.Gender,
			&p.Contact,
			&p.Notes,
			&p.Created,
			&p.Updated,
			&p.OwnerID,
			&p.PhotoID,
			&p.PlaceID,
			&u.ID,
			&u.GithubID,
			&u.Name,
			&u.Login,
			&u.Email,
			&u.Created,
			&u.Updated,
			&pl.ID,
			&pl.Key,
			&pl.Name,
			&pl.GroupID,
		); err != nil {
			return nil, err
		}
		p.Owner = &u
		p.Place = &pl
		pets = append(pets, &p)
	}
	return pets, nil
}
