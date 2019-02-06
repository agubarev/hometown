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
// TODO: allow duplicate usernames, this shouldn't be a problem if users want the same username
// TODO: add default groups which need not to be assigned
// TODO: add contexts for cancellation
// TODO: use radix tree for id, username and email indexing
type UserContainer struct {
	domain      *Domain
	users       []*User
	idMap       map[ulid.ULID]*User
	usernameMap map[string]*User
	emailMap    map[string]*User
	store       UserStore
	index       bleve.Index
	passwords   PasswordManager

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

func (c *UserContainer) loadAllUsers() error {
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

// CheckAvailability tests whether someone with such username or email
// is already registered
func (c *UserContainer) CheckAvailability(username string, email string) error {
	if c.store == nil {
		return ErrNilUserStore
	}

	// runtime checks first
	_, err := c.GetByUsername(username)
	if err != nil {
		if err == ErrUserNotFound {
			return ErrUsernameTaken
		}

		return err
	}

	_, err = c.GetByEmail(email)
	if err != nil {
		if err == ErrUserNotFound {
			return ErrEmailTaken
		}

		return err
	}

	// checking storage for just in case
	_, err = c.store.GetByIndex("username", username)
	if err != nil {
		if err == ErrUserNotFound {
			return ErrUsernameTaken
		}

		return err
	}

	_, err = c.store.GetByIndex("email", email)
	if err != nil {
		if err == ErrUserNotFound {
			return ErrEmailTaken
		}

		return err
	}

	return nil
}

// Create new user
func (c *UserContainer) Create(username string, email string, userinfo map[string]string) (*User, error) {
	if c.store == nil {
		return nil, ErrNilUserContainer
	}

	// checking the availability of username and email
	if err := c.CheckAvailability(username, email); err != nil {
		return nil, err
	}

	// initializing new user
	u, err := NewUser(username, email)
	if err != nil {
		return nil, err
	}

	// adding to container's registry
	if err = c.Add(u); err != nil {
		return nil, err
	}

	// saving to the store
	if err = c.store.Put(u); err != nil {
		return nil, err
	}

	return u, nil
}

// CreateWithPassword creates a new user with a password
func (c *UserContainer) CreateWithPassword(username, email, password string, userinfo map[string]string) (*User, error) {
	// copying userinfo values into slice to improve password strength evaluation
	userdata := make([]string, 0)
	for _, v := range userinfo {
		userdata = append(userdata, v)
	}

	err := EvaluatePasswordStrength(password, userdata)
	if err != nil {
		return nil, err
	}

	// attempting to create the user first
	u, err := c.Create(username, email, userinfo)
	if err != nil {
		return nil, err
	}

	// assigning a password to this user
	if err = c.SetPassword(u, password); err != nil {
		// TODO: possibly delete the unfinished user or figure out a better way to handle

		return nil, err
	}

	return u, nil
}

// Add user to the container
func (c *UserContainer) Add(u *User) error {
	if u == nil {
		return ErrNilUser
	}

	// paranoid check
	err := u.Validate()
	if err != nil {
		return err
	}

	// looking inside the container itself
	if _, err = c.Get(u.ID); err != ErrUserNotFound {
		return ErrUserIsRegistered
	}

	c.Lock()
	c.users = append(c.users, u)
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

// SetPassword sets a new password for the user
func (c *UserContainer) SetPassword(u *User, newpass string) error {
	if u == nil {
		return ErrNilUser
	}

	if u.Domain() == nil {
		return ErrNilDomain
	}

	return nil
}

// Remove user from the container
func (c *UserContainer) Remove(id ulid.ULID) error {
	// just being explicit about the error for consistency
	// not returning just nil if the user isn't found
	if _, err := c.Get(id); err != nil {
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

// Get returns a group by ID
func (c *UserContainer) Get(id ulid.ULID) (*User, error) {
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

// SetPasswordManager assigns a password manager for this container
func (c *UserContainer) SetPasswordManager(pm PasswordManager) error {
	if pm == nil {
		return ErrNilPasswordManager
	}

	c.Passwords = pm

	return nil
}
