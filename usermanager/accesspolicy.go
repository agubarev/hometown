package usermanager

import (
	"sync"

	"github.com/oklog/ulid"
)

// Actor represents anything that can be an owner, assignor ar assignee
// TODO: develop the idea
type Actor interface {
	ULID() ulid.ULID
	Roles() []*Group
	Groups() []*Group
}

// AccessRight is a single permission set
type AccessRight uint16

// declaring discrete rights for all cases
const (
	APNoAccess = AccessRight(0)
	APView     = AccessRight(1 << (iota - 1))
	APCreate
	APChange
	APDelete
	APCopy
	APMove
	APManageRights
	APFullAccess = ^AccessRight(0)
)

// RightsRoster denotes who has what rights
type RightsRoster struct {
	Everyone AccessRight               `json:"everyone"`
	Role     map[ulid.ULID]AccessRight `json:"role"`
	Group    map[ulid.ULID]AccessRight `json:"group"`
	User     map[ulid.ULID]AccessRight `json:"user"`
}

// NewRightsRoster is a shorthand initializer function
func NewRightsRoster() *RightsRoster {
	return &RightsRoster{
		Everyone: APNoAccess,
		Group:    make(map[ulid.ULID]AccessRight),
		Role:     make(map[ulid.ULID]AccessRight),
		User:     make(map[ulid.ULID]AccessRight),
	}
}

// Summarize summarizing the resulting access right flags
func (rr *RightsRoster) Summarize(u *User) AccessRight {
	r := rr.Everyone

	// calculating role rights
	for _, ur := range u.Groups(GKRole) {
		if _, ok := rr.Role[ur.ID]; ok {
			r |= rr.Role[ur.ID]
		}
	}

	// same with groups
	for _, ug := range u.Groups(GKGroup) {
		if _, ok := rr.Group[ug.ID]; ok {
			r |= rr.Group[ug.ID]
		}
	}

	// user-specific rights
	if _, ok := rr.User[u.ID]; ok {
		r |= rr.User[u.ID]
	}

	return r
}

// AccessPolicy is a generalized ruleset for an object
// if IsInherited is true, then the policy's own roster will point to it's parent
// and everything else will be ignored as long as it's true
// NOTE: policy may be shared by multiple entities
// NOTE: policy ownership basically is the ownership of it's main entity and only affects the very object alone
// NOTE: owner is the original creator of an entity and has full rights for it
// TODO: calculate extended rights instantly. rights must be recalculated through all the tree after each change
// TODO: add caching mechanism to skip rights summarization
// TODO: disable inheritance if anything is changed about the current policy and create its own rights roster and enable extension by default
type AccessPolicy struct {
	ID           ulid.ULID     `json:"id"`
	Namespace    string        `json:"namespace"`
	OwnerID      ulid.ULID     `json:"owner_id"`
	Owner        *User         `json:"-"`
	ParentID     ulid.ULID     `json:"parent_id"`
	Parent       *AccessPolicy `json:"-"`
	IsExtended   bool          `json:"is_extended"`
	IsInherited  bool          `json:"is_inherited"`
	RightsRoster *RightsRoster `json:"-"`
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

// Seal the policy to prevent further changes
// the idea is to make it modifiable only by its owner
// TODO: do I really want this?
func (ap *AccessPolicy) Seal() error {
	panic("not implemented")

	return nil
}

// CreateSnapshot returns a snapshot copy of the access rights roster for this policy
func (ap *AccessPolicy) CreateSnapshot() (*RightsRoster, error) {
	if ap == nil {
		return nil, ErrNilAccessPolicy
	}

	// must be unforgiving and explicit, returning an error
	if ap.RightsRoster == nil {
		return nil, ErrNilRightsRoster
	}

	// initializing new roster
	ss := NewRightsRoster()

	// copying roster values
	ap.RLock()
	ss.Everyone = ap.RightsRoster.Everyone

	// copying group rights
	for gid, right := range ap.RightsRoster.Group {
		ss.Group[gid] = right
	}

	// copying role rights
	for rid, right := range ap.RightsRoster.Role {
		ss.Role[rid] = right
	}

	// copying user rights
	for uid, right := range ap.RightsRoster.User {
		ss.User[uid] = right
	}

	ap.RUnlock()

	return ss, nil
}

// SetParent setting a new parent policy
// WARNING: parent is always intended to be of an object's parent
// i.e. policy of the secret's containing category
// NOTE: if the parent is set to nil, then forcing IsInherited flag to false
func (ap *AccessPolicy) SetParent(parent *AccessPolicy) error {
	ap.Lock()
	ap.Parent = parent

	// disabling inheritance to avoid unexpected behaviour
	// TODO: think it through, is it really obvious to disable inheritance if parent is nil'ed?
	if parent == nil {
		ap.IsInherited = false
	}

	ap.Unlock()

	return nil
}

// UserAccess returns the user access bitmask
func (ap *AccessPolicy) UserAccess(u *User) AccessRight {
	if u == nil {
		return AccessRight(0)
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
// TODO: store changes if the store is not nil
// TODO: store changes if the store is not nil
// TODO: store changes if the store is not nil
// TODO: store changes if the store is not nil
// TODO: store changes if the store is not nil
// TODO: store changes if the store is not nil
// TODO: store changes if the store is not nil
// TODO: store changes if the store is not nil
// TODO: store changes if the store is not nil
// TODO: store changes if the store is not nil
// TODO: store changes if the store is not nil
// TODO: store changes if the store is not nil
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
