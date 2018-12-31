package usermanager

import (
	"fmt"

	"github.com/oklog/ulid"
	"go.etcd.io/bbolt"
)

// AccessPolicyStore is a storage contract interface for the AccessPolicy objects
type AccessPolicyStore interface {
	Put(p *AccessPolicy)
	GetByID(id ulid.ULID) *AccessPolicy
	GetByUser(u *User) []*AccessPolicy
	GetByNamespace(ns string) []*AccessPolicy
	GetAll() []*AccessPolicy
}

// DefaultAccessPolicyStore is a default access policy store implementation
type DefaultAccessPolicyStore struct {
	db *bbolt.DB
}

// NewDefaultAccessPolicyStore returns an initialized default domain store
func NewDefaultAccessPolicyStore(db *bbolt.DB) (AccessPolicyStore, error) {
	s := &DefaultAccessPolicyStore{db}
	err := s.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("ACCESSPOLICY"))
		if err != nil {
			return fmt.Errorf("failed to create a access policy bucket: %s", err)
		}

		// TODO: implement

		return nil
	})

	return s, err
}
