package usermanager

import (
	"encoding/json"
	"fmt"

	"github.com/oklog/ulid"
	"go.etcd.io/bbolt"
)

// DomainStore handles domain storage
type DomainStore interface {
	Put(d *Domain) error
	GetByID(id ulid.ULID) (*Domain, error)
	GetAll() ([]*Domain, error)
	Delete(id ulid.ULID) error
}

// DefaultDomainStore is the default domain store implementation
type DefaultDomainStore struct {
	db *bbolt.DB
}

// NewDefaultDomainStore returns an initialized default domain store
func NewDefaultDomainStore(db *bbolt.DB) (DomainStore, error) {
	s := &DefaultDomainStore{db}
	err := s.db.Update(func(tx *bbolt.Tx) error {
		// having a separate bucket for safety
		_, err := tx.CreateBucketIfNotExists([]byte("SUPER_DOMAIN"))
		if err != nil {
			return fmt.Errorf("failed to create a super domain bucket: %s", err)
		}

		_, err = tx.CreateBucketIfNotExists([]byte("DOMAIN"))
		if err != nil {
			return fmt.Errorf("failed to create a domain bucket: %s", err)
		}

		if _, err = tx.CreateBucketIfNotExists([]byte("DOMAIN_RELATION")); err != nil {
			return fmt.Errorf("failed to create domain relations bucket: %s", err)
		}

		return nil
	})

	return s, err
}

// Put storing domain
func (s *DefaultDomainStore) Put(d *Domain) error {
	if d == nil {
		return ErrNilDomain
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		domainBucket := tx.Bucket([]byte("DOMAIN"))
		if domainBucket == nil {
			return fmt.Errorf("Put() failed to open domain bucket: %s", ErrBucketNotFound)
		}

		data, err := json.Marshal(d)
		if err != nil {
			return err
		}

		err = domainBucket.Put(d.ID[:], data)
		if err != nil {
			return fmt.Errorf("failed to store domain: %s", err)
		}

		return nil
	})
}

// GetByID retrieving a domain by ID
func (s *DefaultDomainStore) GetByID(id ulid.ULID) (*Domain, error) {
	var d *Domain
	err := s.db.View(func(tx *bbolt.Tx) error {
		domainBucket := tx.Bucket([]byte("DOMAIN"))
		if domainBucket == nil {
			return fmt.Errorf("GetByID(%s) failed to load domain bucket: %s", id, ErrBucketNotFound)
		}

		data := domainBucket.Get(id[:])
		if data == nil {
			return fmt.Errorf("failed to get domain(%s): %s", id, ErrDomainNotFound)
		}

		return json.Unmarshal(data, &d)
	})

	return d, err
}

// GetAll retrieving all domains
func (s *DefaultDomainStore) GetAll() ([]*Domain, error) {
	var domains []*Domain
	err := s.db.View(func(tx *bbolt.Tx) error {
		domainBucket := tx.Bucket([]byte("DOMAIN"))
		if domainBucket == nil {
			return fmt.Errorf("GetAll() failed to load domain bucket: %s", ErrBucketNotFound)
		}

		c := domainBucket.Cursor()
		for k, data := c.First(); k != nil; k, data = c.Next() {
			d := &Domain{}
			err := json.Unmarshal(data, &d)
			if err != nil {
				return fmt.Errorf("GetAll() failed to unmarshal domain [%s]: %s", d.ID, err)
			}

			domains = append(domains, d)
		}

		return nil
	})

	return domains, err
}

// Delete from the store by domain ID
func (s *DefaultDomainStore) Delete(id ulid.ULID) error {
	err := s.db.Update(func(tx *bbolt.Tx) error {
		domainBucket := tx.Bucket([]byte("DOMAIN"))
		if domainBucket == nil {
			return fmt.Errorf("failed to load domain bucket: %s", ErrBucketNotFound)
		}

		// deleting the domain
		err := domainBucket.Delete(id[:])
		if err != nil {
			return fmt.Errorf("Delete() failed to delete domain: %s", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("Delete() failed to delete a domain(%s): %s", id, err)
	}

	return nil
}
