package usermanager

import (
	"fmt"

	"github.com/dgraph-io/badger"

	"github.com/oklog/ulid"
)

// UserStore represents a user storage contract
type UserStore interface {
	Put(u *User) error
	Delete(id ulid.ULID) error
	GetAll() ([]*User, error)
	Get(id ulid.ULID) (*User, error)
	GetByIndex(index string, value string) (*User, error)
}

func userKey(id ulid.ULID) []byte {
	return []byte(fmt.Sprintf("u:%s", id[:]))
}

func userIndexKey(index string, value string) []byte {
	return []byte(fmt.Sprintf("ui:%s:%s", index, value))
}

// DefaultUserStore is a default user store implementation
type DefaultUserStore struct {
	db *badger.DB
}

// NewDefaultUserStore initializing bbolt store
func NewDefaultUserStore(db *badger.DB) (UserStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	s := &DefaultUserStore{
		db: db,
	}

	return s, nil
}

// Put stores a User
func (s *DefaultUserStore) Put(u *User) error {
	if u == nil {
		return ErrNilUser
	}

	return s.db.Update(func(tx *badger.Txn) error {
		payload, err := json.Marshal(u)
		if err != nil {
			return fmt.Errorf("failed to marshal user: %s", err)
		}

		// storing primary value
		primaryKey := userKey(u.ID)
		err = tx.Set(primaryKey, payload)
		if err != nil {
			return fmt.Errorf("failed to store user %s: %s", primaryKey, err)
		}

		// storing username index with a primary key as value
		err = tx.Set(userIndexKey("username", u.Username), primaryKey)
		if err != nil {
			return fmt.Errorf("failed to store username(%s) index: %s", u.Username, err)
		}

		// storing email index, same as above
		err = tx.Set(userIndexKey("email", u.Email), primaryKey)
		if err != nil {
			return fmt.Errorf("failed to store email(%s) index: %s", u.Email, err)
		}

		return nil
	})
}

// Delete a user from the store
func (s *DefaultUserStore) Delete(id ulid.ULID) error {
	u, err := s.Get(id)
	if err != nil {
		return fmt.Errorf("Delete(): %s", err)
	}

	return s.db.Update(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.PrefetchSize = 5
		opts.Prefix = userKey(id)
		it := tx.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			// deleting user indexes
			err = tx.Delete(userIndexKey("username", u.Username))
			if err != nil {
				return fmt.Errorf("Delete(): failed to delete username(%s) index: %s", u.Username, err)
			}

			err = tx.Delete(userIndexKey("email", u.Email))
			if err != nil {
				return fmt.Errorf("Delete(): failed to delete email(%s) index: %s", u.Email, err)
			}

			// deleting user
			return tx.Delete(it.Item().Key())
		}

		return nil
	})
}

// Get returns a User by ID
func (s *DefaultUserStore) Get(id ulid.ULID) (*User, error) {
	return s.getByByteKey(userKey(id))
}

func (s *DefaultUserStore) getByByteKey(key []byte) (*User, error) {
	u := new(User)

	err := s.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(key)
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrUserNotFound
			}

			return fmt.Errorf("getByByteKey(): failed to get stored user by ID %s: %s", key, err)
		}

		return item.Value(func(payload []byte) error {
			if err := json.Unmarshal(payload, u); err != nil {
				return fmt.Errorf("getByByteKey(): failed to unmarshal stored user: %s", err)
			}

			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return u, nil
}

// GetAll returns all stored users
func (s *DefaultUserStore) GetAll() ([]*User, error) {
	us := make([]*User, 0)

	err := s.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("u:")
		it := tx.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			err := it.Item().Value(func(payload []byte) error {
				u := new(User)
				if err := json.Unmarshal(payload, u); err != nil {
					return fmt.Errorf("failed to unmarshal user payload: %s", err)
				}

				us = append(us, u)

				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	return us, err
}

// GetByIndex lookup a user by an index
func (s *DefaultUserStore) GetByIndex(index string, value string) (*User, error) {
	u := &User{}

	err := s.db.View(func(tx *badger.Txn) error {
		// lookup user by ID
		item, err := tx.Get(userIndexKey(index, value))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrUserNotFound
			}

			return fmt.Errorf("GetByIndex(): failed to get stored user by index(%s=%s): %s", index, value, err)
		}

		return item.Value(func(primaryKey []byte) error {
			u, err = s.getByByteKey(primaryKey)
			if err != nil {
				return err
			}

			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return u, nil
}
