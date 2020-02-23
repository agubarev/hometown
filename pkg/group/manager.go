package group

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
	ErrRelationNotFound       = errors.New("relation not found")
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
)

// List is a typed slice of groups to make sorting easier
type List []Group

// userManager is a manager responsible for all operations within its scope
// TODO: add default groups which need not to be assigned
type Manager struct {
	groups map[uint32]Group
	keyMap map[TKey]uint32
	store  Store
	logger *zap.Logger
	sync.RWMutex
}

// NewGroupManager initializing a new group manager
func NewGroupManager(s Store) (*Manager, error) {
	if s == nil {
		log.Println("NewGroupManager: store is not set")
	}

	c := &Manager{
		groups: make(map[uint32]Group, 0),
		keyMap: make(map[TKey]uint32),
		store:  s,
	}

	return c, nil
}

// SetLogger assigns a logger for this manager
func (m *Manager) SetLogger(logger *zap.Logger) error {
	if logger != nil {
		logger = logger.Named("[group]")
	}

	m.logger = logger

	return nil
}

// Logger returns primary logger if is set, otherwise initializing and returning
func (m *Manager) Logger() *zap.Logger {
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
func (m *Manager) Init(ctx context.Context) error {
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
		if err := m.AddGroup(g); err != nil {
			// just warning and moving forward
			l.Info(
				"Init() failed to add group to container",
				zap.Int("group_id", g.ID),
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
func (m *Manager) DistributeMembers(ctx context.Context, members []Member) error {
	if err := m.Validate(); err != nil {
		return err
	}

	l := m.Logger()

	s, err := m.Store()
	if err != nil {
		return err
	}

	// fetching all relations
	l.Info("fetching group member relations")
	rs, err := s.FetchAllRelations(ctx)
	if err != nil {
		return err
	}

	// iterating over injected members
	l.Info("distributing members among their respective groups")
	for _, member := range members {
		// checking whether this member is listed
		// among fetched relations
		if gids, ok := rs[member.GroupMemberID()]; ok {
			// iterating over a slice of group IDs related to thismember
			for _, gid := range gids {
				// obtaining the respective group
				g, err := m.GroupByID(gid)
				if err != nil {
					// continuing even if an associated group is not found
					l.Info(
						"failed to apply group relation due to missing group",
						zap.Int("mid", member.GroupMemberID()),
						zap.Int("gid", gid),
					)
				}

				// linking member to a group
				g.LinkMember(member)
			}
		}
	}

	return nil
}

// Store returns store if set
func (m *Manager) Store() (Store, error) {
	if m.store == nil {
		return nil, ErrNilGroupStore
	}

	return m.store, nil
}

// Validate this group manager
func (m *Manager) Validate() error {
	if m.groups == nil {
		return errors.New("group map is nil")
	}

	if m.keyMap == nil {
		return errors.New("key map is nil")
	}

	return nil
}

// Upsert creates new group
func (m *Manager) Create(kind Kind, key string, name string, parent Group) (g Group, err error) {
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
		return nil, fmt.Errorf("failed to initialize new group(%s): %s", key, err)
	}

	// checking whether there's already some group with such a key
	_, err = m.GroupByKey(key)
	if err != nil {
		// returning on unexpected error
		if err != ErrGroupNotFound {
			return nil, err
		}
	} else {
		// no error means that the group key is already taken, thus canceling group creation
		return nil, ErrGroupKeyTaken
	}

	// creating new group in the store
	g, err = m.store.Put(g)
	if err != nil {
		return nil, err
	}

	// adding new group to manager's registry
	err = m.AddGroup(g)
	if err != nil {
		return nil, err
	}

	return g, nil
}

// AddGroup adds group to the manager
func (m *Manager) AddGroup(g Group) error {
	if g.ID == 0 {
		return ErrZeroID
	}

	_, err := m.GroupByID(g.ID)
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
	m.idMap[g.ID] = g
	m.keyMap[g.Key] = g
	m.Unlock()

	return g.Init()
}

// RemoveGroup removing group from the manager
func (m *Manager) RemoveGroup(id int) error {
	// a bit pedantic but consistent, returning an error if group
	// is already registered within the manager
	g, err := m.GroupByID(id)
	if err != nil {
		return err
	}

	// removing group from index maps and a main slice
	m.Lock()

	// clearing index maps
	delete(m.idMap, g.ID)
	delete(m.keyMap, g.Key)

	// removing the actual item
	for i, fg := range m.groups {
		if g.ID == fg.ID {
			m.groups = append(m.groups[:i], m.groups[i+1:]...)
			break
		}
	}

	m.Unlock()

	// unlinking group from this manager
	g.Lock()
	g.manager = nil
	g.Unlock()

	return nil
}

// List returns all groups inside a manager
func (m *Manager) List(kind Kind) []*Group {
	gs := make([]*Group, 0)
	for _, g := range m.groups {
		if g.Kind&kind != 0 {
			gs = append(gs, g)
		}
	}

	return gs
}

// GroupByID returns a group by GroupMemberID
func (m *Manager) GroupByID(id int) (*Group, error) {
	if g, ok := m.idMap[id]; ok {
		return g, nil
	}

	return nil, ErrGroupNotFound
}

// GetByName returns a group by name
func (m *Manager) GroupByKey(key string) (*Group, error) {
	m.RLock()
	g, ok := m.keyMap[key]
	m.RUnlock()

	if ok {
		return g, nil
	}

	return nil, ErrGroupNotFound
}

// GroupByMember returns a slice of groups to which a given member belongs
func (m *Manager) GroupByMember(k Kind, member Member) ([]*Group, error) {
	if member == nil {
		return nil, ErrNilMember
	}

	m.RLock()

	gs := make([]*Group, 0)
	for _, g := range m.groups {
		if g.Kind == k {
			if g.IsMember(member) {
				gs = append(gs, g)
			}
		}
	}

	m.RUnlock()

	return gs, nil
}

// LinkGroup tracking which groups this member is a member of
func (m *Manager) LinkGroup(g *Group) error {
	if g == nil {
		return ErrNilGroup
	}

	// safeguard in case this slice is not initialized
	if m.groups == nil {
		m.groups = make([]*Group, 0)
	}

	// appending group to slice for easier runtime access
	m.groups = append(m.groups, g)

	return nil
}

// UnlinkGroup removing group from the tracklist
func (m *Manager) UnlinkGroup(g *Group) error {
	if m.groups == nil {
		// initializing just in case
		m.groups = make([]*Group, 0)

		return nil
	}

	// removing group from the tracklist
	for i, ug := range m.groups {
		if ug.ID == g.ID {
			m.groups = append(m.groups[0:i], m.groups[i+1:]...)
			break
		}
	}

	return ErrGroupNotFound
}

// Groups to which the member belongs
func (m *Manager) Groups(mask Kind) []*Group {
	if m.groups == nil {
		m.groups = make([]*Group, 0)
	}

	groups := make([]*Group, 0)
	for _, g := range m.groups {
		if (g.Kind | mask) == mask {
			groups = append(groups, g)
		}
	}

	return groups
}
