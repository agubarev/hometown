package user

import (
	"sync"

	"github.com/agubarev/hometown/pkg/util"
)

// Roster denotes who has what rights
type Roster struct {
	// represents a mixed list of group/role/user rights
	Registry []Cell `json:"registry"`

	// holds a calculated summary cache of rights for a specific group/role/user
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
	backup       *Roster

	// represents the base public access rights
	Everyone Right `json:"everyone"`
}

// Cell represents a single access registry cell
type Cell struct {
	Rights Right       `json:"rights"`
	ID     uint32      `json:"id"`
	Kind   SubjectKind `json:"kind"`
}

// NewRoster is a shorthand initializer function
func NewRoster(regsize int) *Roster {
	return &Roster{
		Registry:        make([]Cell, regsize),
		calculatedCache: make(map[uint64]Right),
		Everyone:        APNoAccess,
	}
}

// HasRights tests whether a given subject of a kind has specific access rights
// NOTE: does not summarize anything, only tests a concrete subject of a kind
func (r *Roster) HasRights(k SubjectKind, subjectID uint32, rights Right) bool {
	return r.Lookup(k, subjectID)&rights == rights
}

// Lookup looks up the isolated rights of a specific subject of a kind
// NOTE: does not summarize any rights, nor includes public access rights
func (r *Roster) Lookup(k SubjectKind, subjectID uint32) (access Right) {
	access, err := r.lookupCache(k, subjectID)
	if err != nil && err != ErrCacheMiss {
		// returning cached value
		return access
	}

	// finding access rights
	r.registryLock.RLock()
	for _, cell := range r.Registry {
		if cell.ID == subjectID && cell.Kind == k {
			access = cell.Rights
			break
		}
	}
	r.registryLock.RUnlock()

	// caching
	r.putCache(k, subjectID, access)

	return access
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

func (r *Roster) unregister(k SubjectKind, id uint32) {
	// searching and removing registry access cell
	r.registryLock.Lock()
	for i, cell := range r.Registry {
		if cell.ID == id && cell.Kind == k {
			r.Registry = append(r.Registry[:i], r.Registry[i+1:]...)
			break
		}
	}
	r.registryLock.Unlock()

	// clearing out specific calculated cache
	r.deleteCache(k, id)
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

// createBackup returns a snapshot copy of the access rights roster for this policy
func (r *Roster) createBackup() {
	// it's fine if this roster already has a backup set,
	// thus doing nothing, allowing roster changes to be accumulated
	if r.backup != nil {
		return
	}

	// initializing backup roster
	backup := NewRoster(len(r.Registry))

	// double-locking registry and cache to freeze
	// the most vital parts of this roster
	r.registryLock.RLock()
	r.cacheLock.RLock()

	// copying public rights
	backup.Everyone = r.Everyone

	// access registry
	for i := range r.Registry {
		backup.Registry[i] = r.Registry[i]
	}

	// copying calculated cache (not essential but still saves redundant re-calculation)
	for k := range r.calculatedCache {
		backup.calculatedCache[k] = r.calculatedCache[k]
	}

	// removing both locks
	r.cacheLock.RUnlock()
	r.registryLock.RUnlock()
}

func (r *Roster) restoreBackup() error {
	// returning with an error if backup isn't found
	// NOTE: doing this explicitly for consistency, because
	// it's important whenever restoration is requested
	if r.backup == nil {
		return ErrNoBackup
	}

	// double-locking registry and cache to freeze
	// the most vital parts of this roster
	r.registryLock.RLock()
	r.cacheLock.RLock()

	// re-initializing fresh registry and a cache
	r.Registry = make([]Cell, len(r.backup.Registry))
	r.calculatedCache = make(map[uint64]Right, len(r.backup.calculatedCache))

	// copying public rights
	r.Everyone = r.backup.Everyone

	// access registry
	for i := range r.backup.Registry {
		r.Registry[i] = r.backup.Registry[i]
	}

	// copying calculated cache (not essential but still saves redundant re-calculation)
	for k := range r.calculatedCache {
		r.backup.calculatedCache[k] = r.calculatedCache[k]
	}

	// removing both locks
	r.cacheLock.RUnlock()
	r.registryLock.RUnlock()
}
