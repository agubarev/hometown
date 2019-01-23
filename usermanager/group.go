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
	Kind         GroupKind     `json:"kind"`
	HasParent    bool          `json:"hasp"`
	ParentID     ulid.ULID     `json:"pid"`
	Parent       *Group        `json:"-"`
	Key          string        `json:"key" valid:"required,ascii"`
	Name         string        `json:"name" valid:"required"`
	Description  string        `json:"desc" valid:"optional,length(0|200)"`
	AccessPolicy *AccessPolicy `json:"-"`

	container *GroupContainer
	store     GroupStore
	members   GroupMembers
	memberMap map[ulid.ULID]*User
	sync.RWMutex
}

// NewGroup initializing a new group struct
// IMPORTANT: group kind is permanent and must never change
func NewGroup(kind GroupKind, key string, name string, parent *Group) (*Group, error) {
	g := &Group{
		ID:        util.NewULID(),
		Kind:      kind,
		Key:       strings.ToLower(key),
		Name:      name,
		members:   make(GroupMembers, 0),
		memberMap: make(map[ulid.ULID]*User),
	}

	if parent != nil {
		if err := parent.Validate(); err != nil {
			return nil, fmt.Errorf("NewGroup() parent validation failed: %s", err)
		}
	}

	if err := g.SetParent(parent); err != nil {
		return nil, err
	}

	return g, g.Validate()
}

// IDString returns short object info
func (g *Group) IDString() string {
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

// SetParent assigning a parent group, could be nil
func (g *Group) SetParent(p *Group) error {
	// since parent could be nil thus it's kind is irrelevant
	if p != nil {
		// checking whether new parent already is set somewhere along the parenthood
		// by tracing backwards until a nil parent is met; at this point only a
		// requested parent is searched and not tested whether the relations
		// are circuited among themselves
		pg := g.Parent
		for {
			// testing equality by comparing each group's ID
			if pg.ID == p.ID {
				return ErrDuplicateParent
			}

			// no more parents, breaking
			if pg.Parent == nil {
				break
			}

			// moving on to a parent's parent
			pg = pg.Parent
		}

		// group kind must be the same all the way back to the top
		if g.Kind != p.Kind {
			return ErrGroupKindMismatch
		}

		// ParentID is used to rebuild parent-child connections after
		// loading groups from the store
		// HasParent flags whether this group has a parent because
		// ULID can't be nil
		g.HasParent = true
		g.ParentID = p.ID
	} else {
		g.HasParent = false
	}

	// assingning a new parent
	g.Parent = p

	return nil
}

// Save this group to the store
func (g *Group) Save() error {
	if g.store == nil {
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

// AddMember adding user to a group
func (g *Group) AddMember(u *User) error {
	if u == nil {
		return ErrNilUser
	}

	if g.IsMember(u) {
		return ErrAlreadyMember
	}

	// if store is set then storing new relation
	if g.store != nil {
		if err := g.store.PutRelation(g.ID, u.ID); err != nil {
			return fmt.Errorf("AddMember(%s) failed to store relation: %s", u.ID, err)
		}
	} else {
		log.Printf("WARNING: AddMember() adding %s member to %s without storing\n", u.IDString(), g.IDString())
	}

	// updating runtime data
	g.Lock()
	g.members = append(g.members, u)
	g.memberMap[u.ID] = u
	g.Unlock()

	// updating group tracklist for this user
	if err := u.TrackGroup(g); err != nil {
		log.Printf("WARNING: AddMember() user failed to track group(%s): %s\n", g.IDString(), err)
	}

	return nil
}

// RemoveMember adding user to a group
func (g *Group) RemoveMember(u *User) error {
	if u == nil {
		return ErrNilUser
	}

	if g.IsMember(u) {
		return ErrNotMember
	}

	// deleting a stored relation
	if err := g.container.store.DeleteRelation(g.ID, u.ID); err != nil {
		return fmt.Errorf("RemoveMember() failed to delete a stored relation: %s", err)
	}

	g.container.Lock()
	defer g.container.Unlock()

	// removing a member
	var pos int
	for i, m := range g.members {
		if m.ID == u.ID {
			pos = i
			break
		}
	}

	// deleting a group from the list
	g.members = append(g.members[0:pos], g.members[pos+1:]...)

	// removing user from the index
	delete(g.memberMap, u.ID)

	// updating group tracklist for this user
	if err := u.UntrackGroup(g.ID); err != nil {
		log.Printf("WARNING: RemoveMember() user failed to untrack group(%s): %s\n", g.IDString(), err)
	}

	return nil
}
