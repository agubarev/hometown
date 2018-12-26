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

// GroupStore describes a storage contract for groups specifically
// TODO add predicates for searching
type GroupStore interface {
	// groups
	PutGroup(ctx context.Context, g *Group) error
	GetGroup(ctx context.Context, id ulid.ULID) error
	GetAllGroups(ctx context.Context) ([]*Group, error)
	DeleteGroup(ctx context.Context, id ulid.ULID) error

	// group relations
	PutGroupRelation(ctx context.Context, g *Group, u *User) error
	GetGroupRelation(ctx context.Context, groupID ulid.ULID, userID ulid.ULID) (bool, error)
	GetGroupRelations(ctx context.Context, groupID ulid.ULID) ([]*Group, error)
	DeleteGroupRelation(ctx context.Context, groupID ulid.ULID, userID ulid.ULID) error
}
