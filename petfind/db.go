package petfind

import (
	"database/sql/driver"
	"fmt"
)

// This file contains implementations of interfaces that allow the Petfind
// types to be persisted in the database.

func (p PetType) Value() (driver.Value, error) { return int64(p), nil }
func (p *PetType) Scan(value interface{}) error {
	if value == nil {
		*p = UnknownType
		return nil
	}
	if v, ok := value.(int64); ok {
		*p = PetType(v)
		return nil
	}
	return fmt.Errorf("cannot scan PetType value")
}

func (p PetSize) Value() (driver.Value, error) { return int64(p), nil }
func (p *PetSize) Scan(value interface{}) error {
	if value == nil {
		*p = UnknownSize
		return nil
	}
	if v, ok := value.(int64); ok {
		*p = PetSize(v)
		return nil
	}
	return fmt.Errorf("cannot scan PetSize value")
}

func (p PetAge) Value() (driver.Value, error) { return int64(p), nil }
func (p *PetAge) Scan(value interface{}) error {
	if value == nil {
		*p = UnknownAge
		return nil
	}
	if v, ok := value.(int64); ok {
		*p = PetAge(v)
		return nil
	}
	return fmt.Errorf("cannot scan PetAge value")
}
