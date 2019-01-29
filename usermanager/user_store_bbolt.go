package usermanager

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/oklog/ulid"
	"go.etcd.io/bbolt"
)

// NewUserStoreBolt initializing bbolt store
func NewUserStoreBolt(db *bbolt.DB, uc UserStoreCache) (UserStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	s := &UserStoreBolt{
		db:        db,
		userCache: uc,
	}

	return s, s.Init()
}

// UserStoreBolt is using bbolt (previously known as BoltDB)
type UserStoreBolt struct {
	db        *bbolt.DB
	userCache UserStoreCache
}

// Init initializing the storage
func (s *UserStoreBolt) Init() error {
	// creating pre-defined buckets if they don't exist yet
	return s.db.Update(func(tx *bbolt.Tx) error {
		// user bucket
		userBucket, err := tx.CreateBucketIfNotExists([]byte("USER"))
		if err != nil {
			return fmt.Errorf("Init() failed to create users bucket: %s", err)
		}

		// username index child bucket
		if _, err = userBucket.CreateBucketIfNotExists([]byte("USERNAME")); err != nil {
			return fmt.Errorf("Init() failed to create username index: %s", err)
		}

		// email index child bucket
		if _, err = userBucket.CreateBucketIfNotExists([]byte("EMAIL")); err != nil {
			return fmt.Errorf("Init() failed to create email index: %s", err)
		}

		// metadata bucket
		_, err = tx.CreateBucketIfNotExists([]byte("METADATA"))
		if err != nil {
			return fmt.Errorf("Init() failed to create metadata bucket: %s", err)
		}

		return nil
	})
}

// GetByID returns a User by ID
func (s *UserStoreBolt) GetByID(ctx context.Context, id ulid.ULID) (*User, error) {
	if len(id) == 0 {
		return nil, ErrInvalidID
	}

	var user *User

	// cache lookup
	if s.userCache != nil {
		if user = s.userCache.GetByID(id); user != nil {
			return user, nil
		}
	}

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("USER"))
		if b == nil {
			return fmt.Errorf("store.GetUserByID(%s): %s", id, ErrBucketNotFound)
		}

		// lookup user by ID
		data := b.Get(id[:])
		if data == nil {
			return ErrUserNotFound
		}

		return json.Unmarshal(data, &user)
	})

	return user, err
}

// GetByIndex lookup a user by an index
func (s *UserStoreBolt) GetByIndex(ctx context.Context, index string, value string) (*User, error) {
	var user *User

	// cache lookup
	if s.userCache != nil {
		if c := s.userCache.GetByIndex(index, value); c != nil {
			// cache hit, returning
			return c, nil
		}
	}

	err := s.db.View(func(tx *bbolt.Tx) error {
		userBucket := tx.Bucket([]byte("USER"))
		if userBucket == nil {
			return fmt.Errorf("store.GetUserByIndex(%s): %s", index, ErrBucketNotFound)
		}

		// retrieving the index bucket
		indexBucket := userBucket.Bucket([]byte(strings.ToUpper(index)))
		if indexBucket == nil {
			return ErrIndexNotFound
		}

		// looking up ID by the index value
		id := indexBucket.Get([]byte(value))
		if id == nil {
			return ErrUserNotFound
		}

		// look up user by ID
		data := userBucket.Get(id)
		if data == nil {
			return ErrUserNotFound
		}

		return json.Unmarshal(data, &user)
	})

	return user, err
}

// Put stores a User
func (s *UserStoreBolt) Put(ctx context.Context, u *User) error {
	if u == nil {
		return ErrNilUser
	}

	if len(u.ID) == 0 {
		return ErrInvalidID
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		userBucket := tx.Bucket([]byte("USER"))
		if userBucket == nil {
			return fmt.Errorf("store.PutUser(): %s", ErrBucketNotFound)
		}

		// marshaling and storing the user
		data, err := json.Marshal(u)
		if err != nil {
			return err
		}

		err = userBucket.Put(u.ID[:], data)
		if err != nil {
			return fmt.Errorf("failed to store user: %s", err)
		}

		// storing username index
		b := userBucket.Bucket([]byte("USERNAME"))
		if b == nil {
			return fmt.Errorf("store.PutUser(username): %s", ErrBucketNotFound)
		}

		if err = b.Put([]byte(u.Username), u.ID[:]); err != nil {
			return err
		}

		// storing email index
		b = userBucket.Bucket([]byte("EMAIL"))
		if b == nil {
			return fmt.Errorf("store.PutUser(email): %s", ErrBucketNotFound)
		}

		if err = b.Put([]byte(u.Email), u.ID[:]); err != nil {
			return err
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
func (s *UserStoreBolt) Delete(ctx context.Context, id ulid.ULID) error {
	if len(id) == 0 {
		return ErrInvalidID
	}

	// clearing cache
	if s.userCache != nil {
		s.userCache.Delete(id)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		userBucket := tx.Bucket([]byte("USER"))
		if userBucket == nil {
			return fmt.Errorf("failed to load users bucket: %s", ErrBucketNotFound)
		}

		return userBucket.Delete(id[:])
	})
}
