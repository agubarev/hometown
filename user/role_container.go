package user

import (
	"github.com/oklog/ulid"
	"gitlab.com/agubarev/hometown/util"
)

// RoleContainer represents a role container and is responsible for all
// role-related operations within its scope
// TODO: add default roles which need not to be assigned
type RoleContainer struct {
	ID ulid.ULID `json:"id"`

	domain     *Domain
	roles      map[ulid.ULID]Role
	nameIndex  map[string]*Role
	userIndex  map[ulid.ULID][]*Role
	groupIndex map[ulid.ULID][]*Role
}

// NewRoleContainer initializing a new role container attached to domain
func NewRoleContainer(d *Domain) (*RoleContainer, error) {
	if d == nil {
		return nil, ErrNilDomain
	}

	c := &RoleContainer{
		ID:        util.NewULID(),
		domain:    d,
		roles:     make(map[ulid.ULID]Role),
		nameIndex: make(map[string]*Role),
		userIndex: make(map[ulid.ULID][]*Role),
	}

	return c, nil
}

// BelongsTo tests whether a given entity belongs to a given role
func (c *RoleContainer) BelongsTo(x interface{}, g *Role) bool {
	if x == nil {
		return false
	}

	switch x.(type) {
	case *User:
		if gs, ok := c.userIndex[x.(*User).ID]; ok {
			for _, fg := range gs {
				if fg.ID == g.ID {
					return true
				}
			}
		}
	}

	return false
}

// GetByID returns a group by ID
func (c *RoleContainer) GetByID(id ulid.ULID) (*Role, error) {
	if g, ok := c.roles[id]; ok {
		return &g, nil
	}

	return nil, ErrGroupNotFound
}

// GetByName returns a group by name
func (c *RoleContainer) GetByName(name string) (*Role, error) {
	if g, ok := c.nameIndex[name]; ok {
		return g, nil
	}

	return nil, ErrGroupNotFound
}

// GetByUser returns a slice of roles to which a given user belongs
func (c *RoleContainer) GetByUser(u *User) ([]*Role, error) {
	if u == nil {
		return nil, ErrNilUser
	}

	return c.userIndex[u.ID], nil
}
