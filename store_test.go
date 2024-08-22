package main

import (
	"database/sql"
	"fmt"
	"maps"
	"math/rand/v2"
	"os"
	"reflect"
	"slices"
	"testing"

	_ "github.com/mattn/go-sqlite3"
)

func setupTestStore() (*Store, func(), error) {
	f, err := os.CreateTemp("", "sqlite-storage-")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create temp file: %v", err)
	}

	db, err := sql.Open("sqlite3", f.Name())
	if err != nil {
		return nil, nil, fmt.Errorf("failed to connect sqlite db: %v", err)
	}

	s, err := NewStore(db, "teststore")
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create new store: %v", err)
	}

	teardown := func() {
		f.Close()
		os.Remove(f.Name())
	}

	return s, teardown, nil
}

func TestStoreReadEmpty(t *testing.T) {
	s, teardown, err := setupTestStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	var nothing struct{}
	if err = s.Get("some key", &nothing); err != ErrNotFound {
		t.Fatalf("expected not found error, received %v", err)
	}
}

func TestStoreWriteAndReadPrimitive(t *testing.T) {
	s, teardown, err := setupTestStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	key := "key"
	val := 1337
	if err = s.Set(key, val); err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	var rtVal int
	if err = s.Get(key, &rtVal); err != nil {
		t.Fatalf("failed to get value: %v", err)
	}

	if val != rtVal {
		t.Fatalf("expected: %v, actual: %v", val, rtVal)
	}
}

func TestStoreWriteAndReadStruct(t *testing.T) {
	s, teardown, err := setupTestStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	type Box struct {
		Name  string
		Array []int64
		AdHoc struct{ Flags []bool }
		Inner *Box
	}

	key := "key"
	var val = Box{
		Name:  "some name",
		Array: []int64{1, 2, 3},
		AdHoc: struct{ Flags []bool }{[]bool{true, false, true, false}},
		Inner: &Box{"other name", nil, struct{ Flags []bool }{[]bool{true}}, nil},
	}
	if err = s.Set(key, val); err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	var rtVal Box
	if err = s.Get(key, &rtVal); err != nil {
		t.Fatalf("failed to get value: %v", err)
	}

	if !reflect.DeepEqual(val, rtVal) {
		t.Fatalf("expected: %v, actual: %v", val, rtVal)
	}
}

func TestStoreWriteAndReadNil(t *testing.T) {
	s, teardown, err := setupTestStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	key := "key"
	val := 1337
	if err = s.Set(key, val); err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	if err = s.Get(key, nil); err != nil {
		t.Fatalf("failed to get value: %v", err)
	}
}

func TestStoreUpdate(t *testing.T) {
	s, teardown, err := setupTestStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	r := rand.New(rand.NewPCG(1, 2))
	key := "key"
	val := r.Int32()

	if err = s.Set(key, val); err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	val = r.Int32()
	if err = s.Set(key, val); err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	var rtVal int32
	if err = s.Get(key, &rtVal); err != nil {
		t.Fatalf("failed to get value: %v", err)
	}

	if val != rtVal {
		t.Fatalf("failed to update value (expected %v, actual %v)", val, rtVal)
	}
}

func TestStoreDeleteMissing(t *testing.T) {
	s, teardown, err := setupTestStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	if err := s.Delete("something"); err != nil {
		t.Fatal(err)
	}
}

func TestStoreDeleteExisting(t *testing.T) {
	s, teardown, err := setupTestStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	key := "key"
	val := 1337
	if err = s.Set(key, val); err != nil {
		t.Fatalf("failed to set value: %v", err)
	}

	if err := s.Delete(key); err != nil {
		t.Fatalf("failed to delete value: %v", err)
	}

	var rtVal int
	err = s.Get(key, &rtVal)
	if err != ErrNotFound {
		if err == nil {
			t.Fatalf("expected to not find value, instead got %v", rtVal)
		} else {
			t.Fatalf("expected to get not found err, instead got %v", err)
		}
	}
}

func TestStoreCount(t *testing.T) {
	s, teardown, err := setupTestStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	rows := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
		"d": 4,
	}
	for key, value := range rows {
		if err := s.Set(key, value); err != nil {
			t.Fatal(err)
		}
	}

	if count, err := s.Count(); err != nil {
		t.Fatal(err)
	} else if count != len(rows) {
		t.Fatalf("have %d, want %d", count, len(rows))
	}

	delete(rows, "a")
	if err := s.Delete("a"); err != nil {
		t.Fatal(err)
	}

	if count, err := s.Count(); err != nil {
		t.Fatal(err)
	} else if count != len(rows) {
		t.Fatalf("have %d, want %d", count, len(rows))
	}
}

func TestStoreGetAllKeys(t *testing.T) {
	s, teardown, err := setupTestStore()
	if err != nil {
		t.Fatal(err)
	}
	defer teardown()

	rows := map[string]int{
		"a": 1,
		"b": 2,
		"c": 3,
		"d": 4,
	}
	for key, value := range rows {
		if err := s.Set(key, value); err != nil {
			t.Fatal(err)
		}
	}

	keys, err := s.GetAllKeys()
	if err != nil {
		t.Fatal(err)
	}
	expectedKeys := slices.Collect(maps.Keys(rows))
	slices.Sort(keys)
	slices.Sort(expectedKeys)
	if !reflect.DeepEqual(keys, expectedKeys) {
		t.Fatalf("have %v, want %v", keys, expectedKeys)
	}
}
