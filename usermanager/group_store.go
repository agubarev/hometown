package usermanager

import (
	"bytes"
	"encoding/gob"
	"fmt"

	"github.com/dgraph-io/badger"
	"github.com/oklog/ulid"
)

// GroupStore describes a storage contract for groups specifically
type GroupStore interface {
	Put(g *Group) error
	Get(groupID ulid.ULID) (*Group, error)
	GetAll() ([]*Group, error)
	Delete(groupID ulid.ULID) error
	PutRelation(groupID ulid.ULID, userID ulid.ULID) error
	HasRelation(groupID ulid.ULID, userID ulid.ULID) (bool, error)
	DeleteRelation(groupID ulid.ULID, userID ulid.ULID) error
}

func groupKey(groupID ulid.ULID) []byte {
	return groupID[:]
}

func groupUserKey(groupID ulid.ULID, userID ulid.ULID) []byte {
	key := make([]byte, 32)
	copy(key[:16], groupID[:])
	copy(key[16:], userID[:])

	return key
}

// defaultGroupStore is the default group store implementation
type defaultGroupStore struct {
	db *badger.DB
}

// NewDefaultGroupStore returns a group store with bbolt used as a backend
func NewDefaultGroupStore(db *badger.DB) (GroupStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &defaultGroupStore{db}, nil
}

// Put storing group
func (s *defaultGroupStore) Put(g *Group) error {
	if g == nil {
		return ErrNilGroup
	}

	return s.db.Update(func(tx *badger.Txn) error {
		// decoding group bytes
		var payload bytes.Buffer
		err := gob.NewEncoder(&payload).Encode(g)
		if err != nil {
			return fmt.Errorf("failed to decode group: %s", err)
		}

		// storing primary value
		key := groupKey(g.ID)
		err = tx.Set(key, payload.Bytes())
		if err != nil {
			return fmt.Errorf("failed to store group %s: %s", key, err)
		}

		return nil
	})
}

// Get retrieving a group by ID
func (s *defaultGroupStore) Get(groupID ulid.ULID) (*Group, error) {
	var g *Group
	err := s.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(groupKey(groupID))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrGroupNotFound
			}

			return fmt.Errorf("failed to get stored user by ID %s: %s", groupID, err)
		}

		return item.Value(func(payload []byte) error {
			if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(g); err != nil {
				return fmt.Errorf("failed to unserialize stored group: %s", err)
			}

			return nil
		})
	})

	return g, err
}

// GetAll retrieving all groups
func (s *defaultGroupStore) GetAll() ([]*Group, error) {
	var g *Group
	var gs []*Group

	err := s.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(fmt.Sprintf("group"))
		it := tx.NewIterator(opts)
		defer it.Close()

		// iterating over keys only
		for it.Rewind(); it.Valid(); it.Next() {
			err := it.Item().Value(func(payload []byte) error {
				// decoding payload buffer
				if err := gob.NewDecoder(bytes.NewReader(payload)).Decode(g); err != nil {
					return fmt.Errorf("failed to decode group payload: %s", err)
				}

				// appending group to result slice
				gs = append(gs, g)
			})
		}

		return nil
	})

	return gs, err
}

// Delete from the store by group ID
func (s *defaultGroupStore) Delete(groupID ulid.ULID) error {
	err := s.db.Update(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.Prefix = groupKey(groupID) // this will also delete all group relations
		it := tx.NewIterator(opts)
		defer it.Close()

		// iterating over found keys and deleting them one by one
		for it.Rewind(); it.Valid(); it.Next() {
			if err := tx.Delete(it.Item().Key()); err != nil {
				return err
			}
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to delete stored group (%s) related key: %s", groupID, err)
	}

	return nil
}

// PutRelation store a relation flagging that user belongs to a group
func (s *defaultGroupStore) PutRelation(groupID ulid.ULID, userID ulid.ULID) error {
	return s.db.Update(func(tx *badger.Txn) error {
		err := tx.Set(groupUserKey(groupID, userID), []byte{1})
		if err != nil {
			return fmt.Errorf("failed to store group(%s) relation: %s", groupID, err)
		}

		return nil
	})
}

// HasRelation returns boolean denoting whether user is related to a group
func (s *defaultGroupStore) HasRelation(groupID ulid.ULID, userID ulid.ULID) (bool, error) {
	result := false
	err := s.db.View(func(tx *badger.Txn) error {
		_, err := tx.Get(groupUserKey(groupID, userID))
		if err != nil {
			// relation exists if the key exists
			if err == badger.ErrKeyNotFound {
				// no relation found, normal return
				// the result is still false
				return nil
			}

			// unexpected error
			return err
		}

		// relation exists
		result = true

		return nil
	})

	return result, err
}

// DeleteRelation deletes a group-user relation
func (s *defaultGroupStore) DeleteRelation(groupID ulid.ULID, userID ulid.ULID) error {
	return s.db.Update(func(tx *badger.Txn) error {
		err := tx.Delete(groupUserKey(groupID, userID))
		if err != nil {
			return fmt.Errorf("failed to delete stored group-user relation (%s -> %s): %s", groupID, userID, err)
		}

		return nil
	})
}
