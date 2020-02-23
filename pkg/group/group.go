package group

import (
	"bytes"
	"fmt"
	"sync"
	"time"

	"github.com/agubarev/hometown/pkg/accesspolicy"
	"go.uber.org/zap"

	"github.com/asaskevich/govalidator"
)

// Member represents a group member contract
type Member interface {
	GroupMemberID() uint32
	GroupMemberKind() uint8
}

// recognized member kinds
const (
	MKUser uint8 = 1 << iota
	MKOther
)

type (
	TKey  = [64]byte
	TName = [256]byte
)

// GroupMemberKind designates a group kind i.e. Group, Role etc...
type Kind uint8

func (k Kind) String() string {
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
	GKGroup Kind = 1 << (iota - Kind(1))
	GKRole
	GKAll = ^Kind(0)
)

// Group represents a member group
// TODO custom JSON marshalling
// TODO add mutex and store to the group; store should be set implicitly upon addition to the container
type Group struct {
	ID          uint32 `db:"id" json:"id"`
	Kind        Kind   `db:"kind" json:"kind"`
	Key         TKey   `db:"key" json:"key" valid:"required,ascii"`
	Name        TName  `db:"name" json:"name"`
	Description []byte `db:"description" json:"description"`

	// these fields are basically just for the storage
	ParentID uint32 `db:"parent_id" json:"parent_id"`

	manager *Manager
	members map[uint32][]uint64
	ap      *accesspolicy.AccessPolicy
	logger  *zap.Logger
	sync.RWMutex
}

// Parent returns parent group or nil
func (g *Group) Parent() Group {
	g.RLock()
	p := g.manager.groups[g.ID]
	g.RUnlock()

	return p
}

// NewGroup initializing a new group struct
// ! IMPORTANT: group kind is permanent and must never change
func NewGroup(kind Kind, key TKey, name TName, parent Group) (g Group, err error) {
	if parent.ID != 0 {
		if err := parent.Validate(); err != nil {
			return g, fmt.Errorf("NewGroup() parent validation failed: %s", err)
		}
	}

	g = Group{
		Kind:    kind,
		Key:     bytes.ToLower(key[:]),
		Name:    bytes.TrimSpace(name[:]),
		members: make(map[uint32]uint64, 0),
	}

	if err := g.SetParent(parent); err != nil {
		return g, err
	}

	return g, g.Validate()
}

// SetLogger assigns a logger for this group
func (g *Group) SetLogger(logger *zap.Logger) error {
	if logger != nil {
		logger = logger.Named(g.StringID())

		logger.With(
			zap.Uint32("id", g.ID),
			zap.ByteString("key", g.Key[:]),
			zap.String("kind", g.Kind.String()),
			zap.ByteString("name", g.Name[:]),
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
		g.members = make(map[uint32]uint64, 0)
	}

	// using manager's logger if it's set
	if g.manager != nil && g.manager.logger != nil {
		g.logger = g.manager.logger.Named(g.StringID())

		g.logger.With(
			zap.Uint32("gid", g.ID),
			zap.ByteString("key", g.Key[:]),
			zap.String("kind", g.Kind.String()),
			zap.ByteString("name", g.Name[:]),
		)
	}

	return nil
}

// IsCircuited tests whether the parents trace back to a nil
func (g *Group) IsCircuited() (bool, error) {
	if g.ParentID == 0 {
		return false, nil
	}

	// moving up a parent tree until nil is reached or the signs of circulation are found
	// TODO add checks to discover possible circulation before the timeout in case of a long parent trail
	p := g.Parent()
	timeout := time.Now().Add(5 * time.Millisecond)
	for !time.Now().After(timeout) {
		if p.ParentID == 0 {
			// it's all good, reached a no-parent
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
func (g *Group) SetParent(p Group) error {
	// since parent could be nil thus it's kind is irrelevant
	if p.ID != 0 {
		// checking whether new parent already is set somewhere along the parenthood
		// by tracing backwards until a no-parent is met; at this point only a
		// requested parent is searched and not tested whether the relations
		// are circuited among themselves
		if p.ParentID != 0 {
			if pg := g.Parent(); pg.ID != 0 {
				for {
					// no more parents, breaking
					if pg.ParentID == 0 {
						break
					}

					// testing equality by comparing each group's GroupMemberID
					if pg.ID == p.ID {
						return ErrDuplicateParent
					}

					// moving on to a parent's parent
					pg = g.Parent()
				}
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

	// assingning a new parent GroupMemberID
	g.ParentID = p.ID

	return nil
}

// UpdateAccessPolicy this group to the store
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

// IsValidMember checks whether a given member is valid for this group
func (g *Group) IsValidMember(member Member) error {
	if member.GroupMemberID() == 0 {
		return ErrZeroMemberID
	}

	if member.GroupMemberKind() == 0 {
		return ErrInvalidMemberKind
	}

	return nil
}

// IsMember tests whether a given member belongs to a given group
func (g *Group) IsMember(member Member) bool {
	if member == nil {
		return false
	}

	// searching member if group slice is not nil
	if mlist := g.members[g.ID]; mlist != nil {
		// wanted mask
		want := (uint64(member.GroupMemberID()) << 8) | uint64(member.GroupMemberKind())

		isFound := false
		g.RLock()
		for _, got := range mlist {
			if got == want {
				isFound = true
				break
			}
		}
		g.RUnlock()

		return isFound
	}

	return false
}

// AddMember adding member to a group
// NOTE: storing relation only if group has a store set is implicit and should at least
// log/print about the occurrence
func (g *Group) AddMember(member Member) (err error) {
	logger := g.Logger()

	// to proceed, the member must be previously properly registered,
	// and persisted to the store
	if err = g.IsValidMember(member); err != nil {
		return err
	}

	// adding member to this group members
	if err = g.LinkMember(member); err != nil {
		return err
	}

	if g.manager != nil {
		if g.manager.store != nil {
			logger.Info("storing group member relation", zap.Uint32("gid", g.ID), zap.Uint32("mid", member.GroupMemberID()))

			// if container is set, then storing group member relation
			if err = g.manager.store.PutRelation(g.ID, member.GroupMemberID()); err != nil {
				logger.Info("failed to store group member relation", zap.Uint32("gid", g.ID), zap.Uint32("mid", member.GroupMemberID()), zap.Error(err))
				return err
			}
		} else {
			logger.Info("removing member member from group while store is not set", zap.Uint32("gid", g.ID), zap.Uint32("mid", member.GroupMemberID()), zap.Error(err))
		}
	}

	return nil
}

// RemoveMember removes member from a group
func (g *Group) RemoveMember(member Member) (err error) {
	l := g.logger

	if err = g.IsValidMember(member); err != nil {
		return err
	}

	// removing member from group members
	if err = g.UnlinkMember(member); err != nil {
		return err
	}

	if g.manager.store != nil {
		l.Info("deleting group member relation", zap.Uint32("gid", g.ID), zap.Uint32("mid", member.GroupMemberID()))

		// deleting a stored relation
		if err := g.manager.store.DeleteRelation(g.ID, member.GroupMemberID()); err != nil {
			return err
		}
	} else {
		l.Info("removing member member from group while store is not set", zap.Uint32("gid", g.ID), zap.Uint32("mid", member.GroupMemberID()))
	}

	return nil
}

// LinkMember adds a member to the group members
// NOTE: does not affect the store
func (g *Group) LinkMember(m Member) (err error) {
	if err = g.IsValidMember(m); err != nil {
		return err
	}

	if g.IsMember(m) {
		return ErrAlreadyMember
	}

	g.Lock()

	// initializing new slice if it's nil
	// NOTE: mask layout is: first byte = member kind (uint8), last 4 bytes = member id (uint32)
	if g.members[g.ID] == nil {
		g.members[g.ID] = []uint64{(uint64(m.GroupMemberID()) << 8) | uint64(m.GroupMemberKind())}
	} else {
		// appending relation mask to a group slice
		g.members[g.ID] = append(g.members[g.ID], (uint64(m.GroupMemberID())<<8)|uint64(m.GroupMemberKind()))
	}

	g.Unlock()

	return nil
}

// UnlinkMember removes a member from the group members
// NOTE: does not affect the store
func (g *Group) UnlinkMember(member Member) (err error) {
	if !g.IsMember(member) {
		return ErrNotMember
	}

	g.Lock()

	// removing group from the main slice
	for i, m := range g.members {
		if m.ID() == member.GroupMemberID {
			// deleting a group from the list
			g.members = append(g.members[0:i], g.members[i+1:]...)
			break
		}
	}

	// removing member from the group members
	delete(g.idMap, member.GroupMemberID())

	g.Unlock()

	// creating a backlink
	if err = member.UnlinkGroup(g); err != nil {
		return fmt.Errorf("failed to link group to the member: %s", err)
	}

	return nil
}
