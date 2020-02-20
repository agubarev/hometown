package accesspolicy

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/cespare/xxhash"
	"github.com/oklog/ulid"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

// Actor represents anything that can be an owner, assignor ar assignee
// TODO: develop this idea
type Actor interface {
	ID() int
	UID() ulid.ULID
	Roles() []*group.Group
	Groups() []*group.Group
}

// everyone, user, group, role_group, etc...
type SubjectKind = uint8

const (
	SKNone     SubjectKind = 0
	SKEveryone SubjectKind = 1 << (iota - SubjectKind(1))
	SKUser
	SKGroup
	SKRoleGroup
)

// access policy name
type TAPName = [255]byte
type TAPObjectType = [64]byte

// AccessRight is a single permission set
type AccessRight uint64

type accessChange struct {
	// denotes an action that occurred: -1 deleted, 0 updated, 1 created
	action      uint8
	subjectKind uint8
	subjectID   int
	accessRight AccessRight
}

// declaring discrete rights for all cases
const (
	APUnrecognizedFlag = "unrecognized access flag"

	APNoAccess = AccessRight(0)
	APView     = AccessRight(1 << uint64(iota-1))
	APCreate
	APChange
	APDelete
	APCopy
	APMove
	APManageRights
	APFullAccess = ^AccessRight(0)
)

func (r AccessRight) String() string {
	switch r {
	case APNoAccess:
		return "no access"
	case APView:
		return "view"
	case APCreate:
		return "create"
	case APChange:
		return "change"
	case APDelete:
		return "delete"
	case APCopy:
		return "copy"
	case APMove:
		return "move"
	case APManageRights:
		return "manage_rights"
	case APFullAccess:
		return "full access"
	default:
		return APUnrecognizedFlag
	}
}

// PropertyDictionary returns a map of property flag values to their respective names
func AccessPolicyDictionary() map[uint64]string {
	dict := make(map[uint64]string)

	for bit := AccessRight(1 << 63); bit > 0; bit >>= 1 {
		if s := bit.String(); s != APUnrecognizedFlag {
			dict[uint64(bit)] = bit.String()
		}
	}

	return dict
}

// AccessExplained returns a human-readable conjunction of comma-separated
// access names for this given context namespace
func (r AccessRight) Explain() string {
	s := make([]string, 0)

	for i := 0; i < 63; i++ {
		if bit := AccessRight(1 << i); r&bit != 0 {
			s = append(s, bit.String())
		}
	}

	if len(s) == 0 {
		return ""
	}

	return strings.Join(s, ",")
}

// RightsRoster denotes who has what rights
type RightsRoster struct {
	Everyone AccessRight         `json:"everyone"`
	Role     map[int]AccessRight `json:"role"`
	Group    map[int]AccessRight `json:"group"`
	User     map[int]AccessRight `json:"user"`

	changes []accessChange
	sync.RWMutex
}

// NewRightsRoster is a shorthand initializer function
func NewRightsRoster() *RightsRoster {
	return &RightsRoster{
		Everyone: APNoAccess,
		Group:    make(map[int]AccessRight),
		Role:     make(map[int]AccessRight),
		User:     make(map[int]AccessRight),
	}
}

// addChange adds a single change for further storing
func (rr *RightsRoster) addChange(action uint8, subjectKind uint8, subjectID int, rights AccessRight) {
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
func (rr *RightsRoster) Summarize(u *user.User) AccessRight {
	r := rr.Everyone

	// calculating standard and role group rights
	// NOTE: if some group doesn't have explicitly set rights, then
	// attempting to obtain the rights of a first ancestor group,
	// that has specific rights set
	for _, g := range u.Groups(group.GKRole | group.GKGroup) {
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
func (rr *RightsRoster) GroupRights(g *group.Group) AccessRight {
	if g == nil {
		return APNoAccess
	}

	var rights AccessRight
	var ok bool

	rr.RLock()

	switch g.Kind {
	case group.GKGroup:
		rights, ok = rr.Group[g.ID]
	case group.GKRole:
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
	ID          int           `db:"id" json:"id"`
	ParentID    int           `db:"parent_id" json:"parent_id"`
	OwnerID     int           `db:"owner_id" json:"owner_id"`
	Name        TAPName       `db:"name" json:"name"`
	ObjectType  TAPObjectType `db:"object_type" json:"object_type"`
	ObjectID    int           `db:"object_id" json:"object_id"`
	IsExtended  bool          `db:"is_extended" json:"is_extended"`
	IsInherited bool          `db:"is_inherited" json:"is_inherited"`
	Checksum    uint64        `db:"checksum" json:"checksum"`

	Parent       *AccessPolicy `json:"-"`
	Owner        *user.User    `json:"-"`
	RightsRoster *RightsRoster `json:"-"`

	keyHash   uint64
	container *Manager
	backup    *AccessPolicy

	sync.RWMutex
}

// NewAccessPolicy create a new AccessPolicy object
// NOTE: the extension of parent's rights has higher precedence over using the inherited rights
// because this allows to create independent policies in the middle of a chain and still
// benefit from using parent's rights as default with it's own corrections/exclusions
func NewAccessPolicy(owner *user.User, parent *AccessPolicy, isInherited bool, isExtended bool) *AccessPolicy {
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

func (ap *AccessPolicy) calculateChecksum() uint64 {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		int64(ap.ParentID),
		int64(ap.OwnerID),
		ap.Name[:],
		ap.ObjectType[:],
		ap.ObjectID,
		ap.IsExtended,
		ap.IsInherited,
	}

	for _, field := range fields {
		if err := binary.Write(buf, binary.LittleEndian, field); err != nil {
			panic(errors.Wrapf(err, "failed to write binary data [%v] to calculate checksum", field))
		}
	}

	// assigning a checksum calculated from a definite list of struct values
	ap.Checksum = xxhash.Sum64(buf.Bytes())

	return ap.Checksum
}

// TODO: use cryptographic hash function to exclude the chance of collision
func (ap *AccessPolicy) hashKey() {
	// panic if ID is zero or a name is empty
	if ap.ID == 0 {
		panic(core.ErrInsufficientDataToHashKey)
	}

	// initializing a key buffer with and assuming the minimum key length
	key := bytes.NewBuffer(make([]byte, 0, 20))

	// composing a key value
	key.Write([]byte("AccessPolicy"))

	if err := binary.Write(key, binary.LittleEndian, int64(ap.ID)); err != nil {
		panic(errors.Wrapf(err, "failed to hash access policy: %d", ap.ID))
	}

	// updating calculated key
	ap.keyHash = xxhash.Sum64(key.Bytes())
}

// Key returns a uint64 key hash to be used as a map/cache key
func (ap AccessPolicy) Key(rehash bool) uint64 {
	// returning a cached key if it's set
	if ap.keyHash == 0 || rehash {
		ap.hashKey()
	}

	return ap.keyHash
}

// ApplyChangelog applies changes described by a diff.Diff()'s changelog
// NOTE: doing a manual update to avoid using reflection
func (ap *AccessPolicy) ApplyChangelog(changelog diff.Changelog) (err error) {
	// it's ok if there's nothing apply
	if len(changelog) == 0 {
		return nil
	}

	for _, change := range changelog {
		switch change.Path[0] {
		case "ParentID":
			ap.ParentID = change.To.(int)
		case "OwnerID":
			ap.OwnerID = change.To.(int)
		case "Name":
			ap.Name = change.To.(TAPName)
		case "ObjectType":
			ap.ObjectType = change.To.(TAPObjectType)
		case "ObjectID":
			ap.ObjectID = change.To.(int)
		case "IsExtended":
			ap.IsExtended = change.To.(bool)
		case "IsInherited":
			ap.IsInherited = change.To.(bool)
		case "Checksum":
			ap.Checksum = change.To.(uint64)
		}
	}

	return nil
}

// Validate validates access policy by performing basic self-check
func (ap *AccessPolicy) Validate() error {
	if ap == nil {
		return core.ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return core.ErrNilRightsRoster
	}

	if (ap.Parent != nil) != (ap.ParentID != 0) {
		return errors.New("parent is set but parent id is not, or vice versa")
	}

	// kind cannot be empty if either key or ID is set
	if len(ap.ObjectType[:]) == 0 && ap.ObjectID != 0 {
		return errors.New("empty kind with a non-zero id")
	}

	// if kind is set, then ID must also be set
	if len(ap.ObjectType[:]) != 0 && ap.ObjectID == 0 {
		return errors.New("non-empty kind with a zero id")
	}

	// policy must have some designators, a key or an ID of its kind
	if len(ap.Name[:]) == 0 && len(ap.ObjectType[:]) == 0 && ap.ObjectID == 0 {
		return core.ErrAccessPolicyEmptyDesignators
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

func (ap *AccessPolicy) StringID() string {
	return fmt.Sprintf("%s_%d", ap.ObjectType, ap.ID)
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
		ObjectType:  ap.ObjectType,
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
		return nil, core.ErrNilAccessPolicy
	}

	// must be unforgiving and explicit, returning an error
	if ap.RightsRoster == nil {
		return nil, core.ErrNilRightsRoster
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
func (ap *AccessPolicy) UserAccess(u *user.User) AccessRight {
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
func (ap *AccessPolicy) SetPublicRights(assignor *user.User, rights AccessRight) error {
	if assignor == nil {
		return core.ErrNilAssignor
	}

	// checking whether the assignor has at least the assigned rights
	if !ap.HasRights(assignor, rights) {
		return core.ErrExcessOfRights
	}

	ap.Lock()
	ap.RightsRoster.Everyone = rights
	ap.Unlock()

	return nil
}

// SetRoleRights setting rights for the role
func (ap *AccessPolicy) SetRoleRights(assignor *user.User, role *group.Group, rights AccessRight) error {
	if assignor == nil {
		return core.ErrNilAssignor
	}

	if role == nil {
		return core.ErrNilRole
	}

	// making sure it's group kind is Role
	if role.Kind != group.GKRole {
		return core.ErrInvalidGroupKind
	}

	// checking whether the assignor has at least the assigned rights
	if !ap.HasRights(assignor, rights) {
		return core.ErrExcessOfRights
	}

	ap.Lock()
	ap.RightsRoster.Role[role.ID] = rights
	ap.Unlock()

	return nil
}

// SetGroupRights setting rights for specific user
func (ap *AccessPolicy) SetGroupRights(assignor *user.User, group *group.Group, rights AccessRight) error {
	if assignor == nil {
		return core.ErrNilAssignor
	}

	if group == nil {
		return core.ErrNilGroup
	}

	// making sure it's group kind is Group
	if group.Kind != group.GKGroup {
		return core.ErrInvalidGroupKind
	}

	// checking whether the assignor has at least the assigned rights
	if !ap.HasRights(assignor, rights) {
		return core.ErrExcessOfRights
	}

	ap.Lock()
	ap.RightsRoster.Group[group.ID] = rights
	ap.Unlock()

	return nil
}

// SetUserRights setting rights for specific user
// TODO: consider whether it's right to turn off inheritance (if enabled) when setting/changing anything on each access policy instance
func (ap *AccessPolicy) SetUserRights(assignor *user.User, assignee *user.User, rights AccessRight) error {
	if assignor == nil {
		return core.ErrNilAssignor
	}

	if assignee == nil {
		return core.ErrNilAssignee
	}

	// the assignor must have a right to set rights (APManageRights) and have all the
	// rights himself that he's attempting to assign to others
	if !ap.HasRights(assignor, APManageRights|rights) {
		return core.ErrExcessOfRights
	}

	ap.Lock()
	ap.RightsRoster.User[assignee.ID] = rights
	ap.Unlock()

	return nil
}

// IsOwner checks whether a given user is the owner of this policy
func (ap *AccessPolicy) IsOwner(u *user.User) bool {
	// owner of the policy (meaning: the main entity) has full rights on it
	if ap.Owner != nil && (ap.Owner.ID == u.ID) {
		return true
	}

	return false
}

// HasRights checks whether the user has specific rights
// NOTE: returns true only if the user has every of specified rights permitted
// TODO: maybe add some sort of a calculated cache with a short livespan, like 100ms or something
func (ap *AccessPolicy) HasRights(user *user.User, rights AccessRight) bool {
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
func (ap *AccessPolicy) HasGroupRights(g *group.Group, rights AccessRight) bool {
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
func (ap *AccessPolicy) UnsetRights(assignor *user.User, assignee interface{}) error {
	if assignor == nil {
		return core.ErrNilAssignor
	}

	if assignee == nil {
		return core.ErrNilAssignee
	}

	// the assignor must have a right to set rights (APManageRights) and have all the
	// rights himself that he's attempting to assign to others
	if !ap.HasRights(assignor, APManageRights) {
		return core.ErrAccessDenied
	}

	ap.Lock()

	// deleting assignee from the roster (depending on its type)
	switch assignee.(type) {
	case *user.User:
		delete(ap.RightsRoster.User, assignee.(*user.User).ID)
	case *group.Group:
		switch group := assignee.(*group.Group); group.Kind {
		case group.GKRole:
			delete(ap.RightsRoster.Role, assignee.(*group.Group).ID)
		case group.GKGroup:
			delete(ap.RightsRoster.Group, assignee.(*group.Group).ID)
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
		return core.ErrAccessPolicyBackupExists
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
		return core.ErrAccessPolicyBackupNotFound
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
	ap.keyHash = ap.backup.keyHash
	ap.ObjectType = ap.backup.ObjectType
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

// UpdateAccessPolicy saves itself via container (if it belongs to any container)
func (ap *AccessPolicy) Save(ctx context.Context) error {
	if ap.container == nil {
		return core.ErrNilAccessPolicyContainer
	}

	return ap.container.Save(ctx, ap)
}
