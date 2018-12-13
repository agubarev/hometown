package user

import (
	"context"
	"encoding/json"
	"errors"

	"go.etcd.io/bbolt"

	"github.com/oklog/ulid"
)

// errors
var (
	ErrNilDB          = errors.New("database is nil")
	ErrIndexNotFound  = errors.New("index not found")
	ErrUserNotFound   = errors.New("user not found")
	ErrEmailNotFound  = errors.New("email not found")
	ErrInvalidID      = errors.New("invalid ID")
	ErrBucketNotFound = errors.New("bucket not found")
)

// Store represents a User storage contract
type Store interface {
	GetByID(ctx context.Context, id ulid.ULID) (*User, error)
	GetByIndex(ctx context.Context, index string, value string) (*User, error)
	Put(ctx context.Context, u *User) error
	Delete(ctx context.Context, id ulid.ULID) error
}

// NewDefaultStore initializing a default User store
func NewDefaultStore(db *bbolt.DB) (Store, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &store{db}, nil
}

type store struct {
	db *bbolt.DB
}

// GetByID returns a User by ID
func (s *store) GetByID(ctx context.Context, id ulid.ULID) (*User, error) {
	if len(id) == 0 {
		return nil, ErrInvalidID
	}

	user := &User{}
	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("users"))
		if b == nil {
			return ErrBucketNotFound
		}

		// lookup user by ID
		data := b.Get(id[:])
		if data == nil {
			return ErrUserNotFound
		}

		return json.Unmarshal(data, user)
	})

	return user, err
}

// GetByIndex lookup a user by an index
func (s *store) GetByIndex(ctx context.Context, index string, value string) (*User, error) {
	user := &User{}

	err := s.db.View(func(tx *bbolt.Tx) error {
		userBucket := tx.Bucket([]byte("users"))
		if userBucket == nil {
			return ErrBucketNotFound
		}

		// retrieving the index bucket
		indexBucket := userBucket.Bucket([]byte(index))
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

		return json.Unmarshal(data, user)
	})

	return user, err
}

// Put stores a User
func (s *store) Put(ctx context.Context, u *User) error {
	if u == nil {
		return ErrNilUser
	}

	if len(u.ID) == 0 {
		return ErrInvalidID
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		userBucket, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}

		// marshaling and storing the user
		data, err := json.Marshal(u)
		userBucket.Put(u.ID[:], data)

		// creating pre-defined indexes
		{
			b, err := userBucket.CreateBucketIfNotExists([]byte("username"))
			if err != nil {
				return err
			}
			if err = b.Put([]byte(u.Username), u.ID[:]); err != nil {
				return err
			}
		}

		{
			b, err := userBucket.CreateBucketIfNotExists([]byte("email"))
			if err != nil {
				return err
			}
			if err = b.Put([]byte(u.Email), u.ID[:]); err != nil {
				return err
			}
		}

		return nil
	})
}

// Delete a user from the store
func (s *store) Delete(ctx context.Context, id ulid.ULID) error {
	if len(id) == 0 {
		return ErrInvalidID
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		userBucket, err := tx.CreateBucketIfNotExists([]byte("users"))
		if err != nil {
			return err
		}
		return userBucket.Delete(id[:])
	})
}
