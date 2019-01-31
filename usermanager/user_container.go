package usermanager

import (
	"errors"
	"log"
	"sync"

	"github.com/blevesearch/bleve"
	"github.com/oklog/ulid"
)

// UserFilterFunc is a filter function parameter to be passed to List() function
type UserFilterFunc func(u []*User) []*User

// UserContainer is a container responsible for all operations within its scope
// TODO add default groups which need not to be assigned
// TODO add contexts for cancellation
// TODO: use radix tree for id, username and email indexing
type UserContainer struct {
	users       []*User
	idMap       map[ulid.ULID]*User
	usernameMap map[string]*User
	emailMap    map[string]*User
	store       UserStore
	index       bleve.Index
	sync.RWMutex
}

// NewUserContainer initializing a new user container
func NewUserContainer(s UserStore, idx bleve.Index) (*UserContainer, error) {
	if s == nil {
		log.Println("WARNING: NewUserContainer() store isn't set")
	}

	if idx == nil {
		return nil, ErrNilUserIndex
	}

	c := &UserContainer{
		users:       make([]*User, 0),
		idMap:       make(map[ulid.ULID]*User),
		usernameMap: make(map[string]*User),
		emailMap:    make(map[string]*User),
		store:       s,
		index:       idx,
	}

	// TODO: initialize bleve index

	return c, nil
}

func (c *UserContainer) loadUsers() error {
	// TODO: implement

	return nil
}

// Validate this group container
func (c *UserContainer) Validate() error {
	if c.users == nil {
		return errors.New("users slice is not initialized")
	}

	if c.idMap == nil {
		return errors.New("id map is nil")
	}

	if c.usernameMap == nil {
		return errors.New("username map is nil")
	}

	if c.emailMap == nil {
		return errors.New("email map is nil")
	}

	return nil
}

// Register user to the container
func (c *UserContainer) Register(user *User) error {
	if user == nil {
		return ErrNilUser
	}

	if _, err := c.GetByID(user.ID); err != ErrUserNotFound {
		return ErrUserExists
	}

	if _, err := c.GetByUsername(user.Username); err != ErrUserNotFound {
		return ErrUsernameTaken
	}

	if _, err := c.GetByEmail(user.Email); err != ErrUserNotFound {
		return ErrEmailTaken
	}

	c.Lock()
	c.users = append(c.users, user)
	c.Unlock()

	return nil
}

// SetDomain is called when this container is attached to a domain
func (c *UserContainer) SetDomain(d *Domain) error {
	if d == nil {
		return ErrNilDomain
	}

	// link this container to a given domain
	c.domain = d

	return nil
}

// Unregister user from the container
func (c *UserContainer) Unregister(id ulid.ULID) error {
	// just being explicit about the error for consistency
	// not returning just nil if the user isn't found
	if _, err := c.GetByID(id); err != nil {
		return err
	}

	c.Lock()
	for i, user := range c.users {
		if user.ID == id {
			// clearing out index maps
			delete(c.idMap, c.users[i].ID)
			delete(c.usernameMap, c.users[i].Username)
			delete(c.emailMap, c.users[i].Email)

			// removing user from the main slice
			c.users = append(c.users[0:i], c.users[i+1:]...)
			break
		}
	}
	c.Unlock()

	return nil
}

// List returns all users by a given filter
// IMPORTANT this function returns values and must be used only for listing
// i.e. returning a list of users via API
func (c *UserContainer) List() []*User {
	// TODO: implement

	return c.users
}

// GetByID returns a group by ID
func (c *UserContainer) GetByID(id ulid.ULID) (*User, error) {
	if user, ok := c.idMap[id]; ok {
		return user, nil
	}

	return nil, ErrUserNotFound
}

// GetByUsername return user by username
func (c *UserContainer) GetByUsername(username string) (*User, error) {
	if user, ok := c.usernameMap[username]; ok {
		return user, nil
	}

	return nil, ErrUserNotFound
}

// GetByEmail return user by email
func (c *UserContainer) GetByEmail(email string) (*User, error) {
	if user, ok := c.emailMap[email]; ok {
		return user, nil
	}

	return nil, ErrUserNotFound
}
