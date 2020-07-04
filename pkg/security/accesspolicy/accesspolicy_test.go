package accesspolicy_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/security/accesspolicy"
	"github.com/stretchr/testify/assert"
)

func TestNewAccessPolicy(t *testing.T) {
	a := assert.New(t)

	// test context
	ctx := context.Background()

	// database instance
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewStore(db)
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
		accesspolicy.NewKey("test_key"), // key
		0,                               // owner ID
		0,                               // parent ID
		0,                               // object ID
		accesspolicy.TObjectType{},      // object type
		0,                               // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.Equal(accesspolicy.NewKey("test_key"), p.Key)
	a.Zero(p.OwnerID)
	a.Zero(p.ParentID)
	a.Zero(p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	p, err = m.Create(
		ctx,
		accesspolicy.NewKey("test_key2"), // key
		1,                                // owner ID
		0,                                // parent ID
		0,                                // object ID
		accesspolicy.TObjectType{},       // object type
		0,                                // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.Equal(accesspolicy.NewKey("test_key2"), p.Key)
	a.Equal(uint32(1), p.OwnerID)
	a.Zero(p.ParentID)
	a.Zero(p.ObjectID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	// with parent
	pWithParent, err := m.Create(
		ctx,
		accesspolicy.NewKey("test_key3"), // key
		1,                                // owner ID
		p.ID,                             // parent ID
		0,                                // object ID
		accesspolicy.TObjectType{},       // object type
		0,                                // flags
	)
	a.NoError(err)
	a.NotNil(pWithParent)
	a.Equal(accesspolicy.NewKey("test_key3"), pWithParent.Key)
	a.Equal(uint32(1), pWithParent.OwnerID)
	a.Equal(p.ID, pWithParent.ParentID)
	a.False(pWithParent.IsInherited())
	a.False(pWithParent.IsExtended())

	// with inheritance (without a parent)
	_, err = m.Create(
		ctx,
		accesspolicy.NewKey("test_key4"), // key
		1,                                // owner ID
		0,                                // parent ID
		1,                                // object ID
		accesspolicy.NewObjectType("test object"), // object type
		accesspolicy.FInherit,                     // flags
	)
	a.Error(err)

	// with extension (without a parent)
	_, err = m.Create(
		ctx,
		accesspolicy.NewKey("test_key5"), // key
		1,                                // owner ID
		0,                                // parent ID
		1,                                // object ID
		accesspolicy.NewObjectType("test object"), // object type
		accesspolicy.FExtend,                      // flags
	)
	a.Error(err)

	// with inheritance (with a parent)
	pInheritedWithParent, err := m.Create(
		ctx,
		accesspolicy.NewKey("test_key6"), // key
		1,                                // owner ID
		p.ID,                             // parent ID
		1,                                // object ID
		accesspolicy.NewObjectType("test object"), // object type
		accesspolicy.FInherit,                     // flags
	)
	a.NoError(err)
	a.NotNil(pInheritedWithParent)

	// with extension (with a parent)
	pExtendedWithParent, err := m.Create(
		ctx,
		accesspolicy.NewKey("test_key7"), // key
		1,                                // owner ID
		p.ID,                             // parent ID
		1,                                // object ID
		accesspolicy.NewObjectType("another test object"), // object type
		accesspolicy.FExtend,                              // flags
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
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewStore(db)
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

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	//---------------------------------------------------------------------------
	p, err := m.Create(
		ctx,
		accesspolicy.NewKey("test_key"), // key
		1,                               // owner ID
		0,                               // parent ID
		0,                               // object ID
		accesspolicy.TObjectType{},      // object type
		0,                               // flags
	)
	a.NoError(err)

	// obtaining corresponding policy roster
	roster, err := m.RosterByPolicyID(ctx, p.ID)
	a.NoError(err)
	a.NotNil(roster)

	a.NoError(m.SetPublicRights(ctx, p.ID, 1, wantedRights))
	a.NoError(m.Update(ctx, p))
	a.Equal(wantedRights, roster.Everyone)
	a.True(m.HasRights(ctx, accesspolicy.SKEveryone, p.ID, 1, wantedRights))

	//---------------------------------------------------------------------------
	// with parent and inheritance only
	//---------------------------------------------------------------------------
	pWithInheritance, err := m.Create(
		ctx,
		accesspolicy.NewKey("test_key_w_inheritance"), // key
		1,                          // owner ID
		p.ID,                       // parent ID
		0,                          // object ID
		accesspolicy.TObjectType{}, // object type
		accesspolicy.FInherit,      // flags
	)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)

	// obtaining corresponding policy roster
	parentRoster, err := m.RosterByPolicyID(ctx, pWithInheritance.ID)
	a.NoError(err)
	a.NotNil(parentRoster)

	parent, err := m.PolicyByID(ctx, pWithInheritance.ParentID)
	a.NoError(err)
	a.True(m.HasRights(ctx, accesspolicy.SKUser, pWithInheritance.ID, 1, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, inheritance false, extend true; using parent's rights no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		accesspolicy.TKey{}, // key
		0,                   // owner ID
		parent.ID,           // parent ID
		1,                   // object ID
		accesspolicy.NewObjectType("some object"), // object type
		accesspolicy.FExtend,                      // flags
	)
	a.NoError(err)

	// obtaining corresponding policy roster
	parentRoster, err = m.RosterByPolicyID(ctx, pExtendedNoOwn.ParentID)
	a.NoError(err)
	a.NotNil(parentRoster)

	parent, err = m.PolicyByID(ctx, pExtendedNoOwn.ParentID)
	a.NoError(err)
	a.True(m.HasRights(ctx, accesspolicy.SKUser, parent.ID, 1, wantedRights))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, pExtendedNoOwn.ID, 1, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, inherit false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		accesspolicy.TKey{}, // key
		1,                   // owner ID
		parent.ID,           // parent ID
		1,                   // object ID
		accesspolicy.NewObjectType("and another object"), // object type
		accesspolicy.FExtend,                             // flags
	)
	a.NoError(err)
	a.NoError(m.SetPublicRights(ctx, pExtendedWithOwn.ID, 1, wantedRights|accesspolicy.APMove))
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
	a.True(m.HasRights(ctx, accesspolicy.SKUser, pExtendedWithOwn.ID, 1, wantedRights|accesspolicy.APMove))
}

func TestSetGroupRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewStore(db)
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

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	// WARNING: "p" will be reused and inherited below in this function
	//---------------------------------------------------------------------------
	basePolicy, err := m.Create(
		ctx,
		accesspolicy.NewKey("parent"), // key
		1,                             // owner ID
		0,                             // parent ID
		0,                             // object ID
		accesspolicy.TObjectType{},    // object type
		0,                             // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// adding the user to 2 groups but setting rights to only one
	//---------------------------------------------------------------------------
	// group 1
	g1, err := gm.Create(ctx, group.GKGroup, 0, "test_group_1", "test group 1", false)
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, g1.ID, 1))
	a.NoError(gm.CreateRelation(ctx, g1.ID, 2))

	// group 2
	g2, err := gm.Create(ctx, group.GKGroup, 0, "test_group_2", "test group 2", false)
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, g2.ID, 1))

	// assigning wanted rights to the first group (1)
	a.NoError(m.SetGroupRights(ctx, basePolicy.ID, 1, g1.ID, wantedRights))
	a.NoError(m.Update(ctx, basePolicy))
	a.True(m.HasRights(ctx, accesspolicy.SKGroup, basePolicy.ID, g1.ID, wantedRights))

	//---------------------------------------------------------------------------
	// now with parent, using inheritance only, no extension
	// NOTE: using previously created policy "p" as a parent here
	// NOTE: not setting it's own rights as it must inherit them from a parent
	//---------------------------------------------------------------------------
	pWithInherit, err := m.Create(
		ctx,
		accesspolicy.NewKey("with inherit"), // key
		0,                                   // owner ID
		basePolicy.ID,                       // parent ID
		0,                                   // object ID
		accesspolicy.TObjectType{},          // object type
		accesspolicy.FInherit,               // flags
	)
	a.NoError(err)

	// checking rights on both parent and its child
	a.True(m.HasRights(ctx, accesspolicy.SKUser, basePolicy.ID, 2, wantedRights))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, pWithInherit.ID, 2, wantedRights))

	//---------------------------------------------------------------------------
	// + with parent
	// - without inherit
	// + with extend
	// NOTE: using parent's rights, no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		accesspolicy.NewKey("with extend, no own rights"), // key
		0,                          // owner ID
		basePolicy.ID,              // parent ID
		0,                          // object ID
		accesspolicy.TObjectType{}, // object type
		accesspolicy.FExtend,       // flags
	)
	a.NoError(err)

	// checking rights on the extended policy
	a.True(m.HasRights(ctx, accesspolicy.SKUser, basePolicy.ID, 2, wantedRights))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, pExtendedNoOwn.ID, 2, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, inheritance false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		accesspolicy.NewKey("with extend and own rights"), // key
		0,                          // owner ID
		basePolicy.ID,              // parent ID
		0,                          // object ID
		accesspolicy.TObjectType{}, // object type
		accesspolicy.FExtend,       // flags
	)
	a.NoError(err)

	// setting own rights for this policy
	a.NoError(m.SetUserRights(ctx, pExtendedWithOwn.ID, 1, 2, accesspolicy.APMove))
	a.NoError(m.Update(ctx, pExtendedWithOwn))

	// expecting a proper blend with parent rights
	a.True(m.HasRights(ctx, accesspolicy.SKUser, basePolicy.ID, 2, wantedRights))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, pExtendedWithOwn.ID, 2, wantedRights|accesspolicy.APMove))
}

func TestSetRoleRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewStore(db)
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

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	// WARNING: "p" will be reused and inherited below in this function
	//---------------------------------------------------------------------------
	basePolicy, err := m.Create(
		ctx,
		accesspolicy.NewKey("parent"), // key
		1,                             // owner ID
		0,                             // parent ID
		0,                             // object ID
		accesspolicy.TObjectType{},    // object type
		0,                             // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// adding the user to 2 groups but setting rights to only one
	//---------------------------------------------------------------------------
	// role 1
	r1, err := gm.Create(ctx, group.GKRole, 0, "test_group_1", "test group 1", false)
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, r1.ID, 1))
	a.NoError(gm.CreateRelation(ctx, r1.ID, 2))

	// role 2
	r2, err := gm.Create(ctx, group.GKRole, 0, "test_group_2", "test group 2", false)
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, r2.ID, 1))

	// assigning wanted rights to the first role (1)
	a.NoError(m.SetRoleRights(ctx, basePolicy.ID, 1, r1.ID, wantedRights))
	a.NoError(m.Update(ctx, basePolicy))
	a.True(m.HasRights(ctx, accesspolicy.SKRoleGroup, basePolicy.ID, r1.ID, wantedRights))

	//---------------------------------------------------------------------------
	// now with parent, using inheritance only, no extension
	// NOTE: using previously created policy "p" as a parent here
	// NOTE: not setting it's own rights as it must inherit them from a parent
	//---------------------------------------------------------------------------
	pWithInherit, err := m.Create(
		ctx,
		accesspolicy.NewKey("with inherit"), // key
		1,                                   // owner ID
		basePolicy.ID,                       // parent ID
		0,                                   // object ID
		accesspolicy.TObjectType{},          // object type
		accesspolicy.FInherit,               // flags
	)
	a.NoError(err)

	// checking rights on both parent and its child
	a.True(m.HasRights(ctx, accesspolicy.SKUser, basePolicy.ID, 2, wantedRights))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, pWithInherit.ID, 2, wantedRights))

	//---------------------------------------------------------------------------
	// + with parent
	// - without inherit
	// + with extend
	// NOTE: using parent's rights, no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		accesspolicy.NewKey("with extend, no own rights"), // key
		1,                          // owner ID
		basePolicy.ID,              // parent ID
		0,                          // object ID
		accesspolicy.TObjectType{}, // object type
		accesspolicy.FExtend,       // flags
	)
	a.NoError(err)

	// checking rights on the extended policy
	a.True(m.HasRights(ctx, accesspolicy.SKUser, basePolicy.ID, 2, wantedRights))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, pExtendedNoOwn.ID, 2, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, inheritance false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		accesspolicy.NewKey("with extend and own rights"), // key
		1,                          // owner ID
		basePolicy.ID,              // parent ID
		0,                          // object ID
		accesspolicy.TObjectType{}, // object type
		accesspolicy.FExtend,       // flags
	)
	a.NoError(err)

	// setting own rights for this policy
	a.NoError(m.SetUserRights(ctx, pExtendedWithOwn.ID, 1, 2, accesspolicy.APCopy))
	a.NoError(m.Update(ctx, pExtendedWithOwn))

	// expecting a proper blend with parent rights
	a.True(m.HasRights(ctx, accesspolicy.SKUser, basePolicy.ID, 2, wantedRights))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, pExtendedWithOwn.ID, 2, wantedRights|accesspolicy.APCopy))
}

func TestSetUserRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// policy store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewStore(db)
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

	//---------------------------------------------------------------------------
	// no parent, not inheriting and not extending
	// WARNING: "p" will be reused and inherited below in this function
	//---------------------------------------------------------------------------
	basePolicy, err := m.Create(
		ctx,
		accesspolicy.NewKey("base policy"), // key
		1,                                  // owner ID
		0,                                  // parent ID
		0,                                  // object ID
		accesspolicy.TObjectType{},         // object type
		0,                                  // flags
	)
	a.NoError(err)
	a.NoError(m.SetUserRights(ctx, basePolicy.ID, 1, 2, wantedRights))
	a.NoError(m.Update(ctx, basePolicy))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, basePolicy.ID, 1, wantedRights))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, basePolicy.ID, 2, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, using inheritance only
	// not setting it's own rights as it must inherit them from a parent
	//---------------------------------------------------------------------------
	pWithInheritance, err := m.Create(
		ctx,
		accesspolicy.NewKey("inheritance only"), // key
		0,                                       // owner ID
		basePolicy.ID,                           // parent ID
		0,                                       // object ID
		accesspolicy.TObjectType{},              // object type
		accesspolicy.FInherit,                   // flags
	)
	a.NoError(err)
	a.True(m.HasRights(ctx, accesspolicy.SKUser, basePolicy.ID, 1, wantedRights))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, pWithInheritance.ID, 2, wantedRights))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, pWithInheritance.ID, 3, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, legacy false, extend true; using parent's rights through
	// extension; no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		accesspolicy.NewKey("extension only"), // key
		0,                                     // owner ID
		basePolicy.ID,                         // parent ID
		0,                                     // object ID
		accesspolicy.TObjectType{},            // object type
		accesspolicy.FExtend,                  // flags
	)
	a.NoError(err)
	a.True(m.HasRights(ctx, accesspolicy.SKUser, basePolicy.ID, 1, wantedRights))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, pExtendedNoOwn.ID, 2, wantedRights))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, pExtendedNoOwn.ID, 3, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, no inherit, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		accesspolicy.NewKey("extension with own rights"), // key
		0,                          // owner ID
		basePolicy.ID,              // parent ID
		0,                          // object ID
		accesspolicy.TObjectType{}, // object type
		accesspolicy.FExtend,       // flags
	)
	a.NoError(err)
	a.NoError(m.SetUserRights(ctx, pExtendedWithOwn.ID, 1, 2, wantedRights|accesspolicy.APMove))
	a.NoError(m.Update(ctx, pExtendedWithOwn))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, basePolicy.ID, 1, wantedRights))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, pExtendedWithOwn.ID, 2, wantedRights|accesspolicy.APMove))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, pExtendedWithOwn.ID, 3, wantedRights))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, pExtendedWithOwn.ID, 3, wantedRights|accesspolicy.APMove))
}

func TestIsOwner(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	// test context
	ctx := context.Background()

	// database instance
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// ap store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewStore(db)
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

	//---------------------------------------------------------------------------
	// test policy
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		accesspolicy.NewKey("test policy"), // key
		1,                                  // owner ID
		0,                                  // parent ID
		0,                                  // object ID
		accesspolicy.TObjectType{},         // object type
		0,                                  // flags
	)
	a.NoError(err)

	// testing owner and testuser rights
	a.NoError(m.SetUserRights(ctx, ap.ID, 1, 2, accesspolicy.APView))
	a.NoError(m.Update(ctx, ap))

	// user 1 (owner)
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 1, accesspolicy.APView))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 1, accesspolicy.APChange))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 1, accesspolicy.APDelete))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 1, accesspolicy.APFullAccess))

	// user 2
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APView))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APChange))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APDelete))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APFullAccess))

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
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// ap store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewStore(db)
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

	//---------------------------------------------------------------------------
	// test policy
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		accesspolicy.NewKey("test policy"), // key
		1,                                  // owner ID
		0,                                  // parent ID
		0,                                  // object ID
		accesspolicy.TObjectType{},         // object type
		0,                                  // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// setting base rights and saving them safely
	//---------------------------------------------------------------------------
	a.NoError(m.SetUserRights(ctx, ap.ID, 1, 2, accesspolicy.APView|accesspolicy.APChange))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APView))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APChange))

	//---------------------------------------------------------------------------
	// setting additional rights correctly and incorrectly
	// NOTE: any faulty assignment MUST restore backup to the last
	// safe point of when it was either loaded or safely stored
	//---------------------------------------------------------------------------
	// assigning a few rights but not updating the store
	a.NoError(m.SetUserRights(ctx, ap.ID, 1, 2, accesspolicy.APView|accesspolicy.APChange|accesspolicy.APDelete))

	// testing the rights which haven't been persisted yet
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 1, accesspolicy.APView))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 1, accesspolicy.APChange))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 1, accesspolicy.APDelete))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 1, accesspolicy.APFullAccess))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APView))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APChange))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APFullAccess))
	a.True(ap.IsOwner(1))
	a.False(ap.IsOwner(2))

	// attempting to set rights as a second user to a third
	// NOTE: must fail and restore backup, clearing out any changes made
	// WARNING: THIS MUST FAIL AND RESTORE BACKUP, ROLLING BACK ANY CHANGES ON THE POLICY ROSTER
	a.EqualError(m.SetUserRights(ctx, ap.ID, 2, 3, accesspolicy.APChange), accesspolicy.ErrExcessOfRights.Error())

	// NOTE: this update must be harmless, because there must be nothing to change
	a.NoError(m.Update(ctx, ap))

	// checking whether the rights were returned to its previous state
	// user 1
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 1, accesspolicy.APView))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 1, accesspolicy.APChange))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 1, accesspolicy.APDelete))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 1, accesspolicy.APFullAccess))

	// user 2
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APView))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APChange))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APDelete))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APFullAccess))

	// user 3 (assignment to this user must've provoked the restoration)
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 3, accesspolicy.APChange))

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
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// ap store
	s, err := accesspolicy.NewDefaultMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	// group store
	gs, err := group.NewStore(db)
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

	//---------------------------------------------------------------------------
	// creating test role and a group
	//---------------------------------------------------------------------------
	// role
	r, err := gm.Create(ctx, group.GKRole, 0, "test_role", "test role", false)
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, r.ID, 1))

	// group
	g, err := gm.Create(ctx, group.GKGroup, 0, "test_group", "test group", false)
	a.NoError(err)
	a.NoError(gm.CreateRelation(ctx, g.ID, 1))

	wantedRights := accesspolicy.APView | accesspolicy.APChange | accesspolicy.APCopy | accesspolicy.APDelete

	//---------------------------------------------------------------------------
	// test policy
	//---------------------------------------------------------------------------
	ap, err := m.Create(
		ctx,
		accesspolicy.NewKey("test policy"), // key
		1,                                  // owner ID
		0,                                  // parent ID
		0,                                  // object ID
		accesspolicy.TObjectType{},         // object type
		0,                                  // flags
	)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// public access
	//---------------------------------------------------------------------------
	// TODO: set/unset public access

	//---------------------------------------------------------------------------
	// user rights
	//---------------------------------------------------------------------------
	// setting
	a.NoError(m.SetUserRights(ctx, ap.ID, 1, 2, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APView))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APChange))

	// unsetting
	a.NoError(m.UnsetRights(ctx, accesspolicy.SKUser, ap.ID, 1, 2))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APView))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APChange))

	//---------------------------------------------------------------------------
	// role rights
	//---------------------------------------------------------------------------
	// setting
	a.NoError(m.SetRoleRights(ctx, ap.ID, 1, r.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, accesspolicy.SKRoleGroup, ap.ID, r.ID, accesspolicy.APView))
	a.True(m.HasRights(ctx, accesspolicy.SKRoleGroup, ap.ID, r.ID, accesspolicy.APChange))

	// unsetting
	a.NoError(m.UnsetRights(ctx, accesspolicy.SKRoleGroup, ap.ID, 1, r.ID))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, accesspolicy.SKRoleGroup, ap.ID, r.ID, accesspolicy.APView))
	a.False(m.HasRights(ctx, accesspolicy.SKRoleGroup, ap.ID, r.ID, accesspolicy.APChange))

	//---------------------------------------------------------------------------
	// group rights
	//---------------------------------------------------------------------------
	// setting
	a.NoError(m.SetGroupRights(ctx, ap.ID, 1, g.ID, wantedRights))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, accesspolicy.SKGroup, ap.ID, g.ID, accesspolicy.APView))
	a.True(m.HasRights(ctx, accesspolicy.SKGroup, ap.ID, g.ID, accesspolicy.APChange))

	// unsetting
	a.NoError(m.UnsetRights(ctx, accesspolicy.SKGroup, ap.ID, 1, g.ID))
	a.NoError(m.Update(ctx, ap))
	a.False(m.HasRights(ctx, accesspolicy.SKGroup, ap.ID, g.ID, accesspolicy.APView))
	a.False(m.HasRights(ctx, accesspolicy.SKGroup, ap.ID, g.ID, accesspolicy.APChange))
}

/*
func TestHasGroupRights(t *testing.T) {
	a := assert.New(t)

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	um, ctx, err := ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)
	a.NotNil(ctx)

	apm := ctx.Value(CKAccessPolicyManager).(*accesspolicy.AccessPolicyManager)
	a.NotNil(apm)

	gm := ctx.Value(CKGroupManager).(*GroupManager)
	a.NotNil(gm)

	// creating test users via manager
	assignor, err := CreateTestUser(ctx, um, "assignor", "assignor@hometown.local", nil)
	a.NoError(err)

	// adding the user to 2 groups but setting rights to only one
	group1, err := gm.Create(ctx, GKGroup, 0, "test_group_1", "test group 1")
	a.NoError(err)

	group2, err := gm.Create(ctx, GKGroup, group1.ID, "test_group_2", "test group 2")
	a.NoError(err)

	group3, err := gm.Create(ctx, GKGroup, group2.ID, "test_group_3", "test group 3")
	a.NoError(err)

	//---------------------------------------------------------------------------
	// setting rights only for the first group, thus group3 must inherit its
	// rights only from group 1
	//---------------------------------------------------------------------------
	ap, err := apm.Create(ctx, assignor.ID, 0, "test_name", "test_type", 1, false, false)
	a.NoError(err)
	a.NotNil(ap)

	wantedRights := accesspolicy.APCreate | accesspolicy.APView

	// setting rights for group 1
	a.NoError(ap.SetGroupRights(ctx, assignor.ID, group1.ID, wantedRights))
	a.True(ap.HasGroupRights(ctx, group1.ID, wantedRights))
	a.True(ap.HasGroupRights(ctx, group2.ID, wantedRights))
	a.True(ap.HasGroupRights(ctx, group3.ID, wantedRights))

	//---------------------------------------------------------------------------
	// setting rights only for the second group, thus group3 must inherit its
	// rights only from group 2, and group 1 must not have the rights of group 2
	//---------------------------------------------------------------------------
	ap, err = apm.Create(ctx, assignor.ID, 0, "test_name 2", "test_type 2", 1, false, false)
	a.NoError(err)
	a.NotNil(ap)

	wantedRights = accesspolicy.APCreate | accesspolicy.APView

	// setting rights for group 2
	a.NoError(ap.SetGroupRights(ctx, assignor.ID, group2.ID, wantedRights))
	a.False(ap.HasGroupRights(ctx, group1.ID, wantedRights))
	a.True(ap.HasGroupRights(ctx, group2.ID, wantedRights))
	a.True(ap.HasGroupRights(ctx, group3.ID, wantedRights))

	//---------------------------------------------------------------------------
	// setting rights only for the third group, group 1 and group 2 must not have
	// these rights
	//---------------------------------------------------------------------------
	ap, err = apm.Create(ctx, assignor.ID, 0, "test_name 3", "test_type 3", 1, false, false)
	a.NoError(err)
	a.NotNil(ap)

	wantedRights = accesspolicy.APCreate | accesspolicy.APView

	// setting rights for group 2
	a.NoError(ap.SetGroupRights(ctx, assignor.ID, group3.ID, wantedRights))
	a.False(ap.HasGroupRights(ctx, group1.ID, wantedRights))
	a.False(ap.HasGroupRights(ctx, group2.ID, wantedRights))
	a.True(ap.HasGroupRights(ctx, group3.ID, wantedRights))

	//---------------------------------------------------------------------------
	// setting rights only for group 1 & 2, group 3 must inherit the rights
	// from its direct ancestor that has its own rights (group 2)
	//---------------------------------------------------------------------------
	ap, err = apm.Create(ctx, assignor.ID, 0, "test_name 4", "test_type 4", 1, false, false)
	a.NoError(err)
	a.NotNil(ap)

	group1Rights := accesspolicy.APView | accesspolicy.APCreate
	wantedRights = accesspolicy.APDelete | accesspolicy.APCopy

	// setting rights for group 1 & 2
	a.NoError(ap.SetGroupRights(ctx, assignor.ID, group1.ID, group1Rights))
	a.NoError(ap.SetGroupRights(ctx, assignor.ID, group2.ID, wantedRights))

	a.True(ap.HasGroupRights(ctx, group1.ID, group1Rights))
	a.False(ap.HasGroupRights(ctx, group1.ID, wantedRights))

	a.True(ap.HasGroupRights(ctx, group2.ID, wantedRights))
	a.False(ap.HasGroupRights(ctx, group2.ID, group1Rights))

	a.True(ap.HasGroupRights(ctx, group3.ID, wantedRights))
	a.False(ap.HasGroupRights(ctx, group3.ID, group1Rights))

	//---------------------------------------------------------------------------
	// not setting any rights, only checking
	//---------------------------------------------------------------------------
	ap, err = apm.Create(ctx, assignor.ID, 0, "test_name 5", "test_type 5", 1, false, false)
	a.NoError(err)
	a.NotNil(ap)

	wantedRights = accesspolicy.APCreate | accesspolicy.APView

	// setting rights for group 2
	a.False(ap.HasGroupRights(ctx, group1.ID, wantedRights))
	a.False(ap.HasGroupRights(ctx, group2.ID, wantedRights))
	a.False(ap.HasGroupRights(ctx, group3.ID, wantedRights))
}
*/
