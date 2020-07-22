package accesspolicy

import (
	"bytes"
	"database/sql/driver"
	"runtime/debug"
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
	Name TObjectName
	ID   uuid.UUID
}

func NewObject(id uuid.UUID, name TObjectName) Object {
	return Object{
		Name: name,
		ID:   id,
	}
}

func NilObject() Object {
	return Object{
		Name: TObjectName{},
		ID:   uuid.Nil,
	}
}

type (
	TKey        [32]byte
	TObjectName [32]byte
)

func Key(s string) (v TKey) {
	copy(v[:], strings.ToLower(strings.TrimSpace(s)))
	return v
}

func ObjectName(s string) (v TObjectName) {
	copy(v[:], strings.TrimSpace(s))
	return v
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
// NOTE: an accesspolicy policy can have only one object identifier set, either ObjectID or a TKey
// TODO: store object rights rosters, name and object name in separate maps
// TODO: calculate extended rights instantly. rights must be recalculated through all the tree after each rosterChange
// TODO: add caching mechanism to skip rights summarization
// TODO: disable inheritance if anything is changed about the current policy and create its own rights rosters and enable extension by default
type Policy struct {
	Key        TKey        `db:"key" json:"key"`
	ObjectName TObjectName `db:"object_name" json:"object_name"`
	ID         uuid.UUID   `db:"id" json:"id"`
	ParentID   uuid.UUID   `db:"parent_id" json:"parent_id"`
	OwnerID    uuid.UUID   `db:"owner_id" json:"owner_id"`
	ObjectID   uuid.UUID   `db:"object_id" json:"object_id"`
	Flags      uint8       `db:"flags" json:"flags"`
	_          struct{}
}

// NewPolicy create a new Policy object
func NewPolicy(key TKey, ownerID, parentID uuid.UUID, obj Object, flags uint8) (p Policy, err error) {
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
	if key[0] != 0 {
		if err = p.SetKey(key, 32); err != nil {
			return p, errors.Wrap(err, "failed to set initial key")
		}
	}

	if err = p.SetObjectName(obj.Name, 32); err != nil {
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
		case "TKey":
			ap.Key = change.To.(TKey)
		case "TObjectName":
			ap.ObjectName = change.To.(TObjectName)
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
	if ap.Key[0] == 0 && ap.ObjectName[0] == 0 {
		debug.PrintStack()
		return errors.Wrap(ErrAccessPolicyEmptyDesignators, "policy cannot have both key and object name empty")
	}

	// making sure that both the object name and ActorID are set,
	// if either one of them is provided
	if ap.ObjectName[0] == 0 && ap.ObjectID != uuid.Nil {
		return errors.New("empty object name with a non-zero object id")
	}

	// if object name is set, then ObjectID must also be set
	if ap.ObjectName[0] != 0 && ap.ObjectID == uuid.Nil {
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
func (ap *Policy) SetKey(key interface{}, maxLen int) error {
	if ap.ID != uuid.Nil {
		return ErrForbiddenChange
	}

	switch v := key.(type) {
	case string:
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" {
			return ErrEmptyKey
		}

		copy(ap.Key[:], v)
	case []byte:
		v = bytes.ToLower(bytes.TrimSpace(v))
		if len(v) == 0 {
			return ErrEmptyKey
		}

		copy(ap.Key[:], v)
	case TKey:
		newKey := v[0:maxLen]
		if len(newKey) == 0 {
			return ErrEmptyKey
		}

		copy(ap.Key[:], newKey)
	}

	return nil
}

// SetObjectName sets an object name name
func (ap *Policy) SetObjectName(objectType interface{}, maxLen int) error {
	if ap.ID != uuid.Nil {
		return ErrForbiddenChange
	}

	switch v := objectType.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return ErrEmptyObjectType
		}

		copy(ap.ObjectName[:], v)
	case []byte:
		v = bytes.TrimSpace(v)
		if len(v) == 0 {
			return ErrEmptyObjectType
		}

		copy(ap.ObjectName[:], v)
	case TKey:
		newName := v[0:maxLen]
		if len(newName) == 0 {
			return ErrEmptyObjectType
		}

		copy(ap.ObjectName[:], newName)
	}

	return nil
}

//---------------------------------------------------------------------------
// conversions
//---------------------------------------------------------------------------

func (key TKey) Value() (driver.Value, error) {
	if key[0] == 0 {
		return nil, nil
	}

	zeroPos := bytes.IndexByte(key[:], byte(0))
	if zeroPos == -1 {
		return key[:], nil
	}

	return key[0:zeroPos], nil
}

func (key *TKey) Scan(v interface{}) error {
	if v == nil {
		key[0] = 0
		return nil
	}

	copy(key[:], v.([]byte))

	return nil
}

func (name TObjectName) Value() (driver.Value, error) {
	// a little hack to store an empty string instead of zeroes
	if name[0] == 0 {
		return nil, nil
	}

	zeroPos := bytes.IndexByte(name[:], byte(0))
	if zeroPos == -1 {
		return name[:], nil
	}

	return name[0:zeroPos], nil
}

func (name *TObjectName) Scan(v interface{}) error {
	if v == nil {
		name[0] = 0
		return nil
	}

	copy(name[:], v.([]byte))

	return nil
}
