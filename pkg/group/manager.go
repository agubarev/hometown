package group

import (
	"context"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)


// errors
var (
	ErrNilDatabase            = errors.New("database is nil")
	ErrNothingChanged         = errors.New("nothing changed")
	ErrNoParent               = errors.New("no parent group")
	ErrZeroGroupID            = errors.New("role group id is zero")
	ErrZeroRoleID             = errors.New("role id is zero")
	ErrZeroID                 = errors.New("id is zero")
	ErrNonZeroID              = errors.New("id is not zero")
	ErrZeroMemberID           = errors.New("member id is zero")
	ErrInvalidMemberKind      = errors.New("member kind is invalid")
	ErrNilGroupStore          = errors.New("group store is nil")
	ErrRelationNotFound          = errors.New("relation not found")
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
	ErrNilGroupManager        = errors.New("group manager is nil")
	ErrGroupNotFound          = errors.New("group not found")
	ErrGroupKeyTaken          = errors.New("group key is already taken")
)

// List is a typed slice of groups to make sorting easier
type List []Group

// userManager is a manager responsible for all operations within its scope
// TODO: add default groups which need not to be assigned explicitly
type Manager struct {
	// group id -> group
	groups map[int64]Group

	// group key -> group ID
	keyMap map[TKey]int64

	// default group ids
	// NOTE: all tracked members belong to the groups whose IDs
	// are mentioned in this slide
	// TODO: unless stated otherwise (work out an exclusion mechanism)
	defaultIDs []int64

	// group ID -> user IDs
	members map[int64][]int64

	// group id -> { key, name, description }
	keys         map[int64]TKey
	names        map[int64]TName
	descriptions map[int64]TDescription

	store  Store
	logger *zap.Logger
	sync.RWMutex
}

// NewManager initializing a new group manager
func NewManager(s Store) (*Manager, error) {
	if s == nil {
		log.Println("NewManager: store is not set")
	}

	c := &Manager{
		groups:     make(map[int64]Group, 0),
		defaultIDs: make([]int64, 0),
		keyMap:     make(map[TKey]int64),
		members:    make(map[int64][]int64),
		store:      s,
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
		if err := m.Put(ctx, g); err != nil {
			// just warning and moving forward
			l.Info(
				"Init() failed to add group to container",
				zap.Int64("group_id", g.ID),
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
// TODO: replace members slice with a channel to support a large numbers of users
func (m *Manager) DistributeMembers(ctx context.Context, members []int64) error {
	l := m.Logger()

	s, err := m.Store()
	if err != nil {
		return err
	}

	// fetching all relations
	l.Info("fetching group relations")
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
						zap.Int64("mid", userID),
						zap.Int64("gid", gid),
					)
				}

				// linking userID to a group
				if err := g.LinkMember(ctx, userID); err != nil {
					l.Warn(
						"failed to link user to group",
						zap.Int64("gid", gid),
						zap.Int64("uid", userID),
						zap.Error(err),
					)
				}
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

// Upsert creates new group
func (m *Manager) Create(ctx context.Context, kind Kind, parentID int64, key string, name string) (g Group, err error) {
	var parent Group

	// obtaining parent group
	if parentID != 0 {
		parent, err = m.GroupByID(ctx, parentID)
		if err != nil {
			return g, errors.Wrap(err, "failed to obtain parent group")
		}

		if err := parent.Validate(ctx); err != nil {
			return g, errors.Wrap(err, "parent group validation failed")
		}

		// groups must be of the same kind
		if parent.Kind != kind {
			return g, ErrGroupKindMismatch
		}
	}

	// initializing new group
	g = Group{
		Kind:    kind,
		Key:     strings.ToLower(strings.TrimSpace(key)),
		Name:    strings.ToLower(strings.TrimSpace(name)),
		members: make(map[int64][]int64, 0),
	}

	// group must have a key
	if g.Key == "" {
		return g, errors.New("group must have a key")
	}

	// group name must not be empty
	if g.Name == "" {
		return g, ErrEmptyGroupName
	}

	// assigning parent group
	if err := g.SetParent(ctx, parent); err != nil {
		return g, err
	}

	if err = g.Validate(ctx); err != nil {
		return g, errors.Wrap(err, "new group validation failed")
	}

	// checking whether there's already some group with such key
	_, err = m.GroupByKey(ctx, g.Key)
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
	g, err = m.store.UpsertGroup(ctx, g)
	if err != nil {
		return g, err
	}

	// adding new group to manager's registry
	err = m.Put(ctx, g)
	if err != nil {
		return g, err
	}

	return g, nil
}

// Put adds group to the manager
func (m *Manager) Put(ctx context.Context, g Group) error {
	if g.ID == 0 {
		return ErrZeroID
	}

	m.Lock()

	// adding group to the map or skipping otherwise
	if _, ok := m.groups[g.ID]; !ok {
		m.groups[g.ID] = g
		m.keyMap[g.Key] = g.ID

		// adding group ID to a slice of defaults if it's marked as default
		if g.IsDefault {
			m.defaultIDs = append(m.defaultIDs, g.ID)
		}
	}

	m.Unlock()

	return g.Init()
}

// Lookup looks up in cache and returns a group if found
func (m *Manager) Lookup(ctx context.Context, groupID int64) (g Group, err error) {
	m.RLock()
	g, ok := m.groups[groupID]
	m.RUnlock()

	if ok {
		return g, nil
	}

	return g, ErrGroupNotFound
}

// Remove removing group from the manager
func (m *Manager) Remove(ctx context.Context, groupID int64) error {
	// a bit pedantic but consistent, returning an error if group
	// is already registered within the manager
	g, err := m.GroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	// clearing internal cache

	m.Lock()

	delete(m.groups, g.ID)
	delete(m.keyMap, g.Key)
	delete(m.keys, g.ID)
	delete(m.names, g.ID)
	delete(m.descriptions, g.ID)

	// discarding group membership cache
	delete(m.members, g.ID)

	m.Unlock()

	return nil
}

// List returns all groups inside a manager
func (m *Manager) List(kind Kind) (gs []Group) {
	gs = make([]Group, 0)

	for _, g := range m.groups {
		if g.Kind&kind != 0 {
			gs = append(gs, g)
		}
	}

	return gs
}

// GroupByID returns a group by ID
func (m *Manager) GroupByID(ctx context.Context, id int64) (g Group, err error) {
	if g, err = m.Lookup(ctx, id); err != ErrGroupNotFound {
		return g, nil
	}

	// fetching group from the store
	g, err = m.store.FetchGroupByID(ctx, id)
	if err != nil {
		return g, err
	}

	// updating group cache
	if err = m.Put(ctx, g); err != nil {
		return g, err
	}

	return g, nil
}

// PolicyByName returns a group by name
func (m *Manager) GroupByKey(ctx context.Context, key string) (g Group, err error) {
	m.RLock()
	g, ok := m.groups[m.keyMap[key]]
	m.RUnlock()

	if ok {
		return g, nil
	}

	return g, ErrGroupNotFound
}

// GroupByName returns an access policy by its key
// TODO: add expirable caching
func (m *Manager) GroupByName(ctx context.Context, name string) (g Group, err error) {
	return m.store.FetchGroupByName(ctx, name)
}

// DeletePolicy returns an access policy by its ObjectID
// NOTE: also deletes all relations and nested groups (should user have
// sufficient access rights to do that)
// TODO: implement recursive deletion
func (m *Manager) DeleteGroup(ctx context.Context, g Group) (err error) {
	if err = m.Validate(ctx, g); err != nil {
		return err
	}

	panic("not implemented")

	return nil
}

// GroupsByUserID returns a slice of groups to which a given member belongs
func (m *Manager) GroupsByUserID(ctx context.Context, userID int64, mask Kind) (gs []Group) {
	if userID == 0 {
		return gs
	}

	gs = make([]Group, 0)

	m.RLock()
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

// Groups to which the member belongs
func (m *Manager) Groups(ctx context.Context, mask Kind) []Group {
	if m.groups == nil {
		m.groups = make(map[int64]Group, 0)
	}

	groups := make([]Group, 0)
	for _, g := range m.groups {
		if (g.Kind | mask) == mask {
			groups = append(groups, g)
		}
	}

	return groups
}

func (m *Manager) setupDefaultGroups(ctx context.Context) error {
	if m.groups == nil {
		return ErrNilGroupManager
	}

	// regular user
	userRole, err := m.Create(ctx, GKRole, 0, "user", "Regular User")
	if err != nil {
		return errors.Wrap(err, "failed to create regular user role")
	}

	// manager
	managerRole, err := m.Create(ctx, GKRole, userRole.ID, "manager", "Manager")
	if err != nil {
		return fmt.Errorf("failed to create manager role: %s", err)
	}

	// super user
	_, err = m.Create(ctx, GKRole, managerRole.ID, "superuser", "Super User")
	if err != nil {
		return fmt.Errorf("failed to create superuser role: %s", err)
	}

	return nil
}

// Parent returns a parent of a given group
func (m *Manager) Parent(ctx context.Context, g Group) (p Group, err error) {
	if g.ParentID == 0 {
		return p, ErrNoParent
	}

	return m.GroupByID(ctx, g.ParentID)
}

// Validate performs an integrity check on a given group
func (m *Manager) Validate(ctx context.Context, g Group) error {
	if g.ID == 0 {
		return ErrZeroGroupID
	}

	// checking for parent circulation
	if isCircuited, err := g.IsCircuited(ctx); isCircuited || (err != nil) {
		if err != nil {
			return errors.Wrapf(err, "%s validation failed", g.StringID())
		}

		if isCircuited {
			return errors.Wrapf(err, "%s validation failed: %s", g.StringID(), ErrCircuitedParent)
		}
	}

	// general field validations
	if ok, err := govalidator.ValidateStruct(g); !ok || err != nil {
		return errors.Wrapf(err, "%s validation failed", g.StringID())
	}

	return nil
}

// IsCircuited tests whether the parents trace back to a nil
func (g *Group) IsCircuited(ctx context.Context) (bool, error) {
	if g.ParentID == 0 {
		return false, nil
	}

	// moving up a parent tree until nil is reached or the signs of circulation are found
	// TODO add checks to discover possible circulation before the timeout in case of a long parent trail
	// TODO: obtain timeout from the context or use its deadline
	timeout := time.Now().Add(5 * time.Millisecond)
	for p, err := g.Parent(ctx); err == nil && !time.Now().After(timeout); p, err = p.Parent(ctx) {
		if p.ParentID == 0 {
			// it's all good, reached a no-parent
			return false, nil
		}
	}

	return false, ErrCircuitCheckTimeout
}

// SetDescription sets text description for this group
func (g *Group) SetDescription(text string) error {
	// TODO: implement

	return nil
}

// SetParentID assigning a parent group, could be nil
func (g *Group) SetParent(ctx context.Context, p Group) error {
	// since parent could be nil thus it's kind is irrelevant
	if p.ID != 0 {
		// checking whether new parent already is set somewhere along the parenthood
		// by tracing backwards until a no-parent is met; at this point only a
		// requested parent is searched and not tested whether the relations
		// are circuited among themselves
		if p.ParentID != 0 {
			for pg, err := g.Parent(ctx); err == nil && pg.ID != 0; pg, err = pg.Parent(ctx) {
				// no more parents, breaking
				if pg.ParentID == 0 {
					break
				}

				// testing equality by comparing each group's ObjectID
				if pg.ID == p.ID {
					return ErrDuplicateParent
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
⏰
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
			if err = g.manager.store.CreateRelation(ctx, g.ID, userID); err != nil {
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
func (m *Manager) UnlinkMember(ctx context.Context, userID int64) (err error) {
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

// Invite an existing user to become a member of the group
// NOTE: this is optional and often can be disabled for better control
func (g *Group) Invite(ctx context.Context, userID int64) (err error) {
	// TODO: implement

	panic("not implemented")

	return nil
}
