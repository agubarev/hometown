package util

import (
	"fmt"

	"github.com/dgraph-io/badger"
)

// CreateRandomBadgerDB creates a badger database with a random filename
func CreateRandomBadgerDB() (*badger.DB, string, error) {
	dbDir := fmt.Sprintf("/tmp/testdb-%s.dat", NewULID())
	db, err := badger.Open(badger.DefaultOptions(dbDir))
	if err != nil {
		return nil, "", err
	}

	return db, dbDir, nil
}
