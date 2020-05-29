package user

import (
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
	usernames map[string]*User
	emails    map[string]*User
	manager   *Manager
	logger    *zap.Logger
	sync.RWMutex
}

// NewContainer initializing a new user container
func NewContainer() (*Container, error) {
	c := &Container{
		users:     make([]*User, 0),
		ids:       make(map[int]*User),
		usernames: make(map[string]*User),
		emails:    make(map[string]*User),
	}

	return c, nil
}

// userManager returns a user manager to which this container is attached
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

// SanitizeAndValidate this group container
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
