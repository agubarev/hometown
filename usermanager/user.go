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
// TODO: workout the length restrictions (not set fixed lengths but need to be checked elsewhere)
type User struct {
	ID     ulid.ULID `json:"id"`
	Domain *Domain   `json:"-"`

	// Username and Email are the primary IDs associated with the user account
	Username string `json:"username"`
	Email    string `json:"email"`

	// the name
	Firstname  string `json:"firstname"`
	Lastname   string `json:"lastname"`
	Middlename string `json:"middlename"`

	// most common flags
	IsConfirmed bool `json:"is_verified"`
	IsSuspended bool `json:"is_suspended"`

	// account metadata
	Metadata *Metadata `json:"-"`

	// tracking all group kinds in one slice
	groups []*Group
}

// Metadata information about the user account
type Metadata struct {
	// general timestamps
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at,omitempty"`
	ConfirmedAt time.Time `json:"confirmed_at,omitempty"`

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
}

// IDString returns short info about the user
func (u User) IDString() string {
	return fmt.Sprintf("user[%s:%s]", u.ID, u.Username)
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
		ID:          util.NewULID(),
		Username:    username,
		Email:       email,
		IsSuspended: false,
		Metadata:    NewMetadata(),
		groups:      make([]*Group, 0),
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
		return fmt.Errorf("user [%s:%s] validation failed: %s", u.ID, u.Username, err)
	}

	return nil
}

// NewMetadata initializing user metadata
func NewMetadata() *Metadata {
	return &Metadata{
		CreatedAt:   time.Now(),
		ConfirmedAt: time.Now(),
	}
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

	// finding position
	// TODO: extract to util function to delete slice items by something i.e. ID
	var pos int
	for i, g := range u.groups {
		if g.ID == id {
			pos = i
			break
		}
	}

	// removing group from the tracklist
	u.groups = append(u.groups[0:pos], u.groups[pos+1:]...)

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
