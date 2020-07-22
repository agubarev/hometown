package password

import (
	"context"

	"github.com/google/uuid"
)

// Store interface
// NOTE: ownerID represents the ObjectID of whoever owns a given password
type Store interface {
	Upsert(ctx context.Context, p Password) error
	Get(ctx context.Context, k Kind, ownerID uuid.UUID) (Password, error)
	Delete(ctx context.Context, k Kind, ownerID uuid.UUID) error
}
