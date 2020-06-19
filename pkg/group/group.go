package group

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

// group kinds
const (
	GKGroup Kind = 1 << iota
	GKRole
	GKAll = ^Kind(0)
)

// Group represents a member group
type Group struct {
	ID        uint32 `db:"id" json:"id"`
	ParentID  uint32 `db:"parent_id" json:"parent_id"`
	Kind      Kind   `db:"kind" json:"kind"`
	Key       TKey   `db:"key" json:"key" valid:"required,ascii"`
	Name      TName  `db:"name" json:"name"`
	IsDefault bool   `db:"is_default" json:"is_default"`
}
