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
	GetByID(id ulid.ULID) (*User, error)
	GetByIndex(index string, value string) (*User, error)
	Delete(id ulid.ULID) error
}

// NewDefaultUserStore initializing bbolt store
func NewDefaultUserStore(db *badger.DB, uc UserStoreCache) (UserStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	s := &defaultUserStore{
		db:        db,
		userCache: uc,
	}

	return s, s.Init()
}

// TODO: do I really need caching here?
// TODO: use sync.Pool
type defaultUserStore struct {
	db        *badger.DB
	userCache UserStoreCache
}

// Put stores a User
func (s *defaultUserStore) Put(u *User) error {
	if u == nil {
		return ErrNilUser
	}

	return s.db.Update(func(tx *badger.Txn) error {
		// serializing user using gob
		var data bytes.Buffer
		err := gob.NewEncoder(data).Encode(u)
		if err != nil {
			return fmt.Errorf("failed to encode user: %s", err)
		}

		// storing primary value
		key := u.ID[:]
		err = tx.Set(key, data)
		if err != nil {
			return fmt.Errorf("failed to store user %s: %s", key, err)
		}

		// storing username index with a primary key as value
		indexKey := []byte(fmt.Sprintf("username:%s", u.Username))
		err = tx.Set(indexKey, key)
		if err != nil {
			return fmt.Errorf("failed to store username index %s: %s", indexKey, err)
		}

		// storing email index, same as above
		indexKey = []byte(fmt.Sprintf("email:%s", u.Email))
		err = tx.Set(indexKey, key)
		if err != nil {
			return fmt.Errorf("failed to store email index %s: %s", indexKey, err)
		}

		// renewing cache
		if s.userCache != nil {
			// doing only PutUser() because it'll reoccupy the existing space anyway
			s.userCache.Put(u)
		}

		return nil
	})
}

// Delete a user from the store
func (s *defaultUserStore) Delete(id ulid.ULID) error {
	// clearing cache
	if s.userCache != nil {
		s.userCache.Delete(id)
	}

	u, err := s.GetByID(id)
	if err != nil {
		return fmt.Errorf("failed to delete stored user: %s", err)
	}

	return s.db.Update(func(tx *badger.Txn) error {
		// deleting primary key
		err := tx.Delete(id[:])
		if err != nil {
			return fmt.Errorf("failed to delete stored user: %s", err)
		}

		// deleting username index
		indexKey := []byte(fmt.Sprintf("username:%s", u.Username))
		err = tx.Delete(indexKey)
		if err != nil {
			return fmt.Errorf("failed to delete username index %s: %s", err)
		}

		// deleting email index
		indexKey = []byte(fmt.Sprintf("email:%s", u))
		err = tx.Delete(indexKey)
		if err != nil {
			return fmt.Errorf("failed to delete email user index %s: %s", err)
		}

		return nil
	})
}

// GetByID returns a User by ID
func (s *defaultUserStore) GetByID(id ulid.ULID) (*User, error) {
	var user *User

	// cache lookup
	if s.userCache != nil {
		if user = s.userCache.GetByID(id); user != nil {
			return user, nil
		}
	}

	err := s.db.View(func(tx *badger.Txn) error {
		// lookup user by ID
		item, err := tx.Get(id[:])
		if err != nil {
			if err == ErrKeyNotFound {
				return ErrUserNotFound
			}

			return fmt.Errorf("failed to get stored user by ID %s: %s", id, err)
		}

		// obtaining value
		return item.Value(func(val []byte) error {
			if err := gob.NewDecoder(user).Decode(val); err != nil {
				return fmt.Errorf("failed to unserialize stored user: %s", err)
			}

			return nil
		})
	})

	return user, err
}

// GetByIndex lookup a user by an index
func (s *defaultUserStore) GetByIndex(index string, value string) (*User, error) {
	var user *User

	// cache lookup
	if s.userCache != nil {
		if c := s.userCache.GetByIndex(index, value); c != nil {
			// cache hit, returning
			return c, nil
		}
	}

	err := s.db.View(func(tx *badger.Txn) error {
		// lookup user by ID
		indexKey := []byte(fmt.Sprintf("%s:%s", index, value))
		item, err := tx.Get(indexKey)
		if err != nil {
			if err == ErrKeyNotFound {
				return ErrUserNotFound
			}

			return fmt.Errorf("failed to get stored user by index %s: %s", indexKey, err)
		}

		// obtaining value
		return item.Value(func(val []byte) error {
			if err := gob.NewDecoder(user).Decode(val); err != nil {
				return fmt.Errorf("failed to unserialize stored user: %s", err)
			}

			return nil
		})
	})

	return user, err
}
