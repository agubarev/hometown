package usermanager

import (
	"sync"
	"time"

	"github.com/oklog/ulid"
)

// UserStoreCache is an internal user caching mechanism for a Store
type UserStoreCache interface {
	GetByID(id ulid.ULID) *User
	GetByIndex(index string, value string) *User
	Put(u *User)
	Delete(id ulid.ULID)
	Cleanup() error
}

// NewUserStoreCache is an internal user cache for default implementation
// a very simple mechanism, returning nil on cache misses
func NewUserStoreCache(threshold int) UserStoreCache {
	return &userCache{
		users:     make(map[ulid.ULID]cachedUser),
		usernames: make(map[string]ulid.ULID),
		emails:    make(map[string]ulid.ULID),
		counter:   0,
		len:       0,
		threshold: 0,
	}
}

type cachedUser struct {
	u         *User
	expiresAt time.Time
}

type userCache struct {
	users     map[ulid.ULID]cachedUser
	usernames map[string]ulid.ULID
	emails    map[string]ulid.ULID
	len       int
	counter   uint64
	cacheTTL  time.Duration
	threshold int
	sync.RWMutex
}

func (c *userCache) Put(u *User) {
	c.Lock()
	c.users[u.ID] = cachedUser{u, time.Now().Add(c.cacheTTL)}
	c.usernames[u.Username] = u.ID
	c.emails[u.Email] = u.ID
	c.len++
	c.Unlock()
}

func (c *userCache) GetByID(id ulid.ULID) (u *User) {
	c.RLock()
	if ci, ok := c.users[id]; ok {
		u = ci.u
	}
	c.RUnlock()

	return
}

func (c *userCache) GetByIndex(index string, value string) (u *User) {
	var id ulid.ULID
	var ok bool

	c.RLock()
	switch index {
	case "username":
		id, ok = c.usernames[value]
	case "email":
		id, ok = c.emails[value]
	}
	if ok {
		u = c.users[id].u
	}
	c.RUnlock()

	return
}

func (c *userCache) Delete(id ulid.ULID) {
	c.Lock()
	if ci, ok := c.users[id]; ok {
		delete(c.usernames, ci.u.Username)
		delete(c.emails, ci.u.Email)
		delete(c.users, id)
		c.len--
	}
	c.Unlock()
}

func (c *userCache) Cleanup() error {
	// TODO: implement

	return nil
}
