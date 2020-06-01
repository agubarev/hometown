package token

import (
	"context"

	"github.com/agubarev/hometown/pkg/util/guard"
	"github.com/gocraft/dbr/v2"
)

// Store describes the token store contract interface
type Store interface {
	Put(ctx context.Context, t *Token) error
	Get(ctx context.Context, token string) (*Token, error)
	DeleteByToken(ctx context.Context, token string) error
}

type tokenStore struct {
	db *dbr.Connection
}

// NewTokenStore initializes and returns a new default token store
func NewTokenStore(db *dbr.Connection) (Store, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}

	return &tokenStore{db}, nil
}

// UpdatePolicy puts token into a store
func (s *tokenStore) Put(ctx context.Context, t *Token) error {
	if t == nil {
		return ErrNilToken
	}

	_, err := s.db.NewSession(nil).
		InsertInto("token").
		Columns(guard.DBColumnsFrom(t)...).
		Record(t).
		ExecContext(ctx)

	if err != nil {
		return err
	}

	return nil
}

// Get retrieves token from a store
func (s *tokenStore) Get(ctx context.Context, token string) (*Token, error) {
	t := new(Token)

	err := s.db.NewSession(nil).
		SelectBySql("SELECT * FROM token WHERE token = ? LIMIT 1", token).
		LoadOneContext(ctx, t)

	if err != nil {
		if err == dbr.ErrNotFound {
			return nil, ErrTokenNotFound
		}

		return nil, err
	}

	return t, nil
}

// DeletePolicy deletes token from a store
func (s *tokenStore) DeleteByToken(ctx context.Context, token string) error {
	result, err := s.db.NewSession(nil).
		DeleteFrom("token").
		Where("token = ?", token).
		ExecContext(ctx)

	if err != nil {
		return err
	}

	// checking whether anything was updated at all
	ra, err := result.RowsAffected()
	if err != nil {
		return err
	}

	// if no rows were affected then returning this as a non-critical error
	if ra == 0 {
		return ErrNothingChanged
	}

	return nil
}
