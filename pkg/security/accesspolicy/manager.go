package user

import (
	"context"
	"fmt"
	"sync"

	"github.com/pkg/errors"
)

// errors
var (
	ErrNilDatabase                  = errors.New("database is nil")
	ErrZeroPolicyID                 = errors.New("id is zero")
	ErrNonZeroID                    = errors.New("id is not zero")
	ErrNilStore                     = errors.New("access policy store is nil")
	ErrNilAccessPolicyManager       = errors.New("access policy container is nil")
	ErrPolicyNotFound               = errors.New("access policy not found")
	ErrAccessPolicyEmptyDesignators = errors.New("both key and kind with id are empty")
	ErrPolicyNameTaken              = errors.New("policy name is taken")
	ErrPolicyKindAndIDTaken         = errors.New("id of a kind is taken")
	ErrEmptyKey                     = errors.New("key is empty")
	ErrEmptyObjectType              = errors.New("object type is empty")
	ErrNilRoster                    = errors.New("rights rosters is nil")
	ErrCacheMiss                    = errors.New("roster cache miss")
	ErrEmptyRoster                  = errors.New("rights rosters is empty")
	ErrNilAccessPolicy              = errors.New("access policy is nil")
	ErrAccessDenied                 = errors.New("access denied")
	ErrNoViewRight                  = errors.New("user is not allowed to view this")
	ErrZeroAssignorID               = errors.New("assignor user id is zero ")
	ErrZeroAssigneeID               = errors.New("assignee user id is zero")
	ErrExcessOfRights               = errors.New("assignor is attempting to set the rights that excess his own")
	ErrSameActor                    = errors.New("assignor and assignee is the same user")
	ErrNoParentPolicy               = errors.New("no parent policy")
	ErrNoParent                     = errors.New("parent is nil")
)



// Manager is the access policy registry
type Manager struct {
	policies map[uint32]AccessPolicy
	roster   map[uint32]*Roster
	keyMap   map[TKey]uint32
	store    Store
	sync.RWMutex
}

// NewManager initializes a new access policy container
func NewManager(store Store) (*Manager, error) {
	if store == nil {
		return nil, ErrNilStore
	}

	c := &Manager{
		policies: make(map[uint32]AccessPolicy),
		roster:   make(map[uint32]*Roster),
		keyMap:   make(map[TKey]uint32),
		store:    store,
	}

	return c, nil
}

// UpsertGroup adds policy to container registry
func (m *Manager) put(ap AccessPolicy, r *Roster) error {
	err := ap.Validate()
	if err != nil {
		return err
	}

	// caching inside container's registry
	m.Lock()
	m.policies[ap.ID] = ap
	m.roster[ap.ID] = r
	m.keyMap[ap.Key] = ap.ID
	m.Unlock()

	return nil
}

// remove removes policy from container registry
func (m *Manager) remove(ap AccessPolicy) error {
	err := ap.Validate()
	if err != nil {
		return err
	}

	// clearing out maps
	m.Lock()
	delete(m.policies, ap.ID)
	delete(m.roster, ap.ID)
	delete(m.keyMap, ap.Key)
	m.Unlock()

	return nil
}

// Upsert creates a new access policy
func (m *Manager) Create(ctx context.Context, key TKey, ownerID, parentID, objectID uint32, objectType TObjectType, flags uint8) (ap AccessPolicy, err error) {
	ap, err = NewAccessPolicy(key, ownerID, parentID, objectID, objectType, flags)
	if err != nil {
		return ap, errors.Wrap(err, "failed to initialize new access policy")
	}

	// checking whether the key is available
	if ap.Key[0] != 0 {
		_, err := m.PolicyByName(ctx, ap.Key)
		if err == nil {
			return ap, ErrPolicyNameTaken
		}

		if err != ErrPolicyNotFound {
			return ap, err
		}
	}

	// checking by an object type and ID
	if ap.ObjectType[0] != 0 && ap.ObjectID != 0 {
		_, err = m.PolicyByObjectTypeAndID(ctx, ap.ObjectType, objectID)
		if err == nil {
			return ap, ErrPolicyKindAndIDTaken
		}

		if err != ErrPolicyNotFound {
			return ap, err
		}
	}

	// initializing or re-using rights rosters, depending
	// on whether this policy has a parent from which it inherits
	if parentID != 0 && ap.IsInherited {
		apm := ctx.Value(CKAccessPolicyManager).(*Manager)
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
		ap.RightsRoster = NewRoster()
	}

	// creating initial key hash
	// TODO: figure out a way to create hash without id
	//ap.hashKey()

	// validating before creation
	if err := ap.Validate(); err != nil {
		return ap, err
	}

	// creating in the store
	ap, err = m.store.CreatePolicy(ctx, ap)
	if err != nil {
		return ap, errors.Wrap(err, "failed to create new access policy")
	}

	// adding new policy to the registry

	if err = m.put(ctx, ap); err != nil {
		return ap, errors.Wrap(err, "failed to add access policy to container registry")
	}

	return ap, nil
}

// UpdatePolicy updates existing access policy
func (m *Manager) Update(ctx context.Context, ap AccessPolicy) (_ AccessPolicy, err error) {
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
	if ap.Key != "" {
		existingPolicy, err := m.PolicyByName(ctx, ap.Key)
		if err != nil {
			if err != ErrPolicyNotFound {
				panic(err)
			}
		} else {
			if existingPolicy.ID != ap.ID {
				panic(ErrPolicyNameTaken)
			}
		}
	}

	// checking by an object, just in case kind and id changes,
	// and new kind and object is already attached to a different access policy
	if ap.ObjectType != "" && ap.ObjectID != 0 {
		existingPolicy, err := m.PolicyByObjectTypeAndID(ctx, ap.ObjectType, ap.ObjectID)
		if err != nil {
			if err != ErrPolicyNotFound {
				panic(err)
			}
		} else {
			if existingPolicy.ID != ap.ID {
				panic(ErrPolicyKindAndIDTaken)
			}
		}
	}

	// creating in the store
	err = m.store.UpdatePolicy(ctx, ap)
	if err != nil {
		panic(fmt.Errorf("failed to save updated access policy: %s", err))
	}

	// at this point it's safe to assume that everything went well,
	// thus, clearing backup and rights rosters changelist
	ap.backup = nil
	ap.RightsRoster.changes = nil

	// updating policy cache
	if err = m.put(ctx, ap); err != nil {
		return ap, err
	}

	return ap, nil
}

// PolicyByID returns an access policy by its ObjectID
func (m *Manager) PolicyByID(ctx context.Context, id uint32) (ap AccessPolicy, r *Roster, err error) {
	// checking cache first
	m.RLock()
	ap, ok := m.policies[id]
	m.RUnlock()

	// return if found in cache
	if ok {
		return ap, m.roster[ap.ID], nil
	}

	// attempting to obtain policy from the store
	ap, err = m.store.FetchPolicyByID(ctx, id)
	if err != nil {
		return ap, r, err
	}

	// adding policy to registry
	if err = m.put(ap, r); err != nil {
		return ap, r, err
	}

	return ap, r, nil
}

// PolicyByName returns an access policy by its key
func (m *Manager) PolicyByName(ctx context.Context, name TKey) (ap AccessPolicy, r *Roster, err error) {
	m.RLock()
	ap, ok := m.policies[m.keyMap[name]]
	m.RUnlock()

	// return if found in cache
	if ok {
		return ap, m.roster[ap.ID], nil
	}

	// attempting to obtain policy from the store
	ap, err = m.store.FetchPolicyByName(ctx, name)
	if err != nil {
		return ap, r, err
	}

	// adding policy to registry
	if err = m.put(ap, r); err != nil {
		return ap, r, err
	}

	return ap, r, nil
}

// PolicyByObjectTypeAndID returns an access policy by its kind and id
func (m *Manager) PolicyByObjectTypeAndID(ctx context.Context, objectType TObjectType, objectID uint32) (ap AccessPolicy, r *Roster, err error) {
	// attempting to obtain policy from the store
	ap, err = m.store.FetchPolicyByObjectTypeAndID(ctx, objectID, objectType)
	if err != nil {
		return ap, r, err
	}

	// fetching rights roster
	r, err = m.store.FetchRosterByID(ctx, ap.ID)
	if err != nil {
		return ap, r, err
	}

	// adding policy and roster to the registry
	if err = m.put(ap, r); err != nil {
		return ap, r, err
	}

	return ap,  r, nil
}

// DeletePolicy returns an access policy by its ObjectID
func (m *Manager) DeletePolicy(ctx context.Context, ap AccessPolicy) (err error) {
	if err = ap.Validate(); err != nil {
		return errors.Wrap(err, "failed to delete access policy")
	}

	// deleting policy from the store
	// NOTE: also deletes roster
	if err = m.store.DeletePolicy(ctx, ap); err != nil {
		return err
	}

	// adding policy to registry
	if err = m.remove(ap); err != nil {
		if err == ErrPolicyNotFound {
			return nil
		}

		return err
	}

	return nil
}

/*
// HasRights checks whether a given subject entity has the inquired rights
func (m *Manager) HasRights(ctx context.Context, ap *AccessPolicy, subject interface{}, rights Right) bool {
	if subject == nil {
		return false
	}

	switch sub := subject.(type) {
	case nil:
	case User:
		return ap.HasRights(ctx, sub.ID, rights)
	case Group:
		return ap.HasGroupRights(ctx, sub.ID, rights)
	}

	return false
}
 */


// HasRights checks whether a given subject entity has the inquired rights
func (m *Manager) HasRights(ctx context.Context, policyID uint32, sk SubjectKind, subjectID uint32, rights Right) bool {
	if policyID == 0 {
		return false
	}

	switch sk {
	case SKUser:
		return m.HasUserRights(ctx, subjectID, rights)
	case SKRoleGroup:
		return m.HasRoleRights(ctx, subjectID, rights)
	case SKGroup:
		return m.HasGroupRights(ctx, subjectID, rights)
	}

	return false
}

// SetRights sets rights on a given policy, to a subject, by an assignor
// NOTE: can be called multiple times before policy changes are persisted
// NOTE: rights rosters changes are not persisted unless explicitly saved
// NOTE: changes made with this function will be cancelled and backup restored
// if there will be any errors when saving this policy
func (m *Manager) SetRights(ctx context.Context, ap *AccessPolicy, assignorID uint32, subject interface{}, rights Right) (err error) {
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
	switch sub := subject.(type) {
	case nil:
		err = ap.SetPublicRights(ctx, assignorID, rights)
	case User:
		err = ap.SetUserRights(ctx, assignorID, sub.ID, rights)
	case Group:
		switch sub.Kind {
		case GKRole:
			err = ap.SetRoleRights(ctx, assignorID, sub.ID, rights)
		case GKGroup:
			err = ap.SetGroupRights(ctx, assignorID, sub.ID, rights)
		}
	}

	// clearing changes in case of an error
	if err != nil {
		ap.RightsRoster.clearChanges()
	}

	return err
}

// UnsetRights removes rights of a given subject to this policy
func (m *Manager) UnsetRights(ctx context.Context, ap AccessPolicy, assignorID uint32, subject interface{}) (err error) {
	if err = ap.Validate(); err != nil {
		return err
	}

	if err = ap.CreateBackup(); err != nil {
		return err
	}

	// setting rights depending on the type of a subject
	switch sub := subject.(type) {
	case nil:
		err = ap.UnsetRights(ctx, assignorID, nil)
	case User:
		err = ap.UnsetRights(ctx, assignorID, sub)
	case Group:
		switch sub.Kind {
		case GKRole:
			err = ap.UnsetRights(ctx, assignorID, sub)
		case GKGroup:
			err = ap.UnsetRights(ctx, assignorID, sub)
		}
	}

	// clearing changes in case of an error
	if err != nil {
		ap.RightsRoster.clearChanges()
	}

	return err
}

// Parent returns a parent policy, if set
func (ap AccessPolicy) Parent(ctx context.Context) (p AccessPolicy, err error) {
	if ap.ParentID == 0 {
		return p, ErrNoParentPolicy
	}

	apm := ctx.Value(CKAccessPolicyManager).(*Manager)
	if apm == nil {
		return p, ErrNilAccessPolicyManager
	}

	return apm.PolicyByID(ctx, ap.ParentID)
}

// SetParentID setting a new parent policy
// NOTE: if the parent is set to nil, then forcing IsInherited flag to false
func (ap *AccessPolicy) SetParentID(parentID uint32) error {
	ap.Lock()

	// disabling inheritance to avoid unexpected behaviour
	// TODO: think it through, is it really obvious to disable inheritance if parent is nil'ed?
	if parentID == 0 {
		ap.IsInherited = false
		ap.ParentID = 0
	} else {
		ap.ParentID = parentID
	}

	ap.Unlock()

	return nil
}

// UserAccess returns a summarized access bitmask for a given user
// TODO: move to the Manager
func (ap *AccessPolicy) UserAccess(ctx context.Context, userID uint32) (rights Right) {
	if userID == 0 {
		return APNoAccess
	}

	// if this user is the owner, then returning maximum possible value for Right type
	if ap.IsOwner(ctx, userID) {
		return APFullAccess
	}

	// calculating parents access if parent ID is set
	if ap.ParentID != 0 {
		apm := ctx.Value(CKAccessPolicyManager).(*Manager)
		if apm == nil {
			panic(ErrNilAccessPolicyManager)
		}

		// obtaining parent object
		p, err := ap.Parent(ctx)
		if err != nil {
			panic(errors.Wrap(err, "access policy has parent id set, but failed to obtain parent policy object"))
		}

		// if IsInherited is true, then calling UserAccess until we reach the actual policy
		if ap.IsInherited {
			rights = p.UserAccess(ctx, userID)
		} else {
			ap.RLock()
			// if extend is true and parent exists, then using parent's rights as a base value
			if p.ID != 0 && ap.IsExtended {
				// addressing the parent because it traces back until it finds
				// the first uninherited, actual policy
				rights = p.RightsRoster.Summarize(ctx, userID)
			}
			ap.RUnlock()
		}

		rights |= ap.RightsRoster.Summarize(ctx, userID)
	}

	return rights
}

// SetPublicRights setting rights for everyone
// TODO: move to the Manager
func (ap *AccessPolicy) SetPublicRights(ctx context.Context, assignorID uint32, rights Right) error {
	if assignorID == 0 {
		return ErrZeroUserID
	}

	// checking whether the assignorID has at least the assigned rights
	if !ap.HasRights(ctx, assignorID, rights) {
		return ErrExcessOfRights
	}

	ap.Lock()
	ap.RightsRoster.Everyone = rights
	ap.Unlock()

	// deferred instruction for change
	ap.RightsRoster.addChange(RSet, SKEveryone, 0, rights)

	return nil
}

// SetRoleRights setting rights for the role
// TODO: move to the Manager
func (ap *AccessPolicy) SetRoleRights(ctx context.Context, assignorID uint32, roleID uint32, rights Right) error {
	if assignorID == 0 {
		return ErrZeroAssignorID
	}

	if roleID == 0 {
		return ErrZeroRoleID
	}

	/*
		// making sure it's group kind is Role
		if roleID.Kind != GKRole {
			return ErrInvalidKind
		}
	*/

	// checking whether the assignorID has at least the assigned rights
	if !ap.HasRights(ctx, assignorID, rights) {
		return ErrExcessOfRights
	}

	ap.Lock()
	ap.RightsRoster.Role[roleID] = rights
	ap.Unlock()

	// deferred instruction for change
	ap.RightsRoster.addChange(RSet, SKRoleGroup, roleID, rights)

	return nil
}

// SetGroupRights setting group rights for specific user
// TODO: move to the Manager
func (ap *AccessPolicy) SetGroupRights(ctx context.Context, assignorID uint32, groupID uint32, rights Right) (err error) {
	if assignorID == 0 {
		return ErrZeroAssignorID
	}

	if groupID == 0 {
		return ErrZeroGroupID
	}

	/*
		// making sure it's g kind is Group
		if groupID.Kind != GKGroup {
			return ErrInvalidKind
		}
	*/

	// checking whether the assignorID has at least the assigned rights
	if !ap.HasRights(ctx, assignorID, rights) {
		return ErrExcessOfRights
	}

	ap.Lock()
	ap.RightsRoster.Group[groupID] = rights
	ap.Unlock()

	// deferred instruction for change
	ap.RightsRoster.addChange(RSet, SKGroup, groupID, rights)

	return nil
}

// SetUserRights setting rights for specific user
// TODO: consider whether it's right to turn off inheritance (if enabled) when setting/changing anything on each access policy instance
func (ap *AccessPolicy) SetUserRights(ctx context.Context, assignorID uint32, assigneeID uint32, rights Right) error {
	if assignorID == 0 {
		return ErrZeroAssignorID
	}

	if assigneeID == 0 {
		return ErrZeroAssigneeID
	}

	// the assignorID must have a right to set rights (APManageRights) and have all the
	// rights himself that he's attempting to assign to others
	if !ap.HasRights(ctx, assignorID, APManageRights|rights) {
		return ErrExcessOfRights
	}

	ap.Lock()
	ap.RightsRoster.User[assigneeID] = rights
	ap.Unlock()

	// deferred instruction for change
	ap.RightsRoster.addChange(RSet, SKUser, assigneeID, rights)

	return nil
}

// IsOwner checks whether a given user is the owner of this policy
func (ap *AccessPolicy) IsOwner(ctx context.Context, userID uint32) bool {
	// owner of the policy (meaning: the main entity) has full rights on it
	if ap.OwnerID != 0 && (ap.OwnerID == userID) {
		return true
	}

	return false
}

// HasRights checks whether the user has specific rights
// NOTE: returns true only if the user has every of specified rights permitted
// TODO: maybe add some sort of a calculated cache with a short livespan, like 10ms or something
func (ap *AccessPolicy) HasRights(ctx context.Context, userID uint32, rights Right) bool {
	if userID == 0 {
		return false
	}

	// allow if this user is an owner
	if ap.IsOwner(ctx, userID) {
		return true
	}

	if ap.RightsRoster == nil {
		return false
	}

	// calculated rights
	var cr Right

	// calculating parent-related rights if possible
	if ap.ParentID != 0 {
		if p, err := ap.Parent(ctx); err == nil {
			if ap.IsInherited {
				return p.HasRights(ctx, userID, rights)
			}

			if ap.IsExtended {
				ap.RLock()
				cr = p.RightsRoster.Summarize(ctx, userID)
				ap.RUnlock()
			}
		}
	}

	// merging with the actual policy's rights rosters rights
	ap.RLock()
	cr |= ap.RightsRoster.Summarize(ctx, userID)
	ap.RUnlock()

	return (cr & rights) == rights
}

// HasGroupRights checks whether a group has the rights
func (ap *AccessPolicy) HasGroupRights(ctx context.Context, groupID uint32, rights Right) bool {
	if groupID == 0 {
		return false
	}

	if ap.RightsRoster == nil {
		return false
	}

	return (ap.RightsRoster.GroupRights(ctx, groupID) & rights) == rights
}

// UnsetRights takes away current rights on this policy,
// from an assignee, as an assignor
// NOTE: this function only removes exclusive rights of this assignee,
// but the assignee still retains its public level rights to this policy
// NOTE: if you wish to completely deny access to this policy, then
// better set exclusive rights explicitly (i.e. APNoAccess, 0)
func (ap *AccessPolicy) UnsetRights(ctx context.Context, assignorID uint32, assignee interface{}) error {
	if assignorID == 0 {
		return ErrZeroUserID
	}

	// the assignorID must have a right to set rights (APManageRights) and have all the
	// rights himself that he's attempting to assign to others
	if !ap.HasRights(ctx, assignorID, APManageRights) {
		return ErrAccessDenied
	}

	ap.Lock()

	// deleting assignee from the rosters (depending on its type)
	switch sub := assignee.(type) {
	case nil:
		ap.RightsRoster.Everyone = APNoAccess
		ap.RightsRoster.addChange(RSet, SKEveryone, 0, 0)
	case User:
		delete(ap.RightsRoster.User, sub.ID)
		ap.RightsRoster.addChange(RUnset, SKUser, sub.ID, 0)
	case Group:
		switch sub.Kind {
		case GKRole:
			delete(ap.RightsRoster.Role, sub.ID)
			ap.RightsRoster.addChange(RUnset, SKRoleGroup, sub.ID, 0)
		case GKGroup:
			delete(ap.RightsRoster.Group, sub.ID)
			ap.RightsRoster.addChange(RUnset, SKGroup, sub.ID, 0)
		}
	}

	ap.Unlock()

	return nil
}

// Summarize summarizing the resulting access rights
func (m *Manager) Summarize(ctx context.Context, userID uint32) (access Right) {
	r = r.Everyone

	// calculating standard and role group rights
	// NOTE: if some group doesn't have explicitly set rights, then
	// attempting to obtain the rights of a first ancestor group,
	// that has specific rights set
	for _, g := range gm.GroupsByUserID(ctx, userID, GKRole|GKGroup) {
		r |= r.GroupRights(ctx, g.ID)
	}

	// user-specific rights
	if _, ok := r.User[userID]; ok {
		r |= r.User[userID]
	}

	return r
}

// GroupRights returns the rights of a given group if set explicitly,
// otherwise returns the rights of the first ancestor group that has
// any rights record explicitly set
func (r *Roster) GroupRights(ctx context.Context, groupID uint32) Right {
	if groupID == 0 {
		return APNoAccess
	}

	var rights Right
	var ok bool

	// obtaining group manager
	gm := ctx.Value(CKGroupManager).(*GroupManager)
	if gm == nil {
		panic(ErrNilGroupManager)
	}

	// obtaining target group
	g, err := gm.GroupByID(ctx, groupID)
	if err != nil {
		return APNoAccess
	}

	r.RLock()

	switch g.Kind {
	case GKGroup:
		rights, ok = r.Group[g.ID]
	case GKRole:
		rights, ok = r.Role[g.ID]
	}

	r.RUnlock()

	if ok {
		return rights
	}

	// now looking for the first set rights by tracing back
	// through its parents
	if g.ParentID != 0 {
		return r.GroupRights(ctx, g.ParentID)
	}

	return APNoAccess
}
