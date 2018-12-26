package usermanager

import (
	"sync"

	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

// UserFilterFunc is a filter function parameter to be passed to List() function
type UserFilterFunc func(u []*User) []*User

// UserContainer is a container responsible for all operations within its scope
// TODO add default groups which need not to be assigned
// TODO add contexts for cancellation
type UserContainer struct {
	ID ulid.ULID `json:"id"`

	domain      *Domain
	users       []*User
	idMap       map[ulid.ULID]*User
	usernameMap map[string]*User
	emailMap    map[string]*User
	store       UserStore
	m           sync.RWMutex
}

// NewUserContainer initializing a new user container
func NewUserContainer(d *Domain) (*UserContainer, error) {
	if d == nil {
		return nil, ErrNilDomain
	}

	c := &UserContainer{
		ID:          util.NewULID(),
		domain:      d,
		users:       make([]*User, 0),
		idMap:       make(map[ulid.ULID]*User),
		usernameMap: make(map[string]*User),
		m:           sync.RWMutex{},
	}

	return c, nil
}

// Persist asks all contained groups to store itself
func (c *UserContainer) Persist() error {
	panic("not implemented")

	return nil
}

// AddUser to the container
func (c *UserContainer) AddUser(g *User) error {
	panic("not implemented")

	return nil
}

// RemoveUser from the container
func (c *UserContainer) RemoveUser(id ulid.ULID) error {
	panic("not implemented")

	return nil
}

// List returns all users by a given filter
// IMPORTANT this function returns values and must be used only for listing
// i.e. returning a list of users via API
func (c *UserContainer) List() []*User {
	return c.users
}

// GetByID returns a group by ID
func (c *UserContainer) GetByID(id ulid.ULID) (*User, error) {
	if g, ok := c.idMap[id]; ok {
		return g, nil
	}

	return nil, ErrGroupNotFound
}

// GetByUsername return user by username
func (c *UserContainer) GetByUsername(username string) (*User, error) {
	if u, ok := c.usernameMap[username]; ok {
		return u, nil
	}

	return nil, ErrUserNotFound
}

// GetByEmail return user by email
func (c *UserContainer) GetByEmail(email string) (*User, error) {
	if u, ok := c.emailMap[email]; ok {
		return u, nil
	}

	return nil, ErrUserNotFound
}
