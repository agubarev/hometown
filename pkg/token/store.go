package token

import (
	"context"
)

// Store describes the token store contract interface
type Store interface {
	Put(ctx context.Context, t Token) error
	Get(ctx context.Context, hash THash) (Token, error)
	Delete(ctx context.Context, hash THash) error
}
