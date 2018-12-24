package user

import (
	"sync"

	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

// GroupList is a typed slice of groups to make sorting easier
type GroupList []Group

// GroupContainer is a container responsible for all operations within its scope
// TODO: add default groups which need not to be assigned
type GroupContainer struct {
	ID ulid.ULID `json:"id"`

	domain *Domain
	groups GroupList
	idMap  map[ulid.ULID]*Group
	keyMap map[string]*Group
	store  GroupStore
	sync.RWMutex
}

// NewGroupContainer initializing a new group container attached to domain
func NewGroupContainer(d *Domain) (*GroupContainer, error) {
	if d == nil {
		return nil, ErrNilDomain
	}

	c := &GroupContainer{
		ID:     util.NewULID(),
		domain: d,
		groups: make(GroupList, 0),
		idMap:  make(map[ulid.ULID]*Group),
		keyMap: make(map[string]*Group),
	}

	return c, nil
}

// List returns all groups inside a container
func (c *GroupContainer) List(mask GroupKind) (GroupList, error) {
	gl := make(GroupList, 0)
	for _, g := range c.groups {
		if (g.Kind & mask) == mask {
			gl = append(gl, g)
		}
	}

	return gl, nil
}

// GetByID returns a group by ID
func (c *GroupContainer) GetByID(id ulid.ULID) (*Group, error) {
	if g, ok := c.idMap[id]; ok {
		return g, nil
	}

	return nil, ErrGroupNotFound
}

// GetByKey returns a group by name
func (c *GroupContainer) GetByKey(key string) (*Group, error) {
	if g, ok := c.keyMap[key]; ok {
		return g, nil
	}

	return nil, ErrGroupNotFound
}

// GetByUser returns a slice of groups to which a given user belongs
func (c *GroupContainer) GetByUser(u *User) ([]*Group, error) {
	if u == nil {
		return nil, ErrNilUser
	}

	gs := make([]*Group, 0)
	for _, g := range c.groups {
		if g.IsMember(u) {
			gs = append(gs, &g)
		}
	}

	return gs, nil
}
