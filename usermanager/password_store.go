package usermanager

import (
	"github.com/dgraph-io/badger"
	"github.com/oklog/ulid"
)

// PasswordStore interface
type PasswordStore interface {
	Put(id ulid.ULID, pass []byte) error
	Get(id ulid.ULID) ([]byte, error)
	Delete(id ulid.ULID) error
}

type defaultPasswordStore struct {
	db *badger.DB
}

// NewDefaultPasswordStore initializes a default password store
func NewDefaultPasswordStore(db *badger.DB) (PasswordStore, error) {
	// reserving error return for the future, just in case
	return &defaultPasswordStore{db}, nil
}

// Put stores password
// ID must be equal to the user's ID
func (s *defaultPasswordStore) Put(id ulid.ULID, pass []byte) error {
	if len(pass) == 0 {
		return ErrEmptyPassword
	}

	return s.db.Update(func(tx *badger.Txn) error {
		return tx.Set(id[:], pass)
	})
}

// Get retrieves a stored password
func (s *defaultPasswordStore) Get(id ulid.ULID) ([]byte, error) {
	var pass []byte

	err := s.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(id[:])
		if err != nil {
			return err
		}

		// copying value and while returning
		return item.Value(func(val []byte) error {
			// badger's freaky way of working with values imo
			pass = append(pass, val...)

			return nil
		})
	})

	return pass, err
}

func (s *defaultPasswordStore) Delete(id ulid.ULID) error {
	return s.db.Update(func(tx *badger.Txn) error {
		return tx.Delete(id[:])
	})
}
