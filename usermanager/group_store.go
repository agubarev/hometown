package usermanager

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"

	"github.com/oklog/ulid"
	"go.etcd.io/bbolt"
)

// GroupStore describes a storage contract for groups specifically
type GroupStore interface {
	Put(g *Group) error
	GetByID(id ulid.ULID) (*Group, error)
	GetAll() ([]*Group, error)
	Delete(id ulid.ULID) error
	PutRelation(id ulid.ULID, userID ulid.ULID) error
	GetAllRelation() (map[ulid.ULID][]ulid.ULID, error)
	GetRelationByGroupID(id ulid.ULID) (map[ulid.ULID][]ulid.ULID, error)
	HasRelation(id ulid.ULID, userID ulid.ULID) (bool, error)
	DeleteRelation(id ulid.ULID, userID ulid.ULID) error
	DeleteRelationByGroupID(id ulid.ULID) error
}

// DefaultGroupStore is the default group store implementation
type DefaultGroupStore struct {
	db *bbolt.DB
}

// NewDefaultGroupStore returns a group store with bbolt used as a backend
func NewDefaultGroupStore(db *bbolt.DB) (GroupStore, error) {
	s := &DefaultGroupStore{db}
	err := s.db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte("GROUP"))
		if err != nil {
			return fmt.Errorf("failed to create a group bucket: %s", err)
		}

		if _, err = tx.CreateBucketIfNotExists([]byte("GROUP_RELATION")); err != nil {
			return fmt.Errorf("failed to create group relations bucket: %s", err)
		}

		return nil
	})

	return s, err
}

// Put storing group
func (s *DefaultGroupStore) Put(g *Group) error {
	if g == nil {
		return ErrNilGroup
	}

	return s.db.Update(func(tx *bbolt.Tx) error {
		groupBucket := tx.Bucket([]byte("GROUP"))
		if groupBucket == nil {
			return fmt.Errorf("Put() failed to open group bucket: %s", ErrBucketNotFound)
		}

		data, err := json.Marshal(g)
		if err != nil {
			return err
		}

		err = groupBucket.Put(g.ID[:], data)
		if err != nil {
			return fmt.Errorf("failed to store group: %s", err)
		}

		return nil
	})
}

// GetByID retrieving a group by ID
func (s *DefaultGroupStore) GetByID(id ulid.ULID) (*Group, error) {
	var g *Group
	err := s.db.View(func(tx *bbolt.Tx) error {
		groupBucket := tx.Bucket([]byte("GROUP"))
		if groupBucket == nil {
			return fmt.Errorf("GetByID(%s) failed to load group bucket: %s", id, ErrBucketNotFound)
		}

		data := groupBucket.Get(id[:])
		if data == nil {
			return ErrGroupNotFound
		}

		return json.Unmarshal(data, &g)
	})

	return g, err
}

// GetAll retrieving all groups
func (s *DefaultGroupStore) GetAll() ([]*Group, error) {
	var groups []*Group
	err := s.db.View(func(tx *bbolt.Tx) error {
		groupBucket := tx.Bucket([]byte("GROUP"))
		if groupBucket == nil {
			return fmt.Errorf("GetAll() failed to load group bucket: %s", ErrBucketNotFound)
		}

		c := groupBucket.Cursor()
		for k, data := c.First(); k != nil; k, data = c.Next() {
			g := &Group{}
			err := json.Unmarshal(data, &g)
			if err != nil {
				// just logging about this error and moving forward
				log.Printf("GetAll() failed to unmarshal group [%s]: %s\n", k, err)
				continue
			}

			groups = append(groups, g)
		}

		return nil
	})

	return groups, err
}

// Delete from the store by group ID
func (s *DefaultGroupStore) Delete(id ulid.ULID) error {
	err := s.db.Update(func(tx *bbolt.Tx) error {
		groupBucket := tx.Bucket([]byte("GROUP"))
		if groupBucket == nil {
			return fmt.Errorf("failed to load group bucket: %s", ErrBucketNotFound)
		}

		// deleting the group
		err := groupBucket.Delete(id[:])
		if err != nil {
			return fmt.Errorf("Delete() failed to delete group: %s", err)
		}

		return nil
	})
	if err != nil {
		return fmt.Errorf("Delete() failed to delete a group(%s): %s", id, err)
	}

	// deleting all of this group's relations
	err = s.DeleteRelationByGroupID(id)
	if err != nil {
		return fmt.Errorf("Delete() failed to delete group relations: %s", err)
	}

	return nil
}

// PutRelation store a relation flagging that user belongs to a group
func (s *DefaultGroupStore) PutRelation(id ulid.ULID, userID ulid.ULID) error {
	// making sure that given ids are not empty, just in case
	if len(id[:]) == 0 {
		return ErrInvalidID
	}

	if len(userID[:]) == 0 {
		return ErrInvalidID
	}

	// storing the relationship
	return s.db.Update(func(tx *bbolt.Tx) error {
		groupRelationBucket := tx.Bucket([]byte("GROUP_RELATION"))
		if groupRelationBucket == nil {
			return fmt.Errorf("PutRelation() failed to open group relation bucket: %s", ErrBucketNotFound)
		}

		err := groupRelationBucket.Put(GroupRelationKey(id, userID), []byte{1})
		if err != nil {
			return fmt.Errorf("PutRelation() failed to store group relation: %s", err)
		}

		return nil
	})
}

// HasRelation returns boolean denoting whether user is related to a group
func (s *DefaultGroupStore) HasRelation(id ulid.ULID, userID ulid.ULID) (bool, error) {
	result := false
	err := s.db.View(func(tx *bbolt.Tx) error {
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
func (s *DefaultGroupStore) GetAllRelation() (map[ulid.ULID][]ulid.ULID, error) {
	ids := make(map[ulid.ULID][]ulid.ULID, 0)
	err := s.db.View(func(tx *bbolt.Tx) error {
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
func (s *DefaultGroupStore) GetRelationByGroupID(id ulid.ULID) (map[ulid.ULID][]ulid.ULID, error) {
	ids := make(map[ulid.ULID][]ulid.ULID, 0)
	err := s.db.View(func(tx *bbolt.Tx) error {
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
func (s *DefaultGroupStore) DeleteRelation(id ulid.ULID, userID ulid.ULID) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
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
func (s *DefaultGroupStore) DeleteRelationByGroupID(id ulid.ULID) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
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
