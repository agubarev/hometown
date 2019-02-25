package usermanager

import (
	"fmt"
	"net"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

// User represents a user account, a unique entity
// TODO: workout the length restrictions
type User struct {
	ID ulid.ULID `json:"id"`

	// Username and Email are the primary IDs associated with the user account
	Username string `json:"username" valid:"required,alphanum"`
	Email    string `json:"email" valid:"required,email"`

	// the name, birthdate, country
	Firstname  string    `json:"firstname" valid:"optional,utfletter"`
	Lastname   string    `json:"lastname" valid:"optional,utfletter"`
	Middlename string    `json:"middlename" valid:"optional,utfletter"`
	Birthdate  time.Time `json:"birthdate,omitempty"`
	Phone      string    `json:"phone" valid:"optional,dialstring"`

	// account confirmation
	EmailConfirmedAt time.Time `json:"email_confirmed_at"`
	PhoneConfirmedAt time.Time `json:"phone_confirmed_at"`

	// general timestamps
	CreatedAt   time.Time `json:"t_cr"`
	UpdatedAt   time.Time `json:"t_up,omitempty"`
	ConfirmedAt time.Time `json:"t_co,omitempty"`

	// the most recent authentication information
	LoginAt       time.Time  `json:"login_at,omitempty"`
	LoginIP       net.IPAddr `json:"login_ip,omitempty"`
	LoginFailedAt time.Time  `json:"login_failed_at,omitempty"`
	LoginFailedIP net.IPAddr `json:"login_failed_ip,omitempty"`
	LoginAttempts int        `json:"login_attempts,omitempty"`

	// account suspension
	IsSuspended         bool      `json:"is_suspended"`
	SuspendedAt         time.Time `json:"suspended_at,omitempty"`
	SuspensionExpiresAt time.Time `json:"suspension_expires_at,omitempty"`
	SuspensionReason    string    `json:"suspension_reason,omitempty"`

	// corresponding container for easier backtracking
	container *UserContainer

	// tracking all group kinds in one slice
	groups []*Group
}

// StringInfo returns short info about the user
func (u *User) StringInfo() string {
	return fmt.Sprintf("user(%s:%s)", u.ID, u.Username)
}

// StringID returns string representation of ULID
func (u *User) StringID() string {
	return u.ID.String()
}

// Container returns the corresponding user container to
// which this user belongs
func (u *User) Container() (*UserContainer, error) {
	if u.container == nil {
		return nil, ErrNilUserContainer
	}

	return u.container, nil
}

// Fullname returns full name of a user
func (u User) Fullname(withMiddlename bool) string {
	if withMiddlename {
		return fmt.Sprintf("%s %s %s", u.Firstname, u.Middlename, u.Lastname)
	}

	return fmt.Sprintf("%s %s", u.Firstname, u.Lastname)
}

// NewUser initializing a new User
// TODO: consider changing userinfo value type to interface{} to allow variable data
func NewUser(username string, email string, userinfo map[string]string) (*User, error) {
	u := &User{
		ID:        util.NewULID(),
		Username:  username,
		Email:     email,
		CreatedAt: time.Now(),

		groups: make([]*Group, 0),
	}

	// processing given userinfo
	for k, v := range userinfo {
		k = strings.ToLower(k)

		// whitelist
		switch k {
		case "firstname":
			u.Firstname = v
		case "lastname":
			u.Lastname = v
		case "middlename":
			u.Middlename = v
		default:
			return nil, fmt.Errorf("unrecognized user info field: %s", k)
		}
	}

	if err := u.Validate(); err != nil {
		return nil, err
	}

	return u, nil
}

// Validate user object
func (u *User) Validate() error {
	if u == nil {
		return ErrNilUser
	}

	if ok, err := govalidator.ValidateStruct(u); !ok || err != nil {
		return fmt.Errorf("%s validation failed: %s", u.StringInfo(), err)
	}

	return nil
}

// HasPassword tests whether this user has password
func (u *User) HasPassword() bool {
	c, err := u.Container()
	if err != nil {
		return false
	}

	if err = c.Validate(); err != nil {
		return false
	}

	if _, err = c.passwords.Get(u); err != nil {
		return false
	}

	return true
}

// Save saves current user to the container's store
func (u *User) Save() error {
	if u == nil {
		return ErrNilUser
	}

	c, err := u.Container()
	if err != nil {
		return err
	}

	err = c.Save(u)
	if err != nil {
		return err
	}

	return nil
}

// TrackGroup tracking which groups this user is a member of
func (u *User) TrackGroup(g *Group) error {
	if g == nil {
		return ErrNilGroup
	}

	// safeguard in case this slice is not initialized
	if u.groups == nil {
		u.groups = make([]*Group, 0)
	}

	// appending group to slice for easier runtime access
	u.groups = append(u.groups, g)

	return nil
}

// UntrackGroup removing group from the tracklist
func (u *User) UntrackGroup(id ulid.ULID) error {
	if u.groups == nil {
		// initializing just in case
		u.groups = make([]*Group, 0)

		return nil
	}

	// removing group from the tracklist
	for i, g := range u.groups {
		if g.ID == id {
			u.groups = append(u.groups[0:i], u.groups[i+1:]...)
			break
		}
	}

	return ErrGroupNotFound
}

// Groups to which the user belongs
func (u *User) Groups(kind GroupKind) []*Group {
	groups := make([]*Group, 0)
	for _, g := range u.groups {
		if g.Kind == kind {
			groups = append(groups, g)
		}
	}

	return groups
}

// TrackContainer links user to container for easier backtracking
func (u *User) TrackContainer(c *UserContainer) error {
	u.container = c

	return nil
}

// IsRegisteredAndStored returns true if the user is both:
// 1. registered by user container
// 2. stored to the database
func (u *User) IsRegisteredAndStored() (bool, error) {
	c, err := u.Container()
	if err != nil {
		return false, fmt.Errorf("IsRegisteredAndStored(): failed to obtain user container: %s", err)
	}

	// checking whether the user is registered during runtime
	if _, err = c.Get(u.ID); err != nil {
		if err == ErrUserNotFound {
			// user isn't registered, normal return
			return false, nil
		}

		// unexpected error
		return false, fmt.Errorf("IsRegisteredAndStored(): %s", err)
	}

	// checking container's store
	if _, err = c.store.Get(u.ID); err != nil {
		if err == ErrUserNotFound {
			// user isn't in the store yet, normal return
			return false, nil
		}

		// unexpected error
		return false, fmt.Errorf("IsRegisteredAndStored(): %s", err)
	}

	return true, nil
}
