package role

import (
	"context"

	"github.com/oklog/ulid"
)

// Store contract for Role manager
type Store interface {
	Init() error
	PutRole(ctx context.Context, r *Role)
	PutDomain(ctx context.Context, r *RoleDomain) error
	GetByID(ctx context.Context, id ulid.ULID) (*Role, error)
	GetByDomainID(ctx context.Context, id ulid.ULID) (*Role, error)
	Delete(ctx context.Context, id ulid.ULID) error
}

type store struct {
}
