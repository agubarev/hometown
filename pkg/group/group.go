package group

import (
	"bytes"
	"database/sql/driver"
	"strconv"
	"strings"

	"github.com/pkg/errors"
)

// Kind designates a group kind i.e. Group, Role etc...
type Kind uint8

func (k Kind) String() string {
	switch k {
	case 1:
		return "group"
	case 2:
		return "role group"
	default:
		return "unknown group kind"
	}
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

// group kinds
const (
	GKGroup Kind = 1 << iota
	GKRole
	GKAll = ^Kind(0)
)

// Group represents a member group
// TODO: replace Kind and IsDefault with a Flags bitmask
// TODO: work out a simple Flags bit layout
type Group struct {
	Name      TName  `db:"name" json:"name"`
	Key       TKey   `db:"key" json:"key" valid:"required,ascii"`
	ID        uint32 `db:"id" json:"id"`
	ParentID  uint32 `db:"parent_id" json:"parent_id"`
	Kind      Kind   `db:"kind" json:"kind"`
	IsDefault bool   `db:"is_default" json:"is_default"`
	_         struct{}
}

func NewGroup(kind Kind, parentID uint32, key TKey, name TName, isDefault bool) (g Group, err error) {
	g = Group{
		Name:      name,
		Key:       key,
		ParentID:  parentID,
		Kind:      kind,
		IsDefault: isDefault,
	}

	return g, g.Validate()
}

// Validate validates itself
func (g *Group) Validate() (err error) {
	if g.Kind != GKRole && g.Kind != GKGroup {
		return ErrUnknownKind
	}

	if g.Key[0] == 0 {
		return ErrEmptyKey
	}

	if g.Name[0] == 0 {
		return ErrEmptyGroupName
	}

	return nil
}

// SetKey assigns a key name to the group
func (g *Group) SetKey(key interface{}) error {
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
		newKey := v[0:bytes.IndexByte(v[:], byte(0))]
		if len(newKey) == 0 {
			return ErrEmptyKey
		}

		copy(g.Key[:], newKey)
	}

	return nil
}

// SetName assigns a new name to the group
func (g *Group) SetName(name interface{}) error {
	switch v := name.(type) {
	case string:
		v = strings.TrimSpace(v)
		if v == "" {
			return ErrEmptyGroupName
		}

		copy(g.Name[:], v)
	case []byte:
		v = bytes.TrimSpace(v)
		if len(v) == 0 {
			return ErrEmptyGroupName
		}

		copy(g.Name[:], v)
	case TKey:
		newName := v[0:bytes.IndexByte(v[:], byte(0))]
		if len(newName) == 0 {
			return ErrEmptyGroupName
		}

		copy(g.Name[:], newName)
	}

	return nil
}

//---------------------------------------------------------------------------
// conversions
//---------------------------------------------------------------------------

func (k Kind) Value() (driver.Value, error) {
	return k, nil
}

func (k *Kind) Scan(src interface{}) error {
	n, err := strconv.ParseUint(string(src.([]byte)), 10, 8)
	if err != nil {
		return errors.Wrapf(err, "failed to scan Kind value: %s", src.([]byte))
	}

	*k = Kind(n)

	return nil
}

func (key TKey) Value() (driver.Value, error) {
	return key[0:bytes.IndexByte(key[:], byte(0))], nil
}

func (key *TKey) Scan(v interface{}) error {
	copy(key[:], v.([]byte))
	return nil
}

func (name TName) Value() (driver.Value, error) {
	return name[0:bytes.IndexByte(name[:], byte(0))], nil
}

func (name *TName) Scan(v interface{}) error {
	copy(name[:], v.([]byte))
	return nil
}
