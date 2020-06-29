package user

import (
	"bytes"
	"database/sql/driver"
	"strings"

	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

// policy flags
const (
	FInherited uint8 = 1 << (iota - uint8(1))
	FExtended
)

type (
	TKey        [32]byte
	TObjectType [32]byte
)

func NewKey(s string) (v TKey) {
	copy(v[:], strings.ToLower(strings.TrimSpace(s)))
	return v
}

func NewObjectType(s string) (v TObjectType) {
	copy(v[:], strings.TrimSpace(s))
	return v
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
		return "unrecognized action"
	}
}

// Right is a single permission set
type Right uint32

type accessChange struct {
	// denotes an action that occurred: -1 deleted, 0 updated, 1 created
	action      RAction
	subjectKind SubjectKind
	subjectID   uint32
	accessRight Right
}

// declaring discrete rights for all cases
const (
	APNoAccess = Right(0)
	APView     = Right(1 << (iota - Right(1)))
	APCreate
	APChange
	APDelete
	APCopy
	APMove
	APManageRights
	APFullAccess = ^Right(0)

	// this flag is used for access bits without translation
	APUnrecognizedFlag = "unrecognized access flag"
)

func (r Right) Translate() string {
	switch r {
	case APNoAccess:
		return "no_access"
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
		return "full_access"
	default:
		return APUnrecognizedFlag
	}
}

// PropertyDictionary returns a map of property flag values to their respective names
func AccessPolicyDictionary() map[uint32]string {
	dict := make(map[uint32]string)

	for bit := Right(1 << 31); bit > 0; bit >>= 1 {
		if s := bit.Translate(); s != APUnrecognizedFlag {
			dict[uint32(bit)] = bit.Translate()
		}
	}

	return dict
}

// AccessExplained returns a human-readable conjunction of comma-separated
// access names for this given context namespace
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

// AccessPolicy is a generalized ruleset for an object
// if IsInherited is true, then the policy's own rosters will point to it's parent
// and everything else will be ignored as long as it's true
// NOTE: policy may be shared by multiple entities
// NOTE: policy ownership basically is the ownership of it's main entity and only affects the very object alone
// NOTE: owner is the original creator of an entity and has full rights for it
// NOTE: an access policy can have only one object identifier set, either ObjectID or a Key
// TODO: store object rights rosters, name and object type in separate maps
// TODO: calculate extended rights instantly. rights must be recalculated through all the tree after each change
// TODO: add caching mechanism to skip rights summarization
// TODO: disable inheritance if anything is changed about the current policy and create its own rights rosters and enable extension by default
type AccessPolicy struct {
	Key        TKey        `db:"key" json:"key"`
	ObjectType TObjectType `db:"object_type" json:"object_type"`
	ID         uint32      `db:"id" json:"id"`
	ParentID   uint32      `db:"parent_id" json:"parent_id"`
	OwnerID    uint32      `db:"owner_id" json:"owner_id"`
	ObjectID   uint32      `db:"object_id" json:"object_id"`
	Flags      uint8       `db:"flags" json:"flags"`
	_          struct{}
}

// NewAccessPolicy create a new AccessPolicy object
func NewAccessPolicy(key TKey, ownerID, parentID, objectID uint32, objectType TObjectType, flags uint8) (ap AccessPolicy, err error) {
	// initializing new policy
	// NOTE: the extension of parent's rights has higher precedence over using the inherited rights
	// because this allows to create independent policies in the middle of a chain and still
	// benefit from using parent's rights as default with it's own corrections/exclusions
	ap = AccessPolicy{
		OwnerID:  ownerID,
		ParentID: parentID,
		ObjectID: objectID,
		Flags:    flags,
	}

	if err = ap.SetKey(key); err != nil {
		return ap, errors.Wrap(err, "failed to set initial key")
	}

	if err = ap.SetObjectType(objectType); err != nil {
		return ap, errors.Wrap(err, "failed to set initial object type")
	}

	return ap, ap.Validate()
}

// ApplyChangelog applies changes described by a diff.Diff()'s changelog
// NOTE: doing a manual update to avoid using reflection
func (ap *AccessPolicy) ApplyChangelog(changelog diff.Changelog) (err error) {
	// it's ok if there are no changes to apply
	if len(changelog) == 0 {
		return nil
	}

	for _, change := range changelog {
		switch change.Path[0] {
		case "ParentID":
			ap.ParentID = change.To.(uint32)
		case "OwnerID":
			ap.OwnerID = change.To.(uint32)
		case "Key":
			ap.Key = change.To.(TKey)
		case "ObjectType":
			ap.ObjectType = change.To.(TObjectType)
		case "ObjectID":
			ap.ObjectID = change.To.(uint32)
		case "Flags":
			ap.Flags = change.To.(uint8)
		}
	}

	return nil
}

// SanitizeAndValidate validates access policy by performing basic self-check
func (ap *AccessPolicy) Validate() error {
	if ap == nil {
		return ErrNilAccessPolicy
	}

	// policy must have some designators
	if ap.Key[0] == 0 && ap.ObjectType[0] == 0 {
		return errors.Wrap(ErrAccessPolicyEmptyDesignators, "policy cannot have both key and object type empty")
	}

	// making sure that both the object type and ID are set,
	// if either one of them is provided
	if ap.ObjectType[0] == 0 && ap.ObjectID != 0 {
		return errors.New("empty object type with a non-zero object id")
	}

	// if object type is set, then ObjectID must also be set
	if ap.ObjectType[0] != 0 && ap.ObjectID == 0 {
		return errors.New("zero object id with a non-empty object type")
	}

	// inherited means that this is not a standalone policy but simply points
	// to its parent policy (first standalone policy to be found)
	if ap.IsInherited() && ap.IsExtended() {
		return errors.New("policy cannot be both inherited and extended at the same time")
	}

	// parent must be set if this policy inherits or extends
	if ap.ParentID == 0 && (ap.IsInherited() || ap.IsExtended()) {
		return errors.New("policy cannot inherit or extend without a parent")
	}

	return nil
}

func (ap AccessPolicy) IsInherited() bool {
	return (ap.Flags & FInherited) == FInherited
}

func (ap AccessPolicy) IsExtended() bool {
	return (ap.Flags & FExtended) == FExtended
}

// SetKey sets a key name to the group
func (ap *AccessPolicy) SetKey(key interface{}) error {
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
		newKey := v[0:bytes.IndexByte(v[:], byte(0))]
		if len(newKey) == 0 {
			return ErrEmptyKey
		}

		copy(ap.Key[:], newKey)
	}

	return nil
}

// SetObjectType sets an object type name
func (ap *AccessPolicy) SetObjectType(objectType interface{}) error {
	switch v := objectType.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return ErrEmptyObjectType
		}

		copy(ap.ObjectType[:], v)
	case []byte:
		v = bytes.TrimSpace(v)
		if len(v) == 0 {
			return ErrEmptyObjectType
		}

		copy(ap.ObjectType[:], v)
	case TKey:
		newName := v[0:bytes.IndexByte(v[:], byte(0))]
		if len(newName) == 0 {
			return ErrEmptyObjectType
		}

		copy(ap.ObjectType[:], newName)
	}

	return nil
}

//---------------------------------------------------------------------------
// conversions
//---------------------------------------------------------------------------

func (key TKey) Value() (driver.Value, error) {
	return key[0:bytes.IndexByte(key[:], byte(0))], nil
}

func (key *TKey) Scan(v interface{}) error {
	copy(key[:], v.([]byte))
	return nil
}

func (typ TObjectType) Value() (driver.Value, error) {
	return typ[0:bytes.IndexByte(typ[:], byte(0))], nil
}

func (typ *TObjectType) Scan(v interface{}) error {
	copy(typ[:], v.([]byte))
	return nil
}
