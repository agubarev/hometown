package user

import (
	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

// GroupContainer represents a group container and is responsible for all
// group-related operations within its scope
// TODO: add default groups which need not to be assigned
type GroupContainer struct {
	ID ulid.ULID `json:"id"`

	domain    *Domain
	groups    map[ulid.ULID]Group
	nameIndex map[string]*Group
	userIndex map[ulid.ULID][]*Group
}

// Group represents a user group
type Group struct {
	ID     ulid.ULID `json:"id"`
	Parent ulid.ULID `json:"parent"`
	Name   string    `json:"name"`

	container *GroupContainer
}

// NewGroupContainer initializing a new group container attached to domain
func NewGroupContainer(d *Domain) (*GroupContainer, error) {
	if domain == nil {
		return nil, ErrNilDomain
	}

	c := &GroupContainer{
		ID:        util.NewULID,
		domain:    d,
		groups:    make(map[ulid.ULID]Group),
		nameIndex: make(map[string]*Group),
		userIndex: make(map[ulid.ULID]*Group),
	}

	return c, nil
}

// NewGroup initializing a new group struct
func NewGroup(name string, parent *Group) *Group {
	return &Group{
		ID:     util.NewULID(),
		Parent: parent,
		Name:   name,
	}
}

// GetByUser returns a slice of groups to which a given user belongs
func (gc *GroupContainer) GetByUser(u *User) []*Group {
	groups := make([]*Group, 0)

	return groups
}
