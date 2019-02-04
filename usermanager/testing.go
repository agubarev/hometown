package usermanager

import (
	"fmt"

	"github.com/dgraph-io/badger"
	"gitlab.com/agubarev/hometown/util"
)

func CreateRandomBadgerDB() (*badger.DB, string, error) {
	dbDir := fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID())
	opts := badger.DefaultOptions
	opts.Dir = dbDir
	opts.ValueDir = dbDir
	db, err := badger.Open(opts)
	if err != nil {
		return nil, "", err
	}

	return db, dbDir, nil
}
