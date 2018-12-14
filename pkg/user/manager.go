package user

import (
	"context"
	"errors"

	"github.com/oklog/ulid"
)

// errors
var (
	ErrUserExists    = errors.New("user already exists")
	ErrNilUser       = errors.New("user is nil")
	ErrNilStore      = errors.New("store is nil")
	ErrUsernameTaken = errors.New("username is already taken")
	ErrEmailTaken    = errors.New("email is already taken")
)

// Manager represents a User manager contract interface
// TODO: add User data validation
type Manager interface {
	Create(ctx context.Context, u *User) (*User, error)
	GetByID(ctx context.Context, id ulid.ULID) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	GetByEmail(ctx context.Context, email string) (*User, error)
	Update(ctx context.Context, u *User) (*User, error)
	Delete(ctx context.Context, u *User) (*User, error)
}

// NewDefaultManager is a default user manager implementation
func NewDefaultManager(s Store) (Manager, error) {
	if s == nil {
		return nil, ErrNilStore
	}

	if err := s.Init(); err != nil {
		return nil, err
	}

	return &manager{s}, nil
}

type manager struct {
	store Store
}

// Create new user
func (m *manager) Create(ctx context.Context, u *User) (*User, error) {
	if u == nil {
		return nil, ErrNilUser
	}

	// existence checks
	_, err := m.store.GetByID(ctx, u.ID)
	if err != ErrUserNotFound {
		return nil, ErrUserExists
	}

	_, err = m.store.GetByIndex(ctx, "email", u.Email)
	if err != ErrUserNotFound {
		return nil, ErrEmailTaken
	}

	_, err = m.store.GetByIndex(ctx, "username", u.Username)
	if err != ErrUserNotFound {
		return nil, ErrUsernameTaken
	}

	// storing user
	err = m.store.Put(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// Get existing user
func (m *manager) GetByID(ctx context.Context, id ulid.ULID) (*User, error) {
	return m.store.GetByID(ctx, id)
}

// GetByUsername returns a user by username
func (m *manager) GetByUsername(ctx context.Context, username string) (*User, error) {
	return m.store.GetByIndex(ctx, "username", username)
}

// GetByEmail returns a user by email
func (m *manager) GetByEmail(ctx context.Context, email string) (*User, error) {
	return m.store.GetByIndex(ctx, "email", email)
}

func (m *manager) Update(ctx context.Context, u *User) (*User, error) {
	panic("not implemented")
}

func (m *manager) Delete(ctx context.Context, u *User) (*User, error) {
	panic("not implemented")
}
