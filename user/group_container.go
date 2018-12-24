package user

import (
	"sync"

	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

type groupList []Group

// GroupContainer represents a group container and is responsible for all
// group-related operations within its scope
// TODO: add default groups which need not to be assigned
type GroupContainer struct {
	ID ulid.ULID `json:"id"`

	domain    *Domain
	groups    groupList
	idIndex   map[ulid.ULID]*Group
	nameIndex map[string]*Group
	store     Store
	sync.RWMutex
}

// NewGroupContainer initializing a new group container attached to domain
func NewGroupContainer(d *Domain) (*GroupContainer, error) {
	if d == nil {
		return nil, ErrNilDomain
	}

	c := &GroupContainer{
		ID:        util.NewULID(),
		domain:    d,
		groups:    make(groupList, 0),
		idIndex:   make(map[ulid.ULID]*Group),
		nameIndex: make(map[string]*Group),
	}

	return c, nil
}

// GetByID returns a group by ID
func (c *GroupContainer) GetByID(id ulid.ULID) (*Group, error) {
	if g, ok := c.idIndex[id]; ok {
		return g, nil
	}

	return nil, ErrGroupNotFound
}

// GetByName returns a group by name
func (c *GroupContainer) GetByName(name string) (*Group, error) {
	if g, ok := c.nameIndex[name]; ok {
		return g, nil
	}

	return nil, ErrGroupNotFound
}

// GetByUser returns a slice of groups to which a given user belongs
func (c *GroupContainer) GetByUser(u *User) ([]*Group, error) {
	if u == nil {
		return nil, ErrNilUser
	}

	return c.userIndex[u.ID], nil
}
