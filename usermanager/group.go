package usermanager

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/asaskevich/govalidator"
)

// GroupMembers is a slice of users, typed for the ease of sorting
type GroupMembers []*User

// GroupKind designates a group kind i.e. Group, Role etc...
type GroupKind int8

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
	GKAll GroupKind = ^GroupKind(0)
)

// Group represents a user group
// TODO custom JSON marshalling
// TODO add mutex and store to the group; store should be set implicitly upon addition to the container
type Group struct {
	ID          int64     `json:"id" db:"id"`
	Kind        GroupKind `json:"k" db:"kind"`
	Key         string    `json:"ky" db:"key" valid:"required,ascii"`
	Name        string    `json:"n" db:"name" valid:"required"`
	Description string    `json:"dsc" db:"description" valid:"optional,length(0|200)"`

	// these fields are basically just for the storage
	ParentID int64 `json:"pid" db:"parent_id"`

	parent  *Group
	manager *GroupManager
	members GroupMembers
	idMap   map[int64]*User
	ap      *AccessPolicy
	logger  *zap.Logger
	sync.RWMutex
}

// Parent returns parent group or nil
func (g *Group) Parent() *Group {
	return g.parent
}

// NewGroup initializing a new group struct
// ! IMPORTANT: group kind is permanent and must never change
func NewGroup(kind GroupKind, key string, name string, parent *Group) (*Group, error) {
	if parent != nil {
		if err := parent.Validate(); err != nil {
			return nil, fmt.Errorf("NewGroup() parent validation failed: %s", err)
		}
	}

	g := &Group{
		ID:      0,
		Kind:    kind,
		Key:     strings.ToLower(key),
		Name:    name,
		members: make(GroupMembers, 0),
		idMap:   make(map[int64]*User),
	}

	if err := g.SetParent(parent); err != nil {
		return nil, err
	}

	return g, g.Validate()
}

// SetLogger assigns a logger for this group
func (g *Group) SetLogger(logger *zap.Logger) error {
	if logger != nil {
		logger = logger.Named(g.StringID())

		logger.With(
			zap.Int64("gid", g.ID),
			zap.String("key", g.Key),
			zap.String("kind", g.Kind.String()),
			zap.String("name", g.Name),
		)
	}

	g.logger = logger

	return nil
}

// Logger returns group logger
func (g *Group) Logger() *zap.Logger {
	if g.logger == nil {
		l, err := zap.NewDevelopment()
		if err != nil {
			panic(fmt.Errorf("failed to initialize group logger: %s", err))
		}

		g.logger = l
	}

	return g.logger
}

// StringID returns short object info
func (g *Group) StringID() string {
	return fmt.Sprintf("%s(%d:%s:%s)", g.Kind, g.ID, g.Key, g.Name)
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

// Init initializes current group
// NOTE: used after loading a group from the store and before
// adding/distributing any members
func (g *Group) Init() error {
	// initializing necessary fields (in case they're not)
	if g.members == nil {
		g.members = make(GroupMembers, 0)
	}

	if g.idMap == nil {
		g.idMap = make(map[int64]*User)
	}

	// using manager's logger if it's set
	if g.manager != nil && g.manager.logger != nil {
		g.logger = g.manager.logger.Named(g.StringID())

		g.logger.With(
			zap.Int64("gid", g.ID),
			zap.String("key", g.Key),
			zap.String("kind", g.Kind.String()),
			zap.String("name", g.Name),
		)
	}

	return nil
}

// IsCircuited tests whether the parents trace back to a nil
func (g *Group) IsCircuited() (bool, error) {
	if g.Parent() == nil {
		return false, nil
	}

	// moving up a parent tree until nil is reached or the signs of circulation are found
	// TODO add checks to discover possible circulation before the timeout in case of a long parent trail
	p := g.Parent()
	timeout := time.Now().Add(5 * time.Millisecond)
	for !time.Now().After(timeout) {
		if p == nil {
			// it's all good, reached a nil parent
			return false, nil
		}

		// next parent
		p = p.Parent()
	}

	return false, ErrCircuitCheckTimeout
}

// SetDescription sets text description for this group
func (g *Group) SetDescription(text string) error {
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
		if pg := g.Parent(); pg != nil {
			for {
				// no more parents, breaking
				if pg.Parent() == nil {
					break
				}

				// testing equality by comparing each group's ID
				if pg.ID == p.ID {
					return ErrDuplicateParent
				}

				// moving on to a parent's parent
				pg = g.Parent()
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
	g.parent = p

	return nil
}

// Save this group to the store
func (g *Group) Save() error {
	if g.manager.store == nil {
		return ErrNilGroupStore
	}

	// saving itself
	newGroup, err := g.manager.store.Put(g)
	if err != nil {
		return fmt.Errorf("failed to store a group: %s", err)
	}

	// replacing itself with retrieved group contents
	// *g = *newGroup
	g.ID = newGroup.ID

	return nil
}

// IsMember tests whether a given user belongs to a given group
func (g *Group) IsMember(u *User) bool {
	if u == nil {
		return false
	}

	g.RLock()
	_, ok := g.idMap[u.ID]
	g.RUnlock()

	if ok {
		return true
	}

	return false
}

// AddMember adding user to a group
// NOTE: storing relation only if group has a store set is implicit and should at least
// log/print about the occurrence
func (g *Group) AddMember(u *User) error {
	logger := g.Logger()
	defer logger.Sync()

	if u == nil {
		return ErrNilUser
	}

	// to proceed, the user must be previously properly registered,
	// and persisted to the store
	ok, err := u.IsRegisteredAndStored()
	if err != nil {
		return err
	}

	if !ok {
		return ErrUserNotEligible
	}

	// adding user to this group members
	if err := g.LinkMember(u); err != nil {
		return err
	}

	if g.manager != nil {
		if g.manager.store != nil {
			logger.Info("storing group user relation", zap.Int64("gid", g.ID), zap.Int64("uid", u.ID))

			// if container is set, then storing group user relation
			if err = g.manager.store.PutRelation(g.ID, u.ID); err != nil {
				logger.Info("failed to store group user relation", zap.Int64("gid", g.ID), zap.Int64("uid", u.ID), zap.Error(err))
				return err
			}
		} else {
			logger.Info("removing member user from group while store is not set", zap.Int64("gid", g.ID), zap.Int64("uid", u.ID), zap.Error(err))
		}
	}

	return nil
}

// RemoveMember removes user from a group
func (g *Group) RemoveMember(u *User) error {
	l := g.logger
	defer l.Sync()

	if u == nil {
		return ErrNilUser
	}

	// removing user from group members
	if err := g.LinkMember(u); err != nil {
		return err
	}

	if g.manager.store != nil {
		l.Info("deleting group user relation", zap.Int64("gid", g.ID), zap.Int64("uid", u.ID))

		// deleting a stored relation
		if err := g.manager.store.DeleteRelation(g.ID, u.ID); err != nil {
			return err
		}
	} else {
		l.Info("removing member user from group while store is not set", zap.Int64("gid", g.ID), zap.Int64("uid", u.ID))
	}

	return nil
}

// LinkMember adds a user to the group members
// NOTE: does not affect the store
func (g *Group) LinkMember(u *User) error {
	if g.IsMember(u) {
		return ErrAlreadyMember
	}

	g.Lock()
	g.members = append(g.members, u)
	g.idMap[u.ID] = u
	g.Unlock()

	// creating a backlink
	if err := u.LinkGroup(g); err != nil {
		return fmt.Errorf("failed to link group to the user: %s", err)
	}

	return nil
}

// UnlinkMember removes a user from the group members
// NOTE: does not affect the store
func (g *Group) UnlinkMember(u *User) error {
	if !g.IsMember(u) {
		return ErrNotMember
	}

	g.Lock()

	// removing group from the main slice
	for i, m := range g.members {
		if m.ID == u.ID {
			// deleting a group from the list
			g.members = append(g.members[0:i], g.members[i+1:]...)
			break
		}
	}

	// removing user from the group members
	delete(g.idMap, u.ID)

	g.Unlock()

	// creating a backlink
	if err := u.UnlinkGroup(g); err != nil {
		return fmt.Errorf("failed to link group to the user: %s", err)
	}

	return nil
}
