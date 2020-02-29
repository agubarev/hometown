package user

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/pkg/errors"
	"go.uber.org/zap"

	"github.com/asaskevich/govalidator"
)

type (
	TGroupKey  = [32]byte
	TGroupName = [256]byte
)

// ObjectKind designates a group kind i.e. Group, Role etc...
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
	GKGroup GroupKind = 1 << iota
	GKRole
	GKAll = ^GroupKind(0)
)

// Group represents a member group
// TODO custom JSON marshalling
// TODO add mutex and store to the group; store should be set implicitly upon addition to the container
type Group struct {
	ID          int64     `db:"id" json:"id"`
	Kind        GroupKind `db:"kind" json:"kind"`
	Key         string    `db:"key" json:"key" valid:"required,ascii"`
	Name        string    `db:"name" json:"name"`
	Description string    `db:"description" json:"description"`

	// these fields are basically just for the storage
	ParentID int64 `db:"parent_id" json:"parent_id"`

	manager *GroupManager
	members map[int64][]int64
	ap      *AccessPolicy
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

// SetLogger assigns a logger for this group
func (g *Group) SetLogger(logger *zap.Logger) error {
	if logger != nil {
		logger = logger.Named(g.StringID())

		logger.With(
			zap.Int64("id", g.ID),
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
			panic(errors.Wrap(err, "failed to initialize group logger"))
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
		g.members = make(map[int64][]int64, 0)
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

// SetParentID assigning a parent group, could be nil
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

					// testing equality by comparing each group's ObjectID
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

	// assingning a new parent ObjectID
	g.ParentID = p.ID

	return nil
}

// UpdatePolicy this group to the store
func (g *Group) Save(ctx context.Context) error {
	if g.manager.store == nil {
		return ErrNilGroupStore
	}

	// saving itself
	newGroup, err := g.manager.store.UpsertGroup(ctx, *g)
	if err != nil {
		return fmt.Errorf("failed to store a group: %s", err)
	}

	// replacing itself with retrieved group contents
	// *g = *newGroup
	g.ID = newGroup.ID

	return nil
}

// IsMember tests whether a given member belongs to a given group
func (g *Group) IsMember(ctx context.Context, userID int64) bool {
	if userID == 0 {
		return false
	}

	// searching userID if group slice is not nil
	if members := g.members[g.ID]; members != nil {
		isFound := false
		g.RLock()
		for _, memberID := range members {
			if memberID == userID {
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
func (g *Group) AddMember(ctx context.Context, userID int64) (err error) {
	// to proceed, the userID must be previously properly registered,
	// and persisted to the store
	if userID == 0 {
		return ErrZeroUserID
	}

	// adding userID to this group members
	if err = g.LinkMember(ctx, userID); err != nil {
		return err
	}

	logger := g.Logger()

	if g.manager != nil {
		if g.manager.store != nil {
			logger.Info("storing group userID relation", zap.Int64("gid", g.ID), zap.Int64("uid", userID))

			// if container is set, then storing group userID relation
			if err = g.manager.store.PutRelation(ctx, g.ID, userID); err != nil {
				logger.Info("failed to store group userID relation", zap.Int64("gid", g.ID), zap.Int64("uid", userID), zap.Error(err))
				return err
			}
		} else {
			logger.Info("removing user ID from group while store is not set", zap.Int64("gid", g.ID), zap.Int64("uid", userID), zap.Error(err))
		}
	}

	return nil
}

// RemoveMember removes member from a group
func (g *Group) RemoveMember(ctx context.Context, userID int64) (err error) {
	if userID == 0 {
		return ErrZeroUserID
	}

	// removing userID from group members
	if err = g.UnlinkMember(ctx, userID); err != nil {
		return err
	}

	l := g.Logger()

	if g.manager.store != nil {
		l.Info("deleting group user relation", zap.Int64("gid", g.ID), zap.Int64("uid", userID))

		// deleting a stored relation
		if err := g.manager.store.DeleteRelation(ctx, g.ID, userID); err != nil {
			return err
		}
	} else {
		l.Info("removing user ID from group while store is not set", zap.Int64("gid", g.ID), zap.Int64("uid", userID))
	}

	return nil
}

// LinkMember adds a member to the group members
// NOTE: does not affect the store
func (g *Group) LinkMember(ctx context.Context, userID int64) (err error) {
	if userID == 0 {
		return ErrZeroUserID
	}

	if g.IsMember(ctx, userID) {
		return ErrAlreadyMember
	}

	g.Lock()

	// initializing new slice if it's nil
	// NOTE: mask layout is: first byte = member kind (uint8), last 4 bytes = member id (int64)
	if g.members[g.ID] == nil {
		g.members[g.ID] = []int64{userID}
	} else {
		// appending relation mask to a group slice
		g.members[g.ID] = append(g.members[g.ID], userID)
	}

	g.Unlock()

	return nil
}

// UnlinkMember removes a member from the group members
// NOTE: does not affect the store
func (g *Group) UnlinkMember(ctx context.Context, userID int64) (err error) {
	if userID == 0 {
		return ErrZeroUserID
	}

	// removing group from the main slice
	if members := g.members[g.ID]; members != nil {
		g.Lock()
		for i, memberID := range members {
			if memberID == userID {
				// deleting member from the list
				g.members[g.ID] = append(g.members[g.ID][0:i], g.members[g.ID][i+1:]...)
				break
			}
		}
		g.Unlock()
	}

	return nil
}
