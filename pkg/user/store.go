package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang/glog"

	"strings"

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
	Init() error
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

	s := &store{
		db:        db,
		userCache: newUserCache(),
	}

	return s, s.Init()
}

type store struct {
	db        *bbolt.DB
	userCache *userCache
}

//---------------------------------------------------------------------------
// internal user cache for default implementation, a very simple mechanism
// this cache can return nil because it doesn't perform any checks
//---------------------------------------------------------------------------
func newUserCache() *userCache {
	return &userCache{
		users:     make(map[ulid.ULID]*User),
		usernames: make(map[string]*User),
		emails:    make(map[string]*User),
		threshold: 1000,
	}
}

// TODO: add worker and prevent it from leaking
type userCache struct {
	users     map[ulid.ULID]*User
	usernames map[string]*User
	emails    map[string]*User
	counter   int
	threshold int
	sync.RWMutex
}

func (c *userCache) put(u *User) {
	c.Lock()
	c.users[u.ID] = u
	c.usernames[u.Username] = u
	c.emails[u.Email] = u
	c.counter++
	c.Unlock()
}

func (c *userCache) get(id ulid.ULID) *User {
	c.RLock()
	defer c.RUnlock()
	return c.users[id]
}

func (c *userCache) getByUsername(username string) *User {
	c.RLock()
	defer c.RUnlock()
	return c.usernames[username]
}

func (c *userCache) getByEmail(email string) *User {
	c.RLock()
	defer c.RUnlock()
	return c.emails[email]
}

func (c *userCache) delete(id ulid.ULID) {
	c.Lock()
	if u, ok := c.users[id]; ok {
		delete(c.usernames, u.Username)
		delete(c.emails, u.Email)
		delete(c.users, id)
		c.counter--
	}
	c.Unlock()
}

func (c *userCache) shrug() {
	fmt.Println("shrugged")
}

//---------------------------------------------------------------------------
// default implementation
//---------------------------------------------------------------------------

// Init initializing the storage
func (s *store) Init() error {
	glog.Info("initializing default user store")
	// starting a maintenance goroutine for the user cache
	go func(c *userCache) {
		glog.Info("started user cache poker")
		poker := time.NewTicker(10 * time.Second)
		for {
			select {
			case <-poker.C:
				fmt.Println("poked")
				c.shrug()
			}
		}
	}(s.userCache)

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

// GetByID returns a User by ID
func (s *store) GetByID(ctx context.Context, id ulid.ULID) (*User, error) {
	if len(id) == 0 {
		return nil, ErrInvalidID
	}

	var user *User

	// cache lookup
	if s.userCache != nil {
		if user = s.userCache.get(id); user != nil {
			return user, nil
		}
	}

	err := s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket([]byte("USER"))
		if b == nil {
			return fmt.Errorf("store.GetByID(%s): %s", id, ErrBucketNotFound)
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
func (s *store) GetByIndex(ctx context.Context, index string, value string) (*User, error) {
	var user *User

	// cache lookup
	if s.userCache != nil {
		switch index {
		case "username":
			user = s.userCache.getByUsername(value)
		case "email":
			user = s.userCache.getByEmail(value)
		}

		// cache hit, returning
		if user != nil {
			return user, nil
		}
	}

	err := s.db.View(func(tx *bbolt.Tx) error {
		userBucket := tx.Bucket([]byte("USER"))
		if userBucket == nil {
			return fmt.Errorf("store.GetByIndex(%s): %s", index, ErrBucketNotFound)
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
func (s *store) Put(ctx context.Context, u *User) error {
	if u == nil {
		return ErrNilUser
	}

	if len(u.ID) == 0 {
		return ErrInvalidID
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		userBucket := tx.Bucket([]byte("USER"))
		if userBucket == nil {
			return fmt.Errorf("store.Put(): %s", ErrBucketNotFound)
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
			return fmt.Errorf("store.Put(username): %s", ErrBucketNotFound)
		}

		if err = b.Put([]byte(u.Username), u.ID[:]); err != nil {
			return err
		}

		// storing email index
		b = userBucket.Bucket([]byte("EMAIL"))
		if b == nil {
			return fmt.Errorf("store.Put(email): %s", ErrBucketNotFound)
		}

		if err = b.Put([]byte(u.Email), u.ID[:]); err != nil {
			return err
		}

		// renewing cache
		if s.userCache != nil {
			s.userCache.delete(u.ID)
			s.userCache.put(u)
		}

		return nil
	})
}

// Delete a user from the store
func (s *store) Delete(ctx context.Context, id ulid.ULID) error {
	if len(id) == 0 {
		return ErrInvalidID
	}

	// clearing cache
	if s.userCache != nil {
		s.userCache.delete(id)
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		userBucket := tx.Bucket([]byte("USER"))
		if userBucket == nil {
			return fmt.Errorf("failed to load users bucket: %s", ErrBucketNotFound)
		}

		return userBucket.Delete(id[:])
	})
}
