package main

import (
	"bytes"
	"database/sql"
	"fmt"
	"sync"

	"encoding/gob"
)

type Store struct {
	mu   sync.Mutex
	name string
	db   *sql.DB
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

// Creates a new [Store] instance. name may only contain upper- or lowercase
// Latin letters.
func NewStore(db *sql.DB, name string) (*Store, error) {
	if !isLetters(name) { // HACK this is probably vulnerable to SQL injections
		return nil, ErrBadName
	}

	_, err := db.Exec(`
CREATE TABLE IF NOT EXISTS ` + name + ` (
	key		TEXT PRIMARY KEY,
	value	BLOB
);`)
	if err != nil {
		return nil, err
	}
	s := &Store{name: name, db: db}
	return s, nil
}

// Retrieve a value from the store. Value must be a pointer or nil. If key is
// not present, [ErrNotFound] is returned. If value is nil, data read from store
// is silently discarded.
func (s *Store) Get(key string, value any) error {
	var v []uint8
	if err := s.db.QueryRow(
		`SELECT value FROM `+s.name+` where key = ?;`,
		key).Scan(&v); err == sql.ErrNoRows {
		return ErrNotFound
	} else if err != nil {
		return err
	}
	if value == nil {
		return nil
	}
	dec := gob.NewDecoder(bytes.NewReader(v))
	return dec.Decode(value)
}

// Inserts a new key-value pair or updates an existing one.
func (s *Store) Set(key string, value any) error {
	s.mu.Lock()
	defer s.mu.Unlock()

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
	return err

}

// Deletes key from store without checking if it existed.
func (s *Store) Delete(key string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	_, err := s.db.Exec(`DELETE FROM `+s.name+` WHERE key = ?;`, key)
	return err
}
