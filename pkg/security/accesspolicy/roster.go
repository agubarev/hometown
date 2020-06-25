package user

import (
	"sync"

	"github.com/agubarev/hometown/pkg/util"
)

// Roster denotes who has what rights
type Roster struct {
	// represents a mixed list of group/role/user rights
	Registry []Cell

	// holds a precalculated summary cache of rights for a specific group/role/user
	// NOTE: a key is a packed value of 2 uint32s, left 32 bits are the
	// flags (i.e. a kind like a group/role/user/etc...), and the right 32 bits
	// is the uint32 object ID
	// NOTE: these values are reset should any corresponding value change
	calculatedCache map[uint64]Right

	// this slice accumulates batch changes made to this roster
	changes []accessChange

	registryLock sync.RWMutex
	cacheLock    sync.RWMutex
	changeLock   sync.RWMutex

	// represents the base public access rights
	Everyone Right `json:"everyone"`
}

// Cell represents a single roster registry access cell
type Cell struct {
	Rights Right
	ID     uint32
	Kind   SubjectKind
}

// NewRoster is a shorthand initializer function
func NewRoster() Roster {
	return Roster{
		Registry:        make([]Cell, 0),
		calculatedCache: make(map[uint64]Right),
		Everyone:        APNoAccess,
	}
}

func (r *Roster) register(k SubjectKind, id uint32, rights Right) {
	r.registryLock.Lock()

	r.Registry = append(r.Registry, Cell{
		Rights: rights,
		ID:     id,
		Kind:   k,
	})

	r.registryLock.Unlock()
}

func (r *Roster) putCache(k SubjectKind, id uint32, rights Right) {
	r.cacheLock.Lock()
	r.calculatedCache[util.PackU32s(uint32(k), id)] = rights
	r.cacheLock.Unlock()
}

func (r *Roster) lookupCache(k SubjectKind, id uint32) (Right, error) {
	r.cacheLock.RLock()
	right, ok := r.calculatedCache[util.PackU32s(uint32(k), id)]
	r.cacheLock.RUnlock()

	if !ok {
		return 0, ErrCacheMiss
	}

	return right, nil
}

func (r *Roster) deleteCache(k SubjectKind, id uint32) {
	r.cacheLock.Lock()
	delete(r.calculatedCache, util.PackU32s(uint32(k), id))
	r.cacheLock.Unlock()
}

// addChange adds a single change for further storing
func (r *Roster) addChange(action RAction, subjectKind SubjectKind, subjectID uint32, rights Right) {
	change := accessChange{
		action:      action,
		subjectKind: subjectKind,
		subjectID:   subjectID,
		accessRight: rights,
	}

	r.changeLock.Lock()

	if r.changes == nil {
		r.changes = []accessChange{change}
	} else {
		r.changes = append(r.changes, change)
	}

	r.changeLock.Unlock()
}

func (r *Roster) clearChanges() {
	r.changeLock.Lock()
	r.changes = nil
	r.changeLock.Unlock()
}
