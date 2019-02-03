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
	GetByID(domainID ulid.ULID, groupID ulid.ULID) (*Group, error)
	GetAll(domainID ulid.ULID) ([]*Group, error)
	Delete(domainID ulid.ULID, groupID ulid.ULID) error
	PutRelation(domainID ulid.ULID, groupID ulid.ULID, userID ulid.ULID) error
	GetAllRelation(domainID ulid.ULID) (map[ulid.ULID][]ulid.ULID, error)
	GetRelationByGroupID(domainID ulid.ULID, groupID ulid.ULID) (map[ulid.ULID][]ulid.ULID, error)
	HasRelation(domainID ulid.ULID, groupID ulid.ULID, userID ulid.ULID) (bool, error)
	DeleteRelation(domainID ulid.ULID, groupID ulid.ULID, userID ulid.ULID) error
	DeleteRelationByGroupID(domainID ulid.ULID, groupID ulid.ULID) error
}

func groupKey(domainID ulid.ULID, groupID ulid.ULID) []byte {
	return []byte(fmt.Sprintf("%s:group:%s", domainID[:], groupID[:]))
}

func groupUserKey(domainID ulid.ULID, groupID ulid.ULID, userID ulid.ULID) []byte {
	return []byte(fmt.Sprintf("%s:group:rel:%s:%s", domainID[:], groupID[:], userID[:]))
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
		key := groupKey(g.Domain.ID, g.ID)
		err = tx.Set(key, payload.Bytes())
		if err != nil {
			return fmt.Errorf("failed to store group %s: %s", key, err)
		}

		return nil
	})
}

// GetByID retrieving a group by ID
func (s *defaultGroupStore) GetByID(domainID ulid.ULID, groupID ulid.ULID) (*Group, error) {
	var g *Group
	err := s.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(groupKey(domainID, groupID))
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
func (s *defaultGroupStore) GetAll(domainID ulid.ULID) ([]*Group, error) {
	var g *Group
	var gs []*Group

	err := s.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.Prefix = domainID[:]
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
func (s *defaultGroupStore) Delete(domainID ulid.ULID, groupID ulid.ULID) error {
	err := s.db.Update(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false
		opts.Prefix = []byte(fmt.Sprintf("%s:group:%s:rel:", domainID, groupID))
		it := tx.NewIterator(opts)
		defer it.Close()

		// iterating over found keys and deleting them one by one
		for it.Rewind(); it.Valid(); it.Next() {
			if err := tx.Delete(it.Item().Key()); err != nil {
				return fmt.Errorf("failed to delete stored group relations: %s")
			}
		}

		return tx.Delete(groupKey(domainID, groupID))
	})
	if err != nil {
		return fmt.Errorf("failed to delete stored %s(%s): %s", domainID, groupID, err)
	}

	// deleting all of this group's relations
	err = s.DeleteRelationByGroupID(domainID, groupID)
	if err != nil {
		return fmt.Errorf("failed to delete group relations: %s", err)
	}

	return nil
}

// PutRelation store a relation flagging that user belongs to a group
func (s *defaultGroupStore) PutRelation(domainID ulid.ULID, groupID ulid.ULID, userID ulid.ULID) error {
	return s.db.Update(func(tx *badger.Txn) error {
		err := tx.Set(groupUserKey(domainID, groupID, userID), []byte{1})
		if err != nil {
			return fmt.Errorf("failed to store group relation: %s", err)
		}

		return nil
	})
}

// HasRelation returns boolean denoting whether user is related to a group
func (s *defaultGroupStore) HasRelation(domainID ulid.ULID, groupID ulid.ULID, userID ulid.ULID) (bool, error) {
	result := false
	err := s.db.View(func(tx *badger.Txn) error {
		_, err := tx.Get(groupUserKey(domainID, groupID, userID))
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

// GetAllRelation retrieve all user relations for a given group
// returns a map[groupID][]userID
func (s *defaultGroupStore) GetAllRelation(domainID ulid.ULID) (map[ulid.ULID][]ulid.ULID, error) {
	relations := make(map[ulid.ULID][]ulid.ULID, 0)
	err := s.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(fmt.Sprintf("%s:group:rel:", domainID[:]))
		opts.PrefetchValues = false
		it := tx.NewIterator(opts)

		var groupID, userID ulid.ULID
		for it.Rewind(); it.Valid(); it.Next() {
			key := it.Item().Key()

			err := groupID.UnmarshalBinary(key[:16])
			if err != nil {
				return fmt.Errorf("failed to unmarshal group id from bytes: %s", err)
			}

			err = userID.UnmarshalBinary(key[16:])
			if err != nil {
				return fmt.Errorf("failed to unmarshal user id from bytes: %s", err)
			}

			// creating a user slice for the group
			if _, ok := relations[groupID]; !ok {
				relations[groupID] = make([]ulid.ULID, 0)
			}

			// appending group to user relation
			relations[groupID] = append(relations[groupID], userID)
		}

		return nil
	})

	return relations, err
}

// GetRelationByGroupID returns a map of group id -> user id relations for a given group id
func (s *defaultGroupStore) GetRelationByGroupID(domainID ulid.ULID, groupID ulid.ULID) (map[ulid.ULID][]ulid.ULID, error) {
	relations := make(map[ulid.ULID][]ulid.ULID, 0)
	err := s.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(fmt.Sprintf("%s:group:rel:%s:", domainID[:], groupID[:]))
		opts.PrefetchValues = false
		it := tx.NewIterator(opts)

		var groupID, userID ulid.ULID
		for it.Rewind(); it.Valid(); it.Next() {
			key := it.Item().Key()

			// initializing ULIDs from key bytes
			err := groupID.UnmarshalBinary(key[:16])
			if err != nil {
				return fmt.Errorf("failed to unmarshal group id from bytes: %s", err)
			}

			err = userID.UnmarshalBinary(key[16:])
			if err != nil {
				return fmt.Errorf("failed to unmarshal user id from bytes: %s", err)
			}

			// creating a user slice for the group
			if _, ok := relations[groupID]; !ok {
				relations[groupID] = make([]ulid.ULID, 0)
			}

			// appending group to user relation
			relations[groupID] = append(relations[groupID], userID)

			return nil
		}
	})

	return relations, err
}

// DeleteRelation deletes a group-user relation
func (s *defaultGroupStore) DeleteRelation(domainID ulid.ULID, groupID ulid.ULID, userID ulid.ULID) error {
	return s.db.Update(func(tx *badger.Txn) error {
		err := tx.Delete(groupUserKey(domainID, groupID, userID))
		if err != nil {
			return fmt.Errorf("failed to delete stored group-user relation (%s -> %s): %s", groupID, userID, err)
		}

		return nil
	})
}

// DeleteRelationByGroupID deletes all relations for a given group id
func (s *defaultGroupStore) DeleteRelationByGroupID(domainID ulid.ULID, groupID ulid.ULID) error {
	return s.db.Update(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte(fmt.Sprintf("%s:group:rel:%s:", domainID[:], groupID[:]))
		opts.PrefetchValues = false
		it := tx.NewIterator(opts)

		var groupID, userID ulid.ULID
		for it.Rewind(); it.Valid(); it.Next() {
			if err := tx.Delete(it.Item().Key()); err != nil {
				return fmt.Errorf("failed to delete stored group-user relations for %s: %s", groupID, err)
			}

			return nil
		}

		return nil
	})
}
