package usermanager

import (
	"database/sql"

	"github.com/jmoiron/sqlx"
)

// TokenStore describes the token store contract interface
type TokenStore interface {
	Put(t *Token) error
	Get(token string) (*Token, error)
	Delete(token string) error
}

type tokenStore struct {
	db *sqlx.DB
}

// NewTokenStore initializes and returns a new default token store
func NewTokenStore(db *sqlx.DB) (TokenStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &tokenStore{db}, nil
}

// Put puts token into a store
func (s *tokenStore) Put(t *Token) error {
	if s.db == nil {
		return ErrNilDB
	}

	if t == nil {
		return ErrNilToken
	}

	// query statement
	q := `INSERT INTO tokens(kind, token, payload, c_total, c_remainder, created_at, expire_at) 
			VALUES(:kind, :token, :payload, :c_total, :c_remainder, :created_at, :expire_at)`

	_, err := s.db.NamedExec(q, &t)
	if err != nil {
		return err
	}

	return nil
}

// Get retrieves token from a store
func (s *tokenStore) Get(token string) (*Token, error) {
	if s.db == nil {
		return nil, ErrNilDB
	}

	t := new(Token)

	err := s.db.Get(t, "SELECT * FROM tokens WHERE token = ? LIMIT 1", token)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrTokenNotFound
		}

		return nil, err
	}

	return t, nil
}

// Delete deletes token from a store
func (s *tokenStore) Delete(token string) error {
	if s.db == nil {
		return ErrNilDB
	}

	res, err := s.db.Exec("DELETE FROM tokens WHERE token = ? LIMIT 1", token)
	if err != nil {
		return err
	}

	// checking whether anything was updated at all
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}

	// if no rows were affected then returning this as a non-critical error
	if ra == 0 {
		return ErrNothingChanged
	}

	return nil
}
