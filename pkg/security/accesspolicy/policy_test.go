package accesspolicy_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/security/accesspolicy"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	p, err := m.Create(
		ctx,
		accesspolicy.Key("test_key"), // key
		uuid.Nil,                     // owner
		uuid.Nil,                     // parent
		accesspolicy.NewObject(uuid.Nil, accesspolicy.TObjectName{}),
		0, // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.Equal(accesspolicy.Key("test_key"), p.Key)
	a.Zero(p.OwnerID)
	a.Zero(p.ParentID)
	a.Zero(p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	ownerID := uuid.Nil

	p, err = m.Create(
		ctx,
		accesspolicy.Key("test_key2"), // key
		ownerID,                       // owner
		uuid.Nil,                      // parent
		accesspolicy.NewObject(uuid.Nil, accesspolicy.TObjectName{}),
		0, // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.Equal(accesspolicy.Key("test_key2"), p.Key)
	a.Equal(ownerID, p.OwnerID)
	a.Zero(p.ParentID)
	a.Zero(p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	// with parent
	pWithParent, err := m.Create(
		ctx,
		accesspolicy.Key("test_key3"), // key
		ownerID,                       // owner
		p.ID,                          // parent
		accesspolicy.NewObject(uuid.Nil, accesspolicy.TObjectName{}),
		0, // flags
	)
	a.NoError(err)
	a.NotNil(pWithParent)
	a.Equal(accesspolicy.Key("test_key3"), pWithParent.Key)
	a.Equal(ownerID, pWithParent.OwnerID)
	a.Equal(p.ID, pWithParent.ParentID)
	a.False(pWithParent.IsInherited())
	a.False(pWithParent.IsExtended())

	// with inheritance (without a parent)
	_, err = m.Create(
		ctx,
		accesspolicy.Key("test_key4"), // key
		ownerID,                       // owner
		uuid.Nil,                      // parent
		accesspolicy.NewObject(uuid.New(), accesspolicy.ObjectName("test object")),
		accesspolicy.FInherit, // flags
	)
	a.Error(err)

	// with extension (without a parent)
	_, err = m.Create(
		ctx,
		accesspolicy.Key("test_key5"), // key
		ownerID,                       // owner
		uuid.Nil,                      // parent
		accesspolicy.NewObject(uuid.New(), accesspolicy.ObjectName("test object")),
		accesspolicy.FExtend, // flags
	)
	a.Error(err)

	// with inheritance (with a parent)
	pInheritedWithParent, err := m.Create(
		ctx,
		accesspolicy.Key("test_key6"), // key
		ownerID,                       // owner
		p.ID,                          // parent
		accesspolicy.NewObject(uuid.New(), accesspolicy.ObjectName("test object")),
		accesspolicy.FInherit, // flags
	)
	a.NoError(err)
	a.NotNil(pInheritedWithParent)

	// with extension (with a parent)
	pExtendedWithParent, err := m.Create(
		ctx,
		accesspolicy.Key("test_key7"), // key
		ownerID,                       // owner
		p.ID,                          // parent
		accesspolicy.NewObject(uuid.New(), accesspolicy.ObjectName("another test object")),
		accesspolicy.FExtend, // flags
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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	wantedRights := accesspolicy.APView | accesspolicy.APChange
	ownerID := uuid.Nil

	pact := accesspolicy.PublicActor()
	act1 := accesspolicy.UserActor(uuid.New())

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	//---------------------------------------------------------------------------
	p, err := m.Create(
		ctx,
		accesspolicy.Key("test_key"), // key
		act1.ID,                      // owner
		uuid.Nil,                     // parent
		accesspolicy.NewObject(uuid.Nil, accesspolicy.TObjectName{}),
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
		accesspolicy.Key("test_key_w_inheritance"), // key
		ownerID, // owner
		p.ID,    // parent
		accesspolicy.NewObject(uuid.Nil, accesspolicy.TObjectName{}),
		accesspolicy.FInherit, // flags
	)
	// not granting it's own rights as it must inherit them from a parent
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
		accesspolicy.TKey{}, // key
		uuid.Nil,            // owner
		parent.ID,           // parent
		accesspolicy.NewObject(uuid.New(), accesspolicy.ObjectName("some object")),
		accesspolicy.FExtend, // flags
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
		accesspolicy.TKey{}, // key
		act1.ID,             // owner
		parent.ID,           // parent
		accesspolicy.NewObject(uuid.New(), accesspolicy.ObjectName("and another object")),
		accesspolicy.FExtend, // flags
	)
	a.NoError(err)
	a.NoError(m.GrantPublicAccess(ctx, pExtendedWithOwn.ID, act1, wantedRights|accesspolicy.APMove))
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
	a.True(m.HasRights(ctx, pExtendedWithOwn.ID, act1, wantedRights|accesspolicy.APMove))
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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	wantedRights := accesspolicy.APView | accesspolicy.APChange

	act1 := accesspolicy.UserActor(uuid.New())
	act2 := accesspolicy.UserActor(uuid.New())

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	// WARNING: "p" will be reused and inherited below in this function
	//---------------------------------------------------------------------------
	basePolicy, err := m.Create(
		ctx,
		accesspolicy.Key("parent"), // key
		act1.ID,                    // owner
		uuid.Nil,                   // parent
		accesspolicy.NilObject(),
		0, // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// adding the user to 2 groups but granting rights to only one
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
	a.True(m.HasRights(ctx, basePolicy.ID, accesspolicy.GroupActor(g1.ID), wantedRights))

	//---------------------------------------------------------------------------
	// now with parent, using inheritance only, no extension
	// NOTE: using previously created policy "p" as a parent here
	// NOTE: not granting it's own rights as it must inherit them from a parent
	//---------------------------------------------------------------------------
	pWithInherit, err := m.Create(
		ctx,
		accesspolicy.Key("with inherit"), // key
		act1.ID,                          // owner
		basePolicy.ID,                    // parent
		accesspolicy.NilObject(),
		accesspolicy.FInherit, // flags
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
		accesspolicy.Key("with extend, no own rights"), // key
		act1.ID,       // owner
		basePolicy.ID, // parent
		accesspolicy.NilObject(),
		accesspolicy.FExtend, // flags
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
		accesspolicy.Key("with extend and own rights"), // key
		act1.ID,       // owner
		basePolicy.ID, // parent
		accesspolicy.NilObject(),
		accesspolicy.FExtend, // flags
	)
	a.NoError(err)

	// granting own rights for this policy
	a.NoError(m.GrantUserAccess(ctx, pExtendedWithOwn.ID, act1, act2.ID, accesspolicy.APMove))
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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	wantedRights := accesspolicy.APChange | accesspolicy.APDelete

	act1 := accesspolicy.UserActor(uuid.New())
	act2 := accesspolicy.UserActor(uuid.New())

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	// WARNING: "p" will be reused and inherited below in this function
	//---------------------------------------------------------------------------
	basePolicy, err := m.Create(
		ctx,
		accesspolicy.Key("parent"), // key
		act1.ID,                    // owner
		uuid.Nil,                   // parent
		accesspolicy.NilObject(),
		0, // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// adding the user to 2 groups but granting rights to only one
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
	a.True(m.HasRights(ctx, basePolicy.ID, accesspolicy.RoleActor(r1.ID), wantedRights))

	//---------------------------------------------------------------------------
	// now with parent, using inheritance only, no extension
	// NOTE: using previously created policy "p" as a parent here
	// NOTE: not granting it's own rights as it must inherit them from a parent
	//---------------------------------------------------------------------------
	pWithInherit, err := m.Create(
		ctx,
		accesspolicy.Key("with inherit"), // key
		act1.ID,                          // owner
		basePolicy.ID,                    // parent
		accesspolicy.NilObject(),
		accesspolicy.FInherit, // flags
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
		accesspolicy.Key("with extend, no own rights"), // key
		act1.ID,       // owner
		basePolicy.ID, // parent
		accesspolicy.NilObject(),
		accesspolicy.FExtend, // flags
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
		accesspolicy.Key("with extend and own rights"), // key
		act1.ID,       // owner
		basePolicy.ID, // parent
		accesspolicy.NilObject(),
		accesspolicy.FExtend, // flags
	)
	a.NoError(err)

	// granting own rights for this policy
	a.NoError(m.GrantUserAccess(ctx, pExtendedWithOwn.ID, act1, act2.ID, accesspolicy.APCopy))
	a.NoError(m.Update(ctx, pExtendedWithOwn))

	// expecting a proper blend with parent rights
	a.True(m.HasRights(ctx, basePolicy.ID, act2, accesspolicy.APChange))
	a.True(m.HasRights(ctx, pExtendedWithOwn.ID, act2, accesspolicy.APDelete))
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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	wantedRights := accesspolicy.APChange | accesspolicy.APDelete

	act1 := accesspolicy.UserActor(uuid.New())
	act2 := accesspolicy.UserActor(uuid.New())
	act3 := accesspolicy.UserActor(uuid.New())

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	// WARNING: "p" will be reused and inherited below in this function
	//---------------------------------------------------------------------------
	basePolicy, err := m.Create(
		ctx,
		accesspolicy.Key("base policy"), // key
		act1.ID,                         // owner
		uuid.Nil,                        // parent
		accesspolicy.NilObject(),
		0, // flags
	)
	a.NoError(err)
	a.NoError(m.GrantUserAccess(ctx, basePolicy.ID, act1, act2.ID, wantedRights))
	a.NoError(m.Update(ctx, basePolicy))
	a.True(m.HasRights(ctx, basePolicy.ID, act1, wantedRights))
	a.True(m.HasRights(ctx, basePolicy.ID, act2, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, using inheritance only
	// not granting it's own rights as it must inherit them from a parent
	//---------------------------------------------------------------------------
	pWithInheritance, err := m.Create(
		ctx,
		accesspolicy.Key("inheritance only"), // key
		act1.ID,                              // owner
		basePolicy.ID,                        // parent
		accesspolicy.NilObject(),
		accesspolicy.FInherit, // flags
	)
	a.NoError(err)
	a.True(m.HasRights(ctx, basePolicy.ID, act1, wantedRights))
	a.True(m.HasRights(ctx, pWithInheritance.ID, act2, wantedRights))
	a.False(m.HasRights(ctx, pWithInheritance.ID, act3, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, legacy false, extend true; using parent's rights through
	// extension; no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		accesspolicy.Key("extension only"), // key
		act1.ID,                            // owner
		basePolicy.ID,                      // parent
		accesspolicy.NilObject(),
		accesspolicy.FExtend, // flags
	)
	a.NoError(err)
	a.True(m.HasRights(ctx, basePolicy.ID, act1, wantedRights))
	a.True(m.HasRights(ctx, pExtendedNoOwn.ID, act2, wantedRights))
	a.False(m.HasRights(ctx, pExtendedNoOwn.ID, act3, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, no inherit, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		accesspolicy.Key("extension with own rights"), // key
		act1.ID,       // owner
		basePolicy.ID, // parent
		accesspolicy.NilObject(),
		accesspolicy.FExtend, // flags
	)
	a.NoError(err)
	a.NoError(m.GrantUserAccess(ctx, pExtendedWithOwn.ID, act1, act2.ID, wantedRights|accesspolicy.APMove))
	a.NoError(m.Update(ctx, pExtendedWithOwn))
	a.True(m.HasRights(ctx, basePolicy.ID, act1, wantedRights|accesspolicy.APMove))
	a.True(m.HasRights(ctx, pExtendedWithOwn.ID, act2, wantedRights|accesspolicy.APMove))
	a.False(m.HasRights(ctx, pExtendedWithOwn.ID, act3, wantedRights))
	a.False(m.HasRights(ctx, pExtendedWithOwn.ID, act3, accesspolicy.APMove))
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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	act1 := accesspolicy.UserActor(uuid.New())
	act2 := accesspolicy.UserActor(uuid.New())

	//---------------------------------------------------------------------------
	// test policy
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		accesspolicy.Key("test policy"), // key
		act1.ID,                         // owner
		uuid.Nil,                        // parent
		accesspolicy.NilObject(),
		0, // flags
	)
	a.NoError(err)

	// testing owner and testuser rights
	a.NoError(m.GrantUserAccess(ctx, ap.ID, act1, act2.ID, accesspolicy.APView))
	a.NoError(m.Update(ctx, ap))

	// user 1 (owner)
	a.True(m.HasRights(ctx, ap.ID, act1, accesspolicy.APView))
	a.True(m.HasRights(ctx, ap.ID, act1, accesspolicy.APMove))
	a.True(m.HasRights(ctx, ap.ID, act1, accesspolicy.APDelete))
	a.True(m.HasRights(ctx, ap.ID, act1, accesspolicy.APFullAccess))

	// user 2
	a.True(m.HasRights(ctx, ap.ID, act2, accesspolicy.APView))
	a.False(m.HasRights(ctx, ap.ID, act2, accesspolicy.APMove))
	a.False(m.HasRights(ctx, ap.ID, act2, accesspolicy.APDelete))
	a.False(m.HasRights(ctx, ap.ID, act2, accesspolicy.APFullAccess))

	a.True(ap.IsOwner(act1.ID))
	a.False(ap.IsOwner(act2.ID))
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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	act1 := accesspolicy.UserActor(uuid.New())
	act2 := accesspolicy.UserActor(uuid.New())
	act3 := accesspolicy.UserActor(uuid.New())

	//---------------------------------------------------------------------------
	// test policy
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		accesspolicy.Key("test policy"), // key
		act1.ID,                         // owner
		uuid.Nil,                        // parent
		accesspolicy.NilObject(),
		0, // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// granting base rights and saving them safely
	//---------------------------------------------------------------------------
	a.NoError(m.GrantUserAccess(ctx, ap.ID, act1, act2.ID, accesspolicy.APView|accesspolicy.APChange))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, act2, accesspolicy.APView))
	a.True(m.HasRights(ctx, ap.ID, act2, accesspolicy.APChange))

	//---------------------------------------------------------------------------
	// granting additional rights correctly and incorrectly
	// NOTE: any faulty assignment MUST restore backup to the last
	// safe point of when it was either loaded or safely stored
	//---------------------------------------------------------------------------
	// assigning a few rights but not updating the store
	a.NoError(m.GrantUserAccess(ctx, ap.ID, act1, act2.ID, accesspolicy.APView|accesspolicy.APChange|accesspolicy.APDelete))

	// testing the rights which haven't been persisted yet
	a.True(m.HasRights(ctx, ap.ID, act1, accesspolicy.APView))
	a.True(m.HasRights(ctx, ap.ID, act1, accesspolicy.APChange))
	a.True(m.HasRights(ctx, ap.ID, act1, accesspolicy.APDelete))
	a.True(m.HasRights(ctx, ap.ID, act1, accesspolicy.APFullAccess))
	a.True(m.HasRights(ctx, ap.ID, act2, accesspolicy.APView))
	a.True(m.HasRights(ctx, ap.ID, act2, accesspolicy.APDelete))
	a.False(m.HasRights(ctx, ap.ID, act2, accesspolicy.APMove))
	a.True(ap.IsOwner(act1.ID))
	a.False(ap.IsOwner(act2.ID))

	// attempting to set rights as a second user to a third
	// NOTE: must fail and restore backup, clearing out any changes made
	// WARNING: THIS MUST FAIL AND RESTORE BACKUP, ROLLING BACK ANY CHANGES ON THE POLICY ROSTER
	a.EqualError(m.GrantUserAccess(ctx, ap.ID, act2, act3.ID, accesspolicy.APChange), accesspolicy.ErrExcessOfRights.Error())

	// NOTE: this update must be harmless, because there must be nothing to rosterChange
	a.NoError(m.Update(ctx, ap))

	// checking whether the rights were returned to its previous state
	// user 1
	a.True(m.HasRights(ctx, ap.ID, act1, accesspolicy.APView))
	a.True(m.HasRights(ctx, ap.ID, act1, accesspolicy.APChange))
	a.True(m.HasRights(ctx, ap.ID, act1, accesspolicy.APDelete))
	a.True(m.HasRights(ctx, ap.ID, act1, accesspolicy.APFullAccess))

	// user 2
	a.True(m.HasRights(ctx, ap.ID, act2, accesspolicy.APView))
	a.True(m.HasRights(ctx, ap.ID, act2, accesspolicy.APChange))
	a.False(m.HasRights(ctx, ap.ID, act2, accesspolicy.APDelete))
	a.False(m.HasRights(ctx, ap.ID, act2, accesspolicy.APMove))

	// user 3 (assignment to this user must've provoked the restoration)
	a.False(m.HasRights(ctx, ap.ID, act3, accesspolicy.APChange))

	// just in case
	a.True(ap.IsOwner(act1.ID))
	a.False(ap.IsOwner(act2.ID))
	a.False(ap.IsOwner(act3.ID))
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

	// p store
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	act1 := accesspolicy.UserActor(uuid.New())
	act2 := accesspolicy.UserActor(uuid.New())

	//---------------------------------------------------------------------------
	// creating test role and a group
	//---------------------------------------------------------------------------
	// role
	r, err := gm.Create(ctx, group.FRole, uuid.Nil, group.Key("test_role"), group.Name("test role"))
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, group.NewRelation(r.ID, group.AKUser, act1.ID)))

	// group
	g, err := gm.Create(ctx, group.FGroup, uuid.Nil, group.Key("test_group"), group.Name("test group"))
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, group.NewRelation(g.ID, group.AKUser, act1.ID)))

	wantedRights := accesspolicy.APView | accesspolicy.APChange | accesspolicy.APCopy | accesspolicy.APDelete

	//---------------------------------------------------------------------------
	// test policy
	//---------------------------------------------------------------------------
	p, err := m.Create(
		ctx,
		accesspolicy.Key("test policy"), // key
		act1.ID,                         // owner
		uuid.Nil,                        // parent
		accesspolicy.NilObject(),
		0, // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// public accesspolicy
	//---------------------------------------------------------------------------
	// granting
	a.False(m.HasRights(ctx, p.ID, accesspolicy.PublicActor(), accesspolicy.APView))
	a.False(m.HasRights(ctx, p.ID, accesspolicy.PublicActor(), accesspolicy.APChange))
	a.NoError(m.GrantPublicAccess(ctx, p.ID, act1, wantedRights))
	a.NoError(m.Update(ctx, p))
	a.True(m.HasRights(ctx, p.ID, accesspolicy.PublicActor(), accesspolicy.APView))
	a.True(m.HasRights(ctx, p.ID, accesspolicy.PublicActor(), accesspolicy.APChange))

	// revoking
	a.NoError(m.RevokeAccess(ctx, p.ID, act1, accesspolicy.PublicActor()))
	a.NoError(m.Update(ctx, p))
	a.False(m.HasRights(ctx, p.ID, accesspolicy.PublicActor(), accesspolicy.APView))
	a.False(m.HasRights(ctx, p.ID, accesspolicy.PublicActor(), accesspolicy.APChange))

	//---------------------------------------------------------------------------
	// user rights
	//---------------------------------------------------------------------------
	// granting
	a.False(m.HasRights(ctx, p.ID, act2, accesspolicy.APView))
	a.False(m.HasRights(ctx, p.ID, act2, accesspolicy.APChange))
	a.NoError(m.GrantUserAccess(ctx, p.ID, act1, act2.ID, wantedRights))
	a.NoError(m.Update(ctx, p))
	a.True(m.HasRights(ctx, p.ID, act2, accesspolicy.APView))
	a.True(m.HasRights(ctx, p.ID, act2, accesspolicy.APChange))

	// revoking
	a.NoError(m.RevokeAccess(ctx, p.ID, act1, act2))
	a.NoError(m.Update(ctx, p))
	a.False(m.HasRights(ctx, p.ID, act2, accesspolicy.APView))
	a.False(m.HasRights(ctx, p.ID, act2, accesspolicy.APChange))

	//---------------------------------------------------------------------------
	// role rights
	//---------------------------------------------------------------------------
	// granting
	a.False(m.HasRights(ctx, p.ID, accesspolicy.RoleActor(r.ID), accesspolicy.APView))
	a.False(m.HasRights(ctx, p.ID, accesspolicy.RoleActor(r.ID), accesspolicy.APChange))
	a.NoError(m.GrantRoleAccess(ctx, p.ID, act1, r.ID, wantedRights))
	a.NoError(m.Update(ctx, p))
	a.True(m.HasRights(ctx, p.ID, accesspolicy.RoleActor(r.ID), accesspolicy.APView))
	a.True(m.HasRights(ctx, p.ID, accesspolicy.RoleActor(r.ID), accesspolicy.APChange))

	// revoking
	a.NoError(m.RevokeAccess(ctx, p.ID, act1, accesspolicy.RoleActor(r.ID)))
	a.NoError(m.Update(ctx, p))
	a.False(m.HasRights(ctx, p.ID, accesspolicy.RoleActor(r.ID), accesspolicy.APView))
	a.False(m.HasRights(ctx, p.ID, accesspolicy.RoleActor(r.ID), accesspolicy.APChange))

	//---------------------------------------------------------------------------
	// group rights
	//---------------------------------------------------------------------------
	// granting
	a.False(m.HasRights(ctx, p.ID, accesspolicy.GroupActor(g.ID), accesspolicy.APView))
	a.False(m.HasRights(ctx, p.ID, accesspolicy.GroupActor(g.ID), accesspolicy.APChange))
	a.NoError(m.GrantGroupAccess(ctx, p.ID, act1, g.ID, wantedRights))
	a.NoError(m.Update(ctx, p))
	a.True(m.HasRights(ctx, p.ID, accesspolicy.GroupActor(g.ID), accesspolicy.APView))
	a.True(m.HasRights(ctx, p.ID, accesspolicy.GroupActor(g.ID), accesspolicy.APChange))

	// revoking
	a.NoError(m.RevokeAccess(ctx, p.ID, act1, accesspolicy.GroupActor(g.ID)))
	a.NoError(m.Update(ctx, p))
	a.False(m.HasRights(ctx, p.ID, accesspolicy.GroupActor(g.ID), accesspolicy.APView))
	a.False(m.HasRights(ctx, p.ID, accesspolicy.GroupActor(g.ID), accesspolicy.APChange))
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
	s, err := accesspolicy.NewPostgreSQLStore(db)
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
	m, err := accesspolicy.NewManager(s, gm)
	a.NoError(err)
	a.NotNil(m)

	act1 := accesspolicy.UserActor(uuid.New())
	act2 := accesspolicy.UserActor(uuid.New())

	//---------------------------------------------------------------------------
	// adding the user to 2 groups but granting rights to only one
	//---------------------------------------------------------------------------
	g1, err := gm.Create(ctx, group.FGroup, uuid.Nil, group.Key("test group 1"), group.Name("test group 1"))
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, group.NewRelation(g1.ID, group.AKUser, act2.ID)))

	g2, err := gm.Create(ctx, group.FGroup, g1.ID, group.Key("test group 2"), group.Name("test group 2"))
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, group.NewRelation(g2.ID, group.AKUser, act2.ID)))

	g3, err := gm.Create(ctx, group.FGroup, g2.ID, group.Key("test group 3"), group.Name("test group 3"))
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, group.NewRelation(g3.ID, group.AKUser, act2.ID)))

	// expected rights
	wantedRights := accesspolicy.APCreate | accesspolicy.APView

	//---------------------------------------------------------------------------
	// granting rights only for the g1, thus g3 must inherit its
	// rights only from g1
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		accesspolicy.Key("test policy"), // key
		act1.ID,                         // owner
		uuid.Nil,                        // parent
		accesspolicy.NilObject(),
		0, // flags
	)
	a.NoError(err)

	// granting rights for group 1
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, act1, g1.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g1.ID), wantedRights))
	a.True(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g2.ID), wantedRights))
	a.True(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g3.ID), wantedRights))

	//---------------------------------------------------------------------------
	// granting rights only for the second group, thus g3 must inherit its
	// rights only from group 2, and group 1 must not have the rights of group 2
	//---------------------------------------------------------------------------
	ap, err = m.Create(
		ctx,
		accesspolicy.Key("test policy 2"), // key
		act1.ID,                           // owner
		uuid.Nil,                          // parent
		accesspolicy.NilObject(),
		0, // flags
	)
	a.NoError(err)

	// granting rights for group 2
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, act1, g2.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g1.ID), wantedRights))
	a.True(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g2.ID), wantedRights))
	a.True(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g3.ID), wantedRights))

	//---------------------------------------------------------------------------
	// granting rights only for the third group, group 1 and group 2 must not have
	// these rights
	//---------------------------------------------------------------------------
	ap, err = m.Create(
		ctx,
		accesspolicy.Key("test policy 3"), // key
		act1.ID,                           // owner
		uuid.Nil,                          // parent
		accesspolicy.NilObject(),
		0, // flags
	)
	a.NoError(err)

	// granting rights for group 3
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, act1, g3.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g1.ID), wantedRights))
	a.False(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g2.ID), wantedRights))
	a.True(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g3.ID), wantedRights))

	//---------------------------------------------------------------------------
	// granting rights only for group 1 & 2, group 3 must inherit the rights
	// from its direct ancestor that has its own rights (group 2)
	//---------------------------------------------------------------------------
	ap, err = m.Create(
		ctx,
		accesspolicy.Key("test policy 4"), // key
		act1.ID,                           // owner
		uuid.Nil,                          // parent
		accesspolicy.NilObject(),
		0, // flags
	)
	a.NoError(err)

	// rights to be used
	group1Rights := accesspolicy.APView | accesspolicy.APCreate
	wantedRights = accesspolicy.APDelete | accesspolicy.APCopy

	// granting rights for group 1 and 2
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, act1, g1.ID, group1Rights))
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, act1, g2.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))

	a.True(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g1.ID), group1Rights))
	a.False(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g1.ID), wantedRights))

	a.True(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g2.ID), wantedRights))
	a.False(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g2.ID), group1Rights))

	a.True(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g3.ID), wantedRights))
	a.False(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g3.ID), group1Rights))

	//---------------------------------------------------------------------------
	// not granting any rights, only checking
	//---------------------------------------------------------------------------
	ap, err = m.Create(
		ctx,
		accesspolicy.Key("test policy 5"), // key
		act1.ID,                           // owner
		uuid.Nil,                          // parent
		accesspolicy.NilObject(),
		0, // flags
	)
	a.NoError(err)

	// test rights
	wantedRights = accesspolicy.APCreate | accesspolicy.APView

	// all check results must be false
	a.False(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g1.ID), wantedRights))
	a.False(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g2.ID), wantedRights))
	a.False(m.HasRights(ctx, ap.ID, accesspolicy.GroupActor(g3.ID), wantedRights))
}
