package main

import (
	"bytes"
	"database/sql"
	"fmt"

	"encoding/gob"
)

type Store struct {
	db   *sql.DB
	name string
}

var (
	ErrBadName  = fmt.Errorf("bad name for store")
	ErrNotFound = fmt.Errorf("value not found")
)

func isLetter(c rune) bool {
	return 'a' <= c && c <= 'z' || 'A' <= c && c <= 'Z'
}

func isLetters(s string) bool {
	for _, c := range s {
		if !isLetter(c) {
			return false
		}
	}
	return true
}

// Creates a new [Store] instance. name may only contain upper- or lowercase Latin letters.
func NewStore(db *sql.DB, name string) (*Store, error) {
	if !isLetters(name) { // HACK this is probably vulnerable to SQL injections
		return nil, ErrBadName
	}

	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS `+name+` (
	key		TEXT PRIMARY KEY,
	value	BLOB
);`, name)
	if err != nil {
		return nil, err
	}
	s := &Store{db: db, name: name}
	return s, nil
}

func (s *Store) Set(key string, value any) error {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(value)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`
INSERT INTO `+s.name+` (key, value)
VALUES(?, ?) 
ON CONFLICT(key) 
DO UPDATE SET value=excluded.value;`,
		key, buf.Bytes())
	if err != nil {
		return err
	}
	return nil
}

// Retrieve a value from the store. Value must be a pointer or nil. If key is
// not present, [ErrNotFound] is returned. If value is nil, data read from store
// is silently discarded.
func (s *Store) Get(key string, value any) error {
	var v []uint8
	err := s.db.QueryRow(`SELECT value FROM `+s.name+` where key = ?`, key).Scan(&v)
	if err != nil {
		if err == sql.ErrNoRows {
			return ErrNotFound
		}
		return err
	}
	if value == nil {
		return nil
	}
	dec := gob.NewDecoder(bytes.NewReader(v))
	return dec.Decode(value)
}

func (s *Store) Delete(key string) error {
	_, err := s.db.Exec(`DELETE FROM `+s.name+` WHERE key = ?`, key)
	return err
}
