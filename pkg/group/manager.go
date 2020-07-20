package group

import (
	"bytes"
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// errors
var (
	ErrNilDatabase            = errors.New("database is nil")
	ErrNothingChanged         = errors.New("nothing changed")
	ErrNoParent               = errors.New("no parent group")
	ErrNilGroupID             = errors.New("role group id is zero")
	ErrZeroRoleID             = errors.New("role id is zero")
	ErrZeroID                 = errors.New("id is zero")
	ErrNonZeroID              = errors.New("id is not zero")
	ErrNilAssetID             = errors.New("asset id is zero")
	ErrInvalidAssetKind       = errors.New("asset kind is invalid")
	ErrNilStore               = errors.New("group store is nil")
	ErrRelationNotFound       = errors.New("relation not found")
	ErrGroupAlreadyRegistered = errors.New("group is already registered")
	ErrEmptyKey               = errors.New("empty group key")
	ErrEmptyGroupName         = errors.New("empty group name")
	ErrDuplicateGroup         = errors.New("duplicate group")
	ErrDuplicateParent        = errors.New("duplicate parent")
	ErrDuplicateRelation      = errors.New("duplicate relation")
	ErrAssetNotEligible       = errors.New("asset is not eligible for this operation")
	ErrGroupKindMismatch      = errors.New("group kinds mismatch")
	ErrInvalidKind            = errors.New("invalid group kind")
	ErrNotAsset               = errors.New("asset is not a asset")
	ErrAlreadyAsset           = errors.New("already a asset")
	ErrCircuitedParent        = errors.New("circuited parenting")
	ErrCircuitCheckTimeout    = errors.New("circuit check timed out")
	ErrNilManager             = errors.New("group manager is nil")
	ErrGroupNotFound          = errors.New("group not found")
	ErrGroupKeyTaken          = errors.New("group key is already taken")
	ErrUnknownKind            = errors.New("unknown group kind")
	ErrInvalidGroupKey        = errors.New("invalid group key")
	ErrInvalidGroupName       = errors.New("invalid group name")
	ErrEmptyGroupKey          = errors.New("group key is empty")
	ErrAmbiguousKind          = errors.New("group kind is ambiguous")
)

type AssetKind uint8

const (
	AKUser AssetKind = iota
)

func (ak AssetKind) String() string {
	switch ak {
	case AKUser:
		return "user"
	default:
		return "unrecognized asset kind"
	}
}

type Asset struct {
	Kind AssetKind
	ID   uuid.UUID
}

func NewAsset(k AssetKind, id uuid.UUID) Asset {
	return Asset{
		Kind: k,
		ID:   id,
	}
}

type Relation struct {
	GroupID uuid.UUID
	Asset   Asset
}

func NewRelation(gid uuid.UUID, k AssetKind, aid uuid.UUID) Relation {
	return Relation{
		GroupID: gid,
		Asset: Asset{
			Kind: k,
			ID:   aid,
		},
	}
}

// Registry is a typed slice of groups to make sorting easier
type List []Group

// Manager is a group manager
// TODO: add default groups which need not to be assigned explicitly
type Manager struct {
	// group id -> group
	groups map[uuid.UUID]Group

	// group key -> group ActorID
	keyMap map[TKey]uuid.UUID

	// default group ids
	// NOTE: all tracked assets belong to the groups whose IDs
	// are mentioned in this slide
	// TODO: unless stated otherwise (work out an exclusion mechanism)
	defaultIDs []uuid.UUID

	// relationships
	assetGroups map[Asset][]uuid.UUID // asset -> slice of group IDs
	groupAssets map[uuid.UUID][]Asset // group ActorID -> slice of asset IDs

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
		groups:      make(map[uuid.UUID]Group, 0),
		keyMap:      make(map[TKey]uuid.UUID),
		defaultIDs:  make([]uuid.UUID, 0),
		assetGroups: make(map[Asset][]uuid.UUID),
		groupAssets: make(map[uuid.UUID][]Asset),
		store:       s,
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
		// adding asset to container
		if err := m.Put(ctx, g); err != nil {
			// just warning and moving forward
			m.Logger().Info(
				"Init() failed to add group to container",
				zap.String("group_id", g.ID.String()),
				zap.String("flags", g.Flags.Translate()),
				zap.ByteString("key", g.Key[:]),
				zap.Error(err),
			)
		}
	}

	// fetching and distributing group asset relations
	relations, err := s.FetchAllRelations(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to fetch all relations")
	}

	// creating runtime links of groups and assets
	for _, rel := range relations {
		if err = m.LinkAsset(ctx, rel.GroupID, rel.Asset); err != nil {
			m.Logger().Info(
				"Init() failed to add group to container",
				zap.String("group_id", rel.GroupID.String()),
				zap.String("asset_id", rel.Asset.ID.String()),
				zap.String("asset_kind", rel.Asset.Kind.String()),
				zap.Error(err),
			)
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
func (m *Manager) Create(ctx context.Context, flags Flags, parentID uuid.UUID, key TKey, name TName) (g Group, err error) {
	// checking parent id
	if parentID != uuid.Nil {
		parent, err := m.GroupByID(ctx, parentID)
		if err != nil {
			return g, errors.Wrap(err, "failed to obtain parent group")
		}

		// validating parent group
		if err := m.Validate(ctx, parent.ID); err != nil {
			return g, errors.Wrap(err, "parent group validation failed")
		}

		// groups must be of the same flags
		if parent.Flags != flags {
			return g, ErrGroupKindMismatch
		}
	}

	// initializing new group
	g, err = NewGroup(flags, parentID, key, name)
	if err != nil {
		return g, errors.Wrap(err, "failed to initialize new group")
	}

	// converting string key and a name into array values
	copy(g.Key[:], bytes.TrimSpace(key[:]))
	copy(g.DisplayName[:], bytes.TrimSpace(name[:]))

	// basic field validation
	if ok, err := govalidator.ValidateStruct(g); !ok || err != nil {
		return g, errors.Wrap(err, "new group validation failed")
	}

	// checking whether there's already some group with such key
	if _, err = m.GroupByKey(ctx, g.Key); err != nil {
		// returning on unexpected error
		if errors.Cause(err) != ErrGroupNotFound {
			return g, err
		}
	} else {
		// no error means that the group key is already taken, thus canceling group creation
		return g, ErrGroupKeyTaken
	}

	// generating new UUID before creating
	g.ID = uuid.New()

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

// put adds group to the manager
func (m *Manager) Put(ctx context.Context, g Group) error {
	if g.ID == uuid.Nil {
		return ErrZeroID
	}

	m.Lock()

	// adding group to the map or skipping otherwise
	if _, ok := m.groups[g.ID]; !ok {
		m.groups[g.ID] = g
		m.keyMap[g.Key] = g.ID

		// adding group ActorID to a slice of defaults if it's marked as default
		if g.IsDefault() {
			m.defaultIDs = append(m.defaultIDs, g.ID)
		}
	}

	m.Unlock()

	return nil
}

// lookup looks up in cache and returns a group if found
func (m *Manager) Lookup(ctx context.Context, groupID uuid.UUID) (g Group, err error) {
	m.RLock()
	g, ok := m.groups[groupID]
	m.RUnlock()

	if ok {
		return g, nil
	}

	return g, ErrGroupNotFound
}

// Remove removing group from the manager
func (m *Manager) Remove(ctx context.Context, groupID uuid.UUID) error {
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
	// discarding group relationship cache
	//---------------------------------------------------------------------------
	// first, clearing out group ActorID from every related asset
	for _, assetID := range m.groupAssets[groupID] {
		for i, id := range m.assetGroups[assetID] {
			if id == groupID {
				m.assetGroups[assetID] = append(m.assetGroups[assetID][0:i], m.assetGroups[assetID][i+1:]...)
				break
			}
		}
	}

	// and eventually discarding the group to assets relation
	delete(m.groupAssets, groupID)

	m.Unlock()

	return nil
}

// Registry returns all groups inside a manager
func (m *Manager) List(kind Flags) (gs []Group) {
	gs = make([]Group, 0)

	m.RLock()
	for _, g := range m.groups {
		if g.Flags&kind != 0 {
			gs = append(gs, g)
		}
	}
	m.RUnlock()

	return gs
}

// GroupByID returns a group by ActorID
func (m *Manager) GroupByID(ctx context.Context, id uuid.UUID) (g Group, err error) {
	if id == uuid.Nil {
		return g, ErrNilGroupID
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

// PolicyByKey returns a group by name
func (m *Manager) GroupByKey(ctx context.Context, key TKey) (g Group, err error) {
	m.RLock()
	g, ok := m.groups[m.keyMap[key]]
	m.RUnlock()

	if ok {
		return g, nil
	}

	g, err = m.store.FetchGroupByKey(ctx, key)
	if err != nil {
		return g, errors.Wrapf(err, "failed to obtain group by key: %s", key)
	}

	m.Lock()
	m.groups[g.ID] = g
	m.keyMap[g.Key] = g.ID
	m.Unlock()

	return g, ErrGroupNotFound
}

// GroupByName returns an access policy by its key
// TODO: add expirable caching
func (m *Manager) GroupByName(ctx context.Context, name TName) (g Group, err error) {
	return m.store.FetchGroupByName(ctx, name)
}

// DeletePolicy returns an access policy by its ObjectID
// NOTE: also deletes all relations and nested groups (should asset have
// sufficient access rights to do that)
// TODO: implement recursive deletion
func (m *Manager) DeleteGroup(ctx context.Context, groupID uuid.UUID) (err error) {
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

// GroupsByAssetID returns a slice of groups to which a given asset belongs
func (m *Manager) GroupsByAssetID(ctx context.Context, mask Flags, asset Asset) (gs []Group) {
	if asset.ID == uuid.Nil {
		return gs
	}

	gs = make([]Group, 0)

	m.RLock()
	for _, g := range m.groups {
		if g.Flags&mask != 0 {
			if m.IsAsset(ctx, g.ID, asset) {
				gs = append(gs, g)
			}
		}
	}
	m.RUnlock()

	return gs
}

// Groups to which the asset belongs
func (m *Manager) Groups(ctx context.Context, mask Flags) []Group {
	if m.groups == nil {
		m.groups = make(map[uuid.UUID]Group, 0)
	}

	groups := make([]Group, 0)
	for _, g := range m.groups {
		if (g.Flags | mask) == mask {
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
	regularRole, err := m.Create(ctx, FRole, uuid.Nil, NewKey("regular"), NewName("Regular user"))
	if err != nil {
		return errors.Wrap(err, "failed to create regular user role")
	}

	// manager
	managerRole, err := m.Create(ctx, FRole, regularRole.ID, NewKey("manager"), NewName("Manager"))
	if err != nil {
		return fmt.Errorf("failed to create manager role: %s", err)
	}

	// super user
	_, err = m.Create(ctx, FRole, managerRole.ID, NewKey("superuser"), NewName("Super user"))
	if err != nil {
		return fmt.Errorf("failed to create superuser role: %s", err)
	}

	return nil
}

// Parent returns a parent of a given group
func (m *Manager) Parent(ctx context.Context, g Group) (p Group, err error) {
	if g.ParentID == uuid.Nil {
		return p, ErrNoParent
	}

	return m.GroupByID(ctx, g.ParentID)
}

// Validate performs an integrity check on a given group
func (m *Manager) Validate(ctx context.Context, groupID uuid.UUID) (err error) {
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
func (m *Manager) IsCircuited(ctx context.Context, groupID uuid.UUID) (bool, error) {
	g, err := m.GroupByID(ctx, groupID)
	if err != nil {
		return false, err
	}

	// no parent means no opportunity to make circular parenting
	if g.ParentID == uuid.Nil {
		return false, nil
	}

	// moving up a parent tree until nil is reached or the signs of circulation are found
	// TODO add checks to discover possible circulation before the timeout in case of a long parent trail
	// TODO: obtain timeout from the context or use its deadline
	timeout := time.Now().Add(5 * time.Millisecond)
	for p, err := m.Parent(ctx, g); err == nil && !time.Now().After(timeout); p, err = m.Parent(ctx, p) {
		if p.ParentID == uuid.Nil {
			// it's all good, reached a no-parent point
			return false, nil
		}
	}

	return false, ErrCircuitCheckTimeout
}

// SetParent assigns a new parent ActorID
func (m *Manager) SetParent(ctx context.Context, groupID, newParentID uuid.UUID) (err error) {
	g, err := m.GroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	newParent, err := m.GroupByID(ctx, newParentID)
	if err != nil {
		return errors.Wrap(err, "parent group not found")
	}

	// since new parent could be zero then its kind is irrelevant
	if newParent.ID != uuid.Nil {
		// checking whether new parent already is set somewhere along the parenthood
		// by tracing backwards until a no-parent is met; at this point only a
		// requested parent is searched and not tested whether the relations
		// are circuited among themselves
		if newParent.ParentID != uuid.Nil {
			for pg, err := m.Parent(ctx, g); err == nil && pg.ID != uuid.Nil; pg, err = m.Parent(ctx, pg) {
				// no more parents, breaking
				if pg.ParentID == uuid.Nil {
					break
				}

				// testing equality by comparing each group's ObjectID
				if pg.ID == newParent.ID {
					return ErrDuplicateParent
				}
			}
		}

		// group kind must be the same all the way back to the top
		if g.Flags != newParent.Flags {
			return ErrGroupKindMismatch
		}
	}

	// previous checks have passed, thus assingning a new parent ActorID
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

// IsAsset tests whether a given asset belongs to a given group
func (m *Manager) IsAsset(ctx context.Context, groupID uuid.UUID, asset Asset) bool {
	if groupID == uuid.Nil || asset.ID == uuid.Nil {
		return false
	}

	m.RLock()
	for _, gid := range m.assetGroups[asset] {
		if gid == groupID {
			m.RUnlock()
			return true
		}
	}
	m.RUnlock()

	return false
}

// CreateRelation adding asset to a group
// NOTE: storing relation only if group has a store set is implicit and should at least
// log/print about the occurrence
func (m *Manager) CreateRelation(ctx context.Context, rel Relation) (err error) {
	groupOrRole, err := m.GroupByID(ctx, rel.GroupID)
	if err != nil {
		return err
	}

	if rel.Asset.ID == uuid.Nil {
		return ErrNilAssetID
	}

	s, err := m.Store()
	if err != nil && err != ErrNilStore {
		return errors.Wrap(err, "failed to obtain group store")
	}

	l := m.Logger()

	if s != nil {
		l.Info("creating asset relationship",
			zap.String("group_id", rel.GroupID.String()),
			zap.String("asset_id", rel.Asset.ID.String()),
			zap.String("asset_kind", rel.Asset.Kind.String()),
			zap.String("flags", groupOrRole.Flags.Translate()),
		)

		// persisting relation in the store
		if err = s.CreateRelation(ctx, rel); err != nil {
			l.Info("failed to store asset relation",
				zap.String("group_id", rel.GroupID.String()),
				zap.String("asset_id", rel.Asset.ID.String()),
				zap.String("asset_kind", rel.Asset.Kind.String()),
				zap.String("flags", groupOrRole.Flags.Translate()),
				zap.Error(err),
			)

			return err
		}
	} else {
		l.Info("creating asset relationship while store is not set",
			zap.String("group_id", rel.GroupID.String()),
			zap.String("asset_id", rel.Asset.ID.String()),
			zap.String("asset_kind", rel.Asset.Kind.String()),
			zap.String("flags", groupOrRole.Flags.Translate()),
			zap.Error(err),
		)
	}

	// adding asset ActorID to group assets
	if err = m.LinkAsset(ctx, rel.GroupID, rel.Asset); err != nil {
		return err
	}

	return nil
}

// DeleteRelation removes asset from a group
func (m *Manager) DeleteRelation(ctx context.Context, rel Relation) (err error) {
	_, err = m.GroupByID(ctx, rel.GroupID)
	if err != nil {
		return err
	}

	if rel.Asset.ID == uuid.Nil {
		return ErrNilAssetID
	}

	// removing assetID from group assets
	if err = m.UnlinkAsset(ctx, rel.GroupID, rel.Asset); err != nil {
		return err
	}

	s, err := m.Store()
	if err != nil && err != ErrNilStore {
		return errors.Wrap(err, "failed to obtain group store")
	}

	l := m.Logger()

	if s != nil {
		m.Logger().Info("deleting asset from group",
			zap.String("group_id", rel.GroupID.String()),
			zap.String("asset_id", rel.Asset.ID.String()),
			zap.String("asset_kind", rel.Asset.Kind.String()),
		)

		// deleting a stored relation
		if err := s.DeleteRelation(ctx, rel); err != nil {
			return err
		}
	} else {
		l.Info("deleting asset from group while store is not set",
			zap.String("group_id", rel.GroupID.String()),
			zap.String("asset_id", rel.Asset.ID.String()),
			zap.String("asset_kind", rel.Asset.Kind.String()),
		)
	}

	return nil
}

// LinkAsset adds a asset to the group assets
// NOTE: does not affect the store
func (m *Manager) LinkAsset(ctx context.Context, groupID uuid.UUID, asset Asset) (err error) {
	_, err = m.GroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if asset.ID == uuid.Nil {
		return ErrNilAssetID
	}

	if m.IsAsset(ctx, groupID, asset) {
		return ErrAlreadyAsset
	}

	m.Lock()

	// group ActorID -> asset IDs
	if m.groupAssets[groupID] == nil {
		m.groupAssets[groupID] = []Asset{asset}
	} else {
		m.groupAssets[groupID] = append(m.groupAssets[groupID], asset)
	}

	// asset ActorID -> group IDs
	if m.assetGroups[asset] == nil {
		m.assetGroups[asset] = []uuid.UUID{groupID}
	} else {
		m.assetGroups[asset] = append(m.assetGroups[asset], groupID)
	}

	m.Unlock()

	return nil
}

// UnlinkAsset removes a asset from the group assets
// NOTE: does not affect the store
func (m *Manager) UnlinkAsset(ctx context.Context, groupID uuid.UUID, asset Asset) (err error) {
	_, err = m.GroupByID(ctx, groupID)
	if err != nil {
		return err
	}

	if asset.ID == uuid.Nil {
		return ErrNilAssetID
	}

	m.Lock()

	if m.groupAssets[groupID] != nil {
		for i, _asset := range m.groupAssets[groupID] {
			if _asset == _asset {
				m.groupAssets[groupID] = append(m.groupAssets[groupID][0:i], m.groupAssets[groupID][i+1:]...)
				break
			}
		}
	}

	if m.assetGroups[asset] != nil {
		for i, gid := range m.assetGroups[asset] {
			if gid == groupID {
				m.assetGroups[asset] = append(m.assetGroups[asset][0:i], m.assetGroups[asset][i+1:]...)
				break
			}
		}
	}

	m.Unlock()

	return nil
}

// Invite an existing user to become a asset of the group
// NOTE: this is optional and often can be disabled for better control
// TODO: requires careful planning
func (m *Manager) Invite(ctx context.Context, groupID uuid.UUID, asset Asset) (err error) {
	// TODO: implement

	panic("not implemented")

	return nil
}
