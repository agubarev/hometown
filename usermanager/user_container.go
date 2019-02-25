package usermanager

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/blevesearch/bleve"
	"github.com/davecgh/go-spew/spew"
	"github.com/oklog/ulid"
	"go.uber.org/zap"
)

// UserFilterFunc is a filter function parameter to be passed to List() function
type UserFilterFunc func(u []*User) []*User

// UserContainer is a container responsible for all operations within its scope
// TODO: allow duplicate usernames, this shouldn't be a problem if users want the same username
// TODO: add default groups which need not to be assigned
// TODO: add contexts for cancellation
// TODO: use radix tree for id, username and email indexing
// TODO: consider storing user values inside users map and pointers in index maps
type UserContainer struct {
	domain      *Domain
	users       []*User
	idMap       map[ulid.ULID]*User
	usernameMap map[string]*User
	emailMap    map[string]*User
	store       UserStore
	index       bleve.Index
	passwords   PasswordManager
	logger      *zap.Logger
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

	return c, nil
}

// Init initializes user container
func (c *UserContainer) Init() error {
	if c.store == nil {
		return ErrNilUserStore
	}

	storedUsers, err := c.store.GetAll()
	if err != nil {
		return err
	}

	spew.Dump(storedUsers)

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

// Logger returns domain logger if domain is set, otherwise
// returns the logger of this container; will initialize a new one if it's nil
// NOTE: will panic if it finally fails to obtain a logger
func (c *UserContainer) Logger() *zap.Logger {
	if c.domain != nil {
		return c.domain.Logger()
	}

	// domain is nil, attempting to use container's own logger
	// NOTE: this is useful if the container is used outside of domain's context (i.e. another project)
	if c.logger == nil {
		l, err := zap.NewDevelopment()
		if err != nil {
			// having a working logger is crucial, thus must panic() if initialization fails
			panic(fmt.Errorf("failed to initialize container logger: %s", err))
		}

		c.logger = l
	}

	return c.logger
}

// Store returns store if set
func (c *UserContainer) Store() (UserStore, error) {
	if c.store == nil {
		return nil, ErrNilUserStore
	}

	return c.store, nil
}

// CheckAvailability tests whether someone with such username or email
// is already registered
func (c *UserContainer) CheckAvailability(username string, email string) error {
	if c.store == nil {
		return ErrNilUserStore
	}

	// runtime checks first
	_, err := c.GetByUsername(username)
	if err == nil {
		return ErrUsernameTaken
	}

	if err != ErrUserNotFound {
		return err
	}

	_, err = c.GetByEmail(email)
	if err == nil {
		return ErrEmailTaken
	}

	if err != ErrUserNotFound {
		return err
	}

	// checking storage for just in case
	_, err = c.store.GetByIndex("username", username)
	if err == nil {
		return ErrUsernameTaken
	}

	if err != ErrUserNotFound {
		return err
	}

	_, err = c.store.GetByIndex("email", email)
	if err == nil {
		return ErrEmailTaken
	}

	if err != ErrUserNotFound {
		return err
	}

	return nil
}

// Create new user
func (c *UserContainer) Create(username string, email string, userinfo map[string]string) (*User, error) {
	if c.store == nil {
		return nil, ErrNilUserStore
	}

	// checking the availability of username and email
	if err := c.CheckAvailability(username, email); err != nil {
		return nil, err
	}

	// initializing new user
	u, err := NewUser(username, email, userinfo)
	if err != nil {
		return nil, err
	}

	// saving to the store first
	if err = c.Save(u); err != nil {
		return nil, err
	}

	// adding to container's registry
	if err = c.Add(u); err != nil {
		return nil, err
	}

	return u, nil
}

// CreateWithPassword creates a new user with a password
func (c *UserContainer) CreateWithPassword(username, email, rawpass string, userinfo map[string]string) (u *User, err error) {
	if c.passwords == nil {
		return nil, ErrNilPasswordManager
	}

	// attempting to create the user first
	u, err = c.Create(username, email, userinfo)
	if err != nil {
		return nil, err
	}

	// deferring a function to delete this user if there's any error to follow
	defer func() {
		if r := recover(); r != nil {
			// at this point it doesn't matter what caused this panic, it only matters
			// to delete the created user and clean up what's unfinished
			// TODO: devise a contingency plan for OSHI- if the recovery fails
			if err := c.Delete(u); err != nil {
				// OSHI-, it happened, flap your wings
				err = fmt.Errorf("[CRITICAL] FAILED TO CREATE USER WITH PASSWORD AND FAILED TO DELETE USER DURING RECOVERY FROM PANIC: %s", err)
			}

			// attempting to pass the recovery error to an upper function's return
			err = r.(error)
		}
	}()

	// initializing and assigning a password to this user
	// copying userinfo values into slice to improve password strength evaluation
	userdata := make([]string, 0)
	for _, v := range userinfo {
		userdata = append(userdata, v)
	}

	p, err := NewPassword(u.ID, rawpass, userdata)
	if err != nil {
		panic(err)
	}

	if err = c.SetPassword(u, p); err != nil {
		// TODO: possibly delete the unfinished user or figure out a better way to handle
		// NOTE: at this point this account cannot be allowed to stay without a password
		// because there's an explicit attempt to create a new user account WITH PASSWORD,
		// so it has to be deleted in case of an error, panic now.
		// NOTE: runtime must recover from panic, run it's contingency plan and set a proper error
		// for return
		panic(fmt.Errorf("failed to set user password: %s", err))
	}

	return
}

// Delete deletes user from the store and container
func (c *UserContainer) Delete(u *User) error {
	// checking the store in advance
	if c.store == nil {
		return fmt.Errorf("Delete(): %s", ErrNilUserStore)
	}

	// NOTE: if the user doesn't exist then returning an error for
	// consistent explicitness
	_, err := c.Get(u.ID)
	if err != nil {
		return fmt.Errorf("Delete(): failed to delete user %s: %s", u.ID, err)
	}

	// now deleting user from the store
	err = c.store.Delete(u.ID)
	if err != nil {
		return fmt.Errorf("Delete() failed to delete user from the store: %s", err)
	}

	// removing runtime object
	err = c.Remove(u.ID)
	if err != nil {
		return fmt.Errorf("Delete(): failed to delete user %s: %s", u.ID, err)
	}

	// and finally deleting user's password if the password manager is present
	// NOTE: it should be possible that the user could not have a password
	if c.passwords != nil {
		err = c.passwords.Delete(u)
		if err != nil {
			return fmt.Errorf("Delete(): failed to delete user password: %s", err)
		}
	}

	return nil
}

// Add user runtime object to the container
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
		return ErrUserAlreadyRegistered
	}

	// save user to the store if it's set
	if store, err := c.Store(); err == nil {
		// checking by user ID
		_, err := store.Get(u.ID)
		if err != nil {
			if err != ErrUserNotFound {
				return err
			}

			return ErrUserAlreadyRegistered
		}

		// checking username and email
		err = c.CheckAvailability(u.Username, u.Email)
		if err != nil {
			return err
		}

		// saving to the store
		err = store.Put(u)
		if err != nil {
			return err
		}
	}

	// linking user to this container
	if err = u.TrackContainer(c); err != nil {
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
func (c *UserContainer) Remove(id ulid.ULID) error {
	// just being explicit about the error for consistency
	// not returning just nil if the user isn't found
	if _, err := c.Get(id); err != nil {
		return err
	}

	c.Lock()
	for i, user := range c.users {
		if user.ID == id {
			// clearing container reference
			c.idMap[c.users[i].ID].TrackContainer(nil)

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
func (c *UserContainer) SetPassword(u *User, p *Password) error {
	if u == nil {
		return fmt.Errorf("SetPassword(): %s", ErrNilUser)
	}

	// paranoid check of whether the user is eligible to have
	// a password created and stored
	ok, err := u.IsRegisteredAndStored()
	if err != nil {
		return fmt.Errorf("SetPassword(): %s", err)
	}

	if !ok {
		return fmt.Errorf("SetPassword(): %s", ErrUserPasswordNotEligible)
	}

	// storing password
	// NOTE: PasswordManager is responsible for hashing and encryption
	if err = c.passwords.Set(u, p); err != nil {
		return fmt.Errorf("SetPassword(): failed to set password: %s", err)
	}

	return nil
}

// Save saves user to the store, will return an error if store is not set
func (c *UserContainer) Save(u *User) error {
	s, err := c.Store()
	if err != nil {
		return err
	}

	// pre-save modifications
	u.UpdatedAt = time.Now()

	// refreshing index
	if c.index != nil {
		// deleting previous index
		// TODO: workaround error cases, add logging
		err = c.index.Delete(u.StringID())
		if err != nil {
			return fmt.Errorf("failed to delete previous bleve index: %s", err)
		}

		// indexing current user object version
		err = c.index.Index(u.StringID(), u)
		if err != nil {
			return fmt.Errorf("failed to index user object: %s", err)
		}
	}

	// persisting to the store as a final step
	err = s.Put(u)
	if err != nil {
		return err
	}

	c.Logger().Info("user saved", zap.String("id", u.StringID()), zap.String("username", u.Username))

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

	c.passwords = pm

	return nil
}

// ConfirmEmail this function is used only when user's email is confirmed
// TODO: make it one function to confirm by type (i.e. email, phone, etc.)
func (c *UserContainer) ConfirmEmail(u *User) error {
	if u == nil {
		return ErrNilUser
	}

	if !u.EmailConfirmedAt.IsZero() {
		return ErrUserAlreadyConfirmed
	}

	u.EmailConfirmedAt = time.Now()

	if err := c.Save(u); err != nil {
		return fmt.Errorf("failed to confirm user email(%s:%s): %s", u.ID, u.Email, err)
	}

	c.Logger().Info("user email confirmed",
		zap.String("id", u.StringID()),
		zap.String("email", u.Email),
	)

	return nil
}
