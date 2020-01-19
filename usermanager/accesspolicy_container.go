package usermanager

import (
	"fmt"
	"strings"
	"sync"
)

// AccessPolicyContainer is a registry for convenience
type AccessPolicyContainer struct {
	idMap       map[int64]*AccessPolicy
	keyMap      map[string]*AccessPolicy
	objectIDMap map[string]*AccessPolicy
	store       AccessPolicyStore
	sync.RWMutex
}

// NewAccessPolicyContainer initializes a new access policy container
func NewAccessPolicyContainer(store AccessPolicyStore) (*AccessPolicyContainer, error) {
	if store == nil {
		return nil, ErrNilAccessPolicyStore
	}

	c := &AccessPolicyContainer{
		idMap:       make(map[int64]*AccessPolicy),
		keyMap:      make(map[string]*AccessPolicy),
		objectIDMap: make(map[string]*AccessPolicy),
		store:       store,
	}

	return c, nil
}

// Add adds policy to container registry
func (c *AccessPolicyContainer) Add(ap *AccessPolicy) error {
	err := ap.Validate()
	if err != nil {
		return err
	}

	// assigning container to this policy for backreference
	ap.container = c

	// caching inside container's registry
	c.Lock()
	c.idMap[ap.ID] = ap
	c.keyMap[ap.Key] = ap
	c.objectIDMap[ap.objectIDIndex()] = ap
	c.Unlock()

	return nil
}

// Remove removes policy from container registry
func (c *AccessPolicyContainer) Remove(ap *AccessPolicy) error {
	err := ap.Validate()
	if err != nil {
		return err
	}

	// clearing container reference
	ap.container = nil

	// clearing out maps
	c.Lock()
	delete(c.idMap, ap.ID)
	delete(c.keyMap, ap.Key)
	delete(c.objectIDMap, ap.objectIDIndex())
	c.Unlock()

	return nil
}

// Create creates a new access policy
func (c *AccessPolicyContainer) Create(owner *User, parent *AccessPolicy, key string,
	objectKind string, objectID int64, isInherited bool, isExtended bool) (*AccessPolicy, error) {
	// initializing and creating new policy
	ap := NewAccessPolicy(owner, parent, isInherited, isExtended)

	// setting distinctive markers
	ap.Key = strings.ToLower(strings.TrimSpace(key))
	ap.ObjectKind = strings.ToLower(strings.TrimSpace(objectKind))
	ap.ObjectID = objectID

	// validating before creation
	if err := ap.Validate(); err != nil {
		return nil, err
	}

	// checking whether a key is available
	if ap.Key != "" {
		_, err := c.GetByKey(key)
		if err == nil {
			return nil, ErrAccessPolicyKeyTaken
		}

		if err != ErrAccessPolicyNotFound {
			return nil, err
		}
	}

	// checking by an object
	if ap.ObjectKind != "" && ap.ObjectID != 0 {
		_, err := c.GetByKindAndID(objectKind, objectID)
		if err == nil {
			return nil, ErrAccessPolicyKindAndIDTaken
		}

		if err != ErrAccessPolicyNotFound {
			return nil, err
		}
	}

	// creating in the store
	ap, err := c.store.Create(ap)
	if err != nil {
		return nil, fmt.Errorf("failed to create new access policy: %s", err)
	}

	// adding new policy to the registry
	err = c.Add(ap)
	if err != nil {
		return nil, fmt.Errorf("failed to add access policy to container registry: %s", err)
	}

	return ap, nil
}

// Save updates existing access policy
func (c *AccessPolicyContainer) Save(ap *AccessPolicy) (err error) {
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

	// checking whether a key is available, and if it already
	// exists and doesn't belong to this access policy, then
	// returning an error
	if ap.Key != "" {
		existingPolicy, err := c.GetByKey(ap.Key)
		if err != nil {
			if err != ErrAccessPolicyNotFound {
				panic(err)
			}
		} else {
			if err == nil && existingPolicy.ID != ap.ID {
				panic(ErrAccessPolicyKeyTaken)
			}
		}
	}

	// checking by an object, just in case kind and id changes,
	// and new kind and object is already attached to a different
	// access policy
	if ap.ObjectKind != "" && ap.ObjectID != 0 {
		existingPolicy, err := c.GetByKindAndID(ap.ObjectKind, ap.ObjectID)
		if err != nil {
			if err != ErrAccessPolicyNotFound {
				panic(err)
			}
		} else {
			if err == nil && existingPolicy.ID != ap.ID {
				panic(ErrAccessPolicyKindAndIDTaken)
			}
		}
	}

	// creating in the store
	err = c.store.Update(ap)
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
func (c *AccessPolicyContainer) GetByID(id int64) (*AccessPolicy, error) {
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
	err = c.Add(ap)
	if err != nil {
		return nil, err
	}

	return ap, nil
}

// GetByKey returns an access policy by its key
func (c *AccessPolicyContainer) GetByKey(key string) (*AccessPolicy, error) {
	c.RLock()
	ap, ok := c.keyMap[key]
	c.RUnlock()

	// return if found in cache
	if ok {
		return ap, nil
	}

	// attempting to obtain policy from the store
	ap, err := c.store.GetByKey(key)
	if err != nil {
		return nil, err
	}

	// adding policy to registry
	err = c.Add(ap)
	if err != nil {
		return nil, err
	}

	return ap, nil
}

// GetByKindAndID returns an access policy by its kind and id
func (c *AccessPolicyContainer) GetByKindAndID(kind string, id int64) (*AccessPolicy, error) {
	// checking cache first
	c.RLock()
	ap, ok := c.objectIDMap[fmt.Sprintf("%s_%d", kind, id)]
	c.RUnlock()

	// return if found in cache
	if ok {
		return ap, nil
	}

	// attempting to obtain policy from the store
	ap, err := c.store.GetByKindAndID(kind, id)
	if err != nil {
		return nil, err
	}

	// adding policy to registry
	err = c.Add(ap)
	if err != nil {
		return nil, err
	}

	return ap, nil
}

// Delete returns an access policy by its ID
func (c *AccessPolicyContainer) Delete(ap *AccessPolicy) error {
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
		if err == ErrAccessPolicyNotFound {
			return nil
		}

		return err
	}

	return nil
}

// HasRights checks whether a given subject entity has the inquired rights
func (c *AccessPolicyContainer) HasRights(ap *AccessPolicy, subject interface{}, rights AccessRight) bool {
	if subject == nil {
		return false
	}

	switch subject.(type) {
	case nil:
	case *User:
		return ap.HasRights(subject.(*User), rights)
	case *Group:
		return ap.HasGroupRights(subject.(*Group), rights)
	}

	return false
}

// SetRights sets rights on a given policy, to a subject, by an assignor
// NOTE: can be called multiple times before policy changes are persisted
// NOTE: rights roster changes are not persisted unless explicitly saved
// NOTE: changes made with this function will be cancelled and backup restored
// if there will be any errors when saving this policy
func (c *AccessPolicyContainer) SetRights(ap *AccessPolicy, assignor *User, subject interface{}, rights AccessRight) error {
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
		ap.RightsRoster.addChange(1, "everyone", 0, rights)
	case *User:
		user := subject.(*User)
		err = ap.SetUserRights(assignor, user, rights)
		ap.RightsRoster.addChange(1, "user", user.ID, rights)
	case *Group:
		switch group := subject.(*Group); group.Kind {
		case GKRole:
			err = ap.SetRoleRights(assignor, group, rights)
			ap.RightsRoster.addChange(1, "role", group.ID, rights)
		case GKGroup:
			err = ap.SetGroupRights(assignor, group, rights)
			ap.RightsRoster.addChange(1, "group", group.ID, rights)
		}
	}

	// clearing changes in case of an error
	if err != nil {
		ap.RightsRoster.clearChanges()
	}

	return err
}

// UnsetRights removes rights of a given subject to this policy
func (c *AccessPolicyContainer) UnsetRights(ap *AccessPolicy, assignor *User, subject interface{}) error {
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
	case *User:
		err = ap.UnsetRights(assignor, subject.(*User))
		ap.RightsRoster.addChange(0, "user", subject.(*User).ID, 0)
	case *Group:
		switch group := subject.(*Group); group.Kind {
		case GKRole:
			err = ap.UnsetRights(assignor, group)
			ap.RightsRoster.addChange(0, "role", subject.(*Group).ID, 0)
		case GKGroup:
			err = ap.UnsetRights(assignor, group)
			ap.RightsRoster.addChange(0, "group", subject.(*Group).ID, 0)
		}
	}

	// clearing changes in case of an error
	if err != nil {
		ap.RightsRoster.clearChanges()
	}

	return err
}
