package usermanager

import (
	"context"
	"fmt"

	"github.com/oklog/ulid"
)

// Service represents a User manager contract interface
type Service interface {
	Register(ctx context.Context, req NewUserRequest) (*User, error)
	Unregister(ctx context.Context, u *User) error
	SetUsername(ctx context.Context, id ulid.ULID, username string) error
	GetByID(ctx context.Context, id ulid.ULID) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
}

// NewUserRequest holds data necessary to create a new user
type NewUserRequest struct {
	// origin an object that contains identity provider
	// info along with user-specific data
	// TODO: change origin type
	Origin   string `json:"origin"`
	Username string `json:"username" valid:""`
	Email    string `json:"email"`
	Password string `json:"password"`
}

// NewUserService is a default user manager implementation
func NewUserService(m *UserManager) (Service, error) {
	if m == nil {
		return nil, ErrNilUserManager
	}

	return &service{m}, nil
}

type service struct {
	m *UserManager
}

// Register new user
func (s *service) Register(ctx context.Context, d *Domain, req *NewUserRequest) (*User, error) {
	if d == nil {
		return nil, ErrNilDomain
	}

	if req == nil {
		return nil, fmt.Errorf("new user request is nil")
	}

	newUser, err := NewUser(req.Username, req.Email)
	if err != nil {
		return nil, fmt.Errorf("failed to create new user: %s", err)
	}

	err = d.Users.Register(newUser)
	if err != nil {
		return nil, fmt.Errorf("failed to register new user: %s", err)
	}

	// TODO: create password if required
	// TODO: send verification email

	return newUser, nil
}

// Get existing user
func (s *service) GetByID(ctx context.Context, d *Domain, userID ulid.ULID) (*User, error) {
	return d.Users.GetByID(userID)
}

// GetByUsername returns a user by username
func (s *service) GetByUsername(ctx context.Context, d *Domain, username string) (*User, error) {
	return d.Users.GetByUsername(username)
}

// GetByEmail returns a user by email
func (s *service) GetByEmail(ctx context.Context, d *Domain, email string) (*User, error) {
	return d.Users.GetByEmail(email)
}

// SetUsername update username for an existing user
// TODO: username validation
func (s *service) SetUsername(ctx context.Context, d *Domain, userID ulid.ULID, newUsername string) error {
	u, err := d.Users.GetByID(userID)
	if err != nil {
		return ErrUserNotFound
	}

	// lookup existing user by that username
	eu, err := d.Users.GetByUsername(newUsername)
	if eu != nil {
		return ErrUsernameTaken
	}

	// setting new username and updating
	u.Username = newUsername

	if err = u.Save(); err != nil {
		return err
	}

	return nil
}

// Unregister unregisters user
func (s *service) Unregister(ctx context.Context, d *Domain, userID ulid.ULID) error {
	return d.Users.Unregister(userID)
}
