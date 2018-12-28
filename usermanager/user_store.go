package usermanager

import (
	"context"

	"github.com/oklog/ulid"
)

// UserStore represents a user storage contract
type UserStore interface {
	PutUser(ctx context.Context, u *User) error
	GetUserByID(ctx context.Context, id ulid.ULID) (*User, error)
	GetUserByIndex(ctx context.Context, index string, value string) (*User, error)
	DeleteUser(ctx context.Context, id ulid.ULID) error
}

// UserStoreCache is an internal user caching mechanism for a Store
type UserStoreCache interface {
	GetByID(id ulid.ULID) *User
	GetByIndex(index string, value string) *User
	Put(u *User)
	Delete(id ulid.ULID)
	Cleanup() error
}
