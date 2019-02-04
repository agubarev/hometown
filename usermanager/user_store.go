package usermanager

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/dgraph-io/badger"

	"github.com/oklog/ulid"
)

// UserStore represents a user storage contract
type UserStore interface {
	Put(u *User) error
	Delete(id ulid.ULID) error
	Get(id ulid.ULID) (*User, error)
	GetByIndex(index string, value string) (*User, error)
}

func userKey(id ulid.ULID) []byte {
	return []byte(fmt.Sprintf("user:%s", id[:]))
}

func userIndexKey(index string, value string) []byte {
	return []byte(fmt.Sprintf("uidx:%s:%s", index, value))
}

type defaultUserStore struct {
	db *badger.DB
}

// NewDefaultUserStore initializing bbolt store
func NewDefaultUserStore(db *badger.DB) (UserStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	s := &defaultUserStore{
		db: db,
	}

	return s, nil
}

// Put stores a User
func (s *defaultUserStore) Put(u *User) error {
	if u == nil {
		return ErrNilUser
	}

	return s.db.Update(func(tx *badger.Txn) error {
		// decoding user payload bytes using gob
		var payload bytes.Buffer
		err := gob.NewEncoder(&payload).Encode(u)
		if err != nil {
			return fmt.Errorf("failed to encode user: %s", err)
		}

		// storing primary value
		primaryKey := userKey(u.ID)
		err = tx.Set(primaryKey, payload.Bytes())
		if err != nil {
			return fmt.Errorf("failed to store user %s: %s", primaryKey, err)
		}

		// storing username index with a primary key as value
		err = tx.Set(userIndexKey("username", u.Username), primaryKey)
		if err != nil {
			return fmt.Errorf("failed to store username(%s) index %s: %s", u.Username, err)
		}

		// storing email index, same as above
		err = tx.Set(userIndexKey("email", u.Email), primaryKey)
		if err != nil {
			return fmt.Errorf("failed to store email(%s) index %s: %s", u.Email, err)
		}

		return nil
	})
}

// Delete a user from the store
func (s *defaultUserStore) Delete(id ulid.ULID) error {
	return s.db.Update(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.PrefetchSize = 5
		opts.Prefix = userKey(id)
		it := tx.NewIterator(opts)

		for it.Rewind(); it.Valid(); it.Next() {
			if err := tx.Delete(it.Item().Key()); err != nil {
				return fmt.Errorf("failed to delete stored user")
			}
		}

		return nil
	})
}

// Get returns a User by ID
func (s *defaultUserStore) Get(id ulid.ULID) (*User, error) {
	var u *User

	err := s.db.View(func(tx *badger.Txn) error {
		// lookup user by ID
		item, err := tx.Get(userKey(id))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrUserNotFound
			}

			return fmt.Errorf("failed to get stored user by ID %s: %s", id, err)
		}

		// obtaining value
		return item.Value(func(payload []byte) error {
			if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(u); err != nil {
				return fmt.Errorf("failed to decode stored user: %s", err)
			}

			return nil
		})
	})

	return u, err
}

// GetByIndex lookup a user by an index
func (s *defaultUserStore) GetByIndex(index string, value string) (*User, error) {
	var u *User

	err := s.db.View(func(tx *badger.Txn) error {
		// lookup user by ID
		item, err := tx.Get(userIndexKey(index, value))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrUserNotFound
			}

			return fmt.Errorf("failed to get stored user by index(%s=): %s", index, value, err)
		}

		// obtaining value
		return item.Value(func(payload []byte) error {
			if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(u); err != nil {
				return fmt.Errorf("failed to decode stored user: %s", err)
			}

			return nil
		})
	})

	return u, err
}
