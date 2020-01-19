package usermanager

import (
	"errors"
	"sync"

	"go.uber.org/zap"
)

// UserList a named slice type for convenience
type UserList []*User

// Filter filters current user list by a given predicate
func (l UserList) Filter(fn func(*User) bool) UserList {
	result := make(UserList, 0)

	// looping over the current user list and filtering
	// by a given function
	for _, u := range l {
		if fn(u) {
			result = append(result, u)
		}
	}

	return result
}

// UserContainer is a container responsible for all operations within its scope
// TODO allow duplicate usernames, this shouldn't be a problem if users want the same username
// TODO add default groups which need not to be assigned
// TODO add contexts for cancellation
// TODO use radix tree for id, username and email indexing
// TODO consider storing user values inside users map and pointers in index maps
type UserContainer struct {
	users       UserList
	idMap       map[int64]*User
	usernameMap map[string]*User
	emailMap    map[string]*User
	logger      *zap.Logger
	manager     *UserManager
	sync.RWMutex
}

// NewUserContainer initializing a new user container
func NewUserContainer() (*UserContainer, error) {
	c := &UserContainer{
		users:       make([]*User, 0),
		idMap:       make(map[int64]*User),
		usernameMap: make(map[string]*User),
		emailMap:    make(map[string]*User),
	}

	return c, nil
}

// Manager returns a user manager to which this container is attached
func (c *UserContainer) Manager() (*UserManager, error) {
	if c.manager == nil {
		return nil, ErrNilUserManager
	}

	return c.manager, nil
}

// SetUserManager assigns a corresponding user manager to which
// this container is assigned to
func (c *UserContainer) SetUserManager(m *UserManager) error {
	c.manager = m

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

// Add user runtime object to the container
func (c *UserContainer) Add(u *User) error {
	if u == nil {
		return ErrNilUser
	}

	// zero ID means the user hasn't been created and stored to
	// the database yet, so it cannot be added to the container
	// without the ID
	if u.ID == 0 {
		return ErrNoID
	}

	err := u.Validate()
	if err != nil {
		return err
	}

	// looking inside the container itself
	if _, err = c.GetByID(u.ID); err != ErrUserNotFound {
		return ErrUserAlreadyRegistered
	}

	// linking user to this container
	if err = u.LinkContainer(c); err != nil {
		return err
	}

	c.Lock()
	c.users = append(c.users, u)
	c.idMap[u.ID] = u
	c.usernameMap[u.Username] = u
	c.emailMap[u.Email] = u
	c.Unlock()

	return nil
}

// Remove removes runtime object from the container
func (c *UserContainer) Remove(id int64) error {
	// just being explicit about the error for consistency
	// not returning just nil if the user isn't found
	if _, err := c.GetByID(id); err != nil {
		return err
	}

	c.Lock()
	for i, user := range c.users {
		if user.ID == id {
			// clearing container reference
			c.idMap[c.users[i].ID].LinkContainer(nil)

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
func (c *UserContainer) List(fn func(u *User) bool) UserList {
	if fn == nil {
		return c.users
	}

	return c.users.Filter(fn)
}

// GetByID returns a group by ID
func (c *UserContainer) GetByID(id int64) (*User, error) {
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
