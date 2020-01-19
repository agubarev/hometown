package usermanager

import (
	"errors"
	"fmt"
	"sync"

	"github.com/oklog/ulid"
)

// Actor represents anything that can be an owner, assignor ar assignee
// TODO: develop the idea
type Actor interface {
	ID() int64
	UID() ulid.ULID
	Roles() []*Group
	Groups() []*Group
}

// AccessRight is a single permission set
type AccessRight uint64

type accessChange struct {
	// denotes an action that occurred: -1 deleted, 0 updated, 1 created
	action      int8
	subjectKind string
	subjectID   int64
	accessRight AccessRight
}

// declaring discrete rights for all cases
const (
	APNoAccess = AccessRight(0)
	APView     = AccessRight(1 << uint64(iota - 1))
	APCreate
	APChange
	APDelete
	APCopy
	APMove
	APManageRights
	APFullAccess = ^AccessRight(0)
)

// Human returns a human-readable conjunction of comma-separated
// access names for this given context namespace
func (r AccessRight) Human() string {
	s := "not implemented"

	return s
}

// RightsRoster denotes who has what rights
type RightsRoster struct {
	Everyone AccessRight           `json:"everyone"`
	Role     map[int64]AccessRight `json:"role"`
	Group    map[int64]AccessRight `json:"group"`
	User     map[int64]AccessRight `json:"user"`

	changes []accessChange
	sync.RWMutex
}

// NewRightsRoster is a shorthand initializer function
func NewRightsRoster() *RightsRoster {
	return &RightsRoster{
		Everyone: APNoAccess,
		Group:    make(map[int64]AccessRight),
		Role:     make(map[int64]AccessRight),
		User:     make(map[int64]AccessRight),
	}
}

// addChange adds a single change for further storing
func (rr *RightsRoster) addChange(action int8, subjectKind string, subjectID int64, rights AccessRight) {
	change := accessChange{
		action:      action,
		subjectKind: subjectKind,
		subjectID:   subjectID,
		accessRight: rights,
	}

	rr.Lock()

	if rr.changes == nil {
		rr.changes = []accessChange{change}
	} else {
		rr.changes = append(rr.changes, change)
	}

	rr.Unlock()
}

func (rr *RightsRoster) clearChanges() {
	rr.Lock()
	rr.changes = nil
	rr.Unlock()
}

// Summarize summarizing the resulting access right flags
func (rr *RightsRoster) Summarize(u *User) AccessRight {
	r := rr.Everyone

	// calculating standard and role group rights
	// NOTE: if some group doesn't have explicitly set rights, then
	// attempting to obtain the rights of a first ancestor group,
	// that has specific rights set
	for _, g := range u.Groups(GKRole | GKGroup) {
		r |= rr.GroupRights(g)
	}

	// user-specific rights
	if _, ok := rr.User[u.ID]; ok {
		r |= rr.User[u.ID]
	}

	return r
}

// GroupRights returns the rights of a given group if set explicitly,
// otherwise returns the rights of the first ancestor group that has
// any rights record explicitly set
func (rr *RightsRoster) GroupRights(g *Group) AccessRight {
	if g == nil {
		return APNoAccess
	}

	var rights AccessRight
	var ok bool

	rr.RLock()

	switch g.Kind {
	case GKGroup:
		rights, ok = rr.Group[g.ID]
	case GKRole:
		rights, ok = rr.Role[g.ID]
	}

	rr.RUnlock()

	if ok {
		return rights
	}

	// now looking for the first set rights by tracing back
	// through its parents
	if g.Parent() != nil {
		return rr.GroupRights(g.Parent())
	}

	return APNoAccess
}

// AccessPolicy is a generalized ruleset for an object
// if IsInherited is true, then the policy's own roster will point to it's parent
// and everything else will be ignored as long as it's true
// NOTE: policy may be shared by multiple entities
// NOTE: policy ownership basically is the ownership of it's main entity and only affects the very object alone
// NOTE: owner is the original creator of an entity and has full rights for it
// NOTE: an access policy can have only one object identifier set, either ID or a Key
// TODO calculate extended rights instantly. rights must be recalculated through all the tree after each change
// TODO add caching mechanism to skip rights summarization
// TODO disable inheritance if anything is changed about the current policy and create its own rights roster and enable extension by default
// TODO decide whether I want namespaces
type AccessPolicy struct {
	ID          int64  `db:"id" json:"id"`
	ParentID    int64  `db:"parent_id" json:"parent_id"`
	OwnerID     int64  `db:"owner_id" json:"owner_id"`
	Key         string `db:"key" json:"key"`
	ObjectKind  string `db:"object_kind" json:"object_kind"`
	ObjectID    int64  `db:"object_id" json:"object_id"`
	IsExtended  bool   `db:"is_extended" json:"is_extended"`
	IsInherited bool   `db:"is_inherited" json:"is_inherited"`

	Parent       *AccessPolicy `json:"-"`
	Owner        *User         `json:"-"`
	RightsRoster *RightsRoster `json:"-"`

	container *AccessPolicyContainer
	backup    *AccessPolicy

	sync.RWMutex
}

// NewAccessPolicy create a new AccessPolicy object
// NOTE: the extension of parent's rights has higher precedence over using the inherited rights
// because this allows to create independent policies in the middle of a chain and still
// benefit from using parent's rights as default with it's own corrections/exclusions
func NewAccessPolicy(owner *User, parent *AccessPolicy, isInherited bool, isExtended bool) *AccessPolicy {
	ap := &AccessPolicy{
		Owner:        owner,
		Parent:       parent,
		IsInherited:  isInherited,
		IsExtended:   isExtended,
		RightsRoster: NewRightsRoster(),
	}

	if owner != nil {
		ap.OwnerID = owner.ID
	}

	if parent != nil {
		ap.ParentID = parent.ID

		if ap.IsInherited {
			// just using a pointer to parent rights
			ap.RightsRoster = parent.RightsRoster
		}
	}

	return ap
}

// Validate validates access policy by performing basic self-check
func (ap *AccessPolicy) Validate() error {
	if ap == nil {
		return ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return ErrNilRightsRoster
	}

	if (ap.Parent != nil) != (ap.ParentID != 0) {
		return errors.New("parent is set but parent id is not, or vice versa")
	}

	// kind cannot be empty if either key or ID is set
	if ap.ObjectKind == "" && ap.ObjectID != 0 {
		return errors.New("empty kind with a non-zero id")
	}

	// if kind is set, then ID must also be set
	if ap.ObjectKind != "" && ap.ObjectID == 0 {
		return errors.New("non-empty kind with a zero id")
	}

	// policy must have some designators, a key or an ID of its kind
	if ap.Key == "" && ap.ObjectKind == "" && ap.ObjectID == 0 {
		return ErrAccessPolicyEmptyDesignators
	}

	// inherited means that this is not a standalone policy but simply points
	// to its parent policy (first standalone policy to be found)
	if ap.IsInherited && ap.IsExtended {
		return errors.New("policy cannot be both inherited and extended at the same time")
	}

	// parent must be set if this policy inherits or extends
	if ap.Parent == nil && (ap.IsInherited || ap.IsExtended) {
		return errors.New("policy cannot inherit or extend without a parent")
	}

	// making sure that parent is properly set, if set at all
	if ap.Parent != nil && ap.ParentID == 0 {
		return errors.New("parent is set but parentID is 0")
	}

	return nil
}

// Seal the policy to prevent further changes
// the idea is to make it modifiable only by its owner
// TODO: do I really want this?
func (ap *AccessPolicy) Seal() error {
	panic("not implemented")

	return nil
}

func (ap *AccessPolicy) objectIDIndex() string {
	return fmt.Sprintf("%s_%d", ap.ObjectKind, ap.ID)
}

// Clone clones a whole policy
func (ap *AccessPolicy) Clone() (*AccessPolicy, error) {
	rr, err := ap.CloneRightsRoster()
	if err != nil {
		return nil, err
	}

	clone := &AccessPolicy{
		ID:          ap.ID,
		ParentID:    ap.ParentID,
		OwnerID:     ap.OwnerID,
		Key:         ap.Key,
		ObjectKind:  ap.ObjectKind,
		ObjectID:    ap.ObjectID,
		IsInherited: ap.IsInherited,
		IsExtended:  ap.IsExtended,

		Parent:       ap.Parent,
		Owner:        ap.Owner,
		RightsRoster: rr,

		container: ap.container,
	}

	return clone, nil
}

// CloneRightsRoster returns a snapshot copy of the access rights roster for this policy
func (ap *AccessPolicy) CloneRightsRoster() (*RightsRoster, error) {
	if ap == nil {
		return nil, ErrNilAccessPolicy
	}

	// must be unforgiving and explicit, returning an error
	if ap.RightsRoster == nil {
		return nil, ErrNilRightsRoster
	}

	// initializing new roster
	rr := NewRightsRoster()

	// copying roster values
	ap.RLock()
	rr.Everyone = ap.RightsRoster.Everyone

	// copying group rights
	for gid, right := range ap.RightsRoster.Group {
		rr.Group[gid] = right
	}

	// copying role rights
	for rid, right := range ap.RightsRoster.Role {
		rr.Role[rid] = right
	}

	// copying user rights
	for uid, right := range ap.RightsRoster.User {
		rr.User[uid] = right
	}

	ap.RUnlock()

	return rr, nil
}

// SetParent setting a new parent policy
// NOTE: if the parent is set to nil, then forcing IsInherited flag to false
func (ap *AccessPolicy) SetParent(parent *AccessPolicy) error {
	ap.Lock()
	ap.Parent = parent

	// disabling inheritance to avoid unexpected behaviour
	// TODO: think it through, is it really obvious to disable inheritance if parent is nil'ed?
	if parent == nil {
		ap.IsInherited = false
		ap.ParentID = 0
	} else {
		ap.ParentID = parent.ID
	}

	ap.Unlock()

	return nil
}

// UserAccess returns the user access bitmask
func (ap *AccessPolicy) UserAccess(u *User) AccessRight {
	if u == nil {
		return APNoAccess
	}

	// if this u is the owner, then returning maximum possible value for AccessRight type
	if ap.IsOwner(u) {
		return APFullAccess
	}

	var rights AccessRight
	// if IsInherited is true, then calling UserAccess until we reach the actual policy
	if ap.Parent != nil && ap.IsInherited {
		rights = ap.Parent.UserAccess(u)
	} else {
		ap.RLock()
		// if extend is true and parent exists, then using parent's rights as a base value
		if ap.Parent != nil && ap.IsExtended {
			// addressing the parent because it traces back until it finds
			// the first uninherited, actual policy
			rights = ap.Parent.RightsRoster.Summarize(u)
		}

		rights |= ap.RightsRoster.Summarize(u)
		ap.RUnlock()
	}

	return rights
}

// SetPublicRights setting rights for everyone
func (ap *AccessPolicy) SetPublicRights(assignor *User, rights AccessRight) error {
	if assignor == nil {
		return ErrNilAssignor
	}

	// checking whether the assignor has at least the assigned rights
	if !ap.HasRights(assignor, rights) {
		return ErrExcessOfRights
	}

	ap.Lock()
	ap.RightsRoster.Everyone = rights
	ap.Unlock()

	return nil
}

// SetRoleRights setting rights for the role
func (ap *AccessPolicy) SetRoleRights(assignor *User, role *Group, rights AccessRight) error {
	if assignor == nil {
		return ErrNilAssignor
	}

	if role == nil {
		return ErrNilRole
	}

	// making sure it's group kind is Role
	if role.Kind != GKRole {
		return ErrInvalidGroupKind
	}

	// checking whether the assignor has at least the assigned rights
	if !ap.HasRights(assignor, rights) {
		return ErrExcessOfRights
	}

	ap.Lock()
	ap.RightsRoster.Role[role.ID] = rights
	ap.Unlock()

	return nil
}

// SetGroupRights setting rights for specific user
func (ap *AccessPolicy) SetGroupRights(assignor *User, group *Group, rights AccessRight) error {
	if assignor == nil {
		return ErrNilAssignor
	}

	if group == nil {
		return ErrNilGroup
	}

	// making sure it's group kind is Group
	if group.Kind != GKGroup {
		return ErrInvalidGroupKind
	}

	// checking whether the assignor has at least the assigned rights
	if !ap.HasRights(assignor, rights) {
		return ErrExcessOfRights
	}

	ap.Lock()
	ap.RightsRoster.Group[group.ID] = rights
	ap.Unlock()

	return nil
}

// SetUserRights setting rights for specific user
// TODO: consider whether it's right to turn off inheritance (if enabled) when setting/changing anything on each access policy instance
func (ap *AccessPolicy) SetUserRights(assignor *User, assignee *User, rights AccessRight) error {
	if assignor == nil {
		return ErrNilAssignor
	}

	if assignee == nil {
		return ErrNilAssignee
	}

	// the assignor must have a right to set rights (APManageRights) and have all the
	// rights himself that he's attempting to assign to others
	if !ap.HasRights(assignor, APManageRights|rights) {
		return ErrExcessOfRights
	}

	ap.Lock()
	ap.RightsRoster.User[assignee.ID] = rights
	ap.Unlock()

	return nil
}

// IsOwner checks whether a given user is the owner of this policy
func (ap *AccessPolicy) IsOwner(u *User) bool {
	// owner of the policy (meaning: the main entity) has full rights on it
	if ap.Owner != nil && (ap.Owner.ID == u.ID) {
		return true
	}

	return false
}

// HasRights checks whether the user has specific rights
// NOTE: returns true only if the user has every of specified rights permitted
// TODO: maybe add some sort of a calculated cache with a short livespan, like 100ms or something
func (ap *AccessPolicy) HasRights(user *User, rights AccessRight) bool {
	if user == nil {
		return false
	}

	// allow if this user is an owner
	if ap.IsOwner(user) {
		return true
	}

	if ap.RightsRoster == nil {
		return false
	}

	// calculated rights
	var cr AccessRight

	// calculating parent-related rights if possible
	if ap.Parent != nil {
		if ap.IsInherited {
			return ap.Parent.HasRights(user, rights)
		}

		if ap.IsExtended {
			ap.RLock()
			cr = ap.Parent.RightsRoster.Summarize(user)
			ap.RUnlock()
		}
	}

	// merging with the actual policy's rights roster rights
	ap.RLock()
	cr |= ap.RightsRoster.Summarize(user)
	ap.RUnlock()

	return (cr & rights) == rights
}

// HasGroupRights checks whether a group has the rights
func (ap *AccessPolicy) HasGroupRights(g *Group, rights AccessRight) bool {
	if g == nil {
		return false
	}

	if ap.RightsRoster == nil {
		return false
	}

	return (ap.RightsRoster.GroupRights(g) & rights) == rights
}

// UnsetRights takes away current rights on this policy,
// from an assignee, as an assignor
// NOTE: this function only removes exclusive rights of this assignee,
// but the assignee still retains its public level rights to this policy
// NOTE: if you wish to completely deny access to this policy, then
// better set exclusive rights explicitly (i.e. APNoAccess, 0)
func (ap *AccessPolicy) UnsetRights(assignor *User, assignee interface{}) error {
	if assignor == nil {
		return ErrNilAssignor
	}

	if assignee == nil {
		return ErrNilAssignee
	}

	// the assignor must have a right to set rights (APManageRights) and have all the
	// rights himself that he's attempting to assign to others
	if !ap.HasRights(assignor, APManageRights) {
		return ErrAccessDenied
	}

	ap.Lock()

	// deleting assignee from the roster (depending on its type)
	switch assignee.(type) {
	case *User:
		delete(ap.RightsRoster.User, assignee.(*User).ID)
	case *Group:
		switch group := assignee.(*Group); group.Kind {
		case GKRole:
			delete(ap.RightsRoster.Role, assignee.(*Group).ID)
		case GKGroup:
			delete(ap.RightsRoster.Group, assignee.(*Group).ID)
		}
	}

	ap.Unlock()

	return nil
}

// CreateBackup clones itself and stores a copy inside itself
// NOTE: does nothing if backup already exists
func (ap *AccessPolicy) CreateBackup() error {
	// checking whether there already is a copy backed up
	if ap.backup != nil {
		return ErrAccessPolicyBackupExists
	}

	// preserving a copy of this access policy by storing a backup inside itself
	backup, err := ap.Clone()
	if err != nil {
		return err
	}

	ap.backup = backup

	return nil
}

// RestoreBackup restores policy backup and clears changelist
func (ap *AccessPolicy) RestoreBackup() error {
	if ap.backup == nil {
		return ErrAccessPolicyBackupNotFound
	}

	if err := ap.backup.Validate(); err != nil {
		return fmt.Errorf("policy backup validation failed: %s", err)
	}

	if ap.ID != ap.backup.ID {
		return fmt.Errorf("policy ID and backup ID mismatch")
	}

	// restoring backup (restoring manually, field by field)
	ap.Owner = ap.backup.Owner
	ap.OwnerID = ap.backup.OwnerID
	ap.Parent = ap.backup.Parent
	ap.ParentID = ap.backup.ParentID
	ap.Key = ap.backup.Key
	ap.ObjectKind = ap.backup.ObjectKind
	ap.ObjectID = ap.backup.ObjectID
	ap.RightsRoster = ap.backup.RightsRoster
	ap.IsInherited = ap.backup.IsInherited
	ap.IsExtended = ap.backup.IsExtended
	ap.container = ap.backup.container

	// clearing backup
	ap.backup = nil

	// clearing rights roster changelist
	ap.RightsRoster.changes = nil

	return nil
}

// Backup returns backup policy if exists or nil
func (ap *AccessPolicy) Backup() *AccessPolicy {
	return ap.backup
}

// Save saves itself via container (if it belongs to any container)
func (ap *AccessPolicy) Save() error {
	if ap.container == nil {
		return ErrNilAccessPolicyContainer
	}

	return ap.container.Save(ap)
}
