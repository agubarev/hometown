package usermanager

import (
	"fmt"

	"github.com/dgraph-io/badger"
)

// TokenStore describes the token store contract interface
type TokenStore interface {
	Put(t *Token) error
	Get(token string) (*Token, error)
	Delete(token string) error
}

func tokenKey(token string) []byte {
	return []byte(fmt.Sprintf("t:%s", token))
}

type defaultTokenStore struct {
	db *badger.DB
}

// NewDefaultTokenStore initializes and returns a new default token store
func NewDefaultTokenStore(db *badger.DB) (TokenStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &defaultTokenStore{db}, nil
}

// Put puts token into a store
func (s *defaultTokenStore) Put(t *Token) error {
	if s.db == nil {
		return ErrNilDB
	}

	if t == nil {
		return ErrNilToken
	}

	buf, err := json.Marshal(t)
	if err != nil {
		return err
	}

	// it's pretty much straightforward here
	return s.db.Update(func(tx *badger.Txn) error {
		return tx.Set(tokenKey(t.Token), buf)
	})
}

// Get retrieves token from a store
func (s *defaultTokenStore) Get(token string) (*Token, error) {
	if s.db == nil {
		return nil, ErrNilDB
	}

	t := new(Token)

	err := s.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(tokenKey(token))
		if err != nil {
			return err
		}

		return item.Value(func(payload []byte) error {
			return json.Unmarshal(payload, t)
		})
	})

	if err != nil {
		if err != badger.ErrKeyNotFound {
			return nil, err
		}

		return nil, ErrTokenNotFound
	}

	return t, nil
}

// Delete deletes token from a store
func (s *defaultTokenStore) Delete(token string) error {
	if s.db == nil {
		return ErrNilDB
	}

	return s.db.Update(func(tx *badger.Txn) error {
		return tx.Delete(tokenKey(token))
	})
}
