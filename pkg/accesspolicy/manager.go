package accesspolicy

import (
	"bytes"
	"fmt"
	"sync"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/user"
	"golang.org/x/net/context"
)

// Manager is an access policy registry for convenience
type Manager struct {
	idMap    map[int]*AccessPolicy
	namedMap map[TAPName]*AccessPolicy
	keyMap   map[uint64]*AccessPolicy
	store    Store
	sync.RWMutex
}

// NewManager initializes a new access policy container
func NewManager(store Store) (*Manager, error) {
	if store == nil {
		return nil, core.ErrNilAccessPolicyStore
	}

	c := &Manager{
		idMap:    make(map[int]*AccessPolicy),
		namedMap: make(map[TAPName]*AccessPolicy),
		keyMap:   make(map[uint64]*AccessPolicy),
		store:    store,
	}

	return c, nil
}

// Put adds policy to container registry
func (c *Manager) Put(ap *AccessPolicy) error {
	err := ap.Validate()
	if err != nil {
		return err
	}

	// assigning container to this policy for backreference
	ap.container = c

	// caching inside container's registry
	c.Lock()
	c.idMap[ap.ID] = ap
	c.namedMap[ap.Name] = ap
	c.keyMap[ap.Key(true)] = ap
	c.Unlock()

	return nil
}

// Remove removes policy from container registry
func (c *Manager) Remove(ap *AccessPolicy) error {
	err := ap.Validate()
	if err != nil {
		return err
	}

	// clearing container reference
	ap.container = nil

	// clearing out maps
	c.Lock()
	delete(c.idMap, ap.ID)
	delete(c.namedMap, ap.Name)
	delete(c.keyMap, ap.Key(false))
	c.Unlock()

	return nil
}

// Create creates a new access policy
func (c *Manager) Create(ctx context.Context, owner *user.User, parent *AccessPolicy, objectType TAPObjectType, objectID int, isInherited, isExtended bool) (*AccessPolicy, error) {
	// initializing and creating new policy
	ap := NewAccessPolicy(owner, parent, isInherited, isExtended)

	// setting distinctive markers
	copy(ap.ObjectType[:], bytes.ToLower(bytes.TrimSpace(objectType[:])))
	ap.ObjectID = objectID

	// creating initial key hash
	ap.hashKey()

	// validating before creation
	if err := ap.Validate(); err != nil {
		return nil, err
	}

	// checking whether a key is available
	if ap.Name[0] != 0 {
		_, err := c.GetByName(ap.Name)
		if err == nil {
			return nil, core.ErrAccessPolicyNameTaken
		}

		if err != core.ErrAccessPolicyNotFound {
			return nil, err
		}
	}

	// checking by an object
	if ap.ObjectType[0] != 0 && ap.ObjectID != 0 {
		_, err := c.GetByObjectTypeAndID(objectType, objectID)
		if err == nil {
			return nil, core.ErrAccessPolicyKindAndIDTaken
		}

		if err != core.ErrAccessPolicyNotFound {
			return nil, err
		}
	}

	// creating in the store
	ap, err := c.store.Create(ctx, ap)
	if err != nil {
		return nil, fmt.Errorf("failed to create new access policy: %s", err)
	}

	// adding new policy to the registry
	err = c.Put(ap)
	if err != nil {
		return nil, fmt.Errorf("failed to add access policy to container registry: %s", err)
	}

	return ap, nil
}

// UpdateAccessPolicy updates existing access policy
func (c *Manager) Save(ctx context.Context, ap *AccessPolicy) (err error) {
	// deferring a function that will restore backup, in case
	// of any error
	defer func() {
		if r := recover(); r != nil {
			err = r.(error)

			// restoring policy backup
			if backupErr := ap.RestoreBackup(); backupErr != nil {
				err = fmt.Errorf(
					"recovered from panic, failed to restore policy backup (id=%d): %s [policy backup restoration failed: %s]",
					ap.ID,
					r,
					backupErr,
				)
			}
		}
	}()

	// validating before creation
	err = ap.Validate()
	if err != nil {
		panic(err)
	}

	// checking whether name is available, and if it already
	// exists and doesn't belong to this access policy, then
	// returning an error
	if ap.Name[0] != 0 {
		existingPolicy, err := c.GetByName(ap.Name)
		if err != nil {
			if err != core.ErrAccessPolicyNotFound {
				panic(err)
			}
		} else {
			if existingPolicy.ID != ap.ID {
				panic(core.ErrAccessPolicyNameTaken)
			}
		}
	}

	// checking by an object, just in case kind and id changes,
	// and new kind and object is already attached to a different
	// access policy
	if ap.ObjectType[0] != 0 && ap.ObjectID != 0 {
		existingPolicy, err := c.GetByObjectTypeAndID(ap.ObjectType, ap.ObjectID)
		if err != nil {
			if err != core.ErrAccessPolicyNotFound {
				panic(err)
			}
		} else {
			if existingPolicy.ID != ap.ID {
				panic(core.ErrAccessPolicyKindAndIDTaken)
			}
		}
	}

	// creating in the store
	err = c.store.UpdateAccessPolicy(ctx, ap)
	if err != nil {
		panic(fmt.Errorf("failed to save access policy: %s", err))
	}

	// at this point it's safe to assume that everything went well,
	// thus, clearing backup and rights roster changelist
	ap.backup = nil
	ap.RightsRoster.changes = nil

	return nil
}

// GetByID returns an access policy by its ID
func (c *Manager) GetByID(id int) (*AccessPolicy, error) {
	// checking cache first
	c.RLock()
	ap, ok := c.idMap[id]
	c.RUnlock()

	// return if found in cache
	if ok {
		return ap, nil
	}

	// attempting to obtain policy from the store
	ap, err := c.store.GetByID(id)
	if err != nil {
		return nil, err
	}

	// adding policy to registry
	err = c.Put(ap)
	if err != nil {
		return nil, err
	}

	return ap, nil
}

// GetByName returns an access policy by its key
func (c *Manager) GetByName(name TAPName) (*AccessPolicy, error) {
	c.RLock()
	ap, ok := c.namedMap[name]
	c.RUnlock()

	// return if found in cache
	if ok {
		return ap, nil
	}

	// attempting to obtain policy from the store
	ap, err := c.store.GetByName(name)
	if err != nil {
		return nil, err
	}

	// adding policy to registry
	err = c.Put(ap)
	if err != nil {
		return nil, err
	}

	return ap, nil
}

// GetByObjectTypeAndID returns an access policy by its kind and id
// TODO: add cache
func (c *Manager) GetByObjectTypeAndID(objectType TAPObjectType, id int) (*AccessPolicy, error) {
	// attempting to obtain policy from the store
	ap, err := c.store.GetByObjectAndID(objectType, id)
	if err != nil {
		return nil, err
	}

	// adding policy to registry
	err = c.Put(ap)
	if err != nil {
		return nil, err
	}

	return ap, nil
}

// Delete returns an access policy by its ID
func (c *Manager) Delete(ap *AccessPolicy) error {
	err := ap.Validate()
	if err != nil {
		return err
	}

	_, err = c.GetByID(ap.ID)
	if err != nil {
		return err
	}

	// deleting from the store
	err = c.store.Delete(ap)
	if err != nil {
		return err
	}

	// adding policy to registry
	err = c.Remove(ap)
	if err != nil {
		if err == core.ErrAccessPolicyNotFound {
			return nil
		}

		return err
	}

	return nil
}

// HasRights checks whether a given subject entity has the inquired rights
func (c *Manager) HasRights(ap *AccessPolicy, subject interface{}, rights AccessRight) bool {
	if subject == nil {
		return false
	}

	switch subject.(type) {
	case nil:
	case *user.User:
		return ap.HasRights(subject.(*user.User), rights)
	case *group.Group:
		return ap.HasGroupRights(subject.(*group.Group), rights)
	}

	return false
}

// SetRights sets rights on a given policy, to a subject, by an assignor
// NOTE: can be called multiple times before policy changes are persisted
// NOTE: rights roster changes are not persisted unless explicitly saved
// NOTE: changes made with this function will be cancelled and backup restored
// if there will be any errors when saving this policy
func (c *Manager) SetRights(ap *AccessPolicy, assignor *user.User, subject interface{}, rights AccessRight) error {
	if err := ap.Validate(); err != nil {
		return err
	}

	// checking whether there already is a copy backed up
	if ap.backup == nil {
		// preserving a copy of this access policy by storing a backup inside itself
		backup, err := ap.Clone()
		if err != nil {
			return err
		}

		ap.backup = backup
	}

	var err error

	// setting rights depending on the type of a subject
	switch subject.(type) {
	case nil:
		err = ap.SetPublicRights(assignor, rights)
		ap.RightsRoster.addChange(1, SKEveryone, 0, rights)
	case *user.User:
		u := subject.(*user.User)
		err = ap.SetUserRights(assignor, u, rights)
		ap.RightsRoster.addChange(1, SKUser, u.ID, rights)
	case *group.Group:
		switch group := subject.(*group.Group); group.Kind {
		case group.GKRole:
			err = ap.SetRoleRights(assignor, group, rights)
			ap.RightsRoster.addChange(1, SKRoleGroup, group.ID, rights)
		case group.GKGroup:
			err = ap.SetGroupRights(assignor, group, rights)
			ap.RightsRoster.addChange(1, SKGroup, group.ID, rights)
		}
	}

	// clearing changes in case of an error
	if err != nil {
		ap.RightsRoster.clearChanges()
	}

	return err
}

// UnsetRights removes rights of a given subject to this policy
func (c *Manager) UnsetRights(ap *AccessPolicy, assignor *user.User, subject interface{}) error {
	err := ap.Validate()
	if err != nil {
		return err
	}

	err = ap.CreateBackup()
	if err != nil {
		return err
	}

	// setting rights depending on the type of a subject
	switch subject.(type) {
	case *user.User:
		err = ap.UnsetRights(assignor, subject.(*user.User))
		ap.RightsRoster.addChange(0, SKUser, subject.(*user.User).ID, 0)
	case *group.Group:
		switch group := subject.(*group.Group); group.Kind {
		case group.GKRole:
			err = ap.UnsetRights(assignor, group)
			ap.RightsRoster.addChange(0, SKRoleGroup, subject.(*group.Group).ID, 0)
		case group.GKGroup:
			err = ap.UnsetRights(assignor, group)
			ap.RightsRoster.addChange(0, SKGroup, subject.(*group.Group).ID, 0)
		}
	}

	// clearing changes in case of an error
	if err != nil {
		ap.RightsRoster.clearChanges()
	}

	return err
}
