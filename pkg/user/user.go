package user

import (
	"time"

	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/pkg/util"
)

// User is the main entity of this project
type User struct {
	// Universally Unique Lexicographically Sortable Identifier
	ID ulid.ULID `db:"id" json:"id"`

	// either of these can be used as authentication ID
	Username string `db:"username" json:"username"`
	Email    string `db:"email" json:"email"`
}

// Metadata stores system information about a user
type Metadata struct {
	ID                      ulid.ULID `db:"id" json:"id"`
	GodMode                 bool      `db:"godmode" json:"godmode"`
	IsEnabled               bool      `db:"is_enabled" json:"is_enabled"`
	IsBlocked               bool      `db:"is_blocked" json:"is_blocked"`
	IsVerified              bool      `db:"is_verified" json:"is_verified"`
	LastLoggedAt            time.Time `db:"last_logged_at" json:"last_logged_at"`
	CreatedAt               time.Time `db:"created_at" json:"created_at"`
	UpdatedAt               time.Time `db:"updated_at,omitempty" json:"updated_at,omitempty"`
	LastLoginSuccessful     time.Time `db:"last_login_successful_at,omitempty" json:"last_login_successful_at,omitempty"`
	LastLoginFailed         time.Time `db:"last_login_failed_at,omitempty" json:"last_login_failed_at,omitempty"`
	LastLoginFailedAttempts int       `db:"last_login_failed_attempts" json:"last_login_failed_attempts"`
	VerifiedAt              time.Time `db:"verified_at" json:"verified_at"`
	BlockedAt               time.Time `db:"blocked_at" json:"blocked_at"`
	BlockExpiresAt          time.Time `db:"block_expires_at" json:"block_expires_at"`
}

// Profile stores common information about a user
type Profile struct {
	UID       ulid.ULID `db:"uid" json:"uid"`
	Firstname string    `db:"firstname" json:"firstname"`
	Lastname  string    `db:"lastname" json:"lastname"`
	Email     string    `db:"email" json:"email"`
}

// NewUser initializing a new User
func NewUser(username string, email string) *User {
	return &User{
		ID:       util.NewULID(),
		Username: username,
		Email:    email,
	}
}
