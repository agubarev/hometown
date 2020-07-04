package accesspolicy

import (
	"context"
	"log"
	"sync"

	"github.com/agubarev/hometown/pkg/group"
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
	ErrPolicyKeyTaken               = errors.New("policy name is taken")
	ErrPolicyObjectConflict         = errors.New("id of a kind is taken")
	ErrEmptyKey                     = errors.New("key is empty")
	ErrEmptyObjectType              = errors.New("object type is empty")
	ErrNilRoster                    = errors.New("rights rosters is nil")
	ErrCacheMiss                    = errors.New("roster cache miss")
	ErrEmptyRoster                  = errors.New("rights rosters is empty")
	ErrNilAccessPolicy              = errors.New("access policy is nil")
	ErrAccessDenied                 = errors.New("access denied")
	ErrNoViewRight                  = errors.New("user is not allowed to view this")
	ErrZeroAssignorID               = errors.New("assignor user id is zero")
	ErrZeroAssigneeID               = errors.New("assignee user id is zero")
	ErrExcessOfRights               = errors.New("assignor is attempting to set the rights that excess his own")
	ErrSameActor                    = errors.New("assignor and assignee is the same user")
	ErrInvalidParentPolicy          = errors.New("parent policy is invalid")
	ErrNoParent                     = errors.New("parent policy is nil")
	ErrNoBackup                     = errors.New("roster has no backup")
	ErrZeroGroupID                  = errors.New("role group id is zero")
	ErrZeroRoleID                   = errors.New("role id is zero")
	ErrUnrecognizedRosterAction     = errors.New("unrecognized roster action")
)

// Manager is the access policy registry
type Manager struct {
	policies   map[uint32]AccessPolicy
	keyMap     map[TKey]uint32
	roster     map[uint32]*Roster
	groups     *group.Manager
	store      Store
	rosterLock sync.RWMutex
	sync.RWMutex
}

// NewManager initializes a new access policy container
func NewManager(store Store, gm *group.Manager) (*Manager, error) {
	if store == nil {
		return nil, ErrNilStore
	}

	c := &Manager{
		policies: make(map[uint32]AccessPolicy),
		roster:   make(map[uint32]*Roster),
		keyMap:   make(map[TKey]uint32),
		groups:   gm,
		store:    store,
	}

	return c, nil
}

func (m *Manager) putPolicy(ap AccessPolicy, r *Roster) (err error) {
	if err = ap.Validate(); err != nil {
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

func (m *Manager) lookupPolicy(policyID uint32) (ap AccessPolicy, err error) {
	if policyID == 0 {
		return ap, ErrPolicyNotFound
	}

	m.RLock()
	ap, ok := m.policies[policyID]
	m.RUnlock()

	if !ok {
		return ap, ErrPolicyNotFound
	}

	return ap, nil
}

// removePolicy removes policy from container registry
func (m *Manager) removePolicy(policyID uint32) (err error) {
	ap, err := m.lookupPolicy(policyID)
	if err != nil {
		return err
	}

	// clearing out internal cache
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

	// validating new policy object
	if err = ap.Validate(); err != nil {
		return ap, errors.Wrap(err, "new policy validation failed")
	}

	// checking whether the key is available in general
	if ap.Key[0] != 0 {
		_, err = m.PolicyByKey(ctx, ap.Key)
		if err == nil {
			return ap, ErrPolicyKeyTaken
		}

		if err != ErrPolicyNotFound {
			return ap, err
		}
	}

	// checking by an object type and SubjectID
	if ap.ObjectType[0] != 0 && ap.ObjectID != 0 {
		_, err = m.PolicyByObject(ctx, ap.ObjectType, objectID)
		if err == nil {
			return ap, ErrPolicyObjectConflict
		}

		if err != ErrPolicyNotFound {
			return ap, err
		}
	}

	// initializing or re-using rights rosters, depending
	// on whether this policy has a parent from which it inherits
	if parentID != 0 && ap.IsInherited() {
		if _, err = m.lookupPolicy(ap.ParentID); err != nil {
			return ap, errors.Wrapf(err, "failed to obtain parent policy despite having parent id")
		}
	}

	// creating in the store
	ap, r, err := m.store.CreatePolicy(ctx, ap, NewRoster(0))
	if err != nil {
		return ap, errors.Wrap(err, "failed to create new access policy")
	}

	// adding new policy to internal registry
	if err = m.putPolicy(ap, r); err != nil {
		return ap, errors.Wrap(err, "failed to add access policy to container registry")
	}

	return ap, nil
}

// Update updates given access policy
func (m *Manager) Update(ctx context.Context, ap AccessPolicy) (err error) {
	if err = ap.Validate(); err != nil {
		return errors.Wrap(err, "failed to validate access policy before updating")
	}

	// checking whether name is available, and if it already
	// exists and doesn't belong to this access policy, then
	// returning an error
	if ap.Key[0] != 0 {
		existingPolicy, err := m.PolicyByKey(ctx, ap.Key)
		if err != nil {
			if err != ErrPolicyNotFound {
				return errors.Wrapf(err, "failed to obtain policy by key: %s", ap.Key)
			}
		} else {
			if existingPolicy.ID != ap.ID {
				return ErrPolicyKeyTaken
			}
		}
	}

	// checking by an object, just in case kind and id changes,
	// and new kind and object is already attached to a different access policy
	if ap.ObjectType[0] != 0 && ap.ObjectID != 0 {
		existingPolicy, err := m.PolicyByObject(ctx, ap.ObjectType, ap.ObjectID)
		if err != nil {
			if err != ErrPolicyNotFound {
				return errors.Wrapf(err, "failed to obtain policy by object: type=%s, id=%d", ap.ObjectType, ap.ObjectID)
			}
		} else {
			if existingPolicy.ID != ap.ID {
				return ErrPolicyObjectConflict
			}
		}
	}

	r, err := m.RosterByPolicyID(ctx, ap.ID)
	if err != nil {
		return errors.Wrap(err, "failed to obtain policy roster")
	}

	// making changes to the store backend
	if err = m.store.UpdatePolicy(ctx, ap, r); err != nil {
		return errors.Wrap(err, "failed to save updated access policy")
	}

	// clearing roster changes and backup because the policy update was successful
	r.clearChanges()

	return m.putPolicy(ap, r)
}

// PolicyByID returns an access policy by its ObjectID
func (m *Manager) PolicyByID(ctx context.Context, id uint32) (ap AccessPolicy, err error) {
	if id == 0 {
		return ap, ErrZeroPolicyID
	}

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
		return ap, errors.Wrapf(err, "failed to fetch access policy: %d", ap.ID)
	}

	// fetching roster
	r, err := m.store.FetchRosterByPolicyID(ctx, ap.ID)
	if err != nil {
		return ap, errors.Wrapf(err, "failed to fetch rights roster: %d", ap.ID)
	}

	return ap, m.putPolicy(ap, r)
}

// PolicyByKey returns an access policy by its key
func (m *Manager) PolicyByKey(ctx context.Context, name TKey) (ap AccessPolicy, err error) {
	m.RLock()
	ap, ok := m.policies[m.keyMap[name]]
	m.RUnlock()

	// return if found in cache
	if ok {
		return ap, nil
	}

	// attempting to obtain policy from the store
	ap, err = m.store.FetchPolicyByKey(ctx, name)
	if err != nil {
		return ap, err
	}

	// fetching roster
	r, err := m.store.FetchRosterByPolicyID(ctx, ap.ID)
	if err != nil {
		return ap, errors.Wrapf(err, "failed to fetch rights roster: %d", ap.ID)
	}

	// adding policy to registry
	if err = m.putPolicy(ap, r); err != nil {
		return ap, err
	}

	return ap, nil
}

// PolicyByObject returns an access policy by its kind and id
func (m *Manager) PolicyByObject(ctx context.Context, objectType TObjectType, objectID uint32) (ap AccessPolicy, err error) {
	// attempting to obtain policy from the store
	ap, err = m.store.FetchPolicyByObject(ctx, objectID, objectType)
	if err != nil {
		return ap, err
	}

	// fetching roster
	r, err := m.store.FetchRosterByPolicyID(ctx, ap.ID)
	if err != nil {
		return ap, errors.Wrapf(err, "failed to fetch rights roster: %d", ap.ID)
	}

	// adding policy and roster to the registry
	if err = m.putPolicy(ap, r); err != nil {
		return ap, err
	}

	return ap, nil
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
	if err = m.removePolicy(ap.ID); err != nil {
		if err == ErrPolicyNotFound {
			return nil
		}

		return err
	}

	return nil
}

// RosterByPolicy returns the rights roster by its access policy
func (m *Manager) RosterByPolicyID(ctx context.Context, policyID uint32) (r *Roster, err error) {
	if policyID == 0 {
		return nil, ErrZeroPolicyID
	}

	// checking internal cache
	m.rosterLock.RLock()
	r, ok := m.roster[policyID]
	m.rosterLock.RUnlock()

	// returning if cache was found
	if ok {
		return r, nil
	}

	// attempting to obtain policy from the store
	ap, err := m.store.FetchPolicyByID(ctx, policyID)
	if err != nil {
		return r, errors.Wrapf(err, "failed to fetch policy roster: policy_id=%d", policyID)
	}

	// fetching rights roster
	r, err = m.store.FetchRosterByPolicyID(ctx, ap.ID)
	if err != nil {
		// if no roster records are found, then initializing new roster object
		if err == ErrEmptyRoster {
			m.rosterLock.Lock()
			m.roster[ap.ID] = NewRoster(0)
			m.rosterLock.Unlock()
		}

		return r, errors.Wrapf(err, "failed to fetch rights roster: policy_id=%d", ap.ID)
	}

	// adding policy to registry
	if err = m.putPolicy(ap, r); err != nil {
		return r, err
	}

	return r, nil
}

// hasRights checks whether a given subject entity has the inquired rights
func (m *Manager) HasRights(ctx context.Context, kind SubjectKind, policyID, subjectID uint32, rights Right) bool {
	if policyID == 0 {
		return false
	}

	switch kind {
	case SKEveryone:
		return m.HasPublicRights(ctx, policyID, rights)
	case SKUser:
		return m.HasUserRights(ctx, policyID, subjectID, rights)
	case SKRoleGroup:
		return m.HasRoleRights(ctx, policyID, subjectID, rights)
	case SKGroup:
		return m.HasGroupRights(ctx, policyID, subjectID, rights)
	}

	return false
}

// SetRights sets rights on a given policy, to a subject, by an assignor
// NOTE: can be called multiple times before policy changes are persisted
// NOTE: rights rosters changes are not persisted unless explicitly saved
// NOTE: changes made with this function will be cancelled and backup restored
// if there will be any errors when saving this policy
func (m *Manager) SetRights(ctx context.Context, kind SubjectKind, policyID, assignorID, assigneeID uint32, rights Right) (err error) {
	ap, err := m.PolicyByID(ctx, policyID)
	if err != nil {
		return errors.Wrap(err, "failed to obtain access policy")
	}

	r, err := m.RosterByPolicyID(ctx, ap.ID)
	if err != nil {
		return errors.Wrap(err, "failed to obtain rights roster")
	}

	// setting rights depending on the type of a subject
	switch kind {
	case SKEveryone:
		err = m.SetPublicRights(ctx, policyID, assignorID, rights)
	case SKUser:
		err = m.SetUserRights(ctx, policyID, assignorID, assigneeID, rights)
	case SKRoleGroup:
		err = m.SetRoleRights(ctx, policyID, assignorID, assigneeID, rights)
	case SKGroup:
		err = m.SetGroupRights(ctx, policyID, assignorID, assigneeID, rights)
	}

	// clearing changes in case of an error
	if err != nil {
		r.clearChanges()
	}

	return err
}

// UnsetRights takes away current rights on this policy,
// from an assignee, as an assignor
// NOTE: this function only removes exclusive rights of this assignee,
// but the assignee still retains its public level rights to whatver
// that this policy protects
// NOTE: if you wish to completely deny somebody an access through
// this policy, then set exclusive rights explicitly (i.e. APNoAccess, 0)
func (m *Manager) UnsetRights(ctx context.Context, kind SubjectKind, policyID, assignorID, assigneeID uint32) (err error) {
	// safety fuse
	restoreBackup := true

	ap, err := m.PolicyByID(ctx, policyID)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain access policy: policy_id=%d", policyID)
	}

	r, err := m.RosterByPolicyID(ctx, policyID)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain rights roster: policy_id=%d", ap.ID)
	}

	// will restore backup unless successfully cancelled
	defer func() {
		if restoreBackup {
			r.restoreBackup()
		}
	}()

	if assignorID == 0 {
		return ErrZeroAssignorID
	}

	// the assignorID must have a right to set rights (APManageRights) and have all the
	// rights himself that he's attempting to assign to others
	// TODO: consider weighting the rights of who strips whose rights
	if !m.HasRights(ctx, SKUser, policyID, assignorID, APManageRights) {
		return ErrAccessDenied
	}

	// deleting assigneeID from the rosters (depending on its type)
	switch kind {
	case SKEveryone:
		r.change(RSet, SKEveryone, 0, APNoAccess)
	case SKUser:
		r.change(RUnset, SKUser, assigneeID, APNoAccess)
	case SKRoleGroup:
		r.change(RUnset, SKRoleGroup, assigneeID, APNoAccess)
	case SKGroup:
		r.change(RUnset, SKGroup, assigneeID, APNoAccess)
	}

	// all is good, cancelling restoration
	restoreBackup = false

	return nil
}

// SetParentID setting a new parent policy
func (m *Manager) SetParent(ctx context.Context, policyID, parentID uint32) (err error) {
	ap, err := m.lookupPolicy(policyID)
	if err != nil {
		return err
	}

	// disabling inheritance to avoid unexpected behaviour
	// TODO: think it through, is it really obvious to disable inheritance if parent is nil'ed?
	if parentID == 0 {
		ap.Flags &^= FInherit
		ap.ParentID = 0
	} else {
		ap.ParentID = parentID
	}

	return nil
}

// UserAccess returns a summarized access bitmask for a given user
func (m *Manager) UserAccess(ctx context.Context, policyID, userID uint32) (access Right) {
	if userID == 0 {
		return APNoAccess
	}

	// obtaining policy
	ap, err := m.PolicyByID(ctx, policyID)
	if err != nil {
		log.Printf("UserAccess(policy_id=%d, user_id=%d): %s\n", policyID, userID, err)
		return APNoAccess
	}

	// if this user is the owner, then returning maximum possible value for Right type
	if ap.IsOwner(userID) {
		return APFullAccess
	}

	// calculating parents access if parent SubjectID is set
	if ap.ParentID != 0 {
		// obtaining parent object
		parent, err := m.PolicyByID(ctx, ap.ParentID)
		if err != nil {
			panic(errors.Wrap(err, "access policy has parent id set, but failed to obtain parent policy object"))
		}

		// if this policy is flagged as inherited, then
		// calling UserAccess until we reach the actual policy
		if ap.IsInherited() {
			access = m.UserAccess(ctx, ap.ParentID, userID)
		} else {
			// if extend is true and parent exists, then using parent's access as a base value
			if parent.ID != 0 && ap.IsExtended() {
				// addressing the parent because it traces back until it finds
				// the first uninherited, actual policy
				access = m.Summarize(ctx, parent.ID, userID)
			}
		}

		access |= m.Summarize(ctx, ap.ID, userID)
	}

	return access
}

// SetPublicRights setting rights for everyone
func (m *Manager) SetPublicRights(ctx context.Context, policyID, assignorID uint32, rights Right) error {
	// safety fuse
	restoreBackup := true

	r, err := m.RosterByPolicyID(ctx, policyID)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain rights roster: policy_id=%d", policyID)
	}

	// will restore backup unless successfully cancelled
	defer func() {
		if restoreBackup {
			r.restoreBackup()
		}
	}()

	if assignorID == 0 {
		return ErrZeroAssignorID
	}

	// checking whether the assignorID has at least the assigned rights
	if !m.HasRights(ctx, SKUser, policyID, assignorID, APManageRights|rights) {
		return ErrExcessOfRights
	}

	// deferred instruction for change
	r.change(RSet, SKEveryone, 0, rights)

	// all is good, cancelling restoration
	restoreBackup = false

	return nil
}

// SetRoleRights setting rights for the role
func (m *Manager) SetRoleRights(ctx context.Context, policyID, assignorID, roleID uint32, rights Right) error {
	// safety fuse
	restoreBackup := true

	r, err := m.RosterByPolicyID(ctx, policyID)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain rights roster: policy_id=%d", policyID)
	}

	// will restore backup unless successfully cancelled
	defer func() {
		if restoreBackup {
			r.restoreBackup()
		}
	}()

	if assignorID == 0 {
		return ErrZeroAssignorID
	}

	if roleID == 0 {
		return ErrZeroRoleID
	}

	g, err := m.groups.GroupByID(ctx, roleID)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain role group: %d", roleID)
	}

	// making sure it is a role group
	if g.Kind != group.GKRole {
		return errors.Wrapf(
			err,
			"SetRoleRights(policy_id=%d, assignor_id=%d, role_id=%d, rights=%d): expecting %s, got %s",
			policyID, assignorID, roleID, rights, group.GKRole, g.Kind,
		)
	}

	// checking whether assignor has the right to manage,
	// and has at least the assigned rights itself
	if !m.HasRights(ctx, SKUser, policyID, assignorID, APManageRights|rights) {
		return ErrExcessOfRights
	}

	// deferred instruction for change
	r.change(RSet, SKRoleGroup, roleID, rights)

	// all is good, cancelling restoration
	restoreBackup = false

	return nil
}

// SetGroupRights setting group rights for specific user
func (m *Manager) SetGroupRights(ctx context.Context, policyID, assignorID, groupID uint32, rights Right) (err error) {
	// safety fuse
	restoreBackup := true

	r, err := m.RosterByPolicyID(ctx, policyID)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain rights roster: policy_id=%d", policyID)
	}

	// will restore backup unless successfully cancelled
	defer func() {
		if restoreBackup {
			r.restoreBackup()
		}
	}()

	if assignorID == 0 {
		return ErrZeroAssignorID
	}

	if groupID == 0 {
		return ErrZeroGroupID
	}

	g, err := m.groups.GroupByID(ctx, groupID)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain group: %d", groupID)
	}

	// making sure it is a standard group
	if g.Kind != group.GKGroup {
		return errors.Wrapf(
			err,
			"SetGroupRights(policy_id=%d, assignor_id=%d, role_id=%d, rights=%d): expecting %s, got %s",
			policyID, assignorID, groupID, rights, group.GKGroup, g.Kind,
		)
	}

	// checking whether assignor has the right to manage,
	// and has at least the assigned rights itself
	if !m.HasRights(ctx, SKUser, policyID, assignorID, APManageRights|rights) {
		return ErrExcessOfRights
	}

	// deferred instruction for change
	r.change(RSet, SKGroup, groupID, rights)

	// all is good, cancelling restoration
	restoreBackup = false

	return nil
}

// SetUserRights setting rights for specific user
// TODO: consider whether it's right to turn off inheritance (if enabled) when setting/changing anything on each access policy instance
func (m *Manager) SetUserRights(ctx context.Context, policyID, assignorID, assigneeID uint32, rights Right) (err error) {
	// safety fuse
	restoreBackup := true

	r, err := m.RosterByPolicyID(ctx, policyID)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain rights roster: policy_id=%d", policyID)
	}

	// will restore backup unless successfully cancelled
	defer func() {
		if restoreBackup {
			r.restoreBackup()
		}
	}()

	if assignorID == 0 {
		return ErrZeroAssignorID
	}

	if assigneeID == 0 {
		return ErrZeroAssigneeID
	}

	// checking whether assignor has the right to manage,
	// and has at least the assigned rights itself
	if !m.HasRights(ctx, SKUser, policyID, assignorID, APManageRights|rights) {
		return ErrExcessOfRights
	}

	// deferred instruction for change
	r.change(RSet, SKUser, assigneeID, rights)

	// all is good, cancelling restoration
	restoreBackup = false

	return nil
}

// hasRights checks whether the user has specific rights
// NOTE: returns true only if the user has every of specified rights permitted
// TODO: maybe add some sort of a calculated cache with a short lifespan, like 10ms or something
func (m *Manager) HasUserRights(ctx context.Context, policyID, userID uint32, rights Right) bool {
	if userID == 0 {
		return false
	}

	ap, err := m.PolicyByID(ctx, policyID)
	if err != nil {
		return false
	}

	// allow if this user is an owner
	if ap.IsOwner(userID) {
		return true
	}

	// calculated rights
	var cr Right

	// calculating parent-related rights if possible
	if ap.ParentID != 0 {
		// if the current policy is flagged as inherited, then
		// using its parent as the primary source of rights
		if ap.IsInherited() {
			return m.HasUserRights(ctx, ap.ParentID, userID, rights)
		}

		if ap.IsExtended() {
			cr = m.Summarize(ctx, ap.ParentID, userID)
		}
	}

	// merging with the actual policy's rights rosters rights
	// TODO: consider overriding the extended rights with own
	cr |= m.Summarize(ctx, policyID, userID)

	return (cr & rights) == rights
}

// HasPublicRights checks whether a given policy has specific public rights
// NOTE: despite it's narrow purpose, it may still be useful to check public rights alone
func (m *Manager) HasPublicRights(ctx context.Context, policyID uint32, rights Right) bool {
	r, err := m.RosterByPolicyID(ctx, policyID)
	if err != nil {
		return false
	}

	return (r.Everyone & rights) == rights
}

// HasGroupRights checks whether a group has the rights
func (m *Manager) HasGroupRights(ctx context.Context, policyID, groupID uint32, rights Right) bool {
	return (m.GroupRights(ctx, policyID, groupID) & rights) == rights
}

// HasGroupRights checks whether a role has the rights
func (m *Manager) HasRoleRights(ctx context.Context, policyID, groupID uint32, rights Right) bool {
	return (m.GroupRights(ctx, policyID, groupID) & rights) == rights
}

// Summarize summarizing the resulting access rights
func (m *Manager) Summarize(ctx context.Context, policyID, userID uint32) (access Right) {
	ap, err := m.PolicyByID(ctx, policyID)
	if err != nil {
		return APNoAccess
	}

	r, err := m.RosterByPolicyID(ctx, policyID)
	if err != nil {
		return APNoAccess
	}

	// public access is the base right
	access = r.Everyone

	// calculating group rights only if policy manager has a reference
	// to the group manager
	if m.groups != nil {
		// calculating standard and role group rights
		// NOTE: if some group doesn't have explicitly set rights, then
		// attempting to obtain the rights of a first ancestor group,
		// that has specific rights set
		for _, g := range m.groups.GroupsByMemberID(ctx, group.GKRole|group.GKGroup, userID) {
			access |= m.GroupRights(ctx, policyID, g.ID)
		}
	}

	// WARNING: USING USER'S OWNERSHIP TO OVERRIDE ITS ACCESS
	// THIS MEANS THAT OWNERS OF THE PARENT POLICIES WILL HAVE
	// FULL ACCESS TO ITS CHILDREN
	// TODO: CONSIDER THIS VERY STRONGLY
	if ap.IsOwner(userID) {
		access = APFullAccess
	}

	// user-specific rights
	return access | r.lookup(SKUser, userID)
}

// GroupRights returns the rights of a given group if set explicitly,
// otherwise returns the rights of the first ancestor group that has
// any rights record explicitly set
func (m *Manager) GroupRights(ctx context.Context, policyID, groupID uint32) (access Right) {
	if policyID == 0 || groupID == 0 {
		return APNoAccess
	}

	// group manager is mandatory at this point
	if m.groups == nil {
		log.Printf("GroupRights(policy_id=%d, group_id=%d): group manager is nil\n", policyID, groupID)
		return APNoAccess
	}

	// obtaining roster
	r, err := m.RosterByPolicyID(ctx, policyID)
	if err != nil {
		log.Printf("GroupRights(policy_id=%d, group_id=%d): failed to obtain rights roster\n", policyID, groupID)
		return APNoAccess
	}

	// obtaining target group
	g, err := m.groups.GroupByID(ctx, groupID)
	if err != nil {
		return APNoAccess
	}

	switch g.Kind {
	case group.GKGroup:
		access = r.lookup(SKGroup, g.ID)
	case group.GKRole:
		access = r.lookup(SKRoleGroup, g.ID)
	}

	// returning if any positive access right is found
	if access != APNoAccess {
		return access
	}

	// otherwise, looking for the first set access by tracing back
	// through its parents
	if g.ParentID != 0 {
		return m.GroupRights(ctx, policyID, g.ParentID)
	}

	return APNoAccess
}
