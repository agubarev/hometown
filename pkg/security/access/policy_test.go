package access_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/security/access"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"golang.org/x/tools/refactor/rename"
)

func TestNewAccessPolicy(t *testing.T) {
	a := assert.New(t)

	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	p, err := m.Create(
		ctx,
		access.Key("test_key"), // key
		uuid.Nil,               // owner
		uuid.Nil,               // parent
		access.NewObject(uuid.Nil, access.TObjectName{}),
		0, // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.Equal(access.Key("test_key"), p.Key)
	a.Zero(p.OwnerID)
	a.Zero(p.ParentID)
	a.Zero(p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	ownerID := uuid.Nil

	p, err = m.Create(
		ctx,
		access.Key("test_key2"), // key
		ownerID,                 // owner
		uuid.Nil,                // parent
		access.NewObject(uuid.Nil, access.TObjectName{}),
		0, // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.Equal(access.Key("test_key2"), p.Key)
	a.Equal(ownerID, p.OwnerID)
	a.Zero(p.ParentID)
	a.Zero(p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	// with parent
	pWithParent, err := m.Create(
		ctx,
		access.Key("test_key3"), // key
		ownerID,                 // owner
		p.ID,                    // parent
		access.NewObject(uuid.Nil, access.TObjectName{}),
		0, // flags
	)
	a.NoError(err)
	a.NotNil(pWithParent)
	a.Equal(access.Key("test_key3"), pWithParent.Key)
	a.Equal(ownerID, pWithParent.OwnerID)
	a.Equal(p.ID, pWithParent.ParentID)
	a.False(pWithParent.IsInherited())
	a.False(pWithParent.IsExtended())

	objectID := uuid.Nil

	// with inheritance (without a parent)
	_, err = m.Create(
		ctx,
		access.Key("test_key4"), // key
		ownerID,                 // owner
		uuid.Nil,                // parent
		access.NewObject(objectID, access.ObjectName("test object")),
		access.FInherit, // flags
	)
	a.Error(err)

	// with extension (without a parent)
	_, err = m.Create(
		ctx,
		access.Key("test_key5"), // key
		ownerID,                 // owner
		uuid.Nil,                // parent
		access.NewObject(objectID, access.ObjectName("test object")),
		access.FExtend, // flags
	)
	a.Error(err)

	// with inheritance (with a parent)
	pInheritedWithParent, err := m.Create(
		ctx,
		access.Key("test_key6"), // key
		ownerID,                 // owner
		p.ID,                    // parent
		access.NewObject(objectID, access.ObjectName("test object")),
		access.FInherit, // flags
	)
	a.NoError(err)
	a.NotNil(pInheritedWithParent)

	// with extension (with a parent)
	pExtendedWithParent, err := m.Create(
		ctx,
		access.Key("test_key7"), // key
		ownerID,                 // owner
		p.ID,                    // parent
		access.NewObject(objectID, access.ObjectName("another test object")),
		access.FExtend, // flags
	)
	a.NoError(err)
	a.NotNil(pExtendedWithParent)
}

func TestSetPublicRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	wantedRights := access.APView | access.APChange
	ownerID := uuid.Nil

	pact := access.PublicActor()
	act1 := access.UserActor(uuid.New())

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	//---------------------------------------------------------------------------
	p, err := m.Create(
		ctx,
		access.Key("test_key"), // key
		ownerID,                // owner
		uuid.Nil,               // parent
		access.NewObject(uuid.Nil, access.TObjectName{}),
		0, // flags
	)
	a.NoError(err)

	// obtaining corresponding policy roster
	roster, err := m.RosterByPolicyID(ctx, p.ID)
	a.NoError(err)
	a.NotNil(roster)

	a.NoError(m.GrantPublicAccess(ctx, p.ID, act1, wantedRights))
	a.NoError(m.Update(ctx, p))
	a.Equal(wantedRights, roster.Everyone)
	a.True(m.HasRights(ctx, p.ID, pact, wantedRights))
	a.True(m.HasRights(ctx, p.ID, act1, wantedRights))

	//---------------------------------------------------------------------------
	// with parent and inheritance only
	//---------------------------------------------------------------------------
	pWithInheritance, err := m.Create(
		ctx,
		access.Key("test_key_w_inheritance"), // key
		ownerID,                              // owner
		p.ID,                                 // parent
		access.NewObject(uuid.Nil, access.TObjectName{}),
		access.FInherit, // flags
	)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)

	// obtaining corresponding policy roster
	parentRoster, err := m.RosterByPolicyID(ctx, pWithInheritance.ID)
	a.NoError(err)
	a.NotNil(parentRoster)

	parent, err := m.PolicyByID(ctx, pWithInheritance.ParentID)
	a.NoError(err)
	a.True(m.HasRights(ctx, pWithInheritance.ID, act1, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, inheritance false, extend true; using parent's rights no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		access.TKey{}, // key
		uuid.Nil,      // owner
		parent.ID,     // parent
		access.NewObject(uuid.New(), access.ObjectName("some object")),
		access.FExtend, // flags
	)
	a.NoError(err)

	// obtaining corresponding policy roster
	parentRoster, err = m.RosterByPolicyID(ctx, pExtendedNoOwn.ParentID)
	a.NoError(err)
	a.NotNil(parentRoster)

	parent, err = m.PolicyByID(ctx, pExtendedNoOwn.ParentID)
	a.NoError(err)
	a.True(m.HasRights(ctx, parent.ID, act1, wantedRights))
	a.True(m.HasRights(ctx, pExtendedNoOwn.ID, act1, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, inherit false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		access.TKey{}, // key
		1,             // owner
		parent.ID,     // parent
		access.NewObject(uuid.New(), access.ObjectName("and another object")),
		access.FExtend, // flags
	)
	a.NoError(err)
	a.NoError(m.GrantPublicAccess(ctx, pExtendedWithOwn.ID, act1, wantedRights|access.APMove))
	a.NoError(m.Update(ctx, pExtendedWithOwn))

	parent, err = m.PolicyByID(ctx, pExtendedWithOwn.ParentID)
	a.NoError(err)

	roster, err = m.RosterByPolicyID(ctx, pExtendedWithOwn.ID)
	a.NoError(err)
	a.NotNil(roster)

	parentRoster, err = m.RosterByPolicyID(ctx, parent.ID)
	a.NoError(err)
	a.NotNil(parentRoster)

	a.Equal(wantedRights, parentRoster.Everyone)
	a.True(m.HasRights(ctx, pExtendedWithOwn.ID, act1, wantedRights|access.APMove))
}

func TestSetGroupRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	wantedRights := access.APView | access.APChange
	ownerID := uuid.New()

	pact := access.PublicActor()
	act1 := access.UserActor(uuid.New())
	act2 := access.UserActor(uuid.New())

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	// WARNING: "p" will be reused and inherited below in this function
	//---------------------------------------------------------------------------
	basePolicy, err := m.Create(
		ctx,
		access.Key("parent"), // key
		ownerID,              // owner
		uuid.Nil,             // parent
		access.NilObject(),
		0, // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// adding the user to 2 groups but setting rights to only one
	//---------------------------------------------------------------------------
	// group 1
	g1, err := gm.Create(ctx, group.FGroup, uuid.Nil, group.Key("test_group_1"), group.Name("test group 1"))
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, group.NewRelation(g1.ID, group.AKUser, act1.ID)))
	a.NoError(gm.CreateRelation(ctx, group.NewRelation(g1.ID, group.AKUser, act2.ID)))

	// group 2
	g2, err := gm.Create(ctx, group.FGroup, uuid.Nil, group.Key("test_group_2"), group.Name("test group 2"))
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, group.NewRelation(g2.ID, group.AKUser, act1.ID)))

	// assigning wanted rights to the first group (1)
	a.NoError(m.GrantGroupAccess(ctx, basePolicy.ID, act1, g1.ID, wantedRights))
	a.NoError(m.Update(ctx, basePolicy))
	a.True(m.HasRights(ctx, basePolicy.ID, access.GroupActor(g1.ID), wantedRights))

	//---------------------------------------------------------------------------
	// now with parent, using inheritance only, no extension
	// NOTE: using previously created policy "p" as a parent here
	// NOTE: not setting it's own rights as it must inherit them from a parent
	//---------------------------------------------------------------------------
	pWithInherit, err := m.Create(
		ctx,
		access.Key("with inherit"), // key
		uuid.Nil,                   // owner
		basePolicy.ID,              // parent
		access.NilObject(),
		access.FInherit, // flags
	)
	a.NoError(err)

	// checking rights on both parent and its child
	a.True(m.HasRights(ctx, basePolicy.ID, act2, wantedRights))
	a.True(m.HasRights(ctx, pWithInherit.ID, act2, wantedRights))

	//---------------------------------------------------------------------------
	// + with parent
	// - without inherit
	// + with extend
	// NOTE: using parent's rights, no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		access.Key("with extend, no own rights"), // key
		uuid.Nil,                                 // owner
		basePolicy.ID,                            // parent
		access.NilObject(),
		access.FExtend, // flags
	)
	a.NoError(err)

	// checking rights on the extended policy
	a.True(m.HasRights(ctx, basePolicy.ID, act2, wantedRights))
	a.True(m.HasRights(ctx, pExtendedNoOwn.ID, act2, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, inheritance false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		access.Key("with extend and own rights"), // key
		uuid.Nil,                                 // owner
		basePolicy.ID,                            // parent
		access.NilObject(),
		access.FExtend, // flags
	)
	a.NoError(err)

	// setting own rights for this policy
	a.NoError(m.GrantUserAccess(ctx, pExtendedWithOwn.ID, act1, act2.ID, access.APMove))
	a.NoError(m.Update(ctx, pExtendedWithOwn))

	// expecting a proper blend with parent rights
	a.True(m.HasRights(ctx, basePolicy.ID, act2, wantedRights))
	a.True(m.HasRights(ctx, pExtendedWithOwn.ID, act2, wantedRights))
}

func TestSetRoleRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	wantedRights := access.APChange | access.APDelete
	ownerID := uuid.New()

	act1 := access.UserActor(uuid.New())
	act2 := access.UserActor(uuid.New())

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	// WARNING: "p" will be reused and inherited below in this function
	//---------------------------------------------------------------------------
	basePolicy, err := m.Create(
		ctx,
		access.Key("parent"), // key
		ownerID,              // owner
		uuid.Nil,             // parent
		access.NilObject(),
		0, // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// adding the user to 2 groups but setting rights to only one
	//---------------------------------------------------------------------------
	// role 1
	r1, err := gm.Create(ctx, group.FRole, uuid.Nil, group.Key("test_group_1"), group.Name("test group 1"))
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, group.NewRelation(r1.ID, group.AKUser, act1.ID)))
	a.NoError(gm.CreateRelation(ctx, group.NewRelation(r1.ID, group.AKUser, act2.ID)))

	// role 2
	r2, err := gm.Create(ctx, group.FRole, uuid.Nil, group.Key("test_group_2"), group.Name("test group 2"))
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, group.NewRelation(r2.ID, group.AKUser, act1.ID)))

	// assigning wanted rights to the first role (1)
	a.NoError(m.GrantRoleAccess(ctx, basePolicy.ID, act1, r1.ID, wantedRights))
	a.NoError(m.Update(ctx, basePolicy))
	a.True(m.HasRights(ctx, basePolicy.ID, access.RoleActor(r1.ID), wantedRights))

	//---------------------------------------------------------------------------
	// now with parent, using inheritance only, no extension
	// NOTE: using previously created policy "p" as a parent here
	// NOTE: not setting it's own rights as it must inherit them from a parent
	//---------------------------------------------------------------------------
	pWithInherit, err := m.Create(
		ctx,
		access.Key("with inherit"), // key
		ownerID,                    // owner
		basePolicy.ID,              // parent
		access.NilObject(),
		access.FInherit, // flags
	)
	a.NoError(err)

	// checking rights on both parent and its child
	a.True(m.HasRights(ctx, basePolicy.ID, act2, wantedRights))
	a.True(m.HasRights(ctx, pWithInherit.ID, act2, wantedRights))

	//---------------------------------------------------------------------------
	// + with parent
	// - without inherit
	// + with extend
	// NOTE: using parent's rights, no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		access.Key("with extend, no own rights"), // key
		ownerID,                                  // owner
		basePolicy.ID,                            // parent
		access.NilObject(),
		access.FExtend, // flags
	)
	a.NoError(err)

	// checking rights on the extended policy
	a.True(m.HasRights(ctx, basePolicy.ID, act2, wantedRights))
	a.True(m.HasRights(ctx, pExtendedNoOwn.ID, act2, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, inheritance false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		access.Key("with extend and own rights"), // key
		ownerID,                                  // owner
		basePolicy.ID,                            // parent
		access.NilObject(),
		access.FExtend, // flags
	)
	a.NoError(err)

	// setting own rights for this policy
	a.NoError(m.GrantUserAccess(ctx, pExtendedWithOwn.ID, act1, act2.ID, access.APCopy))
	a.NoError(m.Update(ctx, pExtendedWithOwn))

	// expecting a proper blend with parent rights
	a.True(m.HasRights(ctx, basePolicy.ID, act2, access.APCopy))
	a.True(m.HasRights(ctx, pExtendedWithOwn.ID, act2, access.APCopy))
}

func TestSetUserRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	wantedRights := access.APChange | access.APDelete

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	// WARNING: "p" will be reused and inherited below in this function
	//---------------------------------------------------------------------------
	basePolicy, err := m.Create(
		ctx,
		access.Key("base policy"), // key
		1,                         // owner
		0,                         // parent
		0,                         // object ActorID
		access.TObjectName{},      // object type
		0,                         // flags
	)
	a.NoError(err)
	a.NoError(m.GrantUserAccess(ctx, basePolicy.ID, 1, 2, wantedRights))
	a.NoError(m.Update(ctx, basePolicy))
	a.True(m.HasRights(ctx, basePolicy.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, basePolicy.ID, access.AUser, 2))

	//---------------------------------------------------------------------------
	// with parent, using inheritance only
	// not setting it's own rights as it must inherit them from a parent
	//---------------------------------------------------------------------------
	pWithInheritance, err := m.Create(
		ctx,
		access.Key("inheritance only"), // key
		0,                              // owner
		basePolicy.ID,                  // parent
		0,                              // object ActorID
		access.TObjectName{},           // object type
		access.FInherit,                // flags
	)
	a.NoError(err)
	a.True(m.HasRights(ctx, basePolicy.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, pWithInheritance.ID, access.AUser, 2))
	a.False(m.HasRights(ctx, pWithInheritance.ID, access.AUser, 3))

	//---------------------------------------------------------------------------
	// with parent, legacy false, extend true; using parent's rights through
	// extension; no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		access.Key("extension only"), // key
		0,                            // owner
		basePolicy.ID,                // parent
		0,                            // object ActorID
		access.TObjectName{},         // object type
		access.FExtend,               // flags
	)
	a.NoError(err)
	a.True(m.HasRights(ctx, basePolicy.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, pExtendedNoOwn.ID, access.AUser, 2))
	a.False(m.HasRights(ctx, pExtendedNoOwn.ID, access.AUser, 3))

	//---------------------------------------------------------------------------
	// with parent, no inherit, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		access.Key("extension with own rights"), // key
		0,                                       // owner
		basePolicy.ID,                           // parent
		0,                                       // object ActorID
		access.TObjectName{},                    // object type
		access.FExtend,                          // flags
	)
	a.NoError(err)
	a.NoError(m.GrantUserAccess(ctx, pExtendedWithOwn.ID, 1, 2, wantedRights|access.APMove))
	a.NoError(m.Update(ctx, pExtendedWithOwn))
	a.True(m.HasRights(ctx, basePolicy.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, pExtendedWithOwn.ID, access.AUser, 2))
	a.False(m.HasRights(ctx, pExtendedWithOwn.ID, access.AUser, 3))
	a.False(m.HasRights(ctx, pExtendedWithOwn.ID, access.AUser, 3))
}

func TestIsOwner(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// ap store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	//---------------------------------------------------------------------------
	// test policy
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		access.Key("test policy"), // key
		1,                         // owner
		0,                         // parent
		0,                         // object ActorID
		access.TObjectName{},      // object type
		0,                         // flags
	)
	a.NoError(err)

	// testing owner and testuser rights
	a.NoError(m.GrantUserAccess(ctx, ap.ID, 1, 2, access.APView))
	a.NoError(m.Update(ctx, ap))

	// user 1 (owner)
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 1))

	// user 2
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.AUser, 2))

	a.True(ap.IsOwner(1))
	a.False(ap.IsOwner(2))
}

func TestAccessPolicyTestRosterBackup(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// ap store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	//---------------------------------------------------------------------------
	// test policy
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		access.Key("test policy"), // key
		1,                         // owner
		0,                         // parent
		0,                         // object ActorID
		access.TObjectName{},      // object type
		0,                         // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// setting base rights and saving them safely
	//---------------------------------------------------------------------------
	a.NoError(m.GrantUserAccess(ctx, ap.ID, 1, 2, access.APView|access.APChange))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 2))

	//---------------------------------------------------------------------------
	// setting additional rights correctly and incorrectly
	// NOTE: any faulty assignment MUST restore backup to the last
	// safe point of when it was either loaded or safely stored
	//---------------------------------------------------------------------------
	// assigning a few rights but not updating the store
	a.NoError(m.GrantUserAccess(ctx, ap.ID, 1, 2, access.APView|access.APChange|access.APDelete))

	// testing the rights which haven't been persisted yet
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.True(ap.IsOwner(1))
	a.False(ap.IsOwner(2))

	// attempting to set rights as a second user to a third
	// NOTE: must fail and restore backup, clearing out any changes made
	// WARNING: THIS MUST FAIL AND RESTORE BACKUP, ROLLING BACK ANY CHANGES ON THE POLICY ROSTER
	a.EqualError(m.GrantUserAccess(ctx, ap.ID, 2, 3, access.APChange), access.ErrExcessOfRights.Error())

	// NOTE: this update must be harmless, because there must be nothing to rosterChange
	a.NoError(m.Update(ctx, ap))

	// checking whether the rights were returned to its previous state
	// user 1
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 1))

	// user 2
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.AUser, 2))

	// user 3 (assignment to this user must've provoked the restoration)
	a.False(m.HasRights(ctx, ap.ID, access.AUser, 3))

	// just in case
	a.True(ap.IsOwner(1))
	a.False(ap.IsOwner(2))
	a.False(ap.IsOwner(3))
}

func TestAccessPolicyUnsetRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// ap store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	//---------------------------------------------------------------------------
	// creating test role and a group
	//---------------------------------------------------------------------------
	// role
	r, err := gm.Create(ctx, group.FRole, 0, "test_role", "test role")
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, r.ID, 1))

	// group
	g, err := gm.Create(ctx, group.FGroup, 0, "test_group", "test group")
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, g.ID, 1))

	wantedRights := access.APView | access.APChange | access.APCopy | access.APDelete

	//---------------------------------------------------------------------------
	// test policy
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		access.Key("test policy"), // key
		1,                         // owner
		0,                         // parent
		0,                         // object ActorID
		access.TObjectName{},      // object type
		0,                         // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// public access
	//---------------------------------------------------------------------------
	// setting
	a.False(m.HasRights(ctx, ap.ID, access.AEveryone, 2))
	a.False(m.HasRights(ctx, ap.ID, access.AEveryone, 2))
	a.NoError(m.GrantPublicAccess(ctx, ap.ID, 1, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, access.AEveryone, 2))
	a.True(m.HasRights(ctx, ap.ID, access.AEveryone, 2))

	// unsetting
	a.NoError(m.RevokeAccess(ctx, access.AEveryone, ap.ID, 1, 2))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, access.AEveryone, 2))
	a.False(m.HasRights(ctx, ap.ID, access.AEveryone, 2))

	//---------------------------------------------------------------------------
	// user rights
	//---------------------------------------------------------------------------
	// setting
	a.False(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.NoError(m.GrantUserAccess(ctx, ap.ID, 1, 2, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.True(m.HasRights(ctx, ap.ID, access.AUser, 2))

	// unsetting
	a.NoError(m.RevokeAccess(ctx, access.AUser, ap.ID, 1, 2))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, access.AUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.AUser, 2))

	//---------------------------------------------------------------------------
	// role rights
	//---------------------------------------------------------------------------
	// setting
	a.False(m.HasRights(ctx, ap.ID, access.ARoleGroup, r.ID))
	a.False(m.HasRights(ctx, ap.ID, access.ARoleGroup, r.ID))
	a.NoError(m.GrantRoleAccess(ctx, ap.ID, 1, r.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, access.ARoleGroup, r.ID))
	a.True(m.HasRights(ctx, ap.ID, access.ARoleGroup, r.ID))

	// unsetting
	a.NoError(m.RevokeAccess(ctx, access.ARoleGroup, ap.ID, 1, r.ID))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, access.ARoleGroup, r.ID))
	a.False(m.HasRights(ctx, ap.ID, access.ARoleGroup, r.ID))

	//---------------------------------------------------------------------------
	// group rights
	//---------------------------------------------------------------------------
	// setting
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g.ID))
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g.ID))
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, 1, g.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, access.AGroup, g.ID))
	a.True(m.HasRights(ctx, ap.ID, access.AGroup, g.ID))

	// unsetting
	a.NoError(m.RevokeAccess(ctx, access.AGroup, ap.ID, 1, g.ID))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g.ID))
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g.ID))
}

func TestHasGroupRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	// ap store
	s, err := access.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	// group manager
	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// policy manager
	m, err := access.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	//---------------------------------------------------------------------------
	// adding the user to 2 groups but setting rights to only one
	//---------------------------------------------------------------------------
	g1, err := gm.Create(ctx, group.FGroup, 0, "test group 1", "test group 1")
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, g1.ID, 2))

	g2, err := gm.Create(ctx, group.FGroup, g1.ID, "test group 2", "test group 2")
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, g2.ID, 2))

	g3, err := gm.Create(ctx, group.FGroup, g2.ID, "test group 3", "test group 3")
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, g3.ID, 2))

	// expected rights
	wantedRights := access.APCreate | access.APView

	//---------------------------------------------------------------------------
	// setting rights only for the g1, thus g3 must inherit its
	// rights only from g1
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		access.Key("test policy"), // key
		1,                         // owner
		0,                         // parent
		0,                         // object ActorID
		access.TObjectName{},      // object type
		0,                         // flags
	)
	a.NoError(err)

	// setting rights for group 1
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, 1, g1.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, access.AGroup, g1.ID))
	a.True(m.HasRights(ctx, ap.ID, access.AGroup, g2.ID))
	a.True(m.HasRights(ctx, ap.ID, access.AGroup, g3.ID))

	//---------------------------------------------------------------------------
	// setting rights only for the second group, thus g3 must inherit its
	// rights only from group 2, and group 1 must not have the rights of group 2
	//---------------------------------------------------------------------------
	ap, err = m.Create(
		ctx,
		access.Key("test policy 2"), // key
		1,                           // owner
		0,                           // parent
		0,                           // object ActorID
		access.TObjectName{},        // object type
		0,                           // flags
	)
	a.NoError(err)

	// setting rights for group 2
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, 1, g2.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g1.ID))
	a.True(m.HasRights(ctx, ap.ID, access.AGroup, g2.ID))
	a.True(m.HasRights(ctx, ap.ID, access.AGroup, g3.ID))

	//---------------------------------------------------------------------------
	// setting rights only for the third group, group 1 and group 2 must not have
	// these rights
	//---------------------------------------------------------------------------
	ap, err = m.Create(
		ctx,
		access.Key("test policy 3"), // key
		1,                           // owner
		0,                           // parent
		0,                           // object ActorID
		access.TObjectName{},        // object type
		0,                           // flags
	)
	a.NoError(err)

	// setting rights for group 3
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, 1, g3.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g1.ID))
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g2.ID))
	a.True(m.HasRights(ctx, ap.ID, access.AGroup, g3.ID))

	//---------------------------------------------------------------------------
	// setting rights only for group 1 & 2, group 3 must inherit the rights
	// from its direct ancestor that has its own rights (group 2)
	//---------------------------------------------------------------------------
	ap, err = m.Create(
		ctx,
		access.Key("test policy 4"), // key
		1,                           // owner
		0,                           // parent
		0,                           // object ActorID
		access.TObjectName{},        // object type
		0,                           // flags
	)
	a.NoError(err)

	// rights to be used
	group1Rights := access.APView | access.APCreate
	wantedRights = access.APDelete | access.APCopy

	// setting rights for group 1 and 2
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, 1, g1.ID, group1Rights))
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, 1, g2.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))

	a.True(m.HasRights(ctx, ap.ID, access.AGroup, g1.ID))
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g1.ID))

	a.True(m.HasRights(ctx, ap.ID, access.AGroup, g2.ID))
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g2.ID))

	a.True(m.HasRights(ctx, ap.ID, access.AGroup, g3.ID))
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g3.ID))

	//---------------------------------------------------------------------------
	// not setting any rights, only checking
	//---------------------------------------------------------------------------
	ap, err = m.Create(
		ctx,
		access.Key("test policy 5"), // key
		1,                           // owner
		0,                           // parent
		0,                           // object ActorID
		access.TObjectName{},        // object type
		0,                           // flags
	)
	a.NoError(err)

	// test rights
	wantedRights = access.APCreate | access.APView

	// all check results must be false
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g1.ID))
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g2.ID))
	a.False(m.HasRights(ctx, ap.ID, access.AGroup, g3.ID))
}
