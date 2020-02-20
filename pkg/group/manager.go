package group

import (
	"errors"
	"fmt"
	"log"
	"sync"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/user"
	"go.uber.org/zap"
	"golang.org/x/net/context"
)

// List is a typed slice of groups to make sorting easier
type List []Group

// Manager is a manager responsible for all operations within its scope
// TODO: add default groups which need not to be assigned
type Manager struct {
	groups []*Group
	idMap  map[int]*Group
	keyMap map[string]*Group
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
		groups: make([]*Group, 0),
		idMap:  make(map[int]*Group),
		keyMap: make(map[string]*Group),
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
		// adding user to container
		if err := m.AddGroup(g); err != nil {
			// just warning and moving forward
			l.Info(
				"Init() failed to add group to container",
				zap.Int("group_id", g.ID),
				zap.String("kind", g.Kind.String()),
				zap.String("key", g.Key),
				zap.Error(err),
			)
		}
	}

	return nil
}

// DistributeUsers injects given users into the manager, and linking
// them to their respective groups and roles
// NOTE: does not affect the store
func (m *Manager) DistributeUsers(ctx context.Context, users []*user.User) error {
	if err := m.Validate(); err != nil {
		return err
	}

	l := m.Logger()

	s, err := m.Store()
	if err != nil {
		return err
	}

	// fetching all relations
	l.Info("fetching group user relations")
	rs, err := s.FetchAllRelations(ctx)
	if err != nil {
		return err
	}

	// iterating over injected users
	l.Info("distributing users among their respective groups")
	for _, u := range users {
		// checking whether this user is listed
		// among fetched relations
		if gids, ok := rs[u.ID]; ok {
			// iterating over a slice of group IDs related to this user
			for _, gid := range gids {
				// obtaining the respective group
				g, err := m.GetByID(gid)
				if err != nil {
					// continuing even if an associated group is not found
					l.Info(
						"failed to apply group relation due to missing group",
						zap.Int("uid", u.ID),
						zap.Int("gid", gid),
					)
				}

				// linking user to a group
				if err := g.LinkMember(u); err != nil {
					l.Warn("failed to link user to group", zap.Error(err))
				}
			}
		}
	}

	return nil
}

// Store returns store if set
func (m *Manager) Store() (Store, error) {
	if m.store == nil {
		return nil, core.ErrNilGroupStore
	}

	return m.store, nil
}

// Validate this group manager
func (m *Manager) Validate() error {
	if m.groups == nil {
		return errors.New("groups slice is not initialized")
	}

	if m.idMap == nil {
		return errors.New("id map is nil")
	}

	if m.keyMap == nil {
		return errors.New("key map is nil")
	}

	return nil
}

// Create creates new group
func (m *Manager) Create(kind Kind, key string, name string, parent *Group) (*Group, error) {
	// groups must be of the same kind
	if parent != nil && parent.Kind != kind {
		return nil, core.ErrGroupKindMismatch
	}

	// initializing new group
	g, err := NewGroup(kind, key, name, parent)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize new group(%s): %s", key, err)
	}

	// checking whether there's already some group with such a key
	_, err = m.GetByKey(key)
	if err != nil {
		// returning on unexpected error
		if err != core.ErrGroupNotFound {
			return nil, err
		}
	} else {
		// no error means that the group key is already taken, thus canceling group creation
		return nil, core.ErrGroupKeyTaken
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
func (m *Manager) AddGroup(g *Group) error {
	if g == nil {
		return core.ErrNilGroup
	}

	_, err := m.GetByID(g.ID)
	if err != core.ErrGroupNotFound {
		return core.ErrGroupAlreadyRegistered
	}

	// linking group to this manager
	g.Lock()
	g.manager = m
	g.Unlock()

	// adding group to this manager's container
	m.Lock()
	m.groups = append(m.groups, g)
	m.idMap[g.ID] = g
	m.keyMap[g.Key] = g
	m.Unlock()

	return g.Init()
}

// RemoveGroup removing group from the manager
func (m *Manager) RemoveGroup(id int) error {
	// a bit pedantic but consistent, returning an error if group
	// is already registered within the manager
	g, err := m.GetByID(id)
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

// GetByID returns a group by ID
func (m *Manager) GetByID(id int) (*Group, error) {
	if g, ok := m.idMap[id]; ok {
		return g, nil
	}

	return nil, core.ErrGroupNotFound
}

// GetByName returns a group by name
func (m *Manager) GetByKey(key string) (*Group, error) {
	m.RLock()
	g, ok := m.keyMap[key]
	m.RUnlock()

	if ok {
		return g, nil
	}

	return nil, core.ErrGroupNotFound
}

// GetByUser returns a slice of groups to which a given user belongs
func (m *Manager) GetByUser(k Kind, u *user.User) ([]*Group, error) {
	if u == nil {
		return nil, core.ErrNilUser
	}

	m.RLock()

	gs := make([]*Group, 0)
	for _, g := range m.groups {
		if g.Kind == k {
			if g.IsMember(u) {
				gs = append(gs, g)
			}
		}
	}

	m.RUnlock()

	return gs, nil
}
