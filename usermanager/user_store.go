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
	GetByID(domainID ulid.ULID, id ulid.ULID) (*User, error)
	GetByIndex(domainID ulid.ULID, index string, value string) (*User, error)
	Delete(id ulid.ULID) error
}

func userKey(domainID ulid.ULID, userID ulid.ULID) []byte {
	return []byte(fmt.Sprintf("%s:user:%s", domainID[:], userID[:]))
}

func userIndexKey(domainID ulid.ULID, index string, value string) []byte {
	return []byte(fmt.Sprintf("%s:user:index:%s:%s", domainID[:], index, value))
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
		db:        db,
		userCache: uc,
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
		primaryKey = userKey(u.Domain().ID, u.ID)
		err = tx.Set(primaryKey, payload.Bytes())
		if err != nil {
			return fmt.Errorf("failed to store user %s: %s", primaryKey, err)
		}

		// storing username index with a primary key as value
		err = tx.Set(userIndexKey(u.Domain().ID, "username", u.Username), primaryKey)
		if err != nil {
			return fmt.Errorf("failed to store username index %s: %s", indexKey, err)
		}

		// storing email index, same as above
		err = tx.Set(userIndexKey(u.Domain().ID, "email", u.Email), primaryKey)
		if err != nil {
			return fmt.Errorf("failed to store email index %s: %s", indexKey, err)
		}

		return nil
	})
}

// Delete a user from the store
func (s *defaultUserStore) Delete(u *User) error {
	return s.db.Update(func(tx *badger.Txn) error {
		// deleting primary key
		err := tx.Delete(userKey(u.Domain().ID, u.ID))
		if err != nil {
			return fmt.Errorf("failed to delete stored user: %s", err)
		}

		// deleting username index
		err = tx.Delete(userIndexKey(u.Domain().ID, "username", u.Username))
		if err != nil {
			return fmt.Errorf("failed to delete username index %s: %s", err)
		}

		// deleting email index
		err = tx.Delete(userIndexKey(u.Domain().ID, "email", u.Email))
		if err != nil {
			return fmt.Errorf("failed to delete email user index %s: %s", err)
		}

		return nil
	})
}

// GetByID returns a User by ID
func (s *defaultUserStore) GetByID(d *Domain, id ulid.ULID) (*User, error) {
	var user *User

	err := s.db.View(func(tx *badger.Txn) error {
		// lookup user by ID
		item, err := tx.Get(userKey(d.ID, id))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrUserNotFound
			}

			return fmt.Errorf("failed to get stored user by ID %s: %s", id, err)
		}

		// obtaining value
		return item.Value(func(val []byte) error {
			if err := gob.NewDecoder(user).Decode(val); err != nil {
				return fmt.Errorf("failed to decode stored user: %s", err)
			}

			return nil
		})
	})

	return user, err
}

// GetByIndex lookup a user by an index
func (s *defaultUserStore) GetByIndex(domainID ulid.ULID, index string, value string) (*User, error) {
	var user *User
	err := s.db.View(func(tx *badger.Txn) error {
		// lookup user by ID
		item, err := tx.Get(userIndexKey(domainID, index, value))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrUserNotFound
			}

			return fmt.Errorf("failed to get stored user by index %s: %s", indexKey, err)
		}

		// obtaining value
		return item.Value(func(val []byte) error {
			if err := gob.NewDecoder(user).Decode(val); err != nil {
				return fmt.Errorf("failed to decode stored user: %s", err)
			}

			return nil
		})
	})

	return user, err
}
