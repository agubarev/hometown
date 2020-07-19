package access

import (
	"sync"

	"github.com/agubarev/hometown/pkg/util"
	"github.com/pkg/errors"
)

// Roster denotes who has what rights
type Roster struct {
	// represents a mixed list of group/role/user rights
	Registry []Cell `json:"registry"`

	// holds a calculated summary cache of rights for a specific group/role/user
	// NOTE: a key is a packed value of 2 uint32s, left 32 bits are the
	// flags (i.e. a kind like a group/role/user/etc...), and the right 32 bits
	// is the uint32 object SubjectID
	// NOTE: these values are reset should any corresponding value change
	calculatedCache map[uint64]Right

	// this slice accumulates batch changes made to this roster
	changes []change

	registryLock sync.RWMutex
	cacheLock    sync.RWMutex
	changeLock   sync.RWMutex
	backup       *Roster

	// represents the base public access rights
	Everyone Right `json:"everyone"`
}

// Cell represents a single access registry cell
type Cell struct {
	Rights    Right       `json:"rights"`
	SubjectID uint32      `json:"subject_id"`
	Kind      SubjectKind `json:"kind"`
}

// NewRoster is a shorthand initializer function
func NewRoster(regsize int) *Roster {
	return &Roster{
		Registry:        make([]Cell, regsize),
		calculatedCache: make(map[uint64]Right),
		Everyone:        APNoAccess,
	}
}

// put adds a new or alters an existing access cell
func (r *Roster) put(kind SubjectKind, subjectID uint32, rights Right) {
	r.registryLock.Lock()

	// finding existing cell
	for i, cell := range r.Registry {
		if cell.SubjectID == subjectID && cell.Kind == kind {
			// altering the rights of an existing cell
			r.Registry[i].Rights = rights

			// unlocking before early return
			r.registryLock.Unlock()

			return
		}
	}

	// appending new cell because it hasn't been found above
	r.Registry = append(r.Registry, Cell{
		Rights:    rights,
		SubjectID: subjectID,
		Kind:      kind,
	})

	r.registryLock.Unlock()
}

// lookup looks up the isolated rights of a specific subject of a kind
// NOTE: does not summarize any rights, nor includes public access rights
func (r *Roster) lookup(k SubjectKind, subjectID uint32) (access Right) {
	access, err := r.lookupCache(k, subjectID)
	if err != nil && err != ErrCacheMiss {
		// returning cached value
		return access
	}

	// finding access rights
	r.registryLock.RLock()
	for _, cell := range r.Registry {
		if cell.SubjectID == subjectID && cell.Kind == k {
			access = cell.Rights
			break
		}
	}
	r.registryLock.RUnlock()

	// caching
	r.putCache(k, subjectID, access)

	return access
}

// hasRights tests whether a given subject of a kind has specific access rights
// NOTE: does not summarize anything, only tests a concrete subject of a kind
func (r *Roster) hasRights(kind SubjectKind, subjectID uint32, rights Right) bool {
	return r.lookup(kind, subjectID)&rights == rights
}

func (r *Roster) delete(k SubjectKind, subjectID uint32) {
	// searching and removing registry access cell
	r.registryLock.Lock()
	for i, cell := range r.Registry {
		if cell.SubjectID == subjectID && cell.Kind == k {
			r.Registry = append(r.Registry[:i], r.Registry[i+1:]...)
			break
		}
	}
	r.registryLock.Unlock()

	// clearing out specific calculated cache
	r.deleteCache(k, subjectID)
}

// putCache caches calculated access for user or a group/role
// NOTE: this cache is cleared whenever any relevant policy are changed
func (r *Roster) putCache(k SubjectKind, id uint32, rights Right) {
	r.cacheLock.Lock()
	r.calculatedCache[util.PackU32s(uint32(k), id)] = rights
	r.cacheLock.Unlock()
}

// lookupCache returns a cached, calculated access for a given user or group
func (r *Roster) lookupCache(k SubjectKind, id uint32) (Right, error) {
	r.cacheLock.RLock()
	right, ok := r.calculatedCache[util.PackU32s(uint32(k), id)]
	r.cacheLock.RUnlock()

	if !ok {
		return 0, ErrCacheMiss
	}

	return right, nil
}

// deleteCache removes calculated access cache
// NOTE: it must be used for any subject whose rights were altered
// directly or indirectly
func (r *Roster) deleteCache(k SubjectKind, id uint32) {
	r.cacheLock.Lock()
	delete(r.calculatedCache, util.PackU32s(uint32(k), id))
	r.cacheLock.Unlock()
}

// change adds a single deferred action for change and further storing
func (r *Roster) change(action RAction, kind SubjectKind, subjectID uint32, rights Right) {
	// the roster must have a backup before any unsaved changes to be made
	r.createBackup()

	// initializing new change
	change := change{
		action:      action,
		subjectKind: kind,
		subjectID:   subjectID,
		accessRight: rights,
	}

	//---------------------------------------------------------------------------
	// applying the actual change
	//---------------------------------------------------------------------------
	switch action {
	case RSet:
		// if kind is Everyone(public), then there's no need update registry
		if kind == SKEveryone {
			r.Everyone = rights
		} else {
			r.put(kind, subjectID, rights)
		}
	case RUnset:
		if kind == SKEveryone {
			r.Everyone = APNoAccess
		} else {
			r.delete(kind, subjectID)
		}
	default:
		panic(errors.Wrapf(
			ErrUnrecognizedRosterAction,
			"action=%d, kind=%s, subject_id=%d, rights=%d", action, kind, subjectID, rights,
		))
	}

	//---------------------------------------------------------------------------
	// adding a deferred action to store changes
	//---------------------------------------------------------------------------
	r.changeLock.Lock()
	if r.changes == nil {
		r.changes = []change{change}
	} else {
		r.changes = append(r.changes, change)
	}
	r.changeLock.Unlock()
}

func (r *Roster) clearChanges() {
	r.changeLock.Lock()
	r.changes = nil
	r.backup = nil
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

	// storing backup inside the roster itself
	r.backup = backup

	// removing both locks
	r.cacheLock.RUnlock()
	r.registryLock.RUnlock()
}

func (r *Roster) restoreBackup() {
	// nothing to restore if there's no backup
	if r.backup == nil {
		return
	}

	// double-locking registry and cache to freeze
	// the most vital parts of this roster
	r.registryLock.RLock()
	r.cacheLock.RLock()

	// re-initializing fresh registry and a cache
	r.Registry = make([]Cell, len(r.backup.Registry))
	r.calculatedCache = make(map[uint64]Right, len(r.backup.calculatedCache))

	// restoring public rights
	r.Everyone = r.backup.Everyone

	// access registry
	for i := range r.backup.Registry {
		r.Registry[i] = r.backup.Registry[i]
	}

	// copying calculated cache (not essential but still saves redundant re-calculation)
	for k := range r.calculatedCache {
		r.backup.calculatedCache[k] = r.calculatedCache[k]
	}

	// backup is no longer needed at this point,
	// clearing backup and all changes
	r.backup = nil
	r.changes = nil

	// removing both locks
	r.cacheLock.RUnlock()
	r.registryLock.RUnlock()
}
