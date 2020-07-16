package group

import (
	"bytes"
	"database/sql/driver"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

// Flags designates whether a group is enabled, default, a role or a standard group
type Flags uint8

const (
	FEnabled Flags = 1 << iota
	FDefault
	FGroup
	FRole
	FAllGroups = FGroup | FRole

	// this flag is used for group flags without translation
	APUnrecognizedFlag = "unrecognized group flag"
)

func (flags Flags) Translate() string {
	switch flags {
	case FEnabled:
		return "enabled"
	case FGroup:
		return "group"
	case FRole:
		return "group"
	case FAllGroups:
		return "groups and roles"
	default:
		return APUnrecognizedFlag
	}
}

// AccessExplained returns a human-readable conjunction of comma-separated
// access names for this given context namespace
func (flags Flags) String() string {
	s := make([]string, 0)

	for i := 0; i < 31; i++ {
		if bit := Flags(1 << i); flags&bit != 0 {
			s = append(s, bit.Translate())
		}
	}

	if len(s) == 0 {
		return ""
	}

	return strings.Join(s, ",")
}

// FlagDictionary returns a map of property flag values to their respective names
func FlagDictionary() map[uint32]string {
	dict := make(map[uint32]string)

	for bit := Flags(1 << 7); bit > 0; bit >>= 1 {
		if s := bit.Translate(); s != APUnrecognizedFlag {
			dict[uint32(bit)] = bit.Translate()
		}
	}

	return dict
}

// using byte arrays as a replacement for strings
type (
	TKey  [32]byte
	TName [128]byte
)

func NewKey(skey string) (key TKey) {
	copy(key[:], strings.ToLower(strings.TrimSpace(skey)))
	return key
}

func NewName(sname string) (name TName) {
	copy(name[:], strings.TrimSpace(sname))
	return name
}

// Group represents a asset group
// TODO: replace Flags and IsDefault with a Flags bitmask
// TODO: work out a simple Flags bit layout
type Group struct {
	DisplayName TName     `db:"name" json:"name"`
	Key         TKey      `db:"key" json:"key" valid:"required,ascii"`
	ID          uuid.UUID `db:"id" json:"id"`
	ParentID    uuid.UUID `db:"parent_id" json:"parent_id"`
	Flags       Flags     `db:"kind" json:"kind"`
	_           struct{}
}

func NewGroup(flags Flags, parentID uuid.UUID, key TKey, name TName) (g Group, err error) {
	g = Group{
		DisplayName: name,
		Key:         key,
		ParentID:    parentID,
		Flags:       flags,
	}

	return g, g.Validate()
}

// Validate validates itself
func (g *Group) Validate() (err error) {
	// group cannot simultaneously be a role and a standard group
	if g.Flags&FAllGroups == FAllGroups {
		return ErrAmbiguousKind
	}

	// group kind must be one of the defined
	if g.Flags&FAllGroups == 0 {
		return ErrUnknownKind
	}

	// group key must not be empty
	if g.Key[0] == 0 {
		return ErrEmptyKey
	}

	// group display name must not be empty
	if g.DisplayName[0] == 0 {
		return ErrEmptyGroupName
	}

	return nil
}

func (g Group) IsDefault() bool {
	return g.Flags&FDefault == FDefault
}

func (g Group) IsEnabled() bool {
	return g.Flags&FEnabled == FEnabled
}

func (g Group) IsGroup() bool {
	return g.Flags&FGroup == FGroup
}

func (g Group) IsRole() bool {
	return g.Flags&FRole == FRole
}

// SetKey assigns a key name to the group
func (g *Group) SetKey(key interface{}, maxLen int) error {
	switch v := key.(type) {
	case string:
		v = strings.ToLower(strings.TrimSpace(v))
		if v == "" {
			return ErrEmptyKey
		}

		copy(g.Key[:], v)
	case []byte:
		v = bytes.ToLower(bytes.TrimSpace(v))
		if len(v) == 0 {
			return ErrEmptyKey
		}

		copy(g.Key[:], v)
	case TKey:
		newKey := v[0:bytes.IndexByte(v[0:maxLen], byte(0))]
		if len(newKey) == 0 {
			return ErrEmptyKey
		}

		copy(g.Key[:], newKey)
	}

	return nil
}

// SetName assigns a new name to the group
func (g *Group) SetName(name interface{}, maxLen int) error {
	switch v := name.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return ErrEmptyGroupName
		}

		copy(g.DisplayName[:], v)
	case []byte:
		v = bytes.TrimSpace(v)
		if len(v) == 0 {
			return ErrEmptyGroupName
		}

		copy(g.DisplayName[:], v)
	case TKey:
		newName := v[0:maxLen]
		if len(newName) == 0 {
			return ErrEmptyGroupName
		}

		copy(g.DisplayName[:], newName)
	}

	return nil
}

//---------------------------------------------------------------------------
// conversions
//---------------------------------------------------------------------------

func (flags Flags) Value() (driver.Value, error) {
	return flags, nil
}

func (flags *Flags) Scan(src interface{}) error {
	n, err := strconv.ParseUint(string(src.([]byte)), 10, 8)
	if err != nil {
		return errors.Wrapf(err, "failed to scan Flags value: %s", src.([]byte))
	}

	*flags = Flags(n)

	return nil
}

func (key TKey) Value() (driver.Value, error) {
	// a little hack to store an empty string instead of zeroes
	if key[0] == 0 {
		return "", nil
	}

	zeroPos := bytes.IndexByte(key[:], byte(0))
	if zeroPos == -1 {
		return key[:], nil
	}

	return key[0:zeroPos], nil
}

func (key *TKey) Scan(v interface{}) error {
	copy(key[:], v.([]byte))
	return nil
}

func (name TName) Value() (driver.Value, error) {
	// a little hack to store an empty string instead of zeroes
	if name[0] == 0 {
		return "", nil
	}

	zeroPos := bytes.IndexByte(name[:], byte(0))
	if zeroPos == -1 {
		return name[:], nil
	}

	return name[0:zeroPos], nil
}

func (name *TName) Scan(v interface{}) error {
	copy(name[:], v.([]byte))
	return nil
}
