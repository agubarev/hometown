package usermanager

import (
	"fmt"
	"strings"

	"github.com/dgraph-io/badger"
	"github.com/oklog/ulid"
)

// GroupStore describes a storage contract for groups specifically
type GroupStore interface {
	Put(g *Group) error
	Get(groupID ulid.ULID) (*Group, error)
	GetGroups() ([]*Group, error)
	Delete(groupID ulid.ULID) error
	GetRelations(groupID ulid.ULID) ([]ulid.ULID, error)
	PutRelation(groupID ulid.ULID, userID ulid.ULID) error
	HasRelation(groupID ulid.ULID, userID ulid.ULID) (bool, error)
	DeleteRelation(groupID ulid.ULID, userID ulid.ULID) error
}

func groupKey(id ulid.ULID) []byte {
	return []byte(fmt.Sprintf("g:%s", id[:]))
}

func groupUserKey(groupID ulid.ULID, userID ulid.ULID) []byte {
	return []byte(fmt.Sprintf("gr:%s:%s", groupID[:], userID[:]))
}

// DefaultGroupStore is the default group store implementation
type DefaultGroupStore struct {
	db *badger.DB
}

// NewDefaultGroupStore returns a group store with bbolt used as a backend
func NewDefaultGroupStore(db *badger.DB) (GroupStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &DefaultGroupStore{db}, nil
}

// Put storing group
func (s *DefaultGroupStore) Put(g *Group) error {
	if g == nil {
		return ErrNilGroup
	}

	return s.db.Update(func(tx *badger.Txn) error {
		payload, err := json.Marshal(g)
		if err != nil {
			return fmt.Errorf("failed to marshal group: %s", err)
		}

		key := groupKey(g.ID)
		if tx.Set(key, payload) != nil {
			return fmt.Errorf("failed to store group key(%s): %s", key, err)
		}

		return nil
	})
}

// Get retrieving a group by ID
func (s *DefaultGroupStore) Get(id ulid.ULID) (*Group, error) {
	g := new(Group)

	err := s.db.View(func(tx *badger.Txn) error {
		item, err := tx.Get(groupKey(id))
		if err != nil {
			if err == badger.ErrKeyNotFound {
				return ErrGroupNotFound
			}

			return fmt.Errorf("failed to get stored user by ID %s: %s", id, err)
		}

		return item.Value(func(payload []byte) error {
			if err := json.Unmarshal(payload, g); err != nil {
				return fmt.Errorf("failed to unmarshal stored group: %s", err)
			}

			return nil
		})
	})

	if err != nil {
		return nil, err
	}

	return g, nil
}

// GetGroups retrieving all groups
func (s *DefaultGroupStore) GetGroups() ([]*Group, error) {
	gs := make([]*Group, 0)

	err := s.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.Prefix = []byte("g:")
		it := tx.NewIterator(opts)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			err := it.Item().Value(func(payload []byte) error {
				g := new(Group)
				if err := json.Unmarshal(payload, g); err != nil {
					return fmt.Errorf("failed to unmarshal group payload: %s", err)
				}

				// appending group to resulting slice
				gs = append(gs, g)

				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	return gs, err
}

// Delete from the store by group ID
func (s *DefaultGroupStore) Delete(groupID ulid.ULID) error {
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
func (s *DefaultGroupStore) PutRelation(groupID ulid.ULID, userID ulid.ULID) error {
	return s.db.Update(func(tx *badger.Txn) error {
		err := tx.Set(groupUserKey(groupID, userID), []byte{1})
		if err != nil {
			return fmt.Errorf("failed to store group(%s) relation: %s", groupID, err)
		}

		return nil
	})
}

// HasRelation returns boolean denoting whether user is related to a group
func (s *DefaultGroupStore) HasRelation(groupID ulid.ULID, userID ulid.ULID) (bool, error) {
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

// GetRelations retrieving all groups, returns a slice of user IDs
func (s *DefaultGroupStore) GetRelations(groupID ulid.ULID) ([]ulid.ULID, error) {
	relations := make([]ulid.ULID, 0)

	err := s.db.View(func(tx *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		// gr stands for group relations
		opts.PrefetchValues = false
		opts.Prefix = []byte(fmt.Sprintf("gr:%s:", groupID.String()))
		it := tx.NewIterator(opts)
		defer it.Close()

		// iterating over keys only
		for it.Rewind(); it.Valid(); it.Next() {
			// converting key to string and extracting user ID which is a suffix of the key
			skey := string(it.Item().Key())
			lspos := strings.LastIndex(skey, ":") // last separator position
			if lspos == -1 {
				return fmt.Errorf("GetRelations(%s): last separator not found in (%s)", groupID.String(), skey)
			}

			// string user ID
			suid := skey[lspos:]
			if len(suid) != 26 {
				return fmt.Errorf("GetRelations(%s): invalid length of user string ID; must be 26 characters long (%s)", groupID.String(), suid)
			}

			// reconstructing user ID from the string
			uid, err := ulid.Parse(suid)
			if err != nil {
				return fmt.Errorf("GetRelations(%s): failed to parse user ID(%s): %s", groupID.String(), suid, err)
			}

			// appending to resulting slice
			relations = append(relations, uid)
		}

		return nil
	})

	return relations, err
}

// DeleteRelation deletes a group-user relation
func (s *DefaultGroupStore) DeleteRelation(groupID ulid.ULID, userID ulid.ULID) error {
	return s.db.Update(func(tx *badger.Txn) error {
		err := tx.Delete(groupUserKey(groupID, userID))
		if err != nil {
			return fmt.Errorf("failed to delete stored group-user relation (%s -> %s): %s", groupID, userID, err)
		}

		return nil
	})
}
