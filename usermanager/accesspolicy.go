package usermanager

import (
	"github.com/oklog/ulid"
)

// Actor represents anything that can be an owner, assignor ar assignee
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
	Everyone AccessRight            `json:"everyone"`
	Role     map[string]AccessRight `json:"role"`
	Group    map[string]AccessRight `json:"group"`
	User     map[string]AccessRight `json:"user"`
}

// Summarize summarizing the resulting access right flags
func (rr *RightsRoster) Summarize(u *User) AccessRight {
	r := rr.Everyone

	// calculating role rights
	for _, ur := range u.Roles() {
		if _, ok := rr.Role[ur.Key]; ok {
			r |= rr.Role[ur.Key]
		}
	}

	// same with groups
	for _, ug := range u.Groups() {
		if _, ok := rr.Group[ug.Key]; ok {
			r |= rr.Group[ug.Key]
		}
	}

	// user-specific rights
	if _, ok := rr.User[u.Username]; ok {
		r |= rr.User[u.Username]
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
// TODO: consider adding a mutex
// TODO: consider making policy to be completely decoupled and agnostic about the subject types
type AccessPolicy struct {
	Owner        *User         `json:"owner"`
	Parent       *AccessPolicy `json:"-"`
	IsExtended   bool          `json:"is_extend"`
	IsInherited  bool          `json:"is_inherited"`
	RightsRoster *RightsRoster `json:"rights_roster"`
}

// NewAccessPolicy create a new AccessPolicy object
// NOTE: the extension of parent's rights has higher precedence over using the inherited rights
// because this allows to create independent policies in the middle of a chain and still
// benefit from using parent's rights as default with it's own corrections/exclusions
func NewAccessPolicy(owner *User, parent *AccessPolicy, isInherited bool, isExtended bool) *AccessPolicy {
	ap := &AccessPolicy{
		Owner:       owner,
		Parent:      parent,
		IsInherited: isInherited,
		IsExtended:  isExtended,
		RightsRoster: &RightsRoster{
			Everyone: APNoAccess,
			Group:    make(map[string]AccessRight),
			Role:     make(map[string]AccessRight),
			User:     make(map[string]AccessRight),
		},
	}

	// just using a pointer to parent rights
	if ap.IsInherited && parent != nil {
		ap.RightsRoster = parent.RightsRoster
	}

	return ap
}

// SetParent setting a new parent policy
// WARNING: parent is always intended to be of an object's parent
// i.e. policy of the secret's containing category
// NOTE: if the parent is set to nil, then forcing IsInherited flag to false
func (ap *AccessPolicy) SetParent(parent *AccessPolicy) error {
	ap.Parent = parent

	// disabling inheritance to avoid unexpected behaviour
	if parent == nil {
		ap.IsInherited = false
	}

	return nil
}

// UserAccess returns the user access bitmask
func (ap *AccessPolicy) UserAccess(u *User) AccessRight {
	if u == nil {
		return AccessRight(0)
	}

	// if this u is the owner, then returning maximum possible value for AccessRight type
	if ap.IsOwner(u) {
		return ^AccessRight(0)
	}

	var rights AccessRight
	// if IsInherited is true, then calling UserAccess until we reach the actual policy
	if ap.Parent != nil && ap.IsInherited {
		rights = ap.Parent.UserAccess(u)
	} else {
		// if extend is true and parent exists, then using parent's rights as a base value
		if ap.Parent != nil && ap.IsExtended {
			rights = ap.Parent.RightsRoster.Summarize(u)
		}

		rights |= ap.RightsRoster.Summarize(u)
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

	ap.RightsRoster.Everyone = rights

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

	// checking whether the assignor has at least the assigned rights
	if !ap.HasRights(assignor, rights) {
		return ErrExcessOfRights
	}

	ap.RightsRoster.Role[role.Key] = rights

	return nil
}

// SetGroupRights setting rights for specific user
func (ap *AccessPolicy) SetGroupRights(assignor *User, group *Group, rights AccessRight) error {
	if assignor == nil {
		return ErrNilAssignor
	}

	if group == nil {
		return ErrGroupIsNil
	}

	// checking whether the assignor has at least the assigned rights
	if !ap.HasRights(assignor, rights) {
		return ErrExcessOfRights
	}

	ap.RightsRoster.Group[group.Key] = rights

	return nil
}

// SetUserRights setting rights for specific user
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

	ap.RightsRoster.User[assignee.Username] = rights

	return nil
}

// IsOwner checks whether a given user is the owner of this policy
func (ap *AccessPolicy) IsOwner(u *User) bool {
	// owner of the policy (meaning: the main entity) has full rights on it
	// TODO: username comparison is a very bad idea, e.g. username can be changed
	if ap.Owner != nil && (ap.Owner.Username == u.Username) {
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

	// copying for convenience
	rr := ap.RightsRoster

	if rr == nil {
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
			cr = ap.Parent.RightsRoster.Summarize(user)
		}
	}

	// merging with the actual policy's rights roster rights
	cr |= ap.RightsRoster.Summarize(user)

	return (cr & rights) == rights
}
