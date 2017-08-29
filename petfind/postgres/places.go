package postgres

import (
	"database/sql"
	"fmt"

	"github.com/psimika/secure-web-app/petfind"
)

func (db *store) AddPlaceGroups(groups []petfind.PlaceGroup) error {
	const (
		placeGroupInsertStmt = `
	INSERT INTO place_groups(name)
	VALUES ($1)
	RETURNING id
	`
		placeInsertStmt = `
	INSERT INTO places(key, name, group_id)
	VALUES ($1, $2, $3)
	RETURNING id
	`
	)

	tx, err := db.Begin()
	if err != nil {
		return err
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

	for _, g := range groups {
		if err = tx.QueryRow(placeGroupInsertStmt, g.Name).Scan(&g.ID); err != nil {
			return err
		}
		for _, p := range g.Places {
			if err = tx.QueryRow(placeInsertStmt, p.Key, p.Name, g.ID).Scan(&p.ID); err != nil {
				return err
			}
		}
		//if err = tx.QueryRow(placeGroupInsertStmt, groups[i].Name).Scan(&groups[i].ID); err != nil {
		//	return err
		//}
		//for k := range groups[i].Places {
		//	if err = tx.QueryRow(placeInsertStmt, groups[i].Places[k].Key, groups[i].Places[k].Name, groups[i].ID).Scan(&groups[i].Places[k].ID); err != nil {
		//		return err
		//	}
		//}

	}
	return nil
}

func (db *store) GetPlaceGroups() ([]petfind.PlaceGroup, error) {
	//const petGetPlaceGroupsQuery = `
	//SELECT g.id, g.name, p.group_id, p.id, p.name
	//FROM place_groups g
	//  JOIN places p ON p.group_id = g.id
	//`
	const placeGroupsGetQuery = `
	SELECT id, name
	FROM place_groups
	`
	rows, err := db.Query(placeGroupsGetQuery)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); err == nil {
			err = cerr
			return
		}
	}()

	groups := make([]petfind.PlaceGroup, 0)
	for rows.Next() {
		var g petfind.PlaceGroup
		if err := rows.Scan(&g.ID, &g.Name); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}

	for i := range groups {
		places, err := db.getPlacesByGroupID(groups[i].ID)
		if err != nil {
			return nil, err
		}
		groups[i].Places = places
	}
	return groups, nil
}
func (db *store) getPlacesByGroupID(groupID int64) ([]petfind.Place, error) {
	//const petGetPlaceGroupsQuery = `
	//SELECT g.id, g.name, p.group_id, p.id, p.name
	//FROM place_groups g
	//  JOIN places p ON p.group_id = g.id
	//`
	const placesGetByGroupIDQuery = `
	SELECT id, key, name, group_id
	FROM places
	where group_id=$1
	`
	rows, err := db.Query(placesGetByGroupIDQuery, groupID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if cerr := rows.Close(); err == nil {
			err = cerr
			return
		}
	}()

	places := make([]petfind.Place, 0)
	for rows.Next() {
		var p petfind.Place
		if err := rows.Scan(&p.ID, &p.Key, &p.Name, &p.GroupID); err != nil {
			return nil, err
		}
		places = append(places, p)
	}
	return places, nil
}

func (db *store) AddPlaceGroup(g *petfind.PlaceGroup) error {
	const placeGroupInsertStmt = `
	INSERT INTO place_groups(name)
	VALUES ($1)
	RETURNING id
	`
	stmt, err := db.Prepare(placeGroupInsertStmt)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := stmt.Close(); err == nil {
			err = cerr
			return
		}
	}()
	err = stmt.QueryRow(g.Name).Scan(&g.ID)
	if err != nil {
		return err
	}
	return nil
}

func (db *store) AddPlace(p *petfind.Place) error {
	const placeInsertStmt = `
	INSERT INTO places(key, name, group_id)
	VALUES ($1, $2, $3)
	RETURNING id
	`
	stmt, err := db.Prepare(placeInsertStmt)
	if err != nil {
		return err
	}
	defer func() {
		if cerr := stmt.Close(); err == nil {
			err = cerr
			return
		}
	}()
	err = stmt.QueryRow(p.Key, p.Name, p.GroupID).Scan(&p.ID)
	if err != nil {
		return err
	}
	return nil
}

func (db *store) CountPlaces() (int64, error) {
	const placeCountQuery = `
	SELECT COUNT(*)
	FROM places
	`

	var count int64
	err := db.QueryRow(placeCountQuery).Scan(&count)
	if err != nil {
		return -1, err
	}
	return count, nil
}

func (db *store) GetPlace(placeID int64) (*petfind.Place, error) {
	const placeGetQuery = `
	SELECT
	  id,
	  key,
	  group_id,
	  name
	FROM places
	WHERE id = $1
	`
	p := new(petfind.Place)
	err := db.QueryRow(placeGetQuery, placeID).Scan(
		&p.ID,
		&p.Key,
		&p.GroupID,
		&p.Name,
	)
	if err == sql.ErrNoRows {
		return nil, petfind.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil
}

func (db *store) GetPlaceByKey(placeKey string) (*petfind.Place, error) {
	const placeGetQuery = `
	SELECT
	  id,
	  key,
	  group_id,
	  name
	FROM places
	WHERE key = $1
	`
	p := new(petfind.Place)
	err := db.QueryRow(placeGetQuery, placeKey).Scan(
		&p.ID,
		&p.Key,
		&p.GroupID,
		&p.Name,
	)
	if err == sql.ErrNoRows {
		return nil, petfind.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return p, nil

}
