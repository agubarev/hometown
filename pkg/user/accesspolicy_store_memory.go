package user

import (
	context2 "context"
	"fmt"
	"sync"
	"sync/atomic"

	"golang.org/x/net/context"
)

// memoryStore is a default access policy store implementation
type memoryStore struct {
	idCounter   int64
	idMap       map[uint32]*AccessPolicy
	nameMap     map[TPolicyName]*AccessPolicy
	objectIDMap map[string]*AccessPolicy

	sync.RWMutex
}

// NewMemoryStore returns an initialized access policy store
// that stores everything in memory
func NewMemoryStore() (AccessPolicyStore, error) {
	s := &memoryStore{
		idCounter:   0,
		idMap:       make(map[uint32]*AccessPolicy),
		nameMap:     make(map[TPolicyName]*AccessPolicy),
		objectIDMap: make(map[string]*AccessPolicy),
	}

	return s, nil
}

func (s *memoryStore) newID() uint32 {
	return uint32(atomic.AddInt64(&s.idCounter, 1))
}

// Upsert creating access policy
func (s *memoryStore) Create(ctx context2.Context, ap *AccessPolicy) (retap *AccessPolicy, err error) {
	// basic validations
	if ap == nil {
		return nil, ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return nil, ErrNilRightsRoster
	}

	if ap.ID != 0 {
		return nil, ErrZeroID
	}

	ap.ID = s.newID()

	s.Lock()
	s.idMap[ap.ID] = ap
	s.nameMap[ap.Name] = ap
	s.objectIDMap[ap.StringID()] = ap
	s.Unlock()

	return ap, nil
}

// UpdatePolicy updates an existing access policy along with its rights roster
// NOTE: rights roster keeps track of its changes, thus, update will
// only affect changes mentioned by the respective RightsRoster object
func (s *memoryStore) Update(ctx context.Context, ap *AccessPolicy) (err error) {
	// basic validations
	if ap == nil {
		return ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return ErrNilRightsRoster
	}

	if ap.ID == 0 {
		return ErrZeroID
	}

	s.Lock()
	s.idMap[ap.ID] = ap
	s.nameMap[ap.Name] = ap
	s.objectIDMap[ap.StringID()] = ap
	s.Unlock()

	return nil
}

// GroupByID retrieving a access policy by ObjectID
func (s *memoryStore) GetByID(id uint32) (*AccessPolicy, error) {
	s.RLock()
	ap, ok := s.idMap[id]
	s.RUnlock()

	if !ok {
		return nil, ErrAccessPolicyNotFound
	}

	return ap, nil
}

// PolicyByName retrieving a access policy by key
func (s *memoryStore) GetByKey(name TPolicyName) (*AccessPolicy, error) {
	s.RLock()
	ap, ok := s.nameMap[name]
	s.RUnlock()

	if !ok {
		return nil, ErrAccessPolicyNotFound
	}

	return ap, nil
}

// PolicyByObjectTypeAndID retrieving a access policy by a kind and id
func (s *memoryStore) GetByKindAndID(kind string, id uint32) (*AccessPolicy, error) {
	s.RLock()
	ap, ok := s.objectIDMap[fmt.Sprintf("%s_%d", kind, id)]
	s.RUnlock()

	if !ok {
		return nil, ErrAccessPolicyNotFound
	}

	return ap, nil
}

// DeletePolicy access policy
func (s *memoryStore) Delete(ap *AccessPolicy) (err error) {
	// basic validations
	if ap == nil {
		return ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return ErrNilRightsRoster
	}

	if ap.ID == 0 {
		return ErrZeroID
	}

	s.Lock()
	delete(s.idMap, ap.ID)
	delete(s.nameMap, ap.Name)
	delete(s.objectIDMap, ap.StringID())
	s.Unlock()

	return nil
}
