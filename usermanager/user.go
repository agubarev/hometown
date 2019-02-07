package usermanager

import (
	"fmt"
	"net"
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

	// the name
	Firstname  string `json:"firstname" valid:"optional,alpha"`
	Lastname   string `json:"lastname" valid:"optional,alpha"`
	Middlename string `json:"middlename" valid:"optional,alpha"`

	// account related flags
	IsConfirmed bool `json:"is_verified"`
	IsSuspended bool `json:"is_suspended"`

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
	SuspendedAt         time.Time `json:"suspended_at,omitempty"`
	SuspensionExpiresAt time.Time `json:"suspension_expires_at,omitempty"`
	SuspensionReason    string    `json:"suspension_reason,omitempty"`

	// corresponding container
	container *UserContainer

	// tracking all group kinds in one slice
	groups []*Group
}

// StringID returns short info about the user
func (u *User) StringID() string {
	return fmt.Sprintf("user(%s:%s)", u.ID, u.Username)
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
func NewUser(username string, email string) (*User, error) {
	u := &User{
		ID:       util.NewULID(),
		Username: username,
		Email:    email,

		// metadata
		IsConfirmed: false,
		IsSuspended: false,
		CreatedAt:   time.Now(),

		groups: make([]*Group, 0),
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
		return fmt.Errorf("%s validation failed: %s", u.StringID(), err)
	}

	return nil
}

// Save saves current user to the container's store
func (u *User) Save() error {
	if u == nil {
		return ErrNilUser
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

// IsRegisteredAndStored returns true if the user is both:
// 1. registered by user container
// 2. persisted to the store
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
	if _, err = c.Store().Get(u.ID); err != nil {
		if err == ErrUserNotFound {
			// user isn't in the store yet, normal return
			return false, nil
		}

		// unexpected error
		return false, fmt.Errorf("IsRegisteredAndStored(): %s", err)
	}

	return true, nil
}
