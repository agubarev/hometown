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

// type aliases
type (
	TKey         = [32]byte
	TName        = [256]byte
	TDescription = [512]byte
)

// group kinds
const (
	GKGroup Kind = 1 << iota
	GKRole
	GKAll = ^Kind(0)
)

// Group represents a member group
type Group struct {
	ID        int64 `db:"id" json:"id"`
	ParentID  int64 `db:"parent_id" json:"parent_id"`
	Kind      Kind  `db:"kind" json:"kind"`
	Key       TKey  `db:"key" json:"key" valid:"required,ascii"`
	IsDefault bool  `db:"is_default" json:"is_default"`
}

// FullGroup represents a full version of a Group,
// including its key, name and description, which due to
// its size would be more expensive to move around as a whole
type FullGroup struct {
	Group

	Name        TName        `db:"name" json:"name"`
	Description TDescription `db:"description" json:"description"`
}
