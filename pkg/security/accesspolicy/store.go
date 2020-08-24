package accesspolicy

import (
	"context"

	"github.com/google/uuid"
)

// Store is a storage contract interface for the Policy objects
// TODO: keep rights separate and segregated by it's kind i.e. Public, Policy, Role, User etc.
type Store interface {
	CreatePolicy(ctx context.Context, p Policy, r *Roster) (Policy, *Roster, error)
	UpdatePolicy(ctx context.Context, p Policy, r *Roster) error
	FetchPolicyByID(ctx context.Context, id uuid.UUID) (Policy, error)
	FetchPolicyByKey(ctx context.Context, key string) (p Policy, err error)
	FetchPolicyByObject(ctx context.Context, obj Object) (p Policy, err error)
	DeletePolicy(ctx context.Context, p Policy) error
	CreateRoster(ctx context.Context, policyID uuid.UUID, r *Roster) (err error)
	FetchRosterByPolicyID(ctx context.Context, pid uuid.UUID) (r *Roster, err error)
	UpdateRoster(ctx context.Context, pid uuid.UUID, r *Roster) (err error)
	DeleteRoster(ctx context.Context, pid uuid.UUID) (err error)
}
