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
	p, err = m.Create(
		ctx,
		accesspolicy.NewKey("test_key"), // key
		1,                               // owner ID
		1,                               // parent ID
		0,                               // object ID
		accesspolicy.TObjectType{},      // object type
		0,                               // flags
	)
	a.NoError(err)
	a.NotNil(p)
	a.Equal(accesspolicy.NewKey("test_key3"), p.Key)
	a.Zero(p.OwnerID)
	a.Equal(uint32(1), p.ParentID)
	a.False(p.IsInherited())
	a.False(p.IsExtended())

	// with inheritance (without a parent)
	p, err = m.Create(
		ctx,
		accesspolicy.NewKey("test_key4"), // key
		1,                                // owner ID
		0,                                // parent ID
		1,                                // object ID
		accesspolicy.TObjectType{},       // object type
		accesspolicy.FInherit,            // flags
	)
	a.Error(err)

	// with extension (without a parent)
	p, err = m.Create(
		ctx,
		accesspolicy.NewKey("test_key5"), // key
		1,                                // owner ID
		0,                                // parent ID
		1,                                // object ID
		accesspolicy.TObjectType{},       // object type
		accesspolicy.FExtend,             // flags
	)
	a.Error(err)

	// with inheritance (with a parent)
	p, err = m.Create(
		ctx,
		accesspolicy.NewKey("test_key6"), // key
		1,                                // owner ID
		1,                                // parent ID
		1,                                // object ID
		accesspolicy.TObjectType{},       // object type
		accesspolicy.FInherit,            // flags
	)
	a.NoError(err)
	a.NotNil(p)

	// with extension (with a parent)
	p, err = m.Create(
		ctx,
		accesspolicy.NewKey("test_key6"), // key
		1,                                // owner ID
		1,                                // parent ID
		1,                                // object ID
		accesspolicy.TObjectType{},       // object type
		accesspolicy.FExtend,             // flags
	)
	a.NoError(err)
	a.NotNil(p)
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
	a.Equal(wantedRights, parentRoster.Everyone)
	a.True(m.HasRights(ctx, accesspolicy.SKEveryone, pWithInheritance.ID, 1, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, inheritance false, extend true; using parent's rights no own rights
	//---------------------------------------------------------------------------
	pExtendedNoOwn, err := m.Create(
		ctx,
		accesspolicy.TKey{}, // key
		0,                   // owner ID
		1,                   // parent ID
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
	a.Equal(wantedRights, parentRoster.Everyone)
	a.True(m.HasRights(ctx, accesspolicy.SKEveryone, parent.ID, 1, wantedRights))

	parent, err = m.PolicyByID(ctx, pExtendedNoOwn.ParentID)
	a.NoError(err)

	a.Equal(wantedRights, parentRoster.Everyone)
	a.Equal(accesspolicy.APNoAccess, parentRoster.Everyone)
	a.True(m.HasRights(ctx, accesspolicy.SKEveryone, pExtendedNoOwn.ID, 1, wantedRights))

	//---------------------------------------------------------------------------
	// with parent, inherit false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	//---------------------------------------------------------------------------
	pExtendedWithOwn, err := m.Create(
		ctx,
		accesspolicy.TKey{}, // key
		0,                   // owner ID
		parent.ID,           // parent ID
		1,                   // object ID
		accesspolicy.NewObjectType("and another object"), // object type
		accesspolicy.FExtend,                             // flags
	)
	a.NoError(err)

	a.Error(m.SetPublicRights(ctx, pExtendedWithOwn.ID, 1, accesspolicy.APMove))
	a.NoError(err)

	parent, err = m.PolicyByID(ctx, pExtendedWithOwn.ParentID)
	a.NoError(err)

	roster, err = m.RosterByPolicyID(ctx, pExtendedWithOwn.ID)
	a.NoError(err)
	a.NotNil(roster)

	parentRoster, err = m.RosterByPolicyID(ctx, parent.ID)
	a.NoError(err)
	a.NotNil(parentRoster)

	a.Equal(wantedRights, parentRoster.Everyone)
	a.Equal(accesspolicy.APMove, roster.Everyone)
	a.True(m.HasRights(ctx, accesspolicy.SKEveryone, pExtendedWithOwn.ID, 1, wantedRights|accesspolicy.APMove))
}

/*
func TestSetGroupRights(t *testing.T) {
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

	testuser, err := CreateTestUser(ctx, um, "testuser", "testuser@hometown.local", nil)
	a.NoError(err)

	// adding the user to 2 groups but setting rights to only one
	group1, err := gm.Create(ctx, GKGroup, 0, "test_group_1", "test group 1")
	a.NoError(err)

	err = group1.AddMember(ctx, testaccesspolicy.ID)
	a.NoError(err)

	group2, err := gm.Create(ctx, GKGroup, 0, "test_group_2", "test group 2")
	a.NoError(err)

	err = group2.AddMember(ctx, testaccesspolicy.ID)
	a.NoError(err)

	wantedRights := accesspolicy.APView | accesspolicy.APChange

	// no parent, not inheriting and not extending
	// WARNING: "p" will be reused and inherited below in this function
	p, err := apm.Create(ctx, assignor.ID, 0, "test_name", "test_type", 1, false, false)
	a.NoError(err)

	err = p.SetGroupRights(ctx, assignor.ID, group1.ID, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.Group[group1.ID])
	a.True(p.HasRights(ctx, testaccesspolicy.ID, wantedRights))

	// with parent, using legacy only
	pWithInheritance, err := apm.Create(ctx, assignor.ID, 1, "the name", "the type", 1, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)

	parent, err := pWithInheritance.Parent(ctx)
	a.NoError(err)

	a.Equal(wantedRights, parent.RightsRoster.Group[group1.ID])
	a.True(pWithInheritance.HasRights(ctx, testaccesspolicy.ID, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn, err := apm.Create(ctx, assignor.ID, 1, "a name", "another type", 1, false, true)
	a.NoError(err)

	parent, err = pExtendedNoOwn.Parent(ctx)
	a.NoError(err)

	a.Equal(wantedRights, parent.RightsRoster.Group[group1.ID])
	a.Equal(accesspolicy.APNoAccess, pExtendedNoOwn.RightsRoster.Group[group1.ID])
	a.True(pExtendedNoOwn.HasRights(ctx, testaccesspolicy.ID, wantedRights))

	// with parent, inheritance false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	pExtendedWithOwn, err := apm.Create(ctx, assignor.ID, 1, "some name", "non-conflicting object type name etc", 1, false, true)
	a.NoError(err)

	err = pExtendedWithOwn.SetGroupRights(ctx, assignor.ID, group1.ID, accesspolicy.APMove)
	a.NoError(err)

	parent, err = pExtendedWithOwn.Parent(ctx)
	a.NoError(err)

	a.Equal(wantedRights, parent.RightsRoster.Group[group1.ID])
	a.Equal(accesspolicy.APMove, pExtendedWithOwn.RightsRoster.Group[group1.ID])
	a.True(pExtendedWithOwn.HasRights(ctx, testaccesspolicy.ID, wantedRights|accesspolicy.APMove))
}

func TestSetRoleRights(t *testing.T) {
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

	testuser, err := CreateTestUser(ctx, um, "testuser", "testuser@hometown.local", nil)
	a.NoError(err)

	// adding the user to 2 groups but setting rights to only one
	role1, err := gm.Create(ctx, GKRole, 0, "test_role_1", "test role 1")
	a.NoError(err)

	err = role1.AddMember(ctx, testaccesspolicy.ID)
	a.NoError(err)
	a.True(role1.IsMember(ctx, testaccesspolicy.ID))

	role2, err := gm.Create(ctx, GKRole, 0, "test_role_2", "test role 2")
	a.NoError(err)

	err = role2.AddMember(ctx, testaccesspolicy.ID)
	a.NoError(err)
	a.True(role2.IsMember(ctx, testaccesspolicy.ID))

	wantedRights := accesspolicy.APView | accesspolicy.APChange

	// no parent, not inheriting and not extending
	// NOTE: "p" will be reused and inherited below in this function
	p, err := apm.Create(ctx, assignor.ID, 0, "test_name", "test_type", 1, false, false)
	a.NoError(err)

	err = p.SetRoleRights(ctx, assignor.ID, role1.ID, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.Role[role1.ID])
	a.True(p.HasRights(ctx, testaccesspolicy.ID, wantedRights))

	// with parent, using legacy only
	pWithInheritance, err := apm.Create(ctx, assignor.ID, p.ID, "another name", "another type name", 1, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)

	parent, err := pWithInheritance.Parent(ctx)
	a.NoError(err)
	a.Equal(wantedRights, parent.RightsRoster.Role[role1.ID])
	a.True(pWithInheritance.HasRights(ctx, testaccesspolicy.ID, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn, err := apm.Create(ctx, assignor.ID, p.ID, "different name", "different type name", 1, false, true)
	a.NoError(err)

	parent, err = pExtendedNoOwn.Parent(ctx)
	a.NoError(err)
	a.Equal(wantedRights, parent.RightsRoster.Role[role1.ID])
	a.Equal(accesspolicy.APNoAccess, pExtendedNoOwn.RightsRoster.Role[role1.ID])
	a.True(pExtendedNoOwn.HasRights(ctx, testaccesspolicy.ID, wantedRights))

	// with parent, legacy false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	pExtendedWithOwn, err := apm.Create(ctx, assignor.ID, p.ID, "other name", "other type name", 1, false, true)
	a.NoError(err)

	err = pExtendedWithOwn.SetRoleRights(ctx, assignor.ID, role1.ID, accesspolicy.APMove)
	a.NoError(err)

	parent, err = pExtendedWithOwn.Parent(ctx)
	a.NoError(err)
	a.Equal(wantedRights, parent.RightsRoster.Role[role1.ID])
	a.Equal(accesspolicy.APMove, pExtendedWithOwn.RightsRoster.Role[role1.ID])
	a.True(pExtendedWithOwn.HasRights(ctx, testaccesspolicy.ID, wantedRights|accesspolicy.APMove))
}

func TestSetUserRights(t *testing.T) {
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

	testuser, err := CreateTestUser(ctx, um, "testuser", "testuser@hometown.local", nil)
	a.NoError(err)

	wantedRights := accesspolicy.APView | accesspolicy.APChange

	// no parent, not inheriting and not extending
	// WARNING: [p] will be reused and inherited below in this function
	p, err := apm.Create(ctx, assignor.ID, 0, "test_name", "test_type", 1, false, false)
	err = p.SetUserRights(ctx, assignor.ID, testaccesspolicy.ID, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.User[testaccesspolicy.ID])
	a.True(p.HasRights(ctx, testaccesspolicy.ID, wantedRights))

	// with parent, using legacy only
	pWithInheritance, err := apm.Create(ctx, assignor.ID, p.ID, "another name", "another type", 1, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)

	parent, err := pWithInheritance.Parent(ctx)
	a.NoError(err)
	a.Equal(wantedRights, parent.RightsRoster.User[testaccesspolicy.ID])
	a.True(pWithInheritance.HasRights(ctx, testaccesspolicy.ID, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn, err := apm.Create(ctx, assignor.ID, p.ID, "the name", "the type", 1, false, true)
	a.NoError(err)

	parent, err = pExtendedNoOwn.Parent(ctx)
	a.NoError(err)
	a.Equal(wantedRights, parent.RightsRoster.User[testaccesspolicy.ID])
	a.Equal(accesspolicy.APNoAccess, pExtendedNoOwn.RightsRoster.User[testaccesspolicy.ID])
	a.True(pExtendedNoOwn.HasRights(ctx, testaccesspolicy.ID, wantedRights))

	// with parent, legacy false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added core.APMove to own rights
	pExtendedWithOwn, err := apm.Create(ctx, assignor.ID, p.ID, "yet another name", "and type", 1, false, true)
	a.NoError(err)

	err = pExtendedWithOwn.SetUserRights(ctx, assignor.ID, testaccesspolicy.ID, accesspolicy.APMove)
	a.NoError(err)

	parent, err = pExtendedWithOwn.Parent(ctx)
	a.NoError(err)
	a.Equal(wantedRights, parent.RightsRoster.User[testaccesspolicy.ID])
	a.Equal(accesspolicy.APMove, pExtendedWithOwn.RightsRoster.User[testaccesspolicy.ID])
	a.True(pExtendedWithOwn.HasRights(ctx, testaccesspolicy.ID, wantedRights|accesspolicy.APMove))
}

func TestIsOwner(t *testing.T) {
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
	testuser1, err := CreateTestUser(ctx, um, "testuser", "testuser@hometown.local", nil)
	a.NoError(err)

	testuser2, err := CreateTestUser(ctx, um, "testuser2", "testuser2@hometown.local", nil)
	a.NoError(err)

	ap, err := apm.Create(ctx, testuser1.ID, 0, "test_name", "test_type", 1, false, false)
	a.NoError(err)

	// testing owner and testuser rights
	a.NoError(ap.SetUserRights(ctx, testuser1.ID, testuser2.ID, accesspolicy.APView))
	a.Empty(ap.RightsRoster.User[testuser1.ID])
	a.Equal(accesspolicy.APView, ap.RightsRoster.User[testuser2.ID])
	a.NotEqual(accesspolicy.APNoAccess, ap.RightsRoster.User[testuser2.ID])
	a.True(ap.HasRights(ctx, testuser1.ID, accesspolicy.APView))
	a.True(ap.HasRights(ctx, testuser1.ID, accesspolicy.APChange))
	a.True(ap.HasRights(ctx, testuser1.ID, accesspolicy.APDelete))
	a.True(ap.HasRights(ctx, testuser1.ID, accesspolicy.APFullAccess))
	a.True(ap.HasRights(ctx, testuser2.ID, accesspolicy.APView))
	a.False(ap.HasRights(ctx, testuser2.ID, accesspolicy.APFullAccess))
	a.True(ap.IsOwner(ctx, testuser1.ID))
	a.False(ap.IsOwner(ctx, testuser2.ID))
}

func TestAccessPolicyClone(t *testing.T) {
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

	owner, err := CreateTestUser(ctx, um, "owner", "owner@hometown.local", nil)
	a.NoError(err)

	testuser, err := CreateTestUser(ctx, um, "testuser", "testuser@hometown.local", nil)
	a.NoError(err)

	ap, err := apm.Create(ctx, owner.ID, 0, "test_name", "test_type", 1, false, false)
	a.NoError(err)

	// testing owner and testuser rights
	a.NoError(ap.SetUserRights(ctx, owner.ID, testaccesspolicy.ID, accesspolicy.APView))
	a.Empty(ap.RightsRoster.User[owner.ID])
	a.Equal(accesspolicy.APView, ap.RightsRoster.User[testaccesspolicy.ID])
	a.NotEqual(accesspolicy.APNoAccess, ap.RightsRoster.User[testaccesspolicy.ID])
	a.True(ap.HasRights(ctx, owner.ID, accesspolicy.APView))
	a.True(ap.HasRights(ctx, owner.ID, accesspolicy.APChange))
	a.True(ap.HasRights(ctx, owner.ID, accesspolicy.APDelete))
	a.True(ap.HasRights(ctx, owner.ID, accesspolicy.APFullAccess))
	a.True(ap.HasRights(ctx, testaccesspolicy.ID, accesspolicy.APView))
	a.False(ap.HasRights(ctx, testaccesspolicy.ID, accesspolicy.APFullAccess))
	a.True(ap.IsOwner(ctx, owner.ID))
	a.False(ap.IsOwner(ctx, testaccesspolicy.ID))

	// cloning
	clone, err := ap.Clone()
	a.NoError(err)
	a.NotNil(clone)
	a.Equal(ap.ID, clone.ID)
	a.Equal(ap.ParentID, clone.ParentID)
	a.Equal(ap.OwnerID, clone.OwnerID)
	a.Equal(ap.Name, clone.Name)
	a.Equal(ap.ObjectType, clone.ObjectType)
	a.Equal(ap.ObjectID, clone.ObjectID)
	a.Equal(ap.IsInherited, clone.IsInherited)
	a.Equal(ap.IsExtended, clone.IsExtended)

	// testing whether access rights were also successfully cloned
	a.Empty(clone.RightsRoster.User[owner.ID])
	a.Equal(accesspolicy.APView, clone.RightsRoster.User[testaccesspolicy.ID])
	a.NotEqual(accesspolicy.APNoAccess, clone.RightsRoster.User[testaccesspolicy.ID])
	a.True(clone.HasRights(ctx, owner.ID, accesspolicy.APView))
	a.True(clone.HasRights(ctx, owner.ID, accesspolicy.APChange))
	a.True(clone.HasRights(ctx, owner.ID, accesspolicy.APDelete))
	a.True(clone.HasRights(ctx, owner.ID, accesspolicy.APFullAccess))
	a.True(clone.HasRights(ctx, testaccesspolicy.ID, accesspolicy.APView))
	a.False(clone.HasRights(ctx, testaccesspolicy.ID, accesspolicy.APFullAccess))
	a.True(clone.IsOwner(ctx, owner.ID))
	a.False(clone.IsOwner(ctx, testaccesspolicy.ID))
}

func TestAccessPolicyUnsetRights(t *testing.T) {
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

	assignor, err := CreateTestUser(ctx, um, "assignor", "assignor@hometown.local", nil)
	a.NoError(err)

	assignee, err := CreateTestUser(ctx, um, "assignee", "assignee@hometown.local", nil)
	a.NoError(err)

	role, err := gm.Create(ctx, GKRole, 0, "test_role_group", "Test Role Group")
	a.NoError(err)

	group, err := gm.Create(ctx, GKGroup, 0, "test_group", "Test Group")
	a.NoError(err)

	wantedRights := accesspolicy.APView | accesspolicy.APChange | accesspolicy.APCopy | accesspolicy.APDelete

	//---------------------------------------------------------------------------
	// user rights
	//---------------------------------------------------------------------------
	// setting
	ap, err := apm.Create(ctx, assignor.ID, 0, "test_name 1", "test_type 1", 1, false, false)
	a.NoError(err)
	a.NoError(ap.SetUserRights(ctx, assignor.ID, assignee.ID, wantedRights))
	a.Equal(wantedRights, ap.RightsRoster.User[assignee.ID])
	a.True(ap.HasRights(ctx, assignee.ID, wantedRights))

	// unsetting
	a.NoError(ap.UnsetRights(ctx, assignor.ID, assignee))
	a.NotContains(ap.RightsRoster.User, assignee.ID)
	a.False(ap.HasRights(ctx, assignee.ID, wantedRights))

	//---------------------------------------------------------------------------
	// role rights
	//---------------------------------------------------------------------------
	// setting
	ap, err = apm.Create(ctx, assignor.ID, 0, "test_name 2", "test_type 2", 1, false, false)
	a.NoError(err)
	a.NoError(ap.SetRoleRights(ctx, assignor.ID, role.ID, wantedRights))
	a.Equal(wantedRights, ap.RightsRoster.Role[role.ID])

	// unsetting
	a.NoError(ap.UnsetRights(ctx, assignor.ID, role))
	a.NotContains(ap.RightsRoster.Role, role.ID)

	//---------------------------------------------------------------------------
	// group rights
	//---------------------------------------------------------------------------
	// setting
	ap, err = apm.Create(ctx, assignor.ID, 0, "test_name 3", "test_type 3", 1, false, false)
	a.NoError(err)
	a.NoError(ap.SetGroupRights(ctx, assignor.ID, group.ID, wantedRights))
	a.Equal(wantedRights, ap.RightsRoster.Group[group.ID])

	// unsetting
	a.NoError(ap.UnsetRights(ctx, assignor.ID, group))
	a.NotContains(ap.RightsRoster.Group, group.ID)
}

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
