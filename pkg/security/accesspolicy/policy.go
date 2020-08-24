package accesspolicy

import (
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

// policy flags
const (
	FInherit uint8 = 1 << iota
	FExtend
	FSealed
)

type Object struct {
	Name string
	ID   uuid.UUID
}

func NewObject(id uuid.UUID, name string) Object {
	return Object{
		Name: name,
		ID:   id,
	}
}

func NilObject() Object {
	return Object{
		Name: "",
		ID:   uuid.Nil,
	}
}

// everyone, user, group, role_group, etc...
type ActorKind uint8

const (
	AEveryone ActorKind = 1 << iota
	AUser
	AGroup
	ARoleGroup
)

func (k ActorKind) String() string {
	switch k {
	case AEveryone:
		return "everyone"
	case AUser:
		return "user"
	case AGroup:
		return "group"
	case ARoleGroup:
		return "role group"
	default:
		return "unrecognized actor kind"
	}
}

type RAction uint8

const (
	RUnset RAction = iota
	RSet
)

func (a RAction) String() string {
	switch a {
	case RUnset:
		return "unset"
	case RSet:
		return "set"
	default:
		return "unrecognized roster action"
	}
}

// Right is a single permission set
type Right uint32

type rosterChange struct {
	// denotes an action that occurred: -1 deleted, 0 updated, 1 created
	action      RAction
	key         Actor
	accessRight Right
}

// declaring discrete rights for all cases
const (
	APNoAccess = Right(0)
	APView     = Right(1 << (iota - Right(1)))
	APViewDeleted
	APViewHidden
	APCreate
	APChange
	APDelete
	APRestoreDeleted
	APCopy
	APDuplicate
	APMove
	APManageAccess
	APFullAccess = ^Right(0)

	// this flag is used for accesspolicy bits without translation
	APUnrecognizedFlag = "unrecognized accesspolicy flag"
)

func (r Right) Translate() string {
	switch r {
	case APNoAccess:
		return "no_access"
	case APView:
		return "view"
	case APViewDeleted:
		return "view_deleted"
	case APViewHidden:
		return "view_hidden"
	case APCreate:
		return "create"
	case APChange:
		return "rosterChange"
	case APDelete:
		return "delete"
	case APRestoreDeleted:
		return "restore_deleted"
	case APCopy:
		return "copy"
	case APDuplicate:
		return "duplicate"
	case APMove:
		return "move"
	case APManageAccess:
		return "manage_access"
	case APFullAccess:
		return "full_access"
	default:
		return APUnrecognizedFlag
	}
}

// Dictionary returns a map of property flag values to their respective names
func Dictionary() map[uint32]string {
	dict := make(map[uint32]string)

	for bit := Right(1 << 31); bit > 0; bit >>= 1 {
		if s := bit.Translate(); s != APUnrecognizedFlag {
			dict[uint32(bit)] = bit.Translate()
		}
	}

	return dict
}

// AccessExplained returns a human-readable conjunction of comma-separated
// accesspolicy names for this given context namespace
func (r Right) String() string {
	s := make([]string, 0)

	for i := 0; i < 31; i++ {
		if bit := Right(1 << i); r&bit != 0 {
			s = append(s, bit.Translate())
		}
	}

	if len(s) == 0 {
		return ""
	}

	return strings.Join(s, ",")
}

// Policy is a generalized ruleset for an object
// if IsInherited is true, then the policy's own rosters will point to it's parent
// and everything else will be ignored as long as it's true
// NOTE: policy may be shared by multiple entities
// NOTE: policy ownership basically is the ownership of it's main entity and only affects the very object alone
// NOTE: owner is the original creator of an entity and has full rights for it
// NOTE: an accesspolicy policy can have only one object identifier set, either ObjectID or astring
// TODO: store object rights rosters, name and object name in separate maps
// TODO: calculate extended rights instantly. rights must be recalculated through all the tree after each rosterChange
// TODO: add caching mechanism to skip rights summarization
// TODO: disable inheritance if anything is changed about the current policy and create its own rights rosters and enable extension by default
type Policy struct {
	Key        string    `db:"key" json:"key"`
	ObjectName string    `db:"object_name" json:"object_name"`
	ID         uuid.UUID `db:"id" json:"id"`
	ParentID   uuid.UUID `db:"parent_id" json:"parent_id"`
	OwnerID    uuid.UUID `db:"owner_id" json:"owner_id"`
	ObjectID   uuid.UUID `db:"object_id" json:"object_id"`
	Flags      uint8     `db:"flags" json:"flags"`
	_          struct{}
}

// NewPolicy create a new Policy object
func NewPolicy(key string, ownerID, parentID uuid.UUID, obj Object, flags uint8) (p Policy, err error) {
	// initializing new policy
	// NOTE: the extension of parent's rights has higher precedence over using the inherited rights
	// because this allows to create independent policies in the middle of a chain and still
	// benefit from using parent's rights as default with it's own corrections/exclusions
	p = Policy{
		OwnerID:    ownerID,
		ParentID:   parentID,
		Key:        key,
		ObjectID:   obj.ID,
		ObjectName: obj.Name,
		Flags:      flags,
	}

	// NOTE: key may be optional
	if key != "" {
		if err = p.SetKey(key); err != nil {
			return p, errors.Wrap(err, "failed to set initial key")
		}
	}

	if err = p.SetObjectName(obj.Name); err != nil {
		return p, errors.Wrap(err, "failed to set initial object name")
	}

	if err = p.Validate(); err != nil {
		return p, errors.Wrap(err, "validation failed")
	}

	return p, nil
}

// IsOwner checks whether a given user is the owner of this policy
// NOTE: owner of the policy (meaning: the main entity) has full rights on it
// WARNING: *** DO NOT remove owner zero check, to reduce the risk of abuse or mistake ***
func (ap Policy) IsOwner(id uuid.UUID) bool {
	return ap.OwnerID != uuid.Nil && (ap.OwnerID == id)
}

// ApplyChangelog applies changes described by a diff.Diff()'s changelog
// NOTE: doing a manual update to avoid using reflection
func (ap *Policy) ApplyChangelog(changelog diff.Changelog) (err error) {
	// it's ok if there are no changes to apply
	if len(changelog) == 0 {
		return nil
	}

	for _, change := range changelog {
		switch change.Path[0] {
		case "ParentID":
			ap.ParentID = change.To.(uuid.UUID)
		case "OwnerID":
			ap.OwnerID = change.To.(uuid.UUID)
		case "Key":
			ap.Key = change.To.(string)
		case "ObjectName":
			ap.ObjectName = change.To.(string)
		case "ObjectID":
			ap.ObjectID = change.To.(uuid.UUID)
		case "Flags":
			ap.Flags = change.To.(uint8)
		}
	}

	return nil
}

// SanitizeAndValidate validates accesspolicy policy by performing basic self-check
func (ap Policy) Validate() error {
	// policy must have some designators
	if ap.Key == "" && ap.ObjectName == "" {
		return errors.Wrap(ErrAccessPolicyEmptyDesignators, "policy cannot have both key and object name empty")
	}

	// making sure that both the object name and ActorID are set,
	// if either one of them is provided
	if ap.ObjectName == "" && ap.ObjectID != uuid.Nil {
		return errors.New("empty object name with a non-zero object id")
	}

	// if object name is set, then ObjectID must also be set
	if ap.ObjectName != "" && ap.ObjectID == uuid.Nil {
		return errors.New("zero object id with a non-empty object name")
	}

	// inherited means that this is not a standalone policy but simply points
	// to its parent policy (first standalone policy to be found)
	if ap.IsInherited() && ap.IsExtended() {
		return errors.New("policy cannot be both inherited and extended at the same time")
	}

	// parent must be set if this policy inherits or extends
	if ap.ParentID == uuid.Nil && (ap.IsInherited() || ap.IsExtended()) {
		return errors.New("policy cannot inherit or extend without a parent")
	}

	return nil
}

func (ap Policy) IsInherited() bool {
	return (ap.Flags & FInherit) == FInherit
}

func (ap Policy) IsExtended() bool {
	return (ap.Flags & FExtend) == FExtend
}

// SetKey sets a key name to the group
func (ap *Policy) SetKey(key string) error {
	if ap.ID != uuid.Nil {
		return ErrForbiddenChange
	}

	// setting new key
	ap.Key = key

	return nil
}

// SetObjectName sets an object name name
func (ap *Policy) SetObjectName(name string) error {
	if ap.ID != uuid.Nil {
		return ErrForbiddenChange
	}

	// setting new object name
	ap.ObjectName = name

	return nil
}
