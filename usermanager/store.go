package user

import (
	"context"
	"encoding/json"
	"fmt"

	"strings"

	"go.etcd.io/bbolt"

	"github.com/oklog/ulid"
)

// UserStore represents a user storage contract
type UserStore interface {
	PutUser(ctx context.Context, u *User) error
	GetUserByID(ctx context.Context, id ulid.ULID) (*User, error)
	GetUserByIndex(ctx context.Context, index string, value string) (*User, error)
	DeleteUser(ctx context.Context, id ulid.ULID) error
}

// GroupStore describes a storage contract for groups specifically
// TODO add predicates for searching
type GroupStore interface {
	// groups
	PutGroup(g *Group) error
	GetGroup(id ulid.ULID) error
	GetAllGroups() ([]*Group, error)
	DeleteGroup(id ulid.ULID) error

	// group relations
	PutGroupRelation(g *Group, u *User) error
	GetGroupRelation(groupID ulid.ULID, userID ulid.ULID) (bool, error)
	GetGroupRelations(groupID ulid.ULID) ([]*Group, error)
	DeleteGroupRelation(groupID ulid.ULID, userID ulid.ULID) error
}

// NewDefaultStore initializing a default User store
func NewDefaultStore(db *bbolt.DB, sc StoreCache) (UserStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	s := &defaultStore{
		db:        db,
		userCache: sc,
	}

	return s, s.Init()
}

type defaultStore struct {
	db        *bbolt.DB
	userCache StoreCache
}

// Init initializing the storage
func (s *defaultStore) Init() error {
	// creating pre-defined buckets if they don't exist yet
	return s.db.Update(func(tx *bbolt.Tx) error {
		// user bucket
		userBucket, err := tx.CreateBucketIfNotExists([]byte("USER"))
		if err != nil {
			return fmt.Errorf("store.Init() failed to create users bucket: %s", err)
		}

		// username index child bucket
		if _, err = userBucket.CreateBucketIfNotExists([]byte("USERNAME")); err != nil {
			return fmt.Errorf("store.Init() failed to create username index: %s", err)
		}

		// email index child bucket
		if _, err = userBucket.CreateBucketIfNotExists([]byte("EMAIL")); err != nil {
			return fmt.Errorf("store.Init() failed to create email index: %s", err)
		}

		// metadata bucket
		_, err = tx.CreateBucketIfNotExists([]byte("METADATA"))
		if err != nil {
			return fmt.Errorf("store.Init() failed to create metadata bucket: %s", err)
		}

		// profile bucket
		_, err = tx.CreateBucketIfNotExists([]byte("PROFILE"))
		if err != nil {
			return fmt.Errorf("store.Init() failed to create metadata bucket: %s", err)
		}

		return nil
	})
}

// GetUserByID returns a User by ID
func (s *defaultStore) GetUserByID(ctx context.Context, id ulid.ULID) (*User, error) {
	if len(id) == 0 {
		return nil, ErrInvalidID
	}

	var user *User

	// cache lookup
	if s.userCache != nil {
		if user = s.userCache.GetUserByID(id); user != nil {
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

// GetUserByIndex lookup a user by an index
func (s *defaultStore) GetUserByIndex(ctx context.Context, index string, value string) (*User, error) {
	var user *User

	// cache lookup
	if s.userCache != nil {
		if c := s.userCache.GetUserByIndex(index, value); c != nil {
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

// PutUser stores a User
func (s *defaultStore) PutUser(ctx context.Context, u *User) error {
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

		err = userBucket.PutUser(u.ID[:], data)
		if err != nil {
			return fmt.Errorf("failed to store user: %s", err)
		}

		// storing username index
		b := userBucket.Bucket([]byte("USERNAME"))
		if b == nil {
			return fmt.Errorf("store.PutUser(username): %s", ErrBucketNotFound)
		}

		if err = b.PutUser([]byte(u.Username), u.ID[:]); err != nil {
			return err
		}

		// storing email index
		b = userBucket.Bucket([]byte("EMAIL"))
		if b == nil {
			return fmt.Errorf("store.PutUser(email): %s", ErrBucketNotFound)
		}

		if err = b.PutUser([]byte(u.Email), u.ID[:]); err != nil {
			return err
		}

		// renewing cache
		if s.userCache != nil {
			// doing only PutUser() because it'll reoccupy the existing space anyway
			s.userCache.PutUser(u)
		}

		return nil
	})
}

// DeleteUser a user from the store
func (s *defaultStore) DeleteUser(ctx context.Context, id ulid.ULID) error {
	if len(id) == 0 {
		return ErrInvalidID
	}

	// clearing cache
	if s.userCache != nil {
		s.userCache.DeleteUser(id)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		userBucket := tx.Bucket([]byte("USER"))
		if userBucket == nil {
			return fmt.Errorf("failed to load users bucket: %s", ErrBucketNotFound)
		}

		return userBucket.DeleteUser(id[:])
	})
}
