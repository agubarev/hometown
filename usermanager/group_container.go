package usermanager

import (
	"errors"
	"log"
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
	groups []*Group
	idMap  map[ulid.ULID]*Group
	keyMap map[string]*Group
	store  GroupStore
	sync.RWMutex
}

// NewGroupContainer initializing a new group container attached to domain
func NewGroupContainer(s GroupStore) (*GroupContainer, error) {
	if s == nil {
		log.Println("NewGroupContainer: store isn't set")
	}

	c := &GroupContainer{
		ID:     util.NewULID(),
		groups: make([]*Group, 0),
		idMap:  make(map[ulid.ULID]*Group),
		keyMap: make(map[string]*Group),
		store:  s,
	}

	return c, nil
}

// Validate this group container
func (c *GroupContainer) Validate() error {
	if c.domain == nil {
		return ErrNilDomain
	}

	if c.groups == nil {
		return errors.New("groups slice is not initialized")
	}

	if c.idMap == nil {
		return errors.New("id map is nil")
	}

	if c.keyMap == nil {
		return errors.New("key map is nil")
	}

	return nil
}

// Persist asks all contained groups to store itself
func (c *GroupContainer) Persist() error {
	panic("not implemented")

	return nil
}

// AddGroup adding group to a container
func (c *GroupContainer) AddGroup(g *Group) error {
	panic("not implemented")

	return nil
}

// RemoveGroup removing group from a container, by ID
func (c *GroupContainer) RemoveGroup(id ulid.ULID) error {
	panic("not implemented")

	return nil
}

// List returns all groups inside a container
func (c *GroupContainer) List(kind GroupKind) []*Group {
	gs := make([]*Group, 0)
	for _, g := range c.groups {
		if g.Kind == kind {
			gs = append(gs, g)
		}
	}

	return gs
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
func (c *GroupContainer) GetByUser(k GroupKind, u *User) ([]*Group, error) {
	if u == nil {
		return nil, ErrNilUser
	}

	gs := make([]*Group, 0)
	for _, g := range c.groups {
		if g.Kind == k {
			if g.IsMember(u) {
				gs = append(gs, g)
			}
		}
	}

	return gs, nil
}
