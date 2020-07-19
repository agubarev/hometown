package access_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/security/access"
	"github.com/stretchr/testify/assert"
)

func TestNewAccessPolicy(t *testing.T) {
	a := assert.New(t)

	// test context
	ctx := context.Background()

	// database instance
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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
		access.NewKey("test_key"), // key
		0,                         // owner ID
		0,                         // parent ID
		0,                         // object ID
		access.ObjectName{},       // object type
		0,                         // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.Equal(access.NewKey("test_key"), p.Key)
	a.Zero(p.OwnerID)
	a.Zero(p.ParentID)
	a.Zero(p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	p, err = m.Create(
		ctx,
		access.NewKey("test_key2"), // key
		1,                          // owner ID
		0,                          // parent ID
		0,                          // object ID
		access.ObjectName{},        // object type
		0,                          // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.Equal(access.NewKey("test_key2"), p.Key)
	a.Equal(uint32(1), p.OwnerID)
	a.Zero(p.ParentID)
	a.Zero(p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	// with parent
	pWithParent, err := m.Create(
		ctx,
		access.NewKey("test_key3"), // key
		1,                          // owner ID
		p.ID,                       // parent ID
		0,                          // object ID
		access.ObjectName{},        // object type
		0,                          // flags
	)
	a.NoError(err)
	a.NotNil(pWithParent)
	a.Equal(access.NewKey("test_key3"), pWithParent.Key)
	a.Equal(uint32(1), pWithParent.OwnerID)
	a.Equal(p.ID, pWithParent.ParentID)
	a.False(pWithParent.IsInherited())
	a.False(pWithParent.IsExtended())

	// with inheritance (without a parent)
	_, err = m.Create(
		ctx,
		access.NewKey("test_key4"),          // key
		1,                                   // owner ID
		0,                                   // parent ID
		1,                                   // object ID
		access.NewObjectName("test object"), // object type
		access.FInherit,                     // flags
	)
	a.Error(err)

	// with extension (without a parent)
	_, err = m.Create(
		ctx,
		access.NewKey("test_key5"),          // key
		1,                                   // owner ID
		0,                                   // parent ID
		1,                                   // object ID
		access.NewObjectName("test object"), // object type
		access.FExtend,                      // flags
	)
	a.Error(err)

	// with inheritance (with a parent)
	pInheritedWithParent, err := m.Create(
		ctx,
		access.NewKey("test_key6"),          // key
		1,                                   // owner ID
		p.ID,                                // parent ID
		1,                                   // object ID
		access.NewObjectName("test object"), // object type
		access.FInherit,                     // flags
	)
	a.NoError(err)
	a.NotNil(pInheritedWithParent)

	// with extension (with a parent)
	pExtendedWithParent, err := m.Create(
		ctx,
		access.NewKey("test_key7"), // key
		1,                          // owner ID
		p.ID,                       // parent ID
		1,                          // object ID
		access.NewObjectName("another test object"), // object type
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
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	//---------------------------------------------------------------------------
	p, err := m.Create(
		ctx,
		access.NewKey("test_key"), // key
		1,                         // owner ID
		0,                         // parent ID
		0,                         // object ID
		access.ObjectName{},       // object type
		0,                         // flags
	)
	a.NoError(err)

	// obtaining corresponding policy roster
	roster, err := m.RosterByPolicyID(ctx, p.ID)
	a.NoError(err)
	a.NotNil(roster)

	a.NoError(m.GrantPublicAccess(ctx, p.ID, 1, wantedRights))
	a.NoError(m.Update(ctx, p))
	a.Equal(wantedRights, roster.Everyone)
	a.True(m.HasRights(ctx, p.ID, access.SKEveryone, 1))

	//---------------------------------------------------------------------------
	// with parent and inheritance only
	//---------------------------------------------------------------------------
	pWithInheritance, err := m.Create(
		ctx,
		access.NewKey("test_key_w_inheritance"), // key
		1,                                       // owner ID
		p.ID,                                    // parent ID
		0,                                       // object ID
		access.ObjectName{},                     // object type
		access.FInherit,                         // flags
	)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)

	// obtaining corresponding policy roster
	parentRoster, err := m.RosterByPolicyID(ctx, pWithInheritance.ID)
	a.NoError(err)
	a.NotNil(parentRoster)

	parent, err := m.PolicyByID(ctx, pWithInheritance.ParentID)
	a.NoError(err)
	a.True(m.HasRights(ctx, pWithInheritance.ID, access.SKUser, 1))

	//---------------------------------------------------------------------------
	// with parent, inheritance false, extend true; using parent's rights no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		access.Key{},                        // key
		0,                                   // owner ID
		parent.ID,                           // parent ID
		1,                                   // object ID
		access.NewObjectName("some object"), // object type
		access.FExtend,                      // flags
	)
	a.NoError(err)

	// obtaining corresponding policy roster
	parentRoster, err = m.RosterByPolicyID(ctx, pExtendedNoOwn.ParentID)
	a.NoError(err)
	a.NotNil(parentRoster)

	parent, err = m.PolicyByID(ctx, pExtendedNoOwn.ParentID)
	a.NoError(err)
	a.True(m.HasRights(ctx, parent.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, pExtendedNoOwn.ID, access.SKUser, 1))

	//---------------------------------------------------------------------------
	// with parent, inherit false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		access.Key{}, // key
		1,            // owner ID
		parent.ID,    // parent ID
		1,            // object ID
		access.NewObjectName("and another object"), // object type
		access.FExtend, // flags
	)
	a.NoError(err)
	a.NoError(m.GrantPublicAccess(ctx, pExtendedWithOwn.ID, 1, wantedRights|access.APMove))
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
	a.True(m.HasRights(ctx, pExtendedWithOwn.ID, access.SKUser, 1))
}

func TestSetGroupRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	// WARNING: "p" will be reused and inherited below in this function
	//---------------------------------------------------------------------------
	basePolicy, err := m.Create(
		ctx,
		access.NewKey("parent"), // key
		1,                       // owner ID
		0,                       // parent ID
		0,                       // object ID
		access.ObjectName{},     // object type
		0,                       // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// adding the user to 2 groups but setting rights to only one
	//---------------------------------------------------------------------------
	// group 1
	g1, err := gm.Create(ctx, group.FGroup, 0, "test_group_1", "test group 1")
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, g1.ID, 1))
	a.NoError(gm.CreateRelation(ctx, g1.ID, 2))

	// group 2
	g2, err := gm.Create(ctx, group.FGroup, 0, "test_group_2", "test group 2")
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, g2.ID, 1))

	// assigning wanted rights to the first group (1)
	a.NoError(m.GrantGroupAccess(ctx, basePolicy.ID, 1, g1.ID, wantedRights))
	a.NoError(m.Update(ctx, basePolicy))
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKGroup, g1.ID))

	//---------------------------------------------------------------------------
	// now with parent, using inheritance only, no extension
	// NOTE: using previously created policy "p" as a parent here
	// NOTE: not setting it's own rights as it must inherit them from a parent
	//---------------------------------------------------------------------------
	pWithInherit, err := m.Create(
		ctx,
		access.NewKey("with inherit"), // key
		0,                             // owner ID
		basePolicy.ID,                 // parent ID
		0,                             // object ID
		access.ObjectName{},           // object type
		access.FInherit,               // flags
	)
	a.NoError(err)

	// checking rights on both parent and its child
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKUser, 2))
	a.True(m.HasRights(ctx, pWithInherit.ID, access.SKUser, 2))

	//---------------------------------------------------------------------------
	// + with parent
	// - without inherit
	// + with extend
	// NOTE: using parent's rights, no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		access.NewKey("with extend, no own rights"), // key
		0,                   // owner ID
		basePolicy.ID,       // parent ID
		0,                   // object ID
		access.ObjectName{}, // object type
		access.FExtend,      // flags
	)
	a.NoError(err)

	// checking rights on the extended policy
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKUser, 2))
	a.True(m.HasRights(ctx, pExtendedNoOwn.ID, access.SKUser, 2))

	//---------------------------------------------------------------------------
	// with parent, inheritance false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		access.NewKey("with extend and own rights"), // key
		0,                   // owner ID
		basePolicy.ID,       // parent ID
		0,                   // object ID
		access.ObjectName{}, // object type
		access.FExtend,      // flags
	)
	a.NoError(err)

	// setting own rights for this policy
	a.NoError(m.GrantUserAccess(ctx, pExtendedWithOwn.ID, 1, 2, access.APMove))
	a.NoError(m.Update(ctx, pExtendedWithOwn))

	// expecting a proper blend with parent rights
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKUser, 2))
	a.True(m.HasRights(ctx, pExtendedWithOwn.ID, access.SKUser, 2))
}

func TestSetRoleRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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
		access.NewKey("parent"), // key
		1,                       // owner ID
		0,                       // parent ID
		0,                       // object ID
		access.ObjectName{},     // object type
		0,                       // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// adding the user to 2 groups but setting rights to only one
	//---------------------------------------------------------------------------
	// role 1
	r1, err := gm.Create(ctx, group.FRole, 0, "test_group_1", "test group 1")
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, r1.ID, 1))
	a.NoError(gm.CreateRelation(ctx, r1.ID, 2))

	// role 2
	r2, err := gm.Create(ctx, group.FRole, 0, "test_group_2", "test group 2")
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, r2.ID, 1))

	// assigning wanted rights to the first role (1)
	a.NoError(m.GrantRoleAccess(ctx, basePolicy.ID, 1, r1.ID, wantedRights))
	a.NoError(m.Update(ctx, basePolicy))
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKRoleGroup, r1.ID))

	//---------------------------------------------------------------------------
	// now with parent, using inheritance only, no extension
	// NOTE: using previously created policy "p" as a parent here
	// NOTE: not setting it's own rights as it must inherit them from a parent
	//---------------------------------------------------------------------------
	pWithInherit, err := m.Create(
		ctx,
		access.NewKey("with inherit"), // key
		1,                             // owner ID
		basePolicy.ID,                 // parent ID
		0,                             // object ID
		access.ObjectName{},           // object type
		access.FInherit,               // flags
	)
	a.NoError(err)

	// checking rights on both parent and its child
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKUser, 2))
	a.True(m.HasRights(ctx, pWithInherit.ID, access.SKUser, 2))

	//---------------------------------------------------------------------------
	// + with parent
	// - without inherit
	// + with extend
	// NOTE: using parent's rights, no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		access.NewKey("with extend, no own rights"), // key
		1,                   // owner ID
		basePolicy.ID,       // parent ID
		0,                   // object ID
		access.ObjectName{}, // object type
		access.FExtend,      // flags
	)
	a.NoError(err)

	// checking rights on the extended policy
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKUser, 2))
	a.True(m.HasRights(ctx, pExtendedNoOwn.ID, access.SKUser, 2))

	//---------------------------------------------------------------------------
	// with parent, inheritance false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		access.NewKey("with extend and own rights"), // key
		1,                   // owner ID
		basePolicy.ID,       // parent ID
		0,                   // object ID
		access.ObjectName{}, // object type
		access.FExtend,      // flags
	)
	a.NoError(err)

	// setting own rights for this policy
	a.NoError(m.GrantUserAccess(ctx, pExtendedWithOwn.ID, 1, 2, access.APCopy))
	a.NoError(m.Update(ctx, pExtendedWithOwn))

	// expecting a proper blend with parent rights
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKUser, 2))
	a.True(m.HasRights(ctx, pExtendedWithOwn.ID, access.SKUser, 2))
}

func TestSetUserRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := access.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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
		access.NewKey("base policy"), // key
		1,                            // owner ID
		0,                            // parent ID
		0,                            // object ID
		access.ObjectName{},          // object type
		0,                            // flags
	)
	a.NoError(err)
	a.NoError(m.GrantUserAccess(ctx, basePolicy.ID, 1, 2, wantedRights))
	a.NoError(m.Update(ctx, basePolicy))
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKUser, 2))

	//---------------------------------------------------------------------------
	// with parent, using inheritance only
	// not setting it's own rights as it must inherit them from a parent
	//---------------------------------------------------------------------------
	pWithInheritance, err := m.Create(
		ctx,
		access.NewKey("inheritance only"), // key
		0,                                 // owner ID
		basePolicy.ID,                     // parent ID
		0,                                 // object ID
		access.ObjectName{},               // object type
		access.FInherit,                   // flags
	)
	a.NoError(err)
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, pWithInheritance.ID, access.SKUser, 2))
	a.False(m.HasRights(ctx, pWithInheritance.ID, access.SKUser, 3))

	//---------------------------------------------------------------------------
	// with parent, legacy false, extend true; using parent's rights through
	// extension; no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		access.NewKey("extension only"), // key
		0,                               // owner ID
		basePolicy.ID,                   // parent ID
		0,                               // object ID
		access.ObjectName{},             // object type
		access.FExtend,                  // flags
	)
	a.NoError(err)
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, pExtendedNoOwn.ID, access.SKUser, 2))
	a.False(m.HasRights(ctx, pExtendedNoOwn.ID, access.SKUser, 3))

	//---------------------------------------------------------------------------
	// with parent, no inherit, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		access.NewKey("extension with own rights"), // key
		0,                   // owner ID
		basePolicy.ID,       // parent ID
		0,                   // object ID
		access.ObjectName{}, // object type
		access.FExtend,      // flags
	)
	a.NoError(err)
	a.NoError(m.GrantUserAccess(ctx, pExtendedWithOwn.ID, 1, 2, wantedRights|access.APMove))
	a.NoError(m.Update(ctx, pExtendedWithOwn))
	a.True(m.HasRights(ctx, basePolicy.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, pExtendedWithOwn.ID, access.SKUser, 2))
	a.False(m.HasRights(ctx, pExtendedWithOwn.ID, access.SKUser, 3))
	a.False(m.HasRights(ctx, pExtendedWithOwn.ID, access.SKUser, 3))
}

func TestIsOwner(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// ap store
	s, err := access.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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
		access.NewKey("test policy"), // key
		1,                            // owner ID
		0,                            // parent ID
		0,                            // object ID
		access.ObjectName{},          // object type
		0,                            // flags
	)
	a.NoError(err)

	// testing owner and testuser rights
	a.NoError(m.GrantUserAccess(ctx, ap.ID, 1, 2, access.APView))
	a.NoError(m.Update(ctx, ap))

	// user 1 (owner)
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 1))

	// user 2
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.SKUser, 2))

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
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// ap store
	s, err := access.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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
		access.NewKey("test policy"), // key
		1,                            // owner ID
		0,                            // parent ID
		0,                            // object ID
		access.ObjectName{},          // object type
		0,                            // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// setting base rights and saving them safely
	//---------------------------------------------------------------------------
	a.NoError(m.GrantUserAccess(ctx, ap.ID, 1, 2, access.APView|access.APChange))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 2))

	//---------------------------------------------------------------------------
	// setting additional rights correctly and incorrectly
	// NOTE: any faulty assignment MUST restore backup to the last
	// safe point of when it was either loaded or safely stored
	//---------------------------------------------------------------------------
	// assigning a few rights but not updating the store
	a.NoError(m.GrantUserAccess(ctx, ap.ID, 1, 2, access.APView|access.APChange|access.APDelete))

	// testing the rights which haven't been persisted yet
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.SKUser, 2))
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
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 1))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 1))

	// user 2
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.SKUser, 2))

	// user 3 (assignment to this user must've provoked the restoration)
	a.False(m.HasRights(ctx, ap.ID, access.SKUser, 3))

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
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// ap store
	s, err := access.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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
		access.NewKey("test policy"), // key
		1,                            // owner ID
		0,                            // parent ID
		0,                            // object ID
		access.ObjectName{},          // object type
		0,                            // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// public access
	//---------------------------------------------------------------------------
	// setting
	a.False(m.HasRights(ctx, ap.ID, access.SKEveryone, 2))
	a.False(m.HasRights(ctx, ap.ID, access.SKEveryone, 2))
	a.NoError(m.GrantPublicAccess(ctx, ap.ID, 1, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, access.SKEveryone, 2))
	a.True(m.HasRights(ctx, ap.ID, access.SKEveryone, 2))

	// unsetting
	a.NoError(m.RevokeAccess(ctx, access.SKEveryone, ap.ID, 1, 2))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, access.SKEveryone, 2))
	a.False(m.HasRights(ctx, ap.ID, access.SKEveryone, 2))

	//---------------------------------------------------------------------------
	// user rights
	//---------------------------------------------------------------------------
	// setting
	a.False(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.NoError(m.GrantUserAccess(ctx, ap.ID, 1, 2, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.True(m.HasRights(ctx, ap.ID, access.SKUser, 2))

	// unsetting
	a.NoError(m.RevokeAccess(ctx, access.SKUser, ap.ID, 1, 2))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, access.SKUser, 2))
	a.False(m.HasRights(ctx, ap.ID, access.SKUser, 2))

	//---------------------------------------------------------------------------
	// role rights
	//---------------------------------------------------------------------------
	// setting
	a.False(m.HasRights(ctx, ap.ID, access.SKRoleGroup, r.ID))
	a.False(m.HasRights(ctx, ap.ID, access.SKRoleGroup, r.ID))
	a.NoError(m.GrantRoleAccess(ctx, ap.ID, 1, r.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, access.SKRoleGroup, r.ID))
	a.True(m.HasRights(ctx, ap.ID, access.SKRoleGroup, r.ID))

	// unsetting
	a.NoError(m.RevokeAccess(ctx, access.SKRoleGroup, ap.ID, 1, r.ID))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, access.SKRoleGroup, r.ID))
	a.False(m.HasRights(ctx, ap.ID, access.SKRoleGroup, r.ID))

	//---------------------------------------------------------------------------
	// group rights
	//---------------------------------------------------------------------------
	// setting
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g.ID))
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g.ID))
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, 1, g.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, access.SKGroup, g.ID))
	a.True(m.HasRights(ctx, ap.ID, access.SKGroup, g.ID))

	// unsetting
	a.NoError(m.RevokeAccess(ctx, access.SKGroup, ap.ID, 1, g.ID))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g.ID))
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g.ID))
}

func TestHasGroupRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// ap store
	s, err := access.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewMySQLStore(db)
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
		access.NewKey("test policy"), // key
		1,                            // owner ID
		0,                            // parent ID
		0,                            // object ID
		access.ObjectName{},          // object type
		0,                            // flags
	)
	a.NoError(err)

	// setting rights for group 1
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, 1, g1.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, ap.ID, access.SKGroup, g1.ID))
	a.True(m.HasRights(ctx, ap.ID, access.SKGroup, g2.ID))
	a.True(m.HasRights(ctx, ap.ID, access.SKGroup, g3.ID))

	//---------------------------------------------------------------------------
	// setting rights only for the second group, thus g3 must inherit its
	// rights only from group 2, and group 1 must not have the rights of group 2
	//---------------------------------------------------------------------------
	ap, err = m.Create(
		ctx,
		access.NewKey("test policy 2"), // key
		1,                              // owner ID
		0,                              // parent ID
		0,                              // object ID
		access.ObjectName{},            // object type
		0,                              // flags
	)
	a.NoError(err)

	// setting rights for group 2
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, 1, g2.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g1.ID))
	a.True(m.HasRights(ctx, ap.ID, access.SKGroup, g2.ID))
	a.True(m.HasRights(ctx, ap.ID, access.SKGroup, g3.ID))

	//---------------------------------------------------------------------------
	// setting rights only for the third group, group 1 and group 2 must not have
	// these rights
	//---------------------------------------------------------------------------
	ap, err = m.Create(
		ctx,
		access.NewKey("test policy 3"), // key
		1,                              // owner ID
		0,                              // parent ID
		0,                              // object ID
		access.ObjectName{},            // object type
		0,                              // flags
	)
	a.NoError(err)

	// setting rights for group 3
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, 1, g3.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g1.ID))
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g2.ID))
	a.True(m.HasRights(ctx, ap.ID, access.SKGroup, g3.ID))

	//---------------------------------------------------------------------------
	// setting rights only for group 1 & 2, group 3 must inherit the rights
	// from its direct ancestor that has its own rights (group 2)
	//---------------------------------------------------------------------------
	ap, err = m.Create(
		ctx,
		access.NewKey("test policy 4"), // key
		1,                              // owner ID
		0,                              // parent ID
		0,                              // object ID
		access.ObjectName{},            // object type
		0,                              // flags
	)
	a.NoError(err)

	// rights to be used
	group1Rights := access.APView | access.APCreate
	wantedRights = access.APDelete | access.APCopy

	// setting rights for group 1 and 2
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, 1, g1.ID, group1Rights))
	a.NoError(m.GrantGroupAccess(ctx, ap.ID, 1, g2.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))

	a.True(m.HasRights(ctx, ap.ID, access.SKGroup, g1.ID))
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g1.ID))

	a.True(m.HasRights(ctx, ap.ID, access.SKGroup, g2.ID))
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g2.ID))

	a.True(m.HasRights(ctx, ap.ID, access.SKGroup, g3.ID))
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g3.ID))

	//---------------------------------------------------------------------------
	// not setting any rights, only checking
	//---------------------------------------------------------------------------
	ap, err = m.Create(
		ctx,
		access.NewKey("test policy 5"), // key
		1,                              // owner ID
		0,                              // parent ID
		0,                              // object ID
		access.ObjectName{},            // object type
		0,                              // flags
	)
	a.NoError(err)

	// test rights
	wantedRights = access.APCreate | access.APView

	// all check results must be false
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g1.ID))
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g2.ID))
	a.False(m.HasRights(ctx, ap.ID, access.SKGroup, g3.ID))
}
