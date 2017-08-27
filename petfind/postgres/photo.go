package postgres

import (
	"database/sql"

	"github.com/psimika/secure-web-app/petfind"
)

func (db *store) AddPhoto(p *petfind.Photo) error {
	const photoInsertStmt = `
	INSERT INTO photos(key, original_filename, content_type, created)
	VALUES ($1, $2, $3, now())
	RETURNING id, created
	`
	stmt, err := db.Prepare(photoInsertStmt)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := stmt.Close(); err == nil {
			err = cerr
			return
		}
	}()
	err = stmt.QueryRow(p.Key, p.OriginalFilename, p.ContentType).Scan(&p.ID, &p.Created)
	if err != nil {
		return err
	}
	return nil
}

func (db *store) GetPhoto(photoID int64) (*petfind.Photo, error) {
	const photoGetQuery = `
	SELECT
	  id,
	  key,
	  original_filename,
	  content_type,
	  created
	FROM photos
	WHERE id = $1
	`
	p := new(petfind.Photo)
	err := db.QueryRow(photoGetQuery, photoID).Scan(
		&p.ID,
		&p.Key,
		&p.OriginalFilename,
		&p.ContentType,
		&p.Created,
	)
	if err == sql.ErrNoRows {
		return nil, petfind.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}
