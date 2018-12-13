package user

import (
	"context"
	"errors"

	"github.com/oklog/ulid"
)

// errors
var (
	ErrUserExists = errors.New("user already exists")
	ErrNilUser    = errors.New("user is nil")
	ErrNilStore   = errors.New("store is nil")
)

// Manager represents a User manager contract interface
// TODO: add User data validation
type Manager interface {
	Exists(ctx context.Context, u *User) bool
	Create(ctx context.Context, u *User) (*User, error)
	GetByID(ctx context.Context, id ulid.ULID) (*User, error)
	Update(ctx context.Context, u *User) (*User, error)
	Delete(ctx context.Context, u *User) (*User, error)
}

// NewDefaultManager is a default user manager implementation
func NewDefaultManager(s Store) Manager {
	// user store must be set
	if s == nil {
		panic(ErrNilStore)
	}

	// default manager
	return &manager{
		store: s,
	}
}

type manager struct {
	store Store
}

// Exists checks whether a given user exists
func (m *manager) Exists(ctx context.Context, u *User) bool {
	var err error

	_, err = m.store.GetByID(ctx, u.ID)
	if err == nil {
		return true
	}

	_, err = m.store.GetByIndex(ctx, "username", u.Username)
	if err == nil {
		return true
	}

	_, err = m.store.GetByIndex(ctx, "email", u.Email)
	if err == nil {
		return true
	}

	return false
}

// Create new user
func (m *manager) Create(ctx context.Context, u *User) (*User, error) {
	if u == nil {
		return nil, ErrNilUser
	}

	// checking for existence
	if m.Exists(ctx, u) {
		return nil, ErrUserExists
	}

	err := m.store.Put(ctx, u)
	if err != nil {
		return nil, err
	}

	return u, nil
}

// Get existing user
func (m *manager) GetByID(ctx context.Context, id ulid.ULID) (*User, error) {
	return m.store.GetByID(ctx, id)
}

func (m *manager) Update(ctx context.Context, u *User) (*User, error) {
	panic("not implemented")
}

func (m *manager) Delete(ctx context.Context, u *User) (*User, error) {
	panic("not implemented")
}
