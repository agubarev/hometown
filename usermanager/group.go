package user

import (
	"fmt"
	"strings"

	"github.com/asaskevich/govalidator"

	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

// GroupMembers is a slice of users, typed for the ease of sorting
type GroupMembers []*User

// GroupKind designates a group kind i.e. Group, Role etc...
type GroupKind int

// group kinds
const (
	GKSystemGroup GroupKind = 1 << iota
	GKStandardGroup
	GKRole
)

// Group represents a user group
// TODO custom JSON marshalling
type Group struct {
	ID           ulid.ULID     `json:"id"`
	Kind         GroupKind     `json:"kind"`
	IsDefault    bool          `json:"is_default"`
	Parent       *Group        `json:"-"`
	Key          string        `json:"key" valid:"required,alphanum"`
	Name         string        `json:"name" valid:"required"`
	Description  string        `json:"description" valid:"optional,length(0|200)"`
	AccessPolicy *AccessPolicy `json:"-"`

	container *GroupContainer
	members   GroupMembers
	memberMap map[ulid.ULID]*User
}

// NewGroup initializing a new group struct
func NewGroup(kind GroupKind, key string, name string, parent *Group) (*Group, error) {
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

// Validate tells a group to perform self-check and return errors if something's wrong
func (g *Group) Validate() error {
	if g == nil {
		return ErrNilGroup
	}

	if ok, err := govalidator.ValidateStruct(g); !ok || err != nil {
		return fmt.Errorf("group [%s:%s] validation failed: %s", g.ID, err)
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
