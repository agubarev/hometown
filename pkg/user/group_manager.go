package user

import (
	"context"
	"errors"
	"fmt"
	"log"
	"sync"

	"go.uber.org/zap"
)

// errors
var (
	ErrNilDatabase            = errors.New("database is nil")
	ErrNothingChanged         = errors.New("nothing changed")
	ErrZeroID                 = errors.New("id is zero")
	ErrNonZeroID              = errors.New("id is not zero")
	ErrZeroMemberID           = errors.New("member id is zero")
	ErrInvalidMemberKind      = errors.New("member kind is invalid")
	ErrNilGroupStore          = errors.New("group store is nil")
	ErrGroupAlreadyRegistered = errors.New("group is already registered")
	ErrEmptyGroupName         = errors.New("empty group name")
	ErrDuplicateGroup         = errors.New("duplicate group")
	ErrDuplicateParent        = errors.New("duplicate parent")
	ErrDuplicateRelation      = errors.New("duplicate relation")
	ErrMemberNotEligible      = errors.New("member is not eligible for this operation")
	ErrGroupKindMismatch      = errors.New("group kinds mismatch")
	ErrInvalidKind            = errors.New("invalid group kind")
	ErrNotMember              = errors.New("member is not a member")
	ErrAlreadyMember          = errors.New("already a member")
	ErrCircuitedParent        = errors.New("circuited parenting")
	ErrCircuitCheckTimeout    = errors.New("circuit check timed out")
	ErrNilRole                = errors.New("role is nil")
	ErrNilGroup               = errors.New("group is nil")
	ErrNilGroupManager        = errors.New("group manager is nil")
	ErrGroupNotFound          = errors.New("group not found")
	ErrGroupKeyTaken          = errors.New("group key is already taken")
	ErrNilMember              = errors.New("member is nil")
)

// GroupList is a typed slice of groups to make sorting easier
type GroupList []Group

// userManager is a manager responsible for all operations within its scope
// TODO: add default groups which need not to be assigned
type GroupManager struct {
	groups map[uint32]Group
	keyMap map[TKey]uint32
	store  GroupStore
	logger *zap.Logger
	sync.RWMutex
}

// NewGroupManager initializing a new group manager
func NewGroupManager(s GroupStore) (*GroupManager, error) {
	if s == nil {
		log.Println("NewGroupManager: store is not set")
	}

	c := &GroupManager{
		groups: make(map[uint32]Group, 0),
		keyMap: make(map[TKey]uint32),
		store:  s,
	}

	return c, nil
}

// SetLogger assigns a logger for this manager
func (m *GroupManager) SetLogger(logger *zap.Logger) error {
	if logger != nil {
		logger = logger.Named("[group]")
	}

	m.logger = logger

	return nil
}

// Logger returns primary logger if is set, otherwise initializing and returning
func (m *GroupManager) Logger() *zap.Logger {
	if m.logger == nil {
		l, err := zap.NewDevelopment()
		if err != nil {
			panic(fmt.Errorf("failed to initialize group manager logger: %s", err))
		}

		m.logger = l
	}

	return m.logger
}

// Init initializes group manager
func (m *GroupManager) Init(ctx context.Context) error {
	if err := m.Validate(); err != nil {
		return err
	}

	l := m.Logger()

	s, err := m.Store()
	if err != nil {
		return err
	}

	// fetching all stored groups
	gs, err := s.FetchAllGroups(ctx)
	if err != nil {
		return err
	}

	// adding groups to a container
	for _, g := range gs {
		// adding member to container
		if err := m.AddGroup(ctx, g); err != nil {
			// just warning and moving forward
			l.Info(
				"Init() failed to add group to container",
				zap.Uint32("group_id", g.ID),
				zap.String("kind", g.Kind.String()),
				zap.ByteString("key", g.Key[:]),
				zap.Error(err),
			)
		}
	}

	return nil
}

// DistributeMembers injects given users into the manager, and linking
// them to their respective groups and roles
// NOTE: does not affect the store
func (m *GroupManager) DistributeMembers(ctx context.Context, members []uint32) error {
	if err := m.Validate(); err != nil {
		return err
	}

	l := m.Logger()

	s, err := m.Store()
	if err != nil {
		return err
	}

	// fetching all relations
	l.Info("fetching group userID relations")
	rs, err := s.FetchAllRelations(ctx)
	if err != nil {
		return err
	}

	// iterating over injected members
	l.Info("distributing members among their respective groups")
	for _, userID := range members {
		// checking whether this userID is listed
		// among fetched relations
		if gids, ok := rs[userID]; ok {
			// iterating over a slice of group IDs related to thismember
			for _, gid := range gids {
				// obtaining the respective group
				g, err := m.GroupByID(ctx, gid)
				if err != nil {
					// continuing even if an associated group is not found
					l.Info(
						"failed to apply group relation due to missing group",
						zap.Uint32("mid", userID),
						zap.Uint32("gid", gid),
					)
				}

				// linking userID to a group
				if err := g.LinkMember(ctx, userID); err != nil {
					l.Warn(
						"failed to link user to group",
						zap.Uint32("gid", gid),
						zap.Uint32("uid", userID),
						zap.Error(err),
					)
				}
			}
		}
	}

	return nil
}

// GroupStore returns store if set
func (m *GroupManager) Store() (GroupStore, error) {
	if m.store == nil {
		return nil, ErrNilGroupStore
	}

	return m.store, nil
}

// Validate this group manager
func (m *GroupManager) Validate() error {
	if m.groups == nil {
		return errors.New("group map is nil")
	}

	if m.keyMap == nil {
		return errors.New("key map is nil")
	}

	return nil
}

// Upsert creates new group
func (m *GroupManager) Create(ctx context.Context, kind GroupKind, key TKey, name TGroupName, parent Group) (g Group, err error) {
	// parent must be already created
	if parent.ID == 0 {
		return
	}

	// groups must be of the same kind
	if parent.Kind != kind {
		return g, ErrGroupKindMismatch
	}

	// initializing new group
	g, err = NewGroup(kind, key, name, parent)
	if err != nil {
		return g, fmt.Errorf("failed to initialize new group(%s): %s", key, err)
	}

	// checking whether there's already some group with such a key
	_, err = m.GroupByKey(ctx, key)
	if err != nil {
		// returning on unexpected error
		if err != ErrGroupNotFound {
			return g, err
		}
	} else {
		// no error means that the group key is already taken, thus canceling group creation
		return g, ErrGroupKeyTaken
	}

	// creating new group in the store
	g, err = m.store.Put(ctx, g)
	if err != nil {
		return g, err
	}

	// adding new group to manager's registry
	err = m.AddGroup(ctx, g)
	if err != nil {
		return g, err
	}

	return g, nil
}

// AddGroup adds group to the manager
func (m *GroupManager) AddGroup(ctx context.Context, g Group) error {
	if g.ID == 0 {
		return ErrZeroID
	}

	_, err := m.GroupByID(ctx, g.ID)
	if err != ErrGroupNotFound {
		return ErrGroupAlreadyRegistered
	}

	// linking group to this manager
	g.Lock()
	g.manager = m
	g.Unlock()

	// adding group to this manager's container
	m.Lock()
	m.groups[g.ID] = g
	m.keyMap[g.Key] = g.ID
	m.Unlock()

	return g.Init()
}

// RemoveGroup removing group from the manager
func (m *GroupManager) RemoveGroup(ctx context.Context, groupID uint32) error {
	// a bit pedantic but consistent, returning an error if group
	// is already registered within the manager
	g, err := m.GroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	// removing group from index maps and a main slice
	m.Lock()

	// clearing index maps
	delete(m.keyMap, g.Key)

	m.Unlock()

	// unlinking group from this manager
	g.Lock()
	g.manager = nil
	g.Unlock()

	return nil
}

// GroupList returns all groups inside a manager
func (m *GroupManager) List(kind GroupKind) (gs []Group) {
	gs = make([]Group, 0)

	for _, g := range m.groups {
		if g.Kind&kind != 0 {
			gs = append(gs, g)
		}
	}

	return gs
}

// GroupByID returns a group by ObjectID
func (m *GroupManager) GroupByID(ctx context.Context, id uint32) (g Group, err error) {
	if g, ok := m.groups[id]; ok {
		return g, nil
	}

	return g, ErrGroupNotFound
}

// PolicyByName returns a group by name
func (m *GroupManager) GroupByKey(ctx context.Context, key TKey) (g Group, err error) {
	m.RLock()
	g, ok := m.groups[m.keyMap[key]]
	m.RUnlock()

	if ok {
		return g, nil
	}

	return g, ErrGroupNotFound
}

// GroupsByUserID returns a slice of groups to which a given member belongs
func (m *GroupManager) GroupsByUserID(ctx context.Context, userID uint32, mask GroupKind) (gs []Group) {
	if userID == 0 {
		return gs
	}

	m.RLock()

	gs = make([]Group, 0)
	for _, g := range m.groups {
		if g.Kind&mask != 0 {
			if g.IsMember(ctx, userID) {
				gs = append(gs, g)
			}
		}
	}

	m.RUnlock()

	return gs
}

// LinkGroup tracking which groups this member is a member of
func (m *GroupManager) LinkGroup(ctx context.Context, g Group) error {
	if g.ID == 0 {
		return ErrZeroID
	}

	// safeguard in case the map is not initialized
	if m.groups == nil {
		m.groups = make(map[uint32]Group, 1)
	}

	m.groups[g.ID] = g

	return nil
}

// UnlinkGroup removing group from the tracklist
func (m *GroupManager) UnlinkGroup(ctx context.Context, groupID uint32) error {
	if m.groups == nil {
		return nil
	}

	delete(m.groups, groupID)

	return ErrGroupNotFound
}

// Groups to which the member belongs
func (m *GroupManager) Groups(ctx context.Context, mask GroupKind) []Group {
	if m.groups == nil {
		m.groups = make(map[uint32]Group, 0)
	}

	groups := make([]Group, 0)
	for _, g := range m.groups {
		if (g.Kind | mask) == mask {
			groups = append(groups, g)
		}
	}

	return groups
}
