package accesspolicy

import (
	"sync"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Roster holds metadata to keep track of who has what access to its
// corresponding access policy
type Roster struct {
	// Resolve calculates the final access right value of a policy
	// which extends (or possibly inherits) from a parent, because sometimes a certain right
	// must be overridden while still preserving extended some values
	Resolve func(extended, current Right) Right

	// represents a mixed list of group/role/user rights
	Registry []Cell `json:"registry"`

	// holds a calculated summary cache of rights for a specific group/role/user
	// NOTE: these values are reset should any related value change
	calculatedCache map[Actor]Right

	// this slice accumulates batch changes made to this roster
	changes []rosterChange

	registryLock sync.RWMutex
	cacheLock    sync.RWMutex
	changeLock   sync.RWMutex
	backup       *Roster

	// represents the base public accesspolicy rights
	Everyone Right `json:"everyone"`
}

type Actor struct {
	ID   uuid.UUID
	Kind ActorKind
}

func NewActor(k ActorKind, id uuid.UUID) Actor {
	return Actor{
		ID:   id,
		Kind: k,
	}
}

func PublicActor() Actor {
	return Actor{
		ID:   uuid.Nil,
		Kind: AKEveryone,
	}
}

func UserActor(id uuid.UUID) Actor {
	return Actor{
		ID:   id,
		Kind: AKUser,
	}
}

func GroupActor(id uuid.UUID) Actor {
	return Actor{
		ID:   id,
		Kind: AKGroup,
	}
}

func RoleActor(id uuid.UUID) Actor {
	return Actor{
		ID:   id,
		Kind: AKRoleGroup,
	}
}

// Cell represents a single access policy registry entry
// TODO: consider overrides
type Cell struct {
	Key    Actor `json:"key"`
	Rights Right `json:"rights"`
}

// NewRoster is a shorthand initializer function
func NewRoster(regsize int) *Roster {
	return &Roster{
		Registry:        make([]Cell, regsize),
		calculatedCache: make(map[Actor]Right),
		Everyone:        APNoAccess,
	}
}

// put adds a new or alters an existing accesspolicy cell
func (r *Roster) put(key Actor, rights Right) {
	r.registryLock.Lock()

	// finding existing cell
	for i, cell := range r.Registry {
		if cell.Key == key {
			// altering the rights of an existing cell
			r.Registry[i].Rights = rights

			// unlocking before early return
			r.registryLock.Unlock()

			return
		}
	}

	// appending new cell because it hasn't been found above
	r.Registry = append(r.Registry, Cell{
		Rights: rights,
		Key:    key,
	})

	r.registryLock.Unlock()
}

// lookup looks up the isolated rights of a specific subject of a kind
// NOTE: does not summarize any rights, nor includes public accesspolicy rights
func (r *Roster) lookup(key Actor) (access Right) {
	access, err := r.lookupCache(key)
	if err != nil && err != ErrCacheMiss {
		return APNoAccess
	}

	// finding accesspolicy rights
	r.registryLock.RLock()
	for _, cell := range r.Registry {
		if cell.Key == key {
			access = cell.Rights
			break
		}
	}
	r.registryLock.RUnlock()

	// caching
	r.putCache(key, access)

	return access
}

// hasRights tests whether a given subject of a kind has specific accesspolicy rights
// NOTE: does not summarize anything, only tests a concrete subject of a kind
func (r *Roster) hasRights(key Actor, rights Right) bool {
	return r.lookup(key)&rights == rights
}

func (r *Roster) delete(key Actor) {
	// searching and removing registry accesspolicy cell
	r.registryLock.Lock()
	for i, cell := range r.Registry {
		if cell.Key == key {
			r.Registry = append(r.Registry[:i], r.Registry[i+1:]...)
			break
		}
	}
	r.registryLock.Unlock()

	// clearing out specific calculated cache
	r.deleteCache(key)
}

// putCache caches calculated accesspolicy for user or a group/role
// NOTE: this cache is cleared whenever any relevant policy are changed
func (r *Roster) putCache(key Actor, rights Right) {
	r.cacheLock.Lock()
	r.calculatedCache[key] = rights
	r.cacheLock.Unlock()
}

// lookupCache returns a cached, calculated accesspolicy for a given user or group
func (r *Roster) lookupCache(key Actor) (Right, error) {
	r.cacheLock.RLock()
	right, ok := r.calculatedCache[key]
	r.cacheLock.RUnlock()

	if !ok {
		return 0, ErrCacheMiss
	}

	return right, nil
}

// deleteCache removes calculated accesspolicy cache
// NOTE: it must be used for any subject whose rights were altered
// directly or indirectly
func (r *Roster) deleteCache(key Actor) {
	r.cacheLock.Lock()
	delete(r.calculatedCache, key)
	r.cacheLock.Unlock()
}

// change adds a single deferred action to change policy before storing
func (r *Roster) change(action RAction, key Actor, rights Right) {
	// the roster must have a backup before any unsaved changes to be made
	r.createBackup()

	// initializing new rosterChange
	change := rosterChange{
		action:      action,
		key:         key,
		accessRight: rights,
	}

	//---------------------------------------------------------------------------
	// applying the actual roster change
	//---------------------------------------------------------------------------
	switch action {
	case RSet:
		// if kind is Everyone(public), then there's no need update registry
		if key.Kind == AKEveryone {
			r.Everyone = rights
		} else {
			r.put(key, rights)
		}
	case RUnset:
		if key.Kind == AKEveryone {
			r.Everyone = APNoAccess
		} else {
			r.delete(key)
		}
	default:
		panic(errors.Wrapf(
			ErrUnrecognizedRosterAction,
			"action=%d, kind=%s, subject_id=%d, rights=%d", action, key.Kind, key.ID, rights,
		))
	}

	//---------------------------------------------------------------------------
	// adding a deferred action to store changes
	//---------------------------------------------------------------------------
	r.changeLock.Lock()
	if r.changes == nil {
		r.changes = []rosterChange{change}
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

// createBackup returns a snapshot copy of the accesspolicy rights roster for this policy
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

	// accesspolicy registry
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
	r.calculatedCache = make(map[Actor]Right, len(r.backup.calculatedCache))

	// restoring public rights
	r.Everyone = r.backup.Everyone

	// accesspolicy registry
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
