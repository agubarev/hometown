package group

import (
	"bytes"
	"context"
	"fmt"
	"log"
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
	ErrNilStore               = errors.New("group store is nil")
	ErrRelationNotFound       = errors.New("relation not found")
	ErrGroupAlreadyRegistered = errors.New("group is already registered")
	ErrEmptyKey               = errors.New("empty group key")
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
	ErrNilManager             = errors.New("group manager is nil")
	ErrGroupNotFound          = errors.New("group not found")
	ErrGroupKeyTaken          = errors.New("group key is already taken")
	ErrUnknownKind            = errors.New("unknown group kind")
	ErrInvalidGroupKey        = errors.New("invalid group key")
	ErrInvalidGroupName       = errors.New("invalid group name")
	ErrEmptyGroupKey          = errors.New("group key is empty")
)

// List is a typed slice of groups to make sorting easier
type List []Group

// Manager is a group manager
// TODO: add default groups which need not to be assigned explicitly
type Manager struct {
	// group id -> group
	groups map[uint32]Group

	// group key -> group ID
	keyMap map[TKey]uint32

	// default group ids
	// NOTE: all tracked members belong to the groups whose IDs
	// are mentioned in this slide
	// TODO: unless stated otherwise (work out an exclusion mechanism)
	defaultIDs []uint32

	// relationships
	memberGroups map[uint32][]uint32 // member ID -> slice of group IDs
	groupMembers map[uint32][]uint32 // group ID -> slice of member IDs

	store  Store
	logger *zap.Logger
	sync.RWMutex
}

// NewManager initializing a new group manager
func NewManager(ctx context.Context, s Store) (m *Manager, err error) {
	if s == nil {
		log.Println("NewManager: store is not set")
	}

	m = &Manager{
		groups:       make(map[uint32]Group, 0),
		keyMap:       make(map[TKey]uint32),
		defaultIDs:   make([]uint32, 0),
		memberGroups: make(map[uint32][]uint32),
		groupMembers: make(map[uint32][]uint32),
		store:        s,
	}

	if err = m.Init(ctx); err != nil {
		return nil, errors.Wrap(err, "failed to initialize group manager")
	}

	return m, nil
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
	s, err := m.Store()
	if err != nil {
		return err
	}

	// fetching all stored groups
	groups, err := s.FetchAllGroups(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to fetch all groups")
	}

	// adding groups to a container
	for _, g := range groups {
		// adding member to container
		if err := m.Put(ctx, g); err != nil {
			// just warning and moving forward
			m.Logger().Info(
				"Init() failed to add group to container",
				zap.Uint32("group_id", g.ID),
				zap.String("kind", g.Kind.String()),
				zap.ByteString("key", g.Key[:]),
				zap.Error(err),
			)
		}
	}

	// fetching and distributing group member relations
	relations, err := s.FetchAllRelations(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to fetch all relations")
	}

	// creating runtime links of groups and members
	for mid, gids := range relations {
		for _, gid := range gids {
			if err = m.LinkMember(ctx, gid, mid); err != nil {
				m.Logger().Info(
					"Init() failed to add group to container",
					zap.Uint32("group_id", gid),
					zap.Uint32("member_id", mid),
					zap.Error(err),
				)
			}
		}
	}

	return nil
}

// DistributeMembers injects given members into the manager, and linking
// them to their respective groups and roles
// NOTE: does not affect the store
// TODO: replace members slice with a channel to support a large numbers of members
func (m *Manager) DistributeMembers(ctx context.Context, members []uint32) (err error) {
	l := m.Logger()

	s, err := m.Store()
	if err != nil {
		return err
	}

	// fetching all relations
	l.Info("fetching group relations")
	relations, err := s.FetchAllRelations(ctx)
	if err != nil {
		return err
	}

	// iterating over injected members
	l.Info("distributing members among their respective groups")
	for _, memberID := range members {
		// checking whether this memberID is listed among fetched relations
		if groupIDs, ok := relations[memberID]; ok {
			// iterating over a slice of group IDs related to this member
			for _, groupID := range groupIDs {
				// obtaining the corresponding group
				if _, err = m.GroupByID(ctx, groupID); err != nil {
					if err != ErrGroupNotFound {
						return errors.Wrapf(err, "failed to obtain group: %d", groupID)
					}

					// continuing even if an associated group is not found
					l.Info(
						"failed to apply group relation due to missing group",
						zap.Uint32("member_id", memberID),
						zap.Uint32("group_id", groupID),
					)
				}

				// linking memberID to a group
				if err = m.LinkMember(ctx, groupID, memberID); err != nil {
					l.Warn(
						"failed to link member to group",
						zap.Uint32("member_id", memberID),
						zap.Uint32("group_id", groupID),
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
		return nil, ErrNilStore
	}

	return m.store, nil
}

// Upsert creates new group
func (m *Manager) Create(ctx context.Context, kind Kind, parentID uint32, key string, name string, isDefault bool) (g Group, err error) {
	// checking parent id
	if parentID != 0 {
		parent, err := m.GroupByID(ctx, parentID)
		if err != nil {
			return g, errors.Wrap(err, "failed to obtain parent group")
		}

		// validating parent group
		if err := m.Validate(ctx, parent.ID); err != nil {
			return g, errors.Wrap(err, "parent group validation failed")
		}

		// groups must be of the same kind
		if parent.Kind != kind {
			return g, ErrGroupKindMismatch
		}
	}

	g, err = NewGroup(kind, parentID, NewKey(key), NewName(name), isDefault)
	if err != nil {
		return g, errors.Wrap(err, "failed to initialize new group")
	}

	// converting string key and a name into array values
	copy(g.Key[:], bytes.TrimSpace([]byte(name)))

	// basic field validation
	if ok, err := govalidator.ValidateStruct(g); !ok || err != nil {
		return g, errors.Wrap(err, "validation failed")
	}

	// checking whether there's already some group with such key
	if _, err = m.GroupByKey(ctx, g.Key); err != nil {
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
	if err = m.Put(ctx, g); err != nil {
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

	return nil
}

// Lookup looks up in cache and returns a group if found
func (m *Manager) Lookup(ctx context.Context, groupID uint32) (g Group, err error) {
	m.RLock()
	g, ok := m.groups[groupID]
	m.RUnlock()

	if ok {
		return g, nil
	}

	return g, ErrGroupNotFound
}

// Remove removing group from the manager
func (m *Manager) Remove(ctx context.Context, groupID uint32) error {
	// a bit pedantic but consistent, returning an error if group
	// is already registered within the manager
	g, err := m.Lookup(ctx, groupID)
	if err != nil {
		return err
	}

	m.Lock()

	//---------------------------------------------------------------------------
	// clearing internal cache
	//---------------------------------------------------------------------------
	delete(m.groups, g.ID)
	delete(m.keyMap, g.Key)

	//---------------------------------------------------------------------------
	// discarding group membership cache
	//---------------------------------------------------------------------------
	// first, clearing out group ID from every related member
	for _, memberID := range m.groupMembers[groupID] {
		for i, id := range m.memberGroups[memberID] {
			if id == groupID {
				m.memberGroups[memberID] = append(m.memberGroups[memberID][0:i], m.memberGroups[memberID][i+1:]...)
				break
			}
		}
	}

	// and eventually discarding the group to members relation
	delete(m.groupMembers, groupID)

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
func (m *Manager) GroupByID(ctx context.Context, id uint32) (g Group, err error) {
	if id == 0 {
		return g, ErrZeroGroupID
	}

	// lookup cache first
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
func (m *Manager) GroupByKey(ctx context.Context, key TKey) (g Group, err error) {
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
func (m *Manager) GroupByName(ctx context.Context, name TName) (g Group, err error) {
	return m.store.FetchGroupByName(ctx, name)
}

// DeletePolicy returns an access policy by its ObjectID
// NOTE: also deletes all relations and nested groups (should member have
// sufficient access rights to do that)
// TODO: implement recursive deletion
func (m *Manager) DeleteGroup(ctx context.Context, groupID uint32) (err error) {
	g, err := m.GroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	s, err := m.Store()
	if err != nil {
		return err
	}

	// deleting from the store backend
	if err = s.DeleteByID(ctx, g.ID); err != nil {
		return errors.Wrapf(err, "failed to delete group: %d", groupID)
	}

	// removing from internal cache
	if err = m.Remove(ctx, g.ID); err != nil {
		return errors.Wrapf(err, "failed to remove cached group after deletion: %d", g.ID)
	}

	return nil
}

// GroupsBymemberID returns a slice of groups to which a given member belongs
func (m *Manager) GroupsBymemberID(ctx context.Context, memberID uint32, mask Kind) (gs []Group) {
	if memberID == 0 {
		return gs
	}

	gs = make([]Group, 0)

	m.RLock()
	for _, g := range m.groups {
		if g.Kind&mask != 0 {
			if m.IsMember(ctx, g.ID, memberID) {
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

func (m *Manager) setupDefaultGroups(ctx context.Context) error {
	if m.groups == nil {
		return ErrNilManager
	}

	//---------------------------------------------------------------------------
	// roles
	//---------------------------------------------------------------------------

	// regular user
	regularRole, err := m.Create(ctx, GKRole, 0, "regular", "Regular user", false)
	if err != nil {
		return errors.Wrap(err, "failed to create regular user role")
	}

	// manager
	managerRole, err := m.Create(ctx, GKRole, regularRole.ID, "manager", "Manager", false)
	if err != nil {
		return fmt.Errorf("failed to create manager role: %s", err)
	}

	// super user
	_, err = m.Create(ctx, GKRole, managerRole.ID, "superuser", "Super user", false)
	if err != nil {
		return fmt.Errorf("failed to create supermember role: %s", err)
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
func (m *Manager) Validate(ctx context.Context, groupID uint32) (err error) {
	g, err := m.GroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	// checking for parent circulation
	if isCircuited, err := m.IsCircuited(ctx, g.ID); isCircuited || (err != nil) {
		if err != nil {
			return errors.Wrapf(err, "group validation failed: %d", g.ID)
		}

		if isCircuited {
			return errors.Wrapf(err, "validation failed: %d (%s)", g.ID, ErrCircuitedParent)
		}
	}

	return nil
}

// IsCircuited tests whether the parents trace back to a nil
func (m *Manager) IsCircuited(ctx context.Context, groupID uint32) (bool, error) {
	g, err := m.GroupByID(ctx, groupID)
	if err != nil {
		return false, err
	}

	// no parent means no opportunity to make circular parenting
	if g.ParentID == 0 {
		return false, nil
	}

	// moving up a parent tree until nil is reached or the signs of circulation are found
	// TODO add checks to discover possible circulation before the timeout in case of a long parent trail
	// TODO: obtain timeout from the context or use its deadline
	timeout := time.Now().Add(5 * time.Millisecond)
	for p, err := m.Parent(ctx, g); err == nil && !time.Now().After(timeout); p, err = m.Parent(ctx, p) {
		if p.ParentID == 0 {
			// it's all good, reached a no-parent point
			return false, nil
		}
	}

	return false, ErrCircuitCheckTimeout
}

// SetParent assigns a new parent ID
func (m *Manager) SetParent(ctx context.Context, groupID, newParentID uint32) (err error) {
	g, err := m.GroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	newParent, err := m.GroupByID(ctx, newParentID)
	if err != nil {
		return errors.Wrap(err, "parent group not found")
	}

	// since new parent could be zero then its kind is irrelevant
	if newParent.ID != 0 {
		// checking whether new parent already is set somewhere along the parenthood
		// by tracing backwards until a no-parent is met; at this point only a
		// requested parent is searched and not tested whether the relations
		// are circuited among themselves
		if newParent.ParentID != 0 {
			for pg, err := m.Parent(ctx, g); err == nil && pg.ID != 0; pg, err = m.Parent(ctx, pg) {
				// no more parents, breaking
				if pg.ParentID == 0 {
					break
				}

				// testing equality by comparing each group's ObjectID
				if pg.ID == newParent.ID {
					return ErrDuplicateParent
				}
			}
		}

		// group kind must be the same all the way back to the top
		if g.Kind != newParent.Kind {
			return ErrGroupKindMismatch
		}
	}

	// previous checks have passed, thus assingning a new parent ID
	// NOTE: ParentID is used to rebuild parent-child connections after
	// loading groups from the store
	g.ParentID = newParent.ID

	// obtaining store
	s, err := m.Store()
	if err != nil {
		return errors.Wrap(err, "failed to obtain group store")
	}

	// saving updated group
	g, err = s.UpsertGroup(ctx, g)
	if err != nil {
		return errors.Wrap(err, "failed to save group after changing new parent")
	}

	return nil
}

// IsMember tests whether a given member belongs to a given group
func (m *Manager) IsMember(ctx context.Context, groupID, memberID uint32) bool {
	if memberID == 0 || groupID == 0 {
		return false
	}

	m.RLock()
	for _, gid := range m.memberGroups[memberID] {
		if gid == groupID {
			m.RUnlock()
			return true
		}
	}
	m.RUnlock()

	return false
}

// CreateRelation adding member to a group
// NOTE: storing relation only if group has a store set is implicit and should at least
// log/print about the occurrence
func (m *Manager) CreateRelation(ctx context.Context, groupID, memberID uint32) (err error) {
	_, err = m.GroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if memberID == 0 {
		return ErrZeroMemberID
	}

	s, err := m.Store()
	if err != nil && err != ErrNilStore {
		return errors.Wrap(err, "failed to obtain group store")
	}

	l := m.Logger()

	if s != nil {
		l.Info("creating group member relationship", zap.Uint32("group_id", groupID), zap.Uint32("member_id", memberID))

		// persisting relation in the store
		if err = s.CreateRelation(ctx, groupID, memberID); err != nil {
			l.Info("failed to store group to member relation", zap.Uint32("group_id", groupID), zap.Uint32("member_id", memberID), zap.Error(err))
			return err
		}
	} else {
		l.Info("creating group member relationship while store is not set", zap.Uint32("group_id", groupID), zap.Uint32("member_id", memberID), zap.Error(err))
	}

	// adding member ID to group members
	if err = m.LinkMember(ctx, groupID, memberID); err != nil {
		return err
	}

	return nil
}

// DeleteRelation removes member from a group
func (m *Manager) DeleteRelation(ctx context.Context, groupID, memberID uint32) (err error) {
	_, err = m.GroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if memberID == 0 {
		return ErrZeroMemberID
	}

	// removing memberID from group members
	if err = m.UnlinkMember(ctx, groupID, memberID); err != nil {
		return err
	}

	s, err := m.Store()
	if err != nil && err != ErrNilStore {
		return errors.Wrap(err, "failed to obtain group store")
	}

	l := m.Logger()

	if s != nil {
		m.Logger().Info("deleting member from group", zap.Uint32("group_id", groupID), zap.Uint32("member_id", memberID))

		// deleting a stored relation
		if err := s.DeleteRelation(ctx, groupID, memberID); err != nil {
			return err
		}
	} else {
		l.Info("deleting member from group while store is not set", zap.Uint32("group_id", groupID), zap.Uint32("member_id", memberID))
	}

	return nil
}

// LinkMember adds a member to the group members
// NOTE: does not affect the store
func (m *Manager) LinkMember(ctx context.Context, groupID, memberID uint32) (err error) {
	_, err = m.GroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if memberID == 0 {
		return ErrZeroMemberID
	}

	if m.IsMember(ctx, groupID, memberID) {
		return ErrAlreadyMember
	}

	m.Lock()

	// group ID -> member IDs
	if m.groupMembers[groupID] == nil {
		m.groupMembers[groupID] = []uint32{memberID}
	} else {
		m.groupMembers[groupID] = append(m.groupMembers[groupID], memberID)
	}

	// member ID -> group IDs
	if m.memberGroups[memberID] == nil {
		m.memberGroups[memberID] = []uint32{groupID}
	} else {
		m.memberGroups[memberID] = append(m.memberGroups[memberID], groupID)
	}

	m.Unlock()

	return nil
}

// UnlinkMember removes a member from the group members
// NOTE: does not affect the store
func (m *Manager) UnlinkMember(ctx context.Context, groupID, memberID uint32) (err error) {
	_, err = m.GroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if memberID == 0 {
		return ErrZeroMemberID
	}

	m.Lock()

	if m.groupMembers[groupID] != nil {
		for i, mid := range m.groupMembers[groupID] {
			if mid == memberID {
				m.groupMembers[groupID] = append(m.groupMembers[groupID][0:i], m.groupMembers[groupID][i+1:]...)
				break
			}
		}
	}

	if m.memberGroups[memberID] != nil {
		for i, gid := range m.memberGroups[memberID] {
			if gid == groupID {
				m.memberGroups[memberID] = append(m.memberGroups[memberID][0:i], m.memberGroups[memberID][i+1:]...)
				break
			}
		}
	}

	m.Unlock()

	return nil
}

// Invite an existing user to become a member of the group
// NOTE: this is optional and often can be disabled for better control
// TODO: requires careful planning
func (m *Manager) Invite(ctx context.Context, groupID, userID uint32) (err error) {
	// TODO: implement

	panic("not implemented")

	return nil
}
