package util

import (
	"fmt"

	"github.com/dgraph-io/badger"
)

func CreateRandomBadgerDB() (*badger.DB, string, error) {
	dbDir := fmt.Sprintf("/tmp/testdb-%s.dat", NewULID())
	opts := badger.DefaultOptions
	opts.Dir = dbDir
	opts.ValueDir = dbDir
	db, err := badger.Open(opts)
	if err != nil {
		return nil, "", err
	}

	return db, dbDir, nil
}
