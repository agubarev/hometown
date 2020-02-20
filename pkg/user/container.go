package user

import (
	"fmt"
	"sync"

	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// List a named slice type for convenience
type List []*User

// Filter filters current user list by a given predicate
func (l List) Filter(fn func(*User) bool) List {
	result := make(List, 0)

	// looping over the current user list and filtering
	// by a given function
	for _, u := range l {
		if fn(u) {
			result = append(result, u)
		}
	}

	return result
}

// Container is a container responsible for all operations within its scope
// TODO allow duplicate usernames, this shouldn't be a problem if users want the same username
// TODO add default groups which need not to be assigned
// TODO add contexts for cancellation
// TODO use radix tree for id, username and email indexing
// TODO consider storing user values inside users map and pointers in index maps
type Container struct {
	users     List
	ids       map[int]*User
	usernames map[TUsername]*User
	emails    map[TEmailAddr]*User
	manager   *Manager
	logger    *zap.Logger
	sync.RWMutex
}

// NewContainer initializing a new user container
func NewContainer() (*Container, error) {
	c := &Container{
		users:     make([]*User, 0),
		ids:       make(map[int]*User),
		usernames: make(map[TUsername]*User),
		emails:    make(map[TEmailAddr]*User),
	}

	return c, nil
}

// Manager returns a user manager to which this container is attached
func (c *Container) Manager() (*Manager, error) {
	if c.manager == nil {
		return nil, ErrNilManager
	}

	return c.manager, nil
}

// SetUserManager assigns a corresponding user manager to which
// this container is assigned to
func (c *Container) SetUserManager(m *Manager) error {
	c.manager = m

	return nil
}

// Validate this group container
func (c *Container) Validate() error {
	if c.users == nil {
		return errors.New("users slice is not initialized")
	}

	if c.ids == nil {
		return errors.New("id map is nil")
	}

	if c.usernames == nil {
		return errors.New("username map is nil")
	}

	if c.emails == nil {
		return errors.New("email map is nil")
	}

	return nil
}

// Put user runtime object to the container
func (c *Container) Add(u *User) error {
	if u == nil {
		return ErrNilUser
	}

	// zero ID means the user hasn't been created and stored to
	// the database yet, so it cannot be added to the container
	// without the ID
	if u.ID == 0 {
		return ErrZeroID
	}

	err := u.Validate()
	if err != nil {
		return err
	}

	// looking inside the container itself
	if _, err = c.GetByID(u.ID); err != ErrUserNotFound {
		return ErrUserExists
	}

	// linking user to this container
	if err = u.LinkContainer(c); err != nil {
		return err
	}

	c.Lock()
	c.users = append(c.users, u)
	c.ids[u.ID] = u
	c.usernames[u.Username] = u
	c.emails[u.Email] = u
	c.Unlock()

	return nil
}

// Remove removes runtime object from the container
func (c *Container) Remove(id int) error {
	// just being explicit about the error for consistency
	// not returning just nil if the user isn't found
	if _, err := c.GetByID(id); err != nil {
		return err
	}

	c.Lock()
	for i, user := range c.users {
		if user.ID == id {
			// clearing container reference
			c.ids[c.users[i].ID].LinkContainer(nil)

			// clearing out index maps
			delete(c.ids, c.users[i].ID)
			delete(c.usernames, c.users[i].Username)
			delete(c.emails, c.users[i].Email)

			// removing user from the main slice
			c.users = append(c.users[0:i], c.users[i+1:]...)

			break
		}
	}
	c.Unlock()

	return nil
}

// List returns all users by a given filter
// IMPORTANT: this function returns values and must be used only for listing
// i.e. returning a list of users via API
func (c *Container) List(fn func(u *User) bool) List {
	if fn == nil {
		return c.users
	}

	return c.users.Filter(fn)
}

// GetByID returns a group by ID
func (c *Container) GetByID(id int) (*User, error) {
	if user, ok := c.ids[id]; ok {
		return user, nil
	}

	return nil, ErrUserNotFound
}

// GetByUsername return user by username
func (c *Container) GetByUsername(username TUsername) (*User, error) {
	if user, ok := c.usernames[username]; ok {
		return user, nil
	}

	return nil, ErrUserNotFound
}

// GetByEmail return user by email
func (c *Container) GetByEmail(email TEmailAddr) (*User, error) {
	if user, ok := c.emails[email]; ok {
		return user, nil
	}

	return nil, ErrUserNotFound
}

// SetUserContainer assigns a user container
func (m *Manager) SetUserContainer(c *UserContainer) error {
	if c == nil {
		return ErrNilUserContainer
	}

	err := c.Validate()
	if err != nil {
		return fmt.Errorf("failed to validate user container: %s", err)
	}

	// intertwining manager and container
	if err := c.SetUserManager(m); err != nil {
		return errors.Wrap(err, "failed to set core on a container")
	}

	m.users = c

	return nil
}
