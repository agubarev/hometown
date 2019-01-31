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
			return fmt.Errorf("PutRelation() failed to store group relation: %s", err)
		}

		return nil
	})
}

// HasRelation returns boolean denoting whether user is related to a group
func (s *defaultGroupStore) HasRelation(id ulid.ULID, userID ulid.ULID) (bool, error) {
	result := false
	err := s.db.View(func(tx *badger.Txn) error {
		groupRelationBucket := tx.Bucket([]byte("GROUP_RELATION"))
		if groupRelationBucket == nil {
			return fmt.Errorf("GetRelation() failed to load group relation bucket: %s", ErrBucketNotFound)
		}

		if !bytes.Equal(groupRelationBucket.Get(GroupRelationKey(id, userID)), []byte{1}) {
			return ErrRelationNotFound
		}

		// relation exists
		result = true

		return nil
	})

	return result, err
}

// GetAllRelation retrieve all user relations for a given group
// returns a map[groupID]userID
func (s *defaultGroupStore) GetAllRelation() (map[ulid.ULID][]ulid.ULID, error) {
	ids := make(map[ulid.ULID][]ulid.ULID, 0)
	err := s.db.View(func(tx *badger.Txn) error {
		groupRelationBucket := tx.Bucket([]byte("GROUP_RELATION"))
		if groupRelationBucket == nil {
			return fmt.Errorf("GetAllRelation() failed to load group relations bucket: %s", ErrBucketNotFound)
		}

		// key holds group and user ids as string of bytes separated by a hyphen
		c := groupRelationBucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			// parsing IDs
			var groupID ulid.ULID
			if err := groupID.Scan(k[0:16]); err != nil {
				return fmt.Errorf("GetAllRelation() failed to parse group id: %s", err)
			}

			var userID ulid.ULID
			if err := userID.Scan(k[16:]); err != nil {
				return fmt.Errorf("GetAllRelation() failed to parse related user id: %s", err)
			}

			ids[groupID] = append(ids[groupID], userID)
		}

		return nil
	})

	return ids, err
}

// GetRelationByGroupID returns a map of group id -> user id relations for a given group id
func (s *defaultGroupStore) GetRelationByGroupID(id ulid.ULID) (map[ulid.ULID][]ulid.ULID, error) {
	ids := make(map[ulid.ULID][]ulid.ULID, 0)
	err := s.db.View(func(tx *badger.Txn) error {
		groupRelationBucket := tx.Bucket([]byte("GROUP_RELATION"))
		if groupRelationBucket == nil {
			return fmt.Errorf("GetRelationByGroupID() failed to load group relations bucket: %s", ErrBucketNotFound)
		}

		// key holds group and user ids as string of bytes separated by a hyphen
		c := groupRelationBucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			if !bytes.HasPrefix(k, id[:]) {
				continue
			}

			var groupID ulid.ULID
			if err := groupID.Scan(k[0:16]); err != nil {
				return fmt.Errorf("GetAllRelation() failed to parse group id: %s", err)
			}

			var userID ulid.ULID
			if err := userID.Scan(k[16:]); err != nil {
				return fmt.Errorf("GetAllRelation() failed to parse related user id: %s", err)
			}

			// paranoid check
			if groupID == id {
				ids[groupID] = append(ids[groupID], userID)
			}
		}

		return nil
	})

	return ids, err
}

// DeleteRelation deletes a group-user relation
func (s *defaultGroupStore) DeleteRelation(id ulid.ULID, userID ulid.ULID) error {
	return s.db.Update(func(tx *badger.Txn) error {
		groupRelationBucket := tx.Bucket([]byte("GROUP_RELATION"))
		if groupRelationBucket == nil {
			return fmt.Errorf("failed to load group relation bucket: %s", ErrBucketNotFound)
		}

		err := groupRelationBucket.Delete(GroupRelationKey(id, userID))
		if err != nil {
			return fmt.Errorf("DeleteRelation() failed to delete group relation (%s -> %s): %s", id, userID, err)
		}

		return nil
	})
}

// DeleteRelationByGroupID deletes all relations for a given group id
func (s *defaultGroupStore) DeleteRelationByGroupID(id ulid.ULID) error {
	return s.db.Update(func(tx *badger.Txn) error {
		groupRelationBucket := tx.Bucket([]byte("GROUP_RELATION"))
		if groupRelationBucket == nil {
			return fmt.Errorf("failed to load group relation bucket: %s", ErrBucketNotFound)
		}

		c := groupRelationBucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			// testing by a prefix
			if bytes.HasPrefix(k, id[:]) {
				// found, deleting
				if err := c.Delete(); err != nil {
					return err
				}
			}
		}

		return nil
	})
}
