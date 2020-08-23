package group

import (
	"database/sql/driver"
	"strings"

	"github.com/google/uuid"
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
// accesspolicy names for this given context namespace
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

// Group represents a asset group
// TODO: replace Flags and IsDefault with a Flags bitmask
// TODO: work out a simple Flags bit layout
type Group struct {
	DisplayName string    `db:"name" json:"name"`
	Key         string    `db:"key" json:"key" valid:"required,ascii"`
	ID          uuid.UUID `db:"id" json:"id"`
	ParentID    uuid.UUID `db:"parent_id" json:"parent_id"`
	Flags       Flags     `db:"kind" json:"kind"`
	_           struct{}
}

func NewGroup(flags Flags, parentID uuid.UUID, key string, name string) (g Group, err error) {
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

func (g Group) IsDefault() bool { return g.Flags&FDefault == FDefault }
func (g Group) IsEnabled() bool { return g.Flags&FEnabled == FEnabled }
func (g Group) IsGroup() bool   { return g.Flags&FGroup == FGroup }
func (g Group) IsRole() bool    { return g.Flags&FRole == FRole }

func (ak AssetKind) Value() (driver.Value, error) {
	return ak, nil
}

func (ak *AssetKind) Scan(data []byte) error {
	*ak = AssetKind(data[0])
	return nil
}

func (flags Flags) Value() (driver.Value, error) {
	return flags, nil
}
