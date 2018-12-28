package usermanager

import (
	"context"

	"github.com/oklog/ulid"
)

// GroupStore describes a storage contract for groups specifically
// TODO add predicates for searching
type GroupStore interface {
	// groups
	Put(ctx context.Context, g *Group) error
	GetByID(ctx context.Context, id ulid.ULID) error
	GetAll(ctx context.Context) ([]*Group, error)
	Delete(ctx context.Context, id ulid.ULID) error

	// group relations
	PutRelation(ctx context.Context, g *Group, u *User) error
	GetAllRelations(ctx context.Context) ([]*Group, error)
	DeleteRelation(ctx context.Context, groupID ulid.ULID, userID ulid.ULID) error
}
