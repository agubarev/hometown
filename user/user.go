package user

import (
	"net"
	"time"

	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

// User represents a user account, a unique entity
// TODO: workout the len restrictions (not set fixed lengths but need to be checked elsewhere)
type User struct {
	ID     ulid.ULID `json:"id"`
	Domain *Domain   `json:"-"`

	// Username and Email are the primary IDs associated with the user account
	Username    string `json:"username"`
	Email       string `json:"email"`
	IsConfirmed bool   `json:"is_verified"`
	IsSuspended bool   `json:"is_suspended"`

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
func NewUser(domain *Domain, username string, email string, isConfirmed bool) *User {
	// TODO: if given domain is nil then default domain must be used, domain must always be present

	return &User{
		Domain:      domain,
		ID:          util.NewULID(),
		Username:    username,
		Email:       email,
		IsConfirmed: isConfirmed,
		IsSuspended: false,
		Metadata:    NewMetadata(),
	}
}

// NewMetadata initializing user metadata
func NewMetadata() *Metadata {
	return &Metadata{
		CreatedAt:   time.Now(),
		ConfirmedAt: time.Now(),
	}
}

// Roles to which the user belongs
func (u *User) Roles() []*Role {
	return u.Domain.RoleContainer().GetByUser(u)
}

// Groups to which the user belongs
func (u *User) Groups() []*Group {
	return u.Domain.GroupContainer().GetByUser(u)
}
