package usermanager

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/oklog/ulid"
	"go.etcd.io/bbolt"
)

// AccessPolicyStore is a storage contract interface for the AccessPolicy objects
type AccessPolicyStore interface {
	Put(ctx context.Context, p *AccessPolicy)
	GetByID(ctx context.Context, id ulid.ULID) *AccessPolicy
	GetByUser(ctx context.Context, userID ulid.ULID) []*AccessPolicy
	GetByNamespace(ctx context.Context, ns string) []*AccessPolicy
	GetRightByUserID(ctx context.Context, userID ulid.ULID) []*RightsRoster
	GetAll(ctx context.Context) []*AccessPolicy
	Delete(ctx context.Context, id ulid.ULID) error
	DeleteRight(ctx context.Context, policyID ulid.ULID, userID ulid.ULID) error
	DeleteRightByPolicyID(ctx context.Context, policyID ulid.ULID) error
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

		_, err = tx.CreateBucketIfNotExists([]byte("ACCESSPOLICY_RIGHT"))
		if err != nil {
			return fmt.Errorf("failed to create a access policy rights bucket: %s", err)
		}

		return nil
	})

	return s, err
}

// Put storing access policy
func (s *DefaultAccessPolicyStore) Put(ctx context.Context, ap *AccessPolicy) error {
	if ap == nil {
		return ErrNilAccessPolicy
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		apBucket := tx.Bucket([]byte("ACCESSPOLICY"))
		if apBucket == nil {
			return fmt.Errorf("Put() failed to open access policy bucket: %s", ErrBucketNotFound)
		}

		data, err := json.Marshal(ap)
		if err != nil {
			return err
		}

		err = apBucket.Put(ap.ID[:], data)
		if err != nil {
			return fmt.Errorf("failed to store access policy: %s", err)
		}

		return nil
	})
}

// GetByID retrieving a access policy by ID
func (s *DefaultAccessPolicyStore) GetByID(ctx context.Context, id ulid.ULID) (*AccessPolicy, error) {
	var g *AccessPolicy
	err := s.db.View(func(tx *bbolt.Tx) error {
		apBucket := tx.Bucket([]byte("ACCESSPOLICY"))
		if apBucket == nil {
			return fmt.Errorf("GetByID(%s) failed to load access policy bucket: %s", id, ErrBucketNotFound)
		}

		data := apBucket.Get(id[:])
		if data == nil {
			return ErrAccessPolicyNotFound
		}

		return json.Unmarshal(data, &g)
	})

	return g, err
}

// GetAll retrieving all access policys
func (s *DefaultAccessPolicyStore) GetAll(ctx context.Context) ([]*AccessPolicy, error) {
	var aps []*AccessPolicy
	err := s.db.View(func(tx *bbolt.Tx) error {
		apBucket := tx.Bucket([]byte("ACCESSPOLICY"))
		if apBucket == nil {
			return fmt.Errorf("GetAll() failed to load access policy bucket: %s", ErrBucketNotFound)
		}

		c := apBucket.Cursor()
		for k, data := c.First(); k != nil; k, data = c.Next() {
			ap := &AccessPolicy{}
			err := json.Unmarshal(data, &ap)
			if err != nil {
				return fmt.Errorf("GetAll() failed to unmarshal access policy [%s]: %s", ap.ID, err)
			}

			aps = append(aps, ap)
		}

		return nil
	})

	return aps, err
}

// Delete from the store by access policy ID
func (s *DefaultAccessPolicyStore) Delete(ctx context.Context, id ulid.ULID) error {
	err := s.db.Update(func(tx *bbolt.Tx) error {
		apBucket := tx.Bucket([]byte("ACCESSPOLICY"))
		if apBucket == nil {
			return fmt.Errorf("failed to load access policy bucket: %s", ErrBucketNotFound)
		}

		// deleting the access policy
		err := apBucket.Delete(id[:])
		if err != nil {
			return fmt.Errorf("Delete() failed to delete access policy: %s", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("Delete() failed to delete a access policy(%s): %s", id, err)
	}

	// deleting all of this access policy's relations
	err = s.DeleteRightByPolicyID(ctx, id)
	if err != nil {
		return fmt.Errorf("Delete() failed to delete access policy relations: %s", err)
	}

	return nil
}
