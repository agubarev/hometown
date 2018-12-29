package usermanager

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/oklog/ulid"
	"go.etcd.io/bbolt"
)

// GroupStore describes a storage contract for groups specifically
// TODO add predicates for searching
type GroupStore interface {
	Put(ctx context.Context, g *Group) error
	GetAll(ctx context.Context) ([]*Group, error)
	GetByID(ctx context.Context, id ulid.ULID) (*Group, error)
	Delete(ctx context.Context, id ulid.ULID) error
	PutRelation(ctx context.Context, id ulid.ULID, userID ulid.ULID) error
	GetAllRelation(ctx context.Context) (map[ulid.ULID]ulid.ULID, error)
	HasRelation(ctx context.Context, id ulid.ULID, userID ulid.ULID) (bool, error)
	DeleteRelation(ctx context.Context, id ulid.ULID, userID ulid.ULID) error
	DeleteRelationByGroupID(ctx context.Context, id ulid.ULID) error
}

// DefaultGroupStore is the default group store implementation
type DefaultGroupStore struct {
	db *bbolt.DB
}

func groupRelationKey(groupID ulid.ULID, userID ulid.ULID) []byte {
	return bytes.Join([][]byte{groupID[:], userID[:]}, []byte("-"))
}

// NewDefaultGroupStore returns a group store with bbolt used as a backend
func NewDefaultGroupStore(db *bbolt.DB) (GroupStore, error) {
	s := &DefaultGroupStore{db}
	err := s.db.Update(func(tx *bbolt.Tx) error {
		groupBucket, err := tx.CreateBucketIfNotExists([]byte("GROUP"))
		if err != nil {
			return fmt.Errorf("failed to create a group bucket: %s", err)
		}

		if _, err = groupBucket.CreateBucketIfNotExists([]byte("GROUP_RELATION")); err != nil {
			return fmt.Errorf("failed to create group relations bucket: %s", err)
		}

		return nil
	})

	return s, err
}

// Put storing group
func (s *DefaultGroupStore) Put(ctx context.Context, g *Group) error {
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
func (s *DefaultGroupStore) GetByID(ctx context.Context, id ulid.ULID) (*Group, error) {
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
func (s *DefaultGroupStore) GetAll(ctx context.Context) ([]*Group, error) {
	var groups []*Group
	err := s.db.View(func(tx *bbolt.Tx) error {
		groupBucket := tx.Bucket([]byte("GROUP"))
		if groupBucket == nil {
			return fmt.Errorf("GetAll() failed to load group bucket: %s", ErrBucketNotFound)
		}

		c := groupBucket.Cursor()
		g := &Group{}
		for k, data := c.First(); k != nil; k, data = c.Next() {
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
func (s *DefaultGroupStore) Delete(ctx context.Context, id ulid.ULID) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		groupBucket := tx.Bucket([]byte("GROUP"))
		if groupBucket == nil {
			return fmt.Errorf("failed to load group bucket: %s", ErrBucketNotFound)
		}

		// deleting the group
		err := groupBucket.Delete(id[:])
		if err != nil {
			return fmt.Errorf("Delete() failed to delete group: %s", err)
		}

		// deleting all of this group's relations
		err = s.DeleteRelationByGroupID(ctx, id)
		if err != nil {
			return fmt.Errorf("Delete() failed to delete group relations: %s", err)
		}

		return nil
	})
}

// PutRelation store a relation flagging that user belongs to a group
func (s *DefaultGroupStore) PutRelation(ctx context.Context, id ulid.ULID, userID ulid.ULID) error {
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

		err := groupRelationBucket.Put(groupRelationKey(id, userID), []byte{1})
		if err != nil {
			return fmt.Errorf("PutRelation() failed to store group relation: %s", err)
		}

		return nil
	})
}

// HasRelation returns boolean denoting whether user is related to a group
func (s *DefaultGroupStore) HasRelation(ctx context.Context, id ulid.ULID, userID ulid.ULID) (bool, error) {
	result := false
	err := s.db.View(func(tx *bbolt.Tx) error {
		groupRelationBucket := tx.Bucket([]byte("GROUP_RELATION"))
		if groupRelationBucket == nil {
			return fmt.Errorf("GetRelation() failed to load group relation bucket: %s", ErrBucketNotFound)
		}

		if !bytes.Equal(groupRelationBucket.Get(groupRelationKey(id, userID)), []byte{1}) {
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
func (s *DefaultGroupStore) GetAllRelation(ctx context.Context) (map[ulid.ULID]ulid.ULID, error) {
	ids := make(map[ulid.ULID]ulid.ULID, 0)
	err := s.db.View(func(tx *bbolt.Tx) error {
		groupRelationBucket := tx.Bucket([]byte("GROUP_RELATION"))
		if groupRelationBucket == nil {
			return fmt.Errorf("GetAllRelation() failed to load group relations bucket: %s", ErrBucketNotFound)
		}

		// key holds group and user ids as string of bytes separated by a hyphen
		c := groupRelationBucket.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			idpair := bytes.Split(k, []byte("-"))
			if len(idpair) != 2 {
				return fmt.Errorf(
					"GetAllRelation() failed to parse group relation key(%s), expecting 2 pieces, got %d",
					string(k),
					len(idpair),
				)
			}

			// parsing ids
			groupID, err := ulid.ParseStrict(string(idpair[0]))
			if err != nil {
				return fmt.Errorf("GetAllRelation() failed to parse group id(%s): %s", string(idpair[0]), err)
			}

			userID, err := ulid.ParseStrict(string(idpair[1]))
			if err != nil {
				return fmt.Errorf("GetAllRelation() failed to parse related user id(%s): %s", string(idpair[1]), err)
			}

			ids[groupID] = userID
		}

		return nil
	})

	return ids, err
}

// DeleteRelation deletes a group-user relation
func (s *DefaultGroupStore) DeleteRelation(ctx context.Context, id ulid.ULID, userID ulid.ULID) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		groupBucket := tx.Bucket([]byte("GROUP"))
		if groupBucket == nil {
			return fmt.Errorf("failed to load group bucket: %s", ErrBucketNotFound)
		}

		err := groupBucket.Delete(groupRelationKey(id, userID))
		if err != nil {
			return fmt.Errorf("DeleteRelation() failed to delete group relation (%s -> %s): %s", id, userID, err)
		}

		return nil
	})
}

// DeleteRelationByGroupID deletes all relations for a given group id
func (s *DefaultGroupStore) DeleteRelationByGroupID(ctx context.Context, id ulid.ULID) error {
	panic("not implemented")
}
