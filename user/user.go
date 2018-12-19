package user

import (
	"time"

	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/user/util"
)

// User represents a user account, a unique entity
// TODO: workout the len restrictions (not set fixed lengths but need to be checked elsewhere)
type User struct {
	// Universally Unique Lexicographically Sortable Identifier
	ID ulid.ULID `json:"id"`

	// either of these can be used as authentication ID
	Username string `json:"username"`
	Email    string `json:"email"`
}

// Metadata about the User
type Metadata struct {
	IsEnabled               bool      `json:"is_enabled"`
	IsBlocked               bool      `json:"is_blocked"`
	IsActivated             bool      `json:"is_activated"`
	IsVerified              bool      `json:"is_verified"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at,omitempty"`
	ActivatedAT             time.Time `json:"activated_at,omitempty"`
	LastLoggedAt            time.Time `json:"last_logged_at,omitempty"`
	LastLoginSuccessful     time.Time `json:"last_login_successful_at,omitempty"`
	LastLoginFailed         time.Time `json:"last_login_failed_at,omitempty"`
	LastLoginFailedAttempts int       `json:"last_login_failed_attempts"`
	VerifiedAt              time.Time `json:"verified_at,omitempty"`
	BlockedAt               time.Time `json:"blocked_at,omitempty"`
	BlockExpiresAt          time.Time `json:"block_expires_at,omitempty"`
}

// NewUser initializing a new User
func NewUser(username string, email string) *User {
	return &User{
		ID:       util.NewULID(),
		Username: username,
		Email:    email,
	}
}
