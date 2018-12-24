package user

import (
	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

type groupMembers []*User

// Group represents a user group
type Group struct {
	ID     ulid.ULID `json:"id"`
	Parent *Group    `json:"parent"`
	Name   string    `json:"name"`

	container   *GroupContainer
	members     groupMembers
	memberIndex map[ulid.ULID]bool
}

// NewGroup initializing a new group struct
func NewGroup(name string, parent *Group, container *GroupContainer) (*Group, error) {
	if container == nil {
		return nil, ErrNilContainer
	}

	g := &Group{
		ID:          util.NewULID(),
		Parent:      parent,
		Name:        name,
		container:   container,
		members:     make(groupMembers, 0),
		memberIndex: make(map[ulid.ULID]bool),
	}

	return g, nil
}

// IsMember tests whether a given user belongs to a given group
func (g *Group) IsMember(u *User) bool {
	if u == nil {
		return false
	}

	g.container.RLock()
	defer g.container.RUnlock()

	if _, ok := g.memberIndex[u.ID]; ok {
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
	g.memberIndex[u.ID] = true
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
	delete(g.memberIndex, u.ID)

	return nil
}
