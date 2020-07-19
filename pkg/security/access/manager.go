package access

import (
	"context"
	"log"
	"sync"

	"github.com/agubarev/hometown/pkg/group"
	"github.com/google/uuid"
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
	ErrKeyTooLong                   = errors.New("key is too long")
	ErrObjectNameTooLong            = errors.New("object name is too long")
	ErrForbiddenChange              = errors.New("access policy key, object name or id is not allowed to rosterChange")
)

// Manager is the access policy registry
type Manager struct {
	policies   map[uuid.UUID]Policy
	keyMap     map[Key]uuid.UUID
	roster     map[uuid.UUID]*Roster
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
		policies: make(map[uuid.UUID]Policy),
		roster:   make(map[uuid.UUID]*Roster),
		keyMap:   make(map[Key]uuid.UUID),
		groups:   gm,
		store:    store,
	}

	return c, nil
}

func (m *Manager) putPolicy(ap Policy, r *Roster) (err error) {
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

func (m *Manager) lookupPolicy(id uuid.UUID) (p Policy, err error) {
	if id == uuid.Nil {
		return p, ErrPolicyNotFound
	}

	m.RLock()
	p, ok := m.policies[id]
	m.RUnlock()

	if !ok {
		return p, ErrPolicyNotFound
	}

	return p, nil
}

// removePolicy removes policy from container registry
func (m *Manager) removePolicy(policyID uuid.UUID) (err error) {
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
func (m *Manager) Create(ctx context.Context, key Key, ownerID, parentID uuid.UUID, obj Object, flags uint8) (ap Policy, err error) {
	ap, err = NewPolicy(key, ownerID, parentID, obj, flags)
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

	// checking by an object type and ID
	if ap.ObjectName[0] != 0 && ap.ObjectID != uuid.Nil {
		_, err = m.PolicyByObject(ctx, obj)
		if err == nil {
			return ap, ErrPolicyObjectConflict
		}

		if err != ErrPolicyNotFound {
			return ap, err
		}
	}

	// initializing or re-using rights rosters, depending
	// on whether this policy has a parent from which it inherits
	if parentID != uuid.Nil {
		if _, err = m.PolicyByID(ctx, ap.ParentID); err != nil {
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
func (m *Manager) Update(ctx context.Context, p Policy) (err error) {
	if err = p.Validate(); err != nil {
		return errors.Wrap(err, "failed to validate access policy before updating")
	}

	currentPolicy, err := m.PolicyByID(ctx, p.ID)
	if err != nil {
		return errors.Wrap(err, "failed to obtain current policy")
	}

	//-!!!-[ WARNING ]-----------------------------------------------------------
	// !!! KEY, OBJECT NAME AND ID ARE NOT ALLOWED TO BE CHANGED BECAUSE CURRENT
	// !!! VALUES ARE/COULD BE RELIED UPON ELSEWHERE AND MUST REMAIN THE SAME
	//-!!!-----------------------------------------------------------------------
	if p.Key != currentPolicy.Key {
		return ErrForbiddenChange
	}

	if p.ObjectID != currentPolicy.ObjectID {
		return ErrForbiddenChange
	}

	if p.ObjectName != currentPolicy.ObjectName {
		return ErrForbiddenChange
	}

	// checking whether name is available, and if it already
	// exists and doesn't belong to this access policy, then
	// returning an error
	if p.Key[0] != 0 {
		existingPolicy, err := m.PolicyByKey(ctx, p.Key)
		if err != nil {
			if err != ErrPolicyNotFound {
				return errors.Wrapf(err, "failed to obtain policy by key: %s", p.Key)
			}
		} else {
			if existingPolicy.ID != p.ID {
				return ErrPolicyKeyTaken
			}
		}
	}

	// checking by an object, just in case kind and id changes,
	// and new kind and object is already attached to a different access policy
	if p.ObjectName[0] != 0 && p.ObjectID != uuid.Nil {
		existingPolicy, err := m.PolicyByObject(ctx, NewObject(p.ObjectID, p.ObjectName))
		if err != nil {
			if err != ErrPolicyNotFound {
				return errors.Wrapf(err, "failed to obtain policy by object: type=%s, id=%d", p.ObjectName, p.ObjectID)
			}
		} else {
			if existingPolicy.ID != p.ID {
				return ErrPolicyObjectConflict
			}
		}
	}

	r, err := m.RosterByPolicyID(ctx, p.ID)
	if err != nil {
		return errors.Wrap(err, "failed to obtain policy roster")
	}

	// making changes to the store backend
	if err = m.store.UpdatePolicy(ctx, p, r); err != nil {
		return errors.Wrap(err, "failed to save updated access policy")
	}

	// clearing roster changes and backup because the policy update was successful
	r.clearChanges()

	return m.putPolicy(p, r)
}

// PolicyByID returns an access policy by its ObjectID
func (m *Manager) PolicyByID(ctx context.Context, id uuid.UUID) (p Policy, err error) {
	if id == uuid.Nil {
		return p, ErrZeroPolicyID
	}

	// checking cache first
	m.RLock()
	p, ok := m.policies[id]
	m.RUnlock()

	// return if found in cache
	if ok {
		return p, nil
	}

	// attempting to obtain policy from the store
	p, err = m.store.FetchPolicyByID(ctx, id)
	if err != nil {
		return p, errors.Wrapf(err, "failed to fetch access policy: %d", id)
	}

	// fetching roster
	r, err := m.store.FetchRosterByPolicyID(ctx, p.ID)
	if err != nil {
		return p, errors.Wrapf(err, "failed to fetch rights roster: %d", p.ID)
	}

	return p, m.putPolicy(p, r)
}

// PolicyByKey returns an access policy by its key
func (m *Manager) PolicyByKey(ctx context.Context, name Key) (p Policy, err error) {
	m.RLock()
	p, ok := m.policies[m.keyMap[name]]
	m.RUnlock()

	// return if found in cache
	if ok {
		return p, nil
	}

	// attempting to obtain policy from the store
	p, err = m.store.FetchPolicyByKey(ctx, name)
	if err != nil {
		return p, err
	}

	// fetching roster
	r, err := m.store.FetchRosterByPolicyID(ctx, p.ID)
	if err != nil {
		return p, errors.Wrapf(err, "failed to fetch rights roster: %d", p.ID)
	}

	// adding policy to registry
	if err = m.putPolicy(p, r); err != nil {
		return p, err
	}

	return p, nil
}

// PolicyByObject returns an access policy by its kind and id
func (m *Manager) PolicyByObject(ctx context.Context, obj Object) (p Policy, err error) {
	// attempting to obtain policy from the store
	p, err = m.store.FetchPolicyByObject(ctx, obj)
	if err != nil {
		return p, err
	}

	// fetching roster
	r, err := m.store.FetchRosterByPolicyID(ctx, p.ID)
	if err != nil {
		return p, errors.Wrapf(err, "failed to fetch rights roster: %d", p.ID)
	}

	// adding policy and roster to the registry
	if err = m.putPolicy(p, r); err != nil {
		return p, err
	}

	return p, nil
}

// DeletePolicy returns an access policy by its ObjectID
func (m *Manager) DeletePolicy(ctx context.Context, p Policy) (err error) {
	if err = p.Validate(); err != nil {
		return errors.Wrap(err, "failed to delete access policy")
	}

	// deleting policy from the store
	// NOTE: also deletes roster
	if err = m.store.DeletePolicy(ctx, p); err != nil {
		return err
	}

	// adding policy to registry
	if err = m.removePolicy(p.ID); err != nil {
		if err == ErrPolicyNotFound {
			return nil
		}

		return err
	}

	return nil
}

// RosterByPolicy returns the rights roster by its access policy
func (m *Manager) RosterByPolicyID(ctx context.Context, id uuid.UUID) (r *Roster, err error) {
	if id == uuid.Nil {
		return nil, ErrZeroPolicyID
	}

	// checking internal cache
	m.rosterLock.RLock()
	r, ok := m.roster[id]
	m.rosterLock.RUnlock()

	// returning if cache was found
	if ok {
		return r, nil
	}

	// attempting to obtain policy from the store
	p, err := m.store.FetchPolicyByID(ctx, id)
	if err != nil {
		return r, errors.Wrapf(err, "failed to fetch policy roster: policy_id=%d", id)
	}

	// fetching rights roster
	r, err = m.store.FetchRosterByPolicyID(ctx, p.ID)
	if err != nil {
		// if no roster records are found, then initializing new roster object
		if err == ErrEmptyRoster {
			m.rosterLock.Lock()
			m.roster[p.ID] = NewRoster(0)
			m.rosterLock.Unlock()
		}

		return r, errors.Wrapf(err, "failed to fetch rights roster: policy_id=%d", p.ID)
	}

	// adding policy to registry
	if err = m.putPolicy(p, r); err != nil {
		return r, err
	}

	return r, nil
}

// hasRights checks whether a given actor entity has the inquired rights
func (m *Manager) HasRights(ctx context.Context, pid uuid.UUID, actor Actor, rights Right) bool {
	if pid == uuid.Nil {
		return false
	}

	switch actor.Kind {
	case SKEveryone:
		return m.HasPublicRights(ctx, pid, rights)
	case SKUser:
		return m.UserHasAccess(ctx, pid, actor.ID, rights)
	case SKRoleGroup:
		return m.HasRoleRights(ctx, pid, actor.ID, rights)
	case SKGroup:
		return m.HasGroupRights(ctx, pid, actor.ID, rights)
	}

	return false
}

// GrantAccess grants access rights on a given policy, by grantor to grantee
// NOTE: can be called multiple times before policy changes are persisted
// NOTE: rights rosters changes are not persisted unless explicitly saved
// NOTE: changes made with this function will be cancelled and backup restored
// if there will be any errors when saving this policy
func (m *Manager) GrantAccess(ctx context.Context, pid uuid.UUID, grantor, grantee Actor, access Right) (err error) {
	p, err := m.PolicyByID(ctx, pid)
	if err != nil {
		return errors.Wrap(err, "failed to obtain access policy")
	}

	r, err := m.RosterByPolicyID(ctx, p.ID)
	if err != nil {
		return errors.Wrap(err, "failed to obtain rights roster")
	}

	// setting rights depending on the type of a subject
	switch grantee.Kind {
	case SKEveryone:
		err = m.GrantPublicAccess(ctx, pid, grantor, access)
	case SKUser:
		err = m.GrantUserAccess(ctx, pid, grantor, grantee.ID, access)
	case SKRoleGroup:
		err = m.GrantRoleAccess(ctx, pid, grantor, grantee.ID, access)
	case SKGroup:
		err = m.GrantGroupAccess(ctx, pid, grantor, grantee.ID, access)
	}

	// clearing changes in case of an error
	if err != nil {
		r.clearChanges()
	}

	return err
}

// RevokeAccess takes away current rights of a kind on this policy,
// from an assignee, as an assignor
// NOTE: this function only removes exclusive rights of this assignee,
// but the assignee still retains its public level rights to whatver
// that this policy protects
// NOTE: if you wish to completely deny somebody an access through
// this policy, then set exclusive rights explicitly (i.e. APNoAccess, 0)
func (m *Manager) RevokeAccess(ctx context.Context, pid uuid.UUID, grantor, grantee Actor) (err error) {
	// safety fuse
	restoreBackup := true

	p, err := m.PolicyByID(ctx, pid)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain access policy: policy_id=%d", pid)
	}

	r, err := m.RosterByPolicyID(ctx, pid)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain rights roster: policy_id=%d", p.ID)
	}

	// will restore backup unless successfully cancelled
	defer func() {
		if restoreBackup {
			r.restoreBackup()
		}
	}()

	if grantor.ID == uuid.Nil {
		return ErrZeroAssignorID
	}

	// the grantor must have a right to manage access rights (APManageAccess) and have all the
	// rights himself that he's attempting to assign to others
	// TODO: consider weighting the rights of who strips whose rights
	if !m.HasRights(ctx, pid, grantor, APManageAccess) {
		return ErrAccessDenied
	}

	// deleting assigneeID from the rosters (depending on its type)
	switch grantee.Kind {
	case SKEveryone:
		r.change(RSet, NewActor(SKEveryone, uuid.Nil), APNoAccess)
	case SKUser, SKRoleGroup, SKGroup:
		r.change(RUnset, grantee, APNoAccess)
	}

	// all is good, cancelling restoration
	restoreBackup = false

	return nil
}

// SetParentID setting a new parent policy
func (m *Manager) SetParent(ctx context.Context, policyID, parentID uuid.UUID) (err error) {
	p, err := m.PolicyByID(ctx, policyID)
	if err != nil {
		return errors.Wrapf(err, "policy_id=%d, new_parent_id=%d", policyID, parentID)
	}

	// disabling inheritance and extension to avoid unexpected behaviour
	if parentID == uuid.Nil {
		// since parent ID is zero, thus disabling inheritance and extension
		p.Flags &^= FInherit | FExtend
		p.ParentID = uuid.Nil
	} else {
		// checking parent policy existence
		if _, err = m.PolicyByID(ctx, parentID); err != nil {
			return errors.Wrapf(err, "failed to obtain new parent policy: policy_id=%d, new_parent_id=%d", policyID, parentID)
		}

		p.ParentID = parentID
	}

	// obtaining roster
	r, err := m.RosterByPolicyID(ctx, p.ID)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain policy roster before setting new parent: policy_id=%d, new_parent_id=%d", policyID, parentID)
	}

	// persisting changes
	if err = m.Update(ctx, p); err != nil {
		return errors.Wrapf(err, "failed to update policy after setting new parent: policy_id=%d, new_parent_id=%d", policyID, parentID)
	}

	// updating cached policy
	m.Lock()
	m.policies[p.ID] = p
	m.Unlock()

	// clearing calculated cache in a roster
	r.cacheLock.Lock()
	r.calculatedCache = make(map[Actor]Right, 0)
	r.cacheLock.Unlock()

	return nil
}

// Access returns a summarized access bitmask for a given actor
func (m *Manager) Access(ctx context.Context, policyID, userID uuid.UUID) (access Right) {
	if userID == uuid.Nil {
		return APNoAccess
	}

	// obtaining policy
	ap, err := m.PolicyByID(ctx, policyID)
	if err != nil {
		log.Printf("Access(policy_id=%d, user_id=%d): %s\n", policyID, userID, err)
		return APNoAccess
	}

	// if this user is the owner, then returning maximum possible value for Right type
	if ap.IsOwner(userID) {
		return APFullAccess
	}

	// calculating parents access if parent ID is set
	if ap.ParentID != uuid.Nil {
		// obtaining parent object
		parent, err := m.PolicyByID(ctx, ap.ParentID)
		if err != nil {
			panic(errors.Wrap(err, "access policy has parent id set, but failed to obtain parent policy object"))
		}

		// if this policy is flagged as inherited, then
		// calling Access until we reach the actual policy
		if ap.IsInherited() {
			access = m.Access(ctx, ap.ParentID, userID)
		} else {
			// if extend is true and parent exists, then using parent's access as a base value
			if parent.ID != uuid.Nil && ap.IsExtended() {
				// addressing the parent because it traces back until it finds
				// the first uninherited, actual policy
				access = m.SummarizedUserAccess(ctx, parent.ID, userID)
			}
		}

		access |= m.SummarizedUserAccess(ctx, ap.ID, userID)
	}

	return access
}

// GrantPublicAccess setting base access rights for everyone
func (m *Manager) GrantPublicAccess(ctx context.Context, pid uuid.UUID, grantor Actor, rights Right) error {
	// safety fuse
	restoreBackup := true

	r, err := m.RosterByPolicyID(ctx, pid)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain rights roster: policy_id=%d", pid)
	}

	// will restore backup unless successfully cancelled
	defer func() {
		if restoreBackup {
			r.restoreBackup()
		}
	}()

	if grantor.ID == uuid.Nil {
		return ErrZeroAssignorID
	}

	// checking whether the assignorID has at least the assigned rights
	if !m.HasRights(ctx, pid, grantor, APManageAccess|rights) {
		return ErrExcessOfRights
	}

	// deferred instruction for rosterChange
	r.change(RSet, NewActor(SKEveryone, uuid.Nil), rights)

	// all is good, cancelling restoration
	restoreBackup = false

	return nil
}

// GrantRoleAccess grants access rights to the role
func (m *Manager) GrantRoleAccess(ctx context.Context, pid uuid.UUID, grantor Actor, roleID uuid.UUID, rights Right) error {
	// safety fuse
	restoreBackup := true

	r, err := m.RosterByPolicyID(ctx, pid)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain rights roster: policy_id=%d", pid)
	}

	// will restore backup unless successfully cancelled
	defer func() {
		if restoreBackup {
			r.restoreBackup()
		}
	}()

	if grantor.ID == uuid.Nil {
		return ErrZeroAssignorID
	}

	if roleID == uuid.Nil {
		return ErrZeroRoleID
	}

	g, err := m.groups.GroupByID(ctx, roleID)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain role group: %d", roleID)
	}

	// making sure it is a role group
	if g.Flags != group.FRole {
		return errors.Wrapf(
			err,
			"GrantRoleAccess(policy_id=%d, assignor_id=%d, role_id=%d, rights=%d): expecting %s, got %s",
			pid, grantor.ID, roleID, rights, group.FRole, g.Flags,
		)
	}

	// checking whether assignor has the right to manage,
	// and has at least the assigned rights itself
	if !m.HasRights(ctx, pid, grantor, APManageAccess|rights) {
		return ErrExcessOfRights
	}

	// deferred instruction for rosterChange
	r.change(RSet, NewActor(SKRoleGroup, roleID), rights)

	// all is good, cancelling restoration
	restoreBackup = false

	return nil
}

// GrantGroupAccess grants access rights to a specific group
func (m *Manager) GrantGroupAccess(ctx context.Context, pid uuid.UUID, grantor Actor, groupID uuid.UUID, rights Right) (err error) {
	// safety fuse
	restoreBackup := true

	r, err := m.RosterByPolicyID(ctx, pid)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain rights roster: policy_id=%d", pid)
	}

	// will restore backup unless successfully cancelled
	defer func() {
		if restoreBackup {
			r.restoreBackup()
		}
	}()

	if grantor.ID == uuid.Nil {
		return ErrZeroAssignorID
	}

	if groupID == uuid.Nil {
		return ErrZeroGroupID
	}

	g, err := m.groups.GroupByID(ctx, groupID)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain group: %d", groupID)
	}

	// making sure it is a standard group
	if g.Flags != group.FGroup {
		return errors.Wrapf(
			err,
			"GrantGroupAccess(policy_id=%d, assignor_id=%d, role_id=%d, rights=%d): expecting %s, got %s",
			pid, grantor.ID, groupID, rights, group.FGroup, g.Flags,
		)
	}

	// checking whether assignor has the right to manage,
	// and has at least the assigned rights itself
	if !m.HasRights(ctx, pid, grantor, APManageAccess|rights) {
		return ErrExcessOfRights
	}

	// deferred instruction for rosterChange
	r.change(RSet, NewActor(SKGroup, groupID), rights)

	// all is good, cancelling restoration
	restoreBackup = false

	return nil
}

// GrantUserAccess grants access rights to a specific user actor
// TODO: consider whether it's right to turn off inheritance (if enabled) when setting/changing anything on each access policy instance
func (m *Manager) GrantUserAccess(ctx context.Context, pid uuid.UUID, grantor Actor, userID uuid.UUID, rights Right) (err error) {
	// safety fuse
	restoreBackup := true

	r, err := m.RosterByPolicyID(ctx, pid)
	if err != nil {
		return errors.Wrapf(err, "failed to obtain rights roster: policy_id=%d", pid)
	}

	// will restore backup unless successfully cancelled
	defer func() {
		if restoreBackup {
			r.restoreBackup()
		}
	}()

	if grantor.ID == uuid.Nil {
		return ErrZeroAssignorID
	}

	if userID == uuid.Nil {
		return ErrZeroAssigneeID
	}

	// checking whether assignor has the right to manage,
	// and has at least the assigned rights itself
	if !m.HasRights(ctx, pid, grantor, APManageAccess|rights) {
		return ErrExcessOfRights
	}

	// deferred instruction for change
	r.change(RSet, NewActor(SKUser, userID), rights)

	// all is good, cancelling restoration
	restoreBackup = false

	return nil
}

// UserHasAccess checks whether the user has specific rights
// NOTE: returns true only if the user has every of specified rights permitted
// TODO: maybe add some sort of a calculated cache with a short lifespan, like 10ms or something
func (m *Manager) UserHasAccess(ctx context.Context, pid uuid.UUID, userID uuid.UUID, rights Right) bool {
	if userID == uuid.Nil {
		return false
	}

	p, err := m.PolicyByID(ctx, pid)
	if err != nil {
		return false
	}

	// allow if this user is an owner
	if p.IsOwner(userID) {
		return true
	}

	// calculated rights
	var cr Right

	// calculating parent-related rights if possible
	if p.ParentID != uuid.Nil {
		// if the current policy is flagged as inherited, then
		// using its parent as the primary source of rights
		if p.IsInherited() {
			return m.UserHasAccess(ctx, p.ParentID, userID, rights)
		}

		if p.IsExtended() {
			cr = m.SummarizedUserAccess(ctx, p.ParentID, userID)
		}
	}

	// merging with the actual policy's rights rosters rights
	// TODO: consider overriding the extended rights with own
	cr |= m.SummarizedUserAccess(ctx, pid, userID)

	return (cr & rights) == rights
}

// HasPublicRights checks whether a given policy has specific public rights
// NOTE: despite it's narrow purpose, it may still be useful to check public rights alone
func (m *Manager) HasPublicRights(ctx context.Context, policyID uuid.UUID, rights Right) bool {
	r, err := m.RosterByPolicyID(ctx, policyID)
	if err != nil {
		return false
	}

	return (r.Everyone & rights) == rights
}

// HasGroupRights checks whether a group has the rights
func (m *Manager) HasGroupRights(ctx context.Context, policyID, groupID uuid.UUID, rights Right) bool {
	return (m.GroupAccess(ctx, policyID, groupID) & rights) == rights
}

// HasGroupRights checks whether a role has the rights
func (m *Manager) HasRoleRights(ctx context.Context, policyID, groupID uuid.UUID, rights Right) bool {
	return (m.GroupAccess(ctx, policyID, groupID) & rights) == rights
}

// SummarizedUserAccess summarizing the resulting access rights of a given user
func (m *Manager) SummarizedUserAccess(ctx context.Context, policyID, userID uuid.UUID) (access Right) {
	p, err := m.PolicyByID(ctx, policyID)
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
		for _, g := range m.groups.GroupsByAssetID(ctx, group.FRole|group.FGroup, group.NewAsset(group.AKUser, userID)) {
			access |= m.GroupAccess(ctx, policyID, g.ID)
		}
	}

	//-!!!-[ WARNING ]-----------------------------------------------------------
	// !!! USING USER'S OWNERSHIP TO OVERRIDE ITS ACCESS
	// !!! THIS MEANS THAT OWNERS OF THE PARENT POLICIES WILL HAVE
	// !!! FULL ACCESS TO ITS CHILDREN
	// !!! TODO: CONSIDER THIS VERY STRONGLY
	//-!!!-----------------------------------------------------------------------
	if p.IsOwner(userID) {
		access = APFullAccess
	}

	// user-specific rights
	return access | r.lookup(NewActor(SKUser, userID))
}

// GroupAccess returns the rights of a given group if set explicitly,
// otherwise returns the rights of the first ancestor group that has
// any rights record explicitly set
func (m *Manager) GroupAccess(ctx context.Context, pid, groupID uuid.UUID) (access Right) {
	if pid == uuid.Nil || groupID == uuid.Nil {
		return APNoAccess
	}

	// group manager is mandatory at this point
	if m.groups == nil {
		log.Printf("GroupAccess(policy_id=%d, group_id=%d): group manager is nil\n", pid, groupID)
		return APNoAccess
	}

	// obtaining roster
	r, err := m.RosterByPolicyID(ctx, pid)
	if err != nil {
		log.Printf("GroupAccess(policy_id=%d, group_id=%d): failed to obtain rights roster\n", pid, groupID)
		return APNoAccess
	}

	// obtaining target group
	g, err := m.groups.GroupByID(ctx, groupID)
	if err != nil {
		return APNoAccess
	}

	switch true {
	case g.IsGroup():
		access = r.lookup(NewActor(SKGroup, g.ID))
	case g.IsRole():
		access = r.lookup(NewActor(SKRoleGroup, g.ID))
	}

	// returning if any positive access right is found
	if access != APNoAccess {
		return access
	}

	// otherwise, looking for the first set access by tracing back
	// through its parents
	if g.ParentID != uuid.Nil {
		return m.GroupAccess(ctx, pid, g.ParentID)
	}

	return APNoAccess
}
