package user

import (
	"context"
	"fmt"
	"sync"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
)

// errors
var (
	ErrNilAccessPolicyStore         = errors.New("access policy store is nil")
	ErrNilAccessPolicyManager       = errors.New("access policy container is nil")
	ErrAccessPolicyNotFound         = errors.New("access policy not found")
	ErrAccessPolicyEmptyDesignators = errors.New("both key and kind with id are empty")
	ErrAccessPolicyNameTaken        = errors.New("policy name is taken")
	ErrAccessPolicyKindAndIDTaken   = errors.New("id of a kind is taken")
	ErrAccessPolicyBackupExists     = errors.New("policy backup is already created")
	ErrAccessPolicyBackupNotFound   = errors.New("policy backup not found")
	ErrAccessPolicyNilSubject       = errors.New("subject entity is nil")
	ErrNilRightsRoster              = errors.New("rights roster is nil")
	ErrEmptyRightsRoster            = errors.New("rights roster is empty")
	ErrNilAccessPolicy              = errors.New("access policy is nil")
	ErrAccessDenied                 = errors.New("user access denied")
	ErrNoViewRight                  = errors.New("user is not allowed to view this")
	ErrNilAssignor                  = errors.New("assignor user is nil")
	ErrNilAssignee                  = errors.New("assignee user is nil")
	ErrExcessOfRights               = errors.New("assignor is attempting to set the rights that excess his own")
	ErrSameUser                     = errors.New("assignor and assignee is the same user")
	ErrNilParent                    = errors.New("parent is nil")
)

// AccessPolicyManager is the access policy registry
type AccessPolicyManager struct {
	policies map[int64]AccessPolicy
	nameMap  map[string]int64
	store    AccessPolicyStore
	sync.RWMutex
}

// NewAccessPolicyManager initializes a new access policy container
func NewAccessPolicyManager(store AccessPolicyStore) (*AccessPolicyManager, error) {
	if store == nil {
		return nil, ErrNilAccessPolicyStore
	}

	c := &AccessPolicyManager{
		policies: make(map[int64]AccessPolicy),
		nameMap:  make(map[string]int64),
		store:    store,
	}

	return c, nil
}

// UpsertGroup adds policy to container registry
func (m *AccessPolicyManager) Put(ctx context.Context, ap AccessPolicy) error {
	err := ap.Validate()
	if err != nil {
		return err
	}

	// caching inside container's registry
	m.Lock()
	m.policies[ap.ID] = ap
	m.nameMap[ap.Name] = ap.ID
	m.Unlock()

	return nil
}

// RemovePolicy removes policy from container registry
func (m *AccessPolicyManager) RemovePolicy(ctx context.Context, ap AccessPolicy) error {
	err := ap.Validate()
	if err != nil {
		return err
	}

	// clearing out maps
	m.Lock()
	delete(m.policies, ap.ID)
	delete(m.nameMap, ap.Name)
	m.Unlock()

	return nil
}

// Upsert creates a new access policy
func (m *AccessPolicyManager) Create(ctx context.Context, ownerID, parentID int64, name string, objectType string, objectID int64, isInherited, isExtended bool) (ap AccessPolicy, err error) {
	ap, err = NewAccessPolicy(ownerID, parentID, name, objectType, objectID, isInherited, isExtended)
	if err != nil {
		return ap, errors.Wrap(err, "failed to initialize new access policy")
	}

	// checking whether the name is available
	if ap.Name != "" {
		_, err := m.PolicyByName(ctx, ap.Name)
		if err == nil {
			return ap, ErrAccessPolicyNameTaken
		}

		if err != ErrAccessPolicyNotFound {
			return ap, err
		}
	}

	// checking by an object type and ID
	if ap.ObjectType != "" && ap.ObjectID != 0 {
		_, err := m.PolicyByObjectTypeAndID(ctx, ap.ObjectType, objectID)
		if err == nil {
			return ap, ErrAccessPolicyKindAndIDTaken
		}

		if err != ErrAccessPolicyNotFound {
			return ap, err
		}
	}

	// initializing or re-using rights roster, depending
	// on whether this policy has a parent from which it inherits
	if parentID != 0 && ap.IsInherited {
		apm := ctx.Value(CKAccessPolicyManager).(*AccessPolicyManager)
		if apm == nil {
			panic(ErrNilAccessPolicyManager)
		}

		p, err := ap.Parent(ctx)
		if err != nil {
			panic(ErrNoParentPolicy)
		}

		// just using a pointer to parent rights
		ap.RightsRoster = p.RightsRoster
	} else {
		ap.RightsRoster = NewRightsRoster()
	}

	// creating initial key hash
	// TODO: figure out a way to create hash without id
	//ap.hashKey()

	// validating before creation
	if err := ap.Validate(); err != nil {
		return ap, err
	}

	spew.Dump(ap)

	// creating in the store
	ap, err = m.store.CreatePolicy(ctx, ap)
	if err != nil {
		return ap, errors.Wrap(err, "failed to create new access policy")
	}

	// adding new policy to the registry
	err = m.Put(ctx, ap)
	if err != nil {
		return ap, errors.Wrap(err, "failed to add access policy to container registry")
	}

	return ap, nil
}

// UpdatePolicy updates existing access policy
func (m *AccessPolicyManager) Update(ctx context.Context, ap AccessPolicy) (_ AccessPolicy, err error) {
	// deferring a function that will restore backup, in case of any error
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(err, "recovering from panic")

			// restoring policy backup
			if backupErr := ap.RestoreBackup(); backupErr != nil {
				err = errors.Wrapf(err, "failed to restore policy backup (id=%d): %s", ap.ID, backupErr)
			}
		}
	}()

	// validating before creation
	if err = ap.Validate(); err != nil {
		panic(errors.Wrap(err, "failed to validate updated access policy"))
	}

	// checking whether name is available, and if it already
	// exists and doesn't belong to this access policy, then
	// returning an error
	if ap.Name != "" {
		existingPolicy, err := m.PolicyByName(ctx, ap.Name)
		if err != nil {
			if err != ErrAccessPolicyNotFound {
				panic(err)
			}
		} else {
			if existingPolicy.ID != ap.ID {
				panic(ErrAccessPolicyNameTaken)
			}
		}
	}

	// checking by an object, just in case kind and id changes,
	// and new kind and object is already attached to a different
	// access policy
	if ap.ObjectType != "" && ap.ObjectID != 0 {
		existingPolicy, err := m.PolicyByObjectTypeAndID(ctx, ap.ObjectType, ap.ObjectID)
		if err != nil {
			if err != ErrAccessPolicyNotFound {
				panic(err)
			}
		} else {
			if existingPolicy.ID != ap.ID {
				panic(ErrAccessPolicyKindAndIDTaken)
			}
		}
	}

	// creating in the store
	err = m.store.UpdatePolicy(ctx, ap)
	if err != nil {
		panic(fmt.Errorf("failed to save updated access policy: %s", err))
	}

	// at this point it's safe to assume that everything went well,
	// thus, clearing backup and rights roster changelist
	ap.backup = nil
	ap.RightsRoster.changes = nil

	return ap, nil
}

// GroupByID returns an access policy by its ObjectID
func (m *AccessPolicyManager) PolicyByID(ctx context.Context, id int64) (ap AccessPolicy, err error) {
	// checking cache first
	m.RLock()
	ap, ok := m.policies[id]
	m.RUnlock()

	// return if found in cache
	if ok {
		return ap, nil
	}

	// attempting to obtain policy from the store
	ap, err = m.store.FetchPolicyByID(ctx, id)
	if err != nil {
		return ap, err
	}

	// adding policy to registry
	err = m.Put(ctx, ap)
	if err != nil {
		return ap, err
	}

	return ap, nil
}

// PolicyByName returns an access policy by its key
func (m *AccessPolicyManager) PolicyByName(ctx context.Context, name string) (ap AccessPolicy, err error) {
	m.RLock()
	ap, ok := m.policies[m.nameMap[name]]
	m.RUnlock()

	// return if found in cache
	if ok {
		return ap, nil
	}

	// attempting to obtain policy from the store
	ap, err = m.store.FetchPolicyByName(ctx, name)
	if err != nil {
		return ap, err
	}

	// adding policy to registry
	err = m.Put(ctx, ap)
	if err != nil {
		return ap, err
	}

	return ap, nil
}

// PolicyByObjectTypeAndID returns an access policy by its kind and id
// TODO: add cache
func (m *AccessPolicyManager) PolicyByObjectTypeAndID(ctx context.Context, objectType string, objectID int64) (ap AccessPolicy, err error) {
	// attempting to obtain policy from the store
	ap, err = m.store.FetchPolicyByObjectTypeAndID(ctx, objectType, objectID)
	if err != nil {
		return ap, err
	}

	// adding policy to registry
	err = m.Put(ctx, ap)
	if err != nil {
		return ap, err
	}

	return ap, nil
}

// DeletePolicy returns an access policy by its ObjectID
func (m *AccessPolicyManager) DeletePolicy(ctx context.Context, ap AccessPolicy) (err error) {
	if err = ap.Validate(); err != nil {
		return err
	}

	// deleting from the store
	err = m.store.DeletePolicy(ctx, ap)
	if err != nil {
		return err
	}

	// adding policy to registry
	err = m.RemovePolicy(ctx, ap)
	if err != nil {
		if err == ErrAccessPolicyNotFound {
			return nil
		}

		return err
	}

	return nil
}

// HasRights checks whether a given subject entity has the inquired rights
func (m *AccessPolicyManager) HasRights(ctx context.Context, ap *AccessPolicy, subject interface{}, rights AccessRight) bool {
	if subject == nil {
		return false
	}

	switch subject.(type) {
	case nil:
	case User:
		return ap.HasRights(ctx, subject.(User).ID, rights)
	case Group:
		return ap.HasGroupRights(ctx, subject.(Group).ID, rights)
	}

	return false
}

// SetRights sets rights on a given policy, to a subject, by an assignor
// NOTE: can be called multiple times before policy changes are persisted
// NOTE: rights roster changes are not persisted unless explicitly saved
// NOTE: changes made with this function will be cancelled and backup restored
// if there will be any errors when saving this policy
func (m *AccessPolicyManager) SetRights(ctx context.Context, ap *AccessPolicy, assignorID int64, subject interface{}, rights AccessRight) (err error) {
	if err = ap.Validate(); err != nil {
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

	// setting rights depending on the type of a subject
	switch subject.(type) {
	case nil:
		err = ap.SetPublicRights(ctx, assignorID, rights)
		ap.RightsRoster.addChange(1, SKEveryone, 0, rights)
	case User:
		u := subject.(User)
		err = ap.SetUserRights(ctx, assignorID, u.ID, rights)
		ap.RightsRoster.addChange(1, SKUser, u.ID, rights)
	case Group:
		switch g := subject.(Group); g.Kind {
		case GKRole:
			err = ap.SetRoleRights(ctx, assignorID, g, rights)
			ap.RightsRoster.addChange(1, SKRoleGroup, g.ID, rights)
		case GKGroup:
			err = ap.SetGroupRights(ctx, assignorID, g, rights)
			ap.RightsRoster.addChange(1, SKGroup, g.ID, rights)
		}
	}

	// clearing changes in case of an error
	if err != nil {
		ap.RightsRoster.clearChanges()
	}

	return err
}

// UnsetRights removes rights of a given subject to this policy
func (m *AccessPolicyManager) UnsetRights(ctx context.Context, ap AccessPolicy, assignorID int64, subject interface{}) error {
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
	case User:
		err = ap.UnsetRights(ctx, assignorID, subject.(User))
		ap.RightsRoster.addChange(0, SKUser, subject.(User).ID, 0)
	case Group:
		switch g := subject.(Group); g.Kind {
		case GKRole:
			err = ap.UnsetRights(ctx, assignorID, g)
			ap.RightsRoster.addChange(0, SKRoleGroup, subject.(Group).ID, 0)
		case GKGroup:
			err = ap.UnsetRights(ctx, assignorID, g)
			ap.RightsRoster.addChange(0, SKGroup, subject.(Group).ID, 0)
		}
	}

	// clearing changes in case of an error
	if err != nil {
		ap.RightsRoster.clearChanges()
	}

	return err
}
