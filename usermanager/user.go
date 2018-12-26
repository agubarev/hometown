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

// NewUser initializing a new User
// TODO consider attaching user to a user container instead of a domain, just like groups
func NewUser(username string, email string) (*User, error) {
	u := &User{
		ID:          util.NewULID(),
		Username:    username,
		Email:       email,
		IsSuspended: false,
		Metadata:    NewMetadata(),
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

// Roles to which the user belongs
func (u *User) Roles() []*Group {
	roleGroups, err := u.Domain.Groups.GetByUser(GKRole, u)
	if err != nil {
		return []*Group{}
	}

	return roleGroups
}

// Groups to which the user belongs
func (u *User) Groups() []*Group {
	groups, err := u.Domain.Groups.GetByUser(GKGroup, u)
	if err != nil {
		return []*Group{}
	}

	return groups
}
