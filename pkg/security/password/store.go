package password

import (
	"context"
)

type Store interface {
	Upsert(ctx context.Context, p Password) error
	Get(ctx context.Context, o Owner) (Password, error)
	Delete(ctx context.Context, o Owner) error
}
