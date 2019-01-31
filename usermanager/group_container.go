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

// Save asks all contained groups to store itself
func (c *GroupContainer) Save() error {

	return nil
}

// SetDomain is called when this container is attached to a domain
func (c *GroupContainer) SetDomain(d *Domain) error {
	if d == nil {
		return ErrNilDomain
	}

	// link this container to a given domain
	c.domain = d

	return nil
}

// Register adding group to a container
func (c *GroupContainer) Register(g *Group) error {
	_, err := c.GetByID(g.ID)
	if err != ErrGroupNotFound {
		return ErrGroupAlreadyRegistered
	}

	// check if the group is in store, otherwise create
	if _, err := c.store.GetByID(g.ID); err == ErrGroupNotFound {
		if err = c.store.Put(g); err != nil {
			return fmt.Errorf("failed to stored registered group [%s]: %s", g.ID, err)
		}
	}

	// linking group to this container
	g.container = c

	c.Lock()
	c.groups = append(c.groups, g)
	c.idMap[g.ID] = g
	c.keyMap[g.Key] = g
	c.Unlock()

	return nil
}

// Unregister removing group from a container by ID
func (c *GroupContainer) Unregister(id ulid.ULID) error {
	// a bit pedantic but consistent, returning an error if the group isn't
	// already registered
	g, err := c.GetByID(id)
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
