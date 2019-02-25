package usermanager

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/oklog/ulid"
)

// GroupList is a typed slice of groups to make sorting easier
type GroupList []Group

// GroupContainer is a container responsible for all operations within its scope
// TODO: add default groups which need not to be assigned
type GroupContainer struct {
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
		groups: make([]*Group, 0),
		idMap:  make(map[ulid.ULID]*Group),
		keyMap: make(map[string]*Group),
		store:  s,
	}

	return c, nil
}

// Store returns store if set
func (c *GroupContainer) Store() (GroupStore, error) {
	if c.store == nil {
		return nil, ErrNilGroupStore
	}

	return c.store, nil
}

// Validate this group container
func (c *GroupContainer) Validate() error {
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

// SetDomain links this container to a given domain, nil is allowed
func (c *GroupContainer) SetDomain(d *Domain) error {
	// merely this atm
	c.domain = d

	return nil
}

// Create creates new group
func (c *GroupContainer) Create(kind GroupKind, key string, name string, parent *Group) (*Group, error) {
	// groups must be of the same kind
	if parent != nil && parent.Kind != kind {
		return nil, ErrGroupKindMismatch
	}

	// initializing new group
	g, err := NewGroup(kind, key, name, parent)
	if err != nil {
		return nil, fmt.Errorf("Create(): failed to initialize new group(%s): %s", key, err)
	}

	// TODO: implement
	// TODO: implement
	// TODO: implement
	// TODO: implement
	// TODO: implement
	// TODO: implement
	// TODO: implement
	// TODO: implement

	return g, nil
}

// Add adds group to the container
func (c *GroupContainer) Add(g *Group) error {
	if g == nil {
		return ErrNilGroup
	}

	_, err := c.Get(g.ID)
	if err != ErrGroupNotFound {
		return ErrGroupAlreadyRegistered
	}

	// TODO: store group

	c.Lock()
	c.groups = append(c.groups, g)
	c.idMap[g.ID] = g
	c.keyMap[g.Key] = g
	c.Unlock()

	return nil
}

// Remove removing group from the container
func (c *GroupContainer) Remove(id ulid.ULID) error {
	// a bit pedantic but consistent, returning an error if group
	// is already registered within the container
	g, err := c.Get(id)
	if err != nil {
		return err
	}

	// removing group from index maps and a main slice
	c.Lock()

	// clearing index maps
	delete(c.idMap, g.ID)
	delete(c.keyMap, g.Key)

	// removing the actual item
	for i, fg := range c.groups {
		if g.ID == fg.ID {
			c.groups = append(c.groups[:i], c.groups[i+1:]...)
			break
		}
	}

	c.Unlock()

	return nil
}

// List returns all groups inside a container
func (c *GroupContainer) List(kind GroupKind) []*Group {
	gs := make([]*Group, 0)
	for _, g := range c.groups {
		if g.Kind&kind != 0 {
			gs = append(gs, g)
		}
	}

	return gs
}

// Get returns a group by ID
func (c *GroupContainer) Get(id ulid.ULID) (*Group, error) {
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
