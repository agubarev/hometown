package usermanager

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"

	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

// GroupMembers is a slice of users, typed for the ease of sorting
type GroupMembers []*User

// GroupKind designates a group kind i.e. Group, Role etc...
type GroupKind string

// group kinds
const (
	GKGroup GroupKind = "group"
	GKRole  GroupKind = "role"
)

// Group represents a user group
// TODO custom JSON marshalling
type Group struct {
	ID           ulid.ULID     `json:"id"`
	Kind         GroupKind     `json:"kind"`
	IsDefault    bool          `json:"is_default"`
	Parent       *Group        `json:"-"`
	Key          string        `json:"key" valid:"required,ascii"`
	Name         string        `json:"name" valid:"required"`
	Description  string        `json:"description" valid:"optional,length(0|200)"`
	AccessPolicy *AccessPolicy `json:"-"`

	container *GroupContainer
	members   GroupMembers
	memberMap map[ulid.ULID]*User
}

// NewGroup initializing a new group struct
func NewGroup(kind GroupKind, key string, name string, parent *Group) (*Group, error) {
	if parent != nil {
		if err := parent.Validate(); err != nil {
			return nil, fmt.Errorf("NewGroup() parent validation failed: %s", err)
		}

		if isCircular, err := parent.IsCircular(); isCircular || (err != nil) {
			if err != nil {
				return nil, err
			}

			if isCircular {
				return nil, ErrCircularParent
			}
		}
	}

	g := &Group{
		ID:        util.NewULID(),
		Parent:    parent,
		Key:       strings.ToLower(key),
		Name:      name,
		members:   make(GroupMembers, 0),
		memberMap: make(map[ulid.ULID]*User),
	}

	return g, g.Validate()
}

func (g *Group) String() string {
	return fmt.Sprintf("%s[%s:%s:%s]", g.KindString(), g.ID, g.Key, g.Name)
}

// KindString returns a group kind as a string
func (g *Group) KindString() string {
	return string(g.Kind)
}

// Validate tells a group to perform self-check and return errors if something's wrong
// TODO check for circular parenting
func (g *Group) Validate() error {
	if g == nil {
		return ErrNilGroup
	}

	if ok, err := govalidator.ValidateStruct(g); !ok || err != nil {
		return fmt.Errorf("%s validation failed: %s", g, err)
	}

	return nil
}

// IsCircular tests whether the parents trace back to a nil
func (g *Group) IsCircular() (bool, error) {
	if g.Parent == nil {
		return false, nil
	}

	// moving up a parent tree until nil is reached or the signs of circulation are found
	// TODO add checks to discover possible circulation before the timeout in case of a long parent trail
	p := g.Parent
	timeout := time.Now().Add(100 * time.Millisecond)
	for !time.Now().After(timeout) {
		if p == nil {
			// it's all good, reached a nil parent
			return false, nil
		}

		// next parent
		p = p.Parent
	}

	return false, ErrCircularCheckTimeout
}

// HasParent tracing parents back to tell whether a given group
// is already among this group's parents
func (g *Group) HasParent(p *Group) bool {
	// nil is not considered as a parent even though the top of a parent tree is nil
	if p == nil {
		return false
	}

	// tracing backwards until a nil parent is met; at this point only a
	// requested parent is searched and not tested whether the relations
	// are circulated among themselves
	pg := g.Parent
	for {
		// testing equality by comparing each group's ID
		if pg.ID == p.ID {
			return true
		}

		// no more parents, returning
		if pg.Parent == nil {
			return false
		}

		// moving on to a parent's parent
		pg = pg.Parent
	}
}

// SetParent assigning a parent group, could be nil
func (g *Group) SetParent(p *Group) error {
	// checking whether new parent already is set somewhere along the parenthood
	if g.HasParent(p) {
		return ErrDuplicateParent
	}

	// group kind must be the same all the way back to the top
	if g.Kind != p.Kind {
		return ErrGroupKindMismatch
	}

	// assingning a new parent
	g.Parent = p

	return nil
}

// Persist this group to storage
func (g *Group) Persist() error {
	if err := g.container.store.PutGroup(context.Background(), g); err != nil {
		return fmt.Errorf("Persist() failed to store a group: %s", err)
	}

	return nil
}

// IsMember tests whether a given user belongs to a given group
func (g *Group) IsMember(u *User) bool {
	if u == nil {
		return false
	}

	g.container.RLock()
	defer g.container.RUnlock()

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

	// storing the relation
	if err := g.container.store.PutGroupRelation(context.Background(), g, u); err != nil {
		return fmt.Errorf("AddMember() failed to store relation: %s", err)
	}

	// updating runtime data
	g.container.Lock()
	g.members = append(g.members, u)
	g.memberMap[u.ID] = u
	g.container.Unlock()

	return nil
}

// RemoveMember adding user to a group
func (g *Group) RemoveMember(u *User) error {
	if u == nil {
		return ErrNilUser
	}

	if g.IsMember(u) {
		return ErrUserNotFound
	}

	// deleting a stored relation
	if err := g.container.store.DeleteGroupRelation(context.Background(), g.ID, u.ID); err != nil {
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

	return nil
}
