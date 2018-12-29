package usermanager

import (
	"context"

	"github.com/oklog/ulid"
)

// Service represents a User manager contract interface
type Service interface {
	Create(ctx context.Context, u *User) (*User, error)
	GetByID(ctx context.Context, id ulid.ULID) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	SetUsername(ctx context.Context, id ulid.ULID, username string) error
	Delete(ctx context.Context, id ulid.ULID) error
}

// NewUserService is a default user manager implementation
func NewUserService(s UserStore) (Service, error) {
	if s == nil {
		return nil, ErrNilUserStore
	}

	return &service{s}, nil
}

type service struct {
	store UserStore
}

// Create new user
func (s *service) Create(ctx context.Context, u *User) (*User, error) {
	if u == nil {
		return nil, ErrNilUser
	}

	// existence checks
	_, err := s.store.GetByID(ctx, u.ID)
	if err != ErrUserNotFound {
		return nil, ErrUserExists
	}

	_, err = s.store.GetByIndex(ctx, "email", u.Email)
	if err != ErrUserNotFound {
		return nil, ErrEmailTaken
	}

	_, err = s.store.GetByIndex(ctx, "username", u.Username)
	if err != ErrUserNotFound {
		return nil, ErrUsernameTaken
	}

	// storing user
	err = s.store.Put(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// Get existing user
func (s *service) GetByID(ctx context.Context, id ulid.ULID) (*User, error) {
	return s.store.GetByID(ctx, id)
}

// GetByUsername returns a user by username
func (s *service) GetByUsername(ctx context.Context, username string) (*User, error) {
	return s.store.GetByIndex(ctx, "username", username)
}

// GetByEmail returns a user by email
func (s *service) GetByEmail(ctx context.Context, email string) (*User, error) {
	return s.store.GetByIndex(ctx, "email", email)
}

// SetUsername update username for an existing user
// TODO: username validation
func (s *service) SetUsername(ctx context.Context, id ulid.ULID, username string) error {
	u, err := s.store.GetByID(ctx, id)
	if err != nil {
		return ErrUserNotFound
	}

	// lookup existing user by that username
	eu, err := s.store.GetByIndex(ctx, "username", username)
	if eu != nil {
		return ErrUsernameTaken
	}

	// setting new username and updating
	u.Username = username

	// updating
	err = s.store.Put(ctx, u)
	if err != nil {
		return err
	}

	return nil
}

// Delete user
func (s *service) Delete(ctx context.Context, id ulid.ULID) error {
	return s.store.Delete(ctx, id)
}