package accesspolicy

import (
	context2 "context"
	"fmt"
	"sync"
	"sync/atomic"

	"github.com/agubarev/hometown/internal/core"
)

// AccessPolicyStoreInMem is a default access policy store implementation
type AccessPolicyStoreInMem struct {
	idCounter   int64
	idMap       map[int]*AccessPolicy
	nameMap     map[TAPName]*AccessPolicy
	objectIDMap map[string]*AccessPolicy

	sync.RWMutex
}

// NewAccessPolicyStoreInMem returns an initialized access policy store
// that stores everything in memory
func NewAccessPolicyStoreInMem() (Store, error) {
	s := &AccessPolicyStoreInMem{
		idCounter:   0,
		idMap:       make(map[int]*AccessPolicy),
		nameMap:     make(map[TAPName]*AccessPolicy),
		objectIDMap: make(map[string]*AccessPolicy),
	}

	return s, nil
}

func (s *AccessPolicyStoreInMem) newID() int {
	return int(atomic.AddInt64(&s.idCounter, 1))
}

// Create creating access policy
func (s *AccessPolicyStoreInMem) Create(ctx context2.Context, ap *AccessPolicy) (retap *AccessPolicy, err error) {
	// basic validations
	if ap == nil {
		return nil, core.ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return nil, core.ErrNilRightsRoster
	}

	if ap.ID != 0 {
		return nil, core.ErrObjectIsNotNew
	}

	ap.ID = s.newID()

	s.Lock()
	s.idMap[ap.ID] = ap
	s.nameMap[ap.Name] = ap
	s.objectIDMap[ap.StringID()] = ap
	s.Unlock()

	return ap, nil
}

// UpdateAccessPolicy updates an existing access policy along with its rights roster
// NOTE: rights roster keeps track of its changes, thus, update will
// only affect changes mentioned by the respective RightsRoster object
func (s *AccessPolicyStoreInMem) Update(ctx context2.Context, ap *AccessPolicy) (err error) {
	// basic validations
	if ap == nil {
		return core.ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return core.ErrNilRightsRoster
	}

	if ap.ID != 0 {
		return core.ErrObjectIsNew
	}

	s.Lock()
	s.idMap[ap.ID] = ap
	s.nameMap[ap.Name] = ap
	s.objectIDMap[ap.StringID()] = ap
	s.Unlock()

	return nil
}

// GetByID retrieving a access policy by ID
func (s *AccessPolicyStoreInMem) GetByID(id int) (*AccessPolicy, error) {
	s.RLock()
	ap, ok := s.idMap[id]
	s.RUnlock()

	if !ok {
		return nil, core.ErrAccessPolicyNotFound
	}

	return ap, nil
}

// GetByName retrieving a access policy by key
func (s *AccessPolicyStoreInMem) GetByKey(name TAPName) (*AccessPolicy, error) {
	s.RLock()
	ap, ok := s.nameMap[name]
	s.RUnlock()

	if !ok {
		return nil, core.ErrAccessPolicyNotFound
	}

	return ap, nil
}

// GetByObjectTypeAndID retrieving a access policy by a kind and id
func (s *AccessPolicyStoreInMem) GetByKindAndID(kind string, id int) (*AccessPolicy, error) {
	s.RLock()
	ap, ok := s.objectIDMap[fmt.Sprintf("%s_%d", kind, id)]
	s.RUnlock()

	if !ok {
		return nil, core.ErrAccessPolicyNotFound
	}

	return ap, nil
}

// Delete access policy
func (s *AccessPolicyStoreInMem) Delete(ap *AccessPolicy) (err error) {
	// basic validations
	if ap == nil {
		return core.ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return core.ErrNilRightsRoster
	}

	if ap.ID != 0 {
		return core.ErrObjectIsNew
	}

	s.Lock()
	delete(s.idMap, ap.ID)
	delete(s.nameMap, ap.Name)
	delete(s.objectIDMap, ap.StringID())
	s.Unlock()

	return nil
}
