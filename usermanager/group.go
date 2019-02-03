package usermanager

import (
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"

	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

// GroupMembers is a slice of users, typed for the ease of sorting
type GroupMembers []*User

// GroupKind designates a group kind i.e. Group, Role etc...
type GroupKind uint8

func (k GroupKind) String() string {
	switch k {
	case 1:
		return "group"
	case 2:
		return "role group"
	default:
		return "unknown group kind"
	}
}

// group kinds
const (
	GKGroup GroupKind = iota + 1
	GKRole
)

// Group represents a user group
// TODO custom JSON marshalling
// TODO add mutex and store to the group; store should be set implicitly upon addition to the container
type Group struct {
	ID           ulid.ULID     `json:"id"`
	Parent       *Group        `json:"-"`
	Domain       *Domain       `json:"-"`
	Kind         GroupKind     `json:"kind"`
	Key          string        `json:"key" valid:"required,ascii"`
	Name         string        `json:"name" valid:"required"`
	Description  string        `json:"desc" valid:"optional,length(0|200)"`
	AccessPolicy *AccessPolicy `json:"-"`

	// these fields are basically just for the storage
	DomainID ulid.ULID `json:"did"`
	ParentID ulid.ULID `json:"pid"`

	container *GroupContainer
	members   GroupMembers
	memberMap map[ulid.ULID]*User
	sync.RWMutex
}

// NewGroup initializing a new group struct
// IMPORTANT: group kind is permanent and must never change
func NewGroup(kind GroupKind, key string, name string, parent *Group) (*Group, error) {
	if parent != nil {
		if err := parent.Validate(); err != nil {
			return nil, fmt.Errorf("NewGroup() parent validation failed: %s", err)
		}
	}

	g := &Group{
		ID:        util.NewULID(),
		Kind:      kind,
		Key:       strings.ToLower(key),
		Name:      name,
		members:   make(GroupMembers, 0),
		memberMap: make(map[ulid.ULID]*User),
	}

	if err := g.SetParent(parent); err != nil {
		return nil, err
	}

	return g, g.Validate()
}

// StringID returns short object info
func (g *Group) StringID() string {
	return fmt.Sprintf("%s(%s:%s:%s)", g.Kind, g.ID, g.Key, g.Name)
}

// Validate tells a group to perform self-check and return errors if something's wrong
func (g *Group) Validate() error {
	if g == nil {
		return ErrNilGroup
	}

	// checking for parent circulation
	if isCircuited, err := g.IsCircuited(); isCircuited || (err != nil) {
		if err != nil {
			return fmt.Errorf("%s validation failed: %s", g.Kind, err)
		}

		if isCircuited {
			return fmt.Errorf("%s validation failed: %s", g.Kind, ErrCircuitedParent)
		}
	}

	// general field validations
	if ok, err := govalidator.ValidateStruct(g); !ok || err != nil {
		return fmt.Errorf("%s validation failed: %s", g.Kind, err)
	}

	return nil
}

// IsCircuited tests whether the parents trace back to a nil
func (g *Group) IsCircuited() (bool, error) {
	if g.Parent == nil {
		return false, nil
	}

	// moving up a parent tree until nil is reached or the signs of circulation are found
	// TODO add checks to discover possible circulation before the timeout in case of a long parent trail
	p := g.Parent
	timeout := time.Now().Add(5 * time.Millisecond)
	for !time.Now().After(timeout) {
		if p == nil {
			// it's all good, reached a nil parent
			return false, nil
		}

		// next parent
		p = p.Parent
	}

	return false, ErrCircuitCheckTimeout
}

// SetDescription sets text description for this domain
func (g *Domain) SetDescription(desc string) error {
	// TODO: implement

	return nil
}

// SetParent assigning a parent group, could be nil
func (g *Group) SetParent(p *Group) error {
	// since parent could be nil thus it's kind is irrelevant
	if p != nil {
		// checking whether new parent already is set somewhere along the parenthood
		// by tracing backwards until a nil parent is met; at this point only a
		// requested parent is searched and not tested whether the relations
		// are circuited among themselves
		if pg := g.Parent; pg != nil {
			for {
				// no more parents, breaking
				if pg.Parent == nil {
					break
				}

				// testing equality by comparing each group's ID
				if pg.ID == p.ID {
					return ErrDuplicateParent
				}

				// moving on to a parent's parent
				pg = pg.Parent
			}
		}

		// group kind must be the same all the way back to the top
		if g.Kind != p.Kind {
			return ErrGroupKindMismatch
		}

		// ParentID is used to rebuild parent-child connections after
		// loading groups from the store
		g.ParentID = p.ID
	}

	// assingning a new parent
	g.Parent = p

	return nil
}

// Save this group to the store
func (g *Group) Save() error {
	if g.container.store == nil {
		return ErrNilGroupStore
	}

	if err := g.container.store.Put(g); err != nil {
		return fmt.Errorf("failed to store a group: %s", err)
	}

	return nil
}

// IsMember tests whether a given user belongs to a given group
func (g *Group) IsMember(u *User) bool {
	if u == nil {
		return false
	}

	g.RLock()
	defer g.RUnlock()

	if _, ok := g.memberMap[u.ID]; ok {
		return true
	}

	return false
}

func (g *Group) validateUser(u *User) error {
	if u == nil {
		return ErrNilUser
	}

	// this group must belong to some domain at this point
	if u.Domain == nil {
		return ErrNilUserDomain
	}

	return nil
}

// Register adding user to a group
func (g *Group) Register(u *User) error {
	if err := g.validateUser(u); err != nil {
		return err
	}

	// returning an error if this user already belongs to this group
	if g.IsMember(u) {
		return ErrAlreadyMember
	}

	// if store is set then storing new relation
	if g.container.store != nil {
		if err := g.container.store.PutRelation(g.Domain.ID, g.ID, u.ID); err != nil {
			return err
		}
	} else {
		log.Printf("WARNING: registering %s member to %s without storing\n", u.StringID(), g.StringID())
	}

	// updating runtime data
	g.Lock()
	g.members = append(g.members, u)
	g.memberMap[u.ID] = u
	g.Unlock()

	// updating group tracklist for this user
	if err := u.TrackGroup(g); err != nil {
		log.Printf("%s failed to track group(%s): %s", u.StringID(), g.StringID(), err)
	}

	return nil
}

// Unregister adding user to a group
func (g *Group) Unregister(u *User) error {
	if err := g.validateUser(u); err != nil {
		return err
	}

	// being consistent and returning an error for explicitness
	if !g.IsMember(u) {
		return ErrNotMember
	}

	if g.container.store != nil {
		// TODO: do not store relation if this user already belongs to this group
		// deleting a stored relation
		if err := g.container.store.DeleteRelation(g.Domain.ID, g.ID, u.ID); err != nil {
			return err
		}
	} else {
		log.Printf("WARNING: unregistering %s member to %s without storing\n", u.StringID(), g.StringID())
	}

	g.container.Lock()

	// removing group from the main slice
	for i, m := range g.members {
		if m.ID == u.ID {
			// deleting a group from the list
			g.members = append(g.members[0:i], g.members[i+1:]...)
			break
		}
	}

	// removing user from the group members
	delete(g.memberMap, u.ID)

	g.container.Unlock()

	// updating group tracklist for this user
	if err := u.UntrackGroup(g.ID); err != nil {
		log.Printf("%s failed to untrack group(%s): %s", u.StringID(), g.StringID(), err)
	}

	return nil
}
