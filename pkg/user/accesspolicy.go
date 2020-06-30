package user

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"strings"
	"sync"

	"github.com/cespare/xxhash"
	"github.com/oklog/ulid"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

// Actor represents anything that can be an owner, assignor ar assignee
// TODO: develop this idea
type Actor interface {
	ID() int64
	UID() ulid.ULID
	Roles() []*Group
	Groups() []*Group
}

// everyone, user, group, role_group, etc...
type SubjectKind uint8

const (
	SKNone     SubjectKind = 0
	SKEveryone SubjectKind = 1 << (iota - SubjectKind(1))
	SKUser
	SKGroup
	SKRoleGroup
)

func (k SubjectKind) String() string {
	switch k {
	case SKEveryone:
		return "everyone"
	case SKUser:
		return "user"
	case SKGroup:
		return "group"
	case SKRoleGroup:
		return "role group"
	default:
		return "unrecognized subject kind"
	}
}

type RRAction uint8

const (
	RRUnset RRAction = iota
	RRSet
)

func (a RRAction) String() string {
	switch a {
	case RRUnset:
		return "unset"
	case RRSet:
		return "set"
	default:
		return "unrecognized action"
	}
}

// AccessRight is a single permission set
type AccessRight uint64

type accessChange struct {
	// denotes an action that occurred: -1 deleted, 0 updated, 1 created
	action      RRAction
	subjectKind SubjectKind
	subjectID   int64
	accessRight AccessRight
}

// declaring discrete rights for all cases
const (
	APNoAccess = AccessRight(0)
	APView     = AccessRight(1 << uint64(iota-1))
	APCreate
	APChange
	APDelete
	APCopy
	APMove
	APManageRights
	APFullAccess = ^AccessRight(0)

	// this flag is used for access bits without translation
	APUnrecognizedFlag = "unrecognized access flag"
)

func (r AccessRight) Translate() string {
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
		if s := bit.Translate(); s != APUnrecognizedFlag {
			dict[uint64(bit)] = bit.Translate()
		}
	}

	return dict
}

// AccessExplained returns a human-readable conjunction of comma-separated
// access names for this given context namespace
func (r AccessRight) String() string {
	s := make([]string, 0)

	for i := 0; i < 63; i++ {
		if bit := AccessRight(1 << i); r&bit != 0 {
			s = append(s, bit.Translate())
		}
	}

	if len(s) == 0 {
		return ""
	}

	return strings.Join(s, ",")
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
func (rr *RightsRoster) addChange(action RRAction, subjectKind SubjectKind, subjectID int64, rights AccessRight) {
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
func (rr *RightsRoster) Summarize(ctx context.Context, userID int64) AccessRight {
	gm := ctx.Value(CKGroupManager).(*GroupManager)
	if gm == nil {
		panic(ErrNilGroupManager)
	}

	r := rr.Everyone

	// calculating standard and role group rights
	// NOTE: if some group doesn't have explicitly set rights, then
	// attempting to obtain the rights of a first ancestor group,
	// that has specific rights set
	for _, g := range gm.GroupsByUserID(ctx, userID, GKRole|GKGroup) {
		r |= rr.GroupRights(ctx, g.ID)
	}

	// user-specific rights
	if _, ok := rr.User[userID]; ok {
		r |= rr.User[userID]
	}

	return r
}

// GroupRights returns the rights of a given group if set explicitly,
// otherwise returns the rights of the first ancestor group that has
// any rights record explicitly set
func (rr *RightsRoster) GroupRights(ctx context.Context, groupID int64) AccessRight {
	if groupID == 0 {
		return APNoAccess
	}

	var rights AccessRight
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
	if g.ParentID != 0 {
		return rr.GroupRights(ctx, g.ParentID)
	}

	return APNoAccess
}

// AccessPolicy is a generalized ruleset for an object
// if IsInherited is true, then the policy's own roster will point to it's parent
// and everything else will be ignored as long as it's true
// NOTE: policy may be shared by multiple entities
// NOTE: policy ownership basically is the ownership of it's main entity and only affects the very object alone
// NOTE: owner is the original creator of an entity and has full rights for it
// NOTE: an access policy can have only one object identifier set, either ObjectID or a Key
// TODO: store object rights roster, name and object type in separate maps
// TODO: calculate extended rights instantly. rights must be recalculated through all the tree after each change
// TODO: add caching mechanism to skip rights summarization
// TODO: disable inheritance if anything is changed about the current policy and create its own rights roster and enable extension by default
type AccessPolicy struct {
	ID          int64  `db:"id" json:"id"`
	ParentID    int64  `db:"parent_id" json:"parent_id"`
	IDPath      string `db:"id_path" json:"id_path"`
	OwnerID     int64  `db:"owner_id" json:"owner_id"`
	Name        string `db:"name" json:"name"`
	ObjectType  string `db:"object_type" json:"object_type"`
	ObjectID    int64  `db:"object_id" json:"object_id"`
	IsExtended  bool   `db:"is_extended" json:"is_extended"`
	IsInherited bool   `db:"is_inherited" json:"is_inherited"`
	Checksum    uint64 `db:"checksum" json:"checksum"`

	RightsRoster *RightsRoster `json:"-"`

	keyHash uint64
	backup  *AccessPolicy

	sync.RWMutex
}

// NewAccessPolicy create a new AccessPolicy object
func NewAccessPolicy(ownerID, parentID int64, name string, objectType string, objectID int64, isInherited, isExtended bool) (ap AccessPolicy, err error) {
	// initializing new policy
	// NOTE: the extension of parent's rights has higher precedence over using the inherited rights
	// because this allows to create independent policies in the middle of a chain and still
	// benefit from using parent's rights as default with it's own corrections/exclusions
	ap = AccessPolicy{
		OwnerID:     ownerID,
		ParentID:    parentID,
		Name:        strings.ToLower(strings.TrimSpace(name)),
		ObjectType:  strings.ToLower(strings.TrimSpace(objectType)),
		ObjectID:    objectID,
		IsInherited: isInherited,
		IsExtended:  isExtended,
	}

	// policy must have some designators
	if ap.Name == "" && ap.ObjectType == "" {
		return ap, errors.Wrap(ErrAccessPolicyEmptyDesignators, "policy cannot have both name and object type empty")
	}

	return ap, nil
}

func (ap *AccessPolicy) calculateChecksum() uint64 {
	buf := new(bytes.Buffer)

	fields := []interface{}{
		ap.ParentID,
		ap.OwnerID,
		[]byte(ap.Name),
		[]byte(ap.ObjectType),
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
	// panic if ObjectID is zero or a name is empty
	if ap.ID == 0 {
		panic(ErrInsufficientDataToHashKey)
	}

	// initializing a key buffer with and assuming the minimum key length
	key := bytes.NewBuffer(make([]byte, 0, 20))

	// composing a key value
	key.WriteString("AccessPolicy")

	if err := binary.Write(key, binary.LittleEndian, ap.ID); err != nil {
		panic(errors.Wrapf(err, "failed to hash access policy: %d", ap.ID))
	}

	// updating calculated key
	ap.keyHash = xxhash.Sum64(key.Bytes())
}

// Key returns a uint64 key hash to be used as a map/cache key
func (ap *AccessPolicy) Key(rehash bool) uint64 {
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
			ap.ParentID = change.To.(int64)
		case "IDPath":
			ap.IDPath = change.To.(string)
		case "OwnerID":
			ap.OwnerID = change.To.(int64)
		case "Key":
			ap.Name = change.To.(string)
		case "ObjectType":
			ap.ObjectType = change.To.(string)
		case "ObjectID":
			ap.ObjectID = change.To.(int64)
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

// SanitizeAndValidate validates access policy by performing basic self-check
func (ap *AccessPolicy) Validate() error {
	if ap == nil {
		return ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return ErrNilRightsRoster
	}

	// kind cannot be empty if either key or ObjectID is set
	if ap.ObjectType == "" && ap.ObjectID != 0 {
		return errors.New("empty object type with a non-zero object id")
	}

	// if object type is set, then ObjectID must also be set
	if ap.ObjectType != "" && ap.ObjectID == 0 {
		return errors.New("non-empty object type with a zero object id")
	}

	// policy must have some designators, a key or an ObjectID of its kind
	if ap.Name == "" && ap.ObjectType == "" && ap.ObjectID == 0 {
		return ErrAccessPolicyEmptyDesignators
	}

	// inherited means that this is not a standalone policy but simply points
	// to its parent policy (first standalone policy to be found)
	if ap.IsInherited && ap.IsExtended {
		return errors.New("policy cannot be both inherited and extended at the same time")
	}

	// parent must be set if this policy inherits or extends
	if ap.ParentID == 0 && (ap.IsInherited || ap.IsExtended) {
		return errors.New("policy cannot inherit or extend without a parent")
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
	return fmt.Sprintf("accesspolicy(%s_%d)", ap.ObjectType, ap.ID)
}

// Clone clones a whole policy
func (ap *AccessPolicy) Clone() (*AccessPolicy, error) {
	rr, err := ap.CloneRightsRoster()
	if err != nil {
		return nil, err
	}

	clone := &AccessPolicy{
		ID:           ap.ID,
		ParentID:     ap.ParentID,
		IDPath:       ap.IDPath,
		OwnerID:      ap.OwnerID,
		Name:         ap.Name,
		ObjectType:   ap.ObjectType,
		ObjectID:     ap.ObjectID,
		IsExtended:   ap.IsExtended,
		IsInherited:  ap.IsInherited,
		Checksum:     0,
		RightsRoster: rr,
		keyHash:      0,
		backup:       nil,
		RWMutex:      sync.RWMutex{},
	}

	return clone, nil
}

// createBackup returns a snapshot copy of the access rights roster for this policy
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

// Parent returns a parent policy, if set
func (ap *AccessPolicy) Parent(ctx context.Context) (p AccessPolicy, err error) {
	if ap.ParentID == 0 {
		return p, ErrNoParentPolicy
	}

	apm := ctx.Value(CKAccessPolicyManager).(*AccessPolicyManager)
	if apm == nil {
		return p, ErrNilAccessPolicyManager
	}

	return apm.PolicyByID(ctx, ap.ParentID)
}

// SetParentID setting a new parent policy
// NOTE: if the parent is set to nil, then forcing IsInherited flag to false
func (ap *AccessPolicy) SetParentID(parentID int64) error {
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
func (ap *AccessPolicy) UserAccess(ctx context.Context, userID int64) (rights AccessRight) {
	if userID == 0 {
		return APNoAccess
	}

	// if this user is the owner, then returning maximum possible value for AccessRight type
	if ap.IsOwner(ctx, userID) {
		return APFullAccess
	}

	// calculating parents access if parent SubjectID is set
	if ap.ParentID != 0 {
		apm := ctx.Value(CKAccessPolicyManager).(*AccessPolicyManager)
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
func (ap *AccessPolicy) SetPublicRights(ctx context.Context, assignorID int64, rights AccessRight) error {
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
	ap.RightsRoster.addChange(RRSet, SKEveryone, 0, rights)

	return nil
}

// SetRoleRights setting rights for the role
// TODO: move to the Manager
func (ap *AccessPolicy) SetRoleRights(ctx context.Context, assignorID int64, roleID int64, rights AccessRight) error {
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
	ap.RightsRoster.addChange(RRSet, SKRoleGroup, roleID, rights)

	return nil
}

// SetGroupRights setting group rights for specific user
// TODO: move to the Manager
func (ap *AccessPolicy) SetGroupRights(ctx context.Context, assignorID int64, groupID int64, rights AccessRight) (err error) {
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
	ap.RightsRoster.addChange(RRSet, SKGroup, groupID, rights)

	return nil
}

// SetUserRights setting rights for specific user
// TODO: consider whether it's right to turn off inheritance (if enabled) when setting/changing anything on each access policy instance
func (ap *AccessPolicy) SetUserRights(ctx context.Context, assignorID int64, assigneeID int64, rights AccessRight) error {
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
	ap.RightsRoster.addChange(RRSet, SKUser, assigneeID, rights)

	return nil
}

// IsOwner checks whether a given user is the owner of this policy
func (ap *AccessPolicy) IsOwner(ctx context.Context, userID int64) bool {
	// owner of the policy (meaning: the main entity) has full rights on it
	if ap.OwnerID != 0 && (ap.OwnerID == userID) {
		return true
	}

	return false
}

// hasRights checks whether the user has specific rights
// NOTE: returns true only if the user has every of specified rights permitted
// TODO: maybe add some sort of a calculated cache with a short livespan, like 10ms or something
func (ap *AccessPolicy) HasRights(ctx context.Context, userID int64, rights AccessRight) bool {
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
	var cr AccessRight

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

	// merging with the actual policy's rights roster rights
	ap.RLock()
	cr |= ap.RightsRoster.Summarize(ctx, userID)
	ap.RUnlock()

	return (cr & rights) == rights
}

// HasGroupRights checks whether a group has the rights
func (ap *AccessPolicy) HasGroupRights(ctx context.Context, groupID int64, rights AccessRight) bool {
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
func (ap *AccessPolicy) UnsetRights(ctx context.Context, assignorID int64, assignee interface{}) error {
	if assignorID == 0 {
		return ErrZeroUserID
	}

	// the assignorID must have a right to set rights (APManageRights) and have all the
	// rights himself that he's attempting to assign to others
	if !ap.HasRights(ctx, assignorID, APManageRights) {
		return ErrAccessDenied
	}

	ap.Lock()

	// deleting assignee from the roster (depending on its type)
	switch sub := assignee.(type) {
	case nil:
		ap.RightsRoster.Everyone = APNoAccess
		ap.RightsRoster.addChange(RRSet, SKEveryone, 0, 0)
	case User:
		delete(ap.RightsRoster.User, sub.ID)
		ap.RightsRoster.addChange(RRUnset, SKUser, sub.ID, 0)
	case Group:
		switch sub.Kind {
		case GKRole:
			delete(ap.RightsRoster.Role, sub.ID)
			ap.RightsRoster.addChange(RRUnset, SKRoleGroup, sub.ID, 0)
		case GKGroup:
			delete(ap.RightsRoster.Group, sub.ID)
			ap.RightsRoster.addChange(RRUnset, SKGroup, sub.ID, 0)
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
func (ap *AccessPolicy) RestoreBackup() (err error) {
	if ap.backup == nil {
		return ErrAccessPolicyBackupNotFound
	}

	if err = ap.backup.Validate(); err != nil {
		return errors.Wrap(err, "policy backup validation failed")
	}

	if ap.ID != ap.backup.ID {
		return errors.New("actual policy and back ids do not match")
	}

	// restoring backup (restoring manually, field by field)
	ap.OwnerID = ap.backup.OwnerID
	ap.ParentID = ap.backup.ParentID
	ap.keyHash = ap.backup.keyHash
	ap.ObjectType = ap.backup.ObjectType
	ap.ObjectID = ap.backup.ObjectID
	ap.RightsRoster = ap.backup.RightsRoster
	ap.IsInherited = ap.backup.IsInherited
	ap.IsExtended = ap.backup.IsExtended

	// clearing backup
	ap.backup = nil

	// clearing rights roster changelist
	ap.RightsRoster.changes = nil

	return nil
}

// Backup returns backup policy if exists or nil
func (ap *AccessPolicy) Backup() (backup AccessPolicy, err error) {
	if ap.backup == nil {
		return backup, ErrNoPolicyBackup
	}

	return *ap.backup, nil
}
