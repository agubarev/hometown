package accesspolicy_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/security/accesspolicy"
	"github.com/stretchr/testify/assert"
)

func TestNewAccessPolicyManager(t *testing.T) {
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
}

func TestAccessPolicyManagerCreate(t *testing.T) {
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

	//---------------------------------------------------------------------------
	// proceeding with the test
	// creating a normal policy with type name and ID set, no key
	//---------------------------------------------------------------------------
	key := accesspolicy.TKey{}
	typeName := accesspolicy.NewObjectName("with type and id, no key")
	objectID := uint32(1)

	ap, err := m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotZero(ap.ID)
	a.Zero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.False(ap.IsInherited())
	a.False(ap.IsExtended())

	// checking rights roster
	roster, err := m.RosterByPolicyID(ctx, ap.ID)
	a.NoError(err)
	a.NotNil(roster)
	a.Equal(accesspolicy.APNoAccess, roster.Everyone)

	//---------------------------------------------------------------------------
	// policy without an owner
	//---------------------------------------------------------------------------
	key = accesspolicy.TKey{}
	typeName = accesspolicy.NewObjectName("policy without an owner")
	objectID = uint32(1)

	ap, err = m.Create(
		ctx,
		key,      // key
		0,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotZero(ap.ID)
	a.Zero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.False(ap.IsInherited())
	a.False(ap.IsExtended())

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, ap.ID)
	a.NoError(err)
	a.NotNil(roster)
	a.Equal(accesspolicy.APNoAccess, roster.Everyone)

	//---------------------------------------------------------------------------
	// creating a policy with a key, object type and ID set
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("test key")
	typeName = accesspolicy.NewObjectName("with type, id and key")
	objectID = uint32(1)

	ap, err = m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotNil(ap)
	a.Zero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.False(ap.IsInherited())
	a.False(ap.IsExtended())

	//---------------------------------------------------------------------------
	// creating the same policy with the same object type and id
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("test key (to attempt duplication)")
	typeName = accesspolicy.NewObjectName("with type and id (to attempt duplication)")
	objectID = uint32(1)

	ap, err = m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotNil(ap)
	a.Zero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.False(ap.IsInherited())
	a.False(ap.IsExtended())

	// attempting to create a policy with a duplicate key
	// NOTE: must fail
	ap, err = m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.Error(err)
	a.EqualError(accesspolicy.ErrPolicyKeyTaken, err.Error())

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, ap.ID)
	a.Error(err)
	a.Nil(roster)

	//---------------------------------------------------------------------------
	// attempting to create a policy with an object name without object ID
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("with name but without id")
	typeName = accesspolicy.NewObjectName("test object")
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.Error(err)

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, ap.ID)
	a.Error(err)
	a.Nil(roster)

	//---------------------------------------------------------------------------
	// attempting to create a policy without an object name but with object ID set
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("without name but with id")
	typeName = accesspolicy.TObjectType{}
	objectID = uint32(1)

	ap, err = m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.Error(err)

	// checking rights roster
	roster, err = m.RosterByPolicyID(ctx, ap.ID)
	a.Error(err)

	//---------------------------------------------------------------------------
	// creating a re-usable parent policy
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("re-usable parent policy")
	typeName = accesspolicy.TObjectType{}
	objectID = uint32(0)

	basePolicy, err := m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotNil(basePolicy)
	a.Zero(basePolicy.ParentID)
	a.Equal(key, basePolicy.Key)
	a.Equal(typeName, basePolicy.ObjectName)
	a.Equal(objectID, basePolicy.ObjectID)
	a.False(basePolicy.IsInherited())
	a.False(basePolicy.IsExtended())

	//---------------------------------------------------------------------------
	// attempting to create a proper policy but with a non-existing parent
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("policy with non-existing parent")
	typeName = accesspolicy.TObjectType{}
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		123321,   // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// inheritance without a parent
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("policy inherits with no parent")
	typeName = accesspolicy.TObjectType{}
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,                   // key
		1,                     // owner ID
		123321,                // parent ID
		objectID,              // object ID
		typeName,              // object type
		accesspolicy.FInherit, // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// extension without a parent
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("policy extends with no parent")
	typeName = accesspolicy.TObjectType{}
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,                  // key
		1,                    // owner ID
		123321,               // parent ID
		objectID,             // object ID
		typeName,             // object type
		accesspolicy.FExtend, // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// proper but inherits and extends at the same time
	// NOTE: must fail
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("policy inherits and extends (must not be created)")
	typeName = accesspolicy.TObjectType{}
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,           // key
		1,             // owner ID
		basePolicy.ID, // parent ID
		objectID,      // object ID
		typeName,      // object type
		accesspolicy.FInherit|accesspolicy.FExtend, // flags
	)
	a.Error(err)

	//---------------------------------------------------------------------------
	// proper creation with inheritance only
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("proper policy with inheritance")
	typeName = accesspolicy.TObjectType{}
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,                   // key
		1,                     // owner ID
		basePolicy.ID,         // parent ID
		objectID,              // object ID
		typeName,              // object type
		accesspolicy.FInherit, // flags
	)
	a.NoError(err)
	a.NotNil(ap)
	a.NotZero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.True(ap.IsInherited())
	a.False(ap.IsExtended())

	//---------------------------------------------------------------------------
	// proper creation with extension only
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("proper policy with extension")
	typeName = accesspolicy.TObjectType{}
	objectID = uint32(0)

	ap, err = m.Create(
		ctx,
		key,                  // key
		1,                    // owner ID
		basePolicy.ID,        // parent ID
		objectID,             // object ID
		typeName,             // object type
		accesspolicy.FExtend, // flags
	)
	a.NoError(err)
	a.NotNil(ap)
	a.NotZero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.False(ap.IsInherited())
	a.True(ap.IsExtended())
}

func TestAccessPolicyManagerUpdate(t *testing.T) {
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

	//---------------------------------------------------------------------------
	// test policy
	//---------------------------------------------------------------------------
	key := accesspolicy.NewKey("test policy")
	typeName := accesspolicy.TObjectType{}
	objectID := uint32(0)

	ap, err := m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotZero(ap.ID)
	a.Zero(ap.ParentID)
	a.Equal(key, ap.Key)
	a.Equal(typeName, ap.ObjectName)
	a.Equal(objectID, ap.ObjectID)
	a.False(ap.IsInherited())
	a.False(ap.IsExtended())

	// checking rights roster
	roster, err := m.RosterByPolicyID(ctx, ap.ID)
	a.NoError(err)
	a.NotNil(roster)
	a.Equal(accesspolicy.APNoAccess, roster.Everyone)

	//---------------------------------------------------------------------------
	// creating base policy (to be used as a parent)
	//---------------------------------------------------------------------------
	key = accesspolicy.NewKey("base policy")
	typeName = accesspolicy.TObjectType{}
	objectID = uint32(0)

	basePolicy, err := m.Create(
		ctx,
		key,      // key
		1,        // owner ID
		0,        // parent ID
		objectID, // object ID
		typeName, // object type
		0,        // flags
	)
	a.NoError(err)
	a.NotZero(basePolicy.ID)
	a.Zero(basePolicy.ParentID)
	a.Equal(key, basePolicy.Key)
	a.Equal(typeName, basePolicy.ObjectName)
	a.Equal(objectID, basePolicy.ObjectID)
	a.False(basePolicy.IsInherited())
	a.False(basePolicy.IsExtended())

	// setting parent
	a.NoError(m.SetParent(ctx, ap.ID, basePolicy.ID))

	// re-obtaining updated policy
	ap, err = m.PolicyByID(ctx, ap.ID)
	a.NoError(err)
	a.Equal(basePolicy.ID, ap.ParentID)

	// re-obtaining updated policy
	// NOTE: parent must be set
	ap, err = m.PolicyByID(ctx, ap.ID)
	a.NoError(err)
	a.Equal(basePolicy.ID, ap.ParentID)
	a.Equal(ap.ParentID, basePolicy.ID)

	// unsetting parent
	a.NoError(m.SetParent(ctx, ap.ID, 0))

	// re-obtaining updated policy
	// NOTE: parent must be cleared
	ap, err = m.PolicyByID(ctx, ap.ID)
	a.NoError(err)
	a.Equal(uint32(0), ap.ParentID)
	a.Zero(ap.ParentID)

	// key, object name and id must not be changeable
	a.Error(ap.SetKey(accesspolicy.NewKey("new key"), 32))
	a.Error(ap.SetObjectType(accesspolicy.NewObjectName("new object name"), 32))

	// attempting to change object id and save
	ap.ObjectName = accesspolicy.NewObjectName("doesn't matter")
	ap.ObjectID = ap.ObjectID + 1
	a.EqualError(accesspolicy.ErrForbiddenChange, m.Update(ctx, ap).Error())

	// re-obtaining policy
	ap, err = m.PolicyByID(ctx, ap.ID)
	a.NoError(err)

	// set assignor rights
	a.NoError(m.SetRights(ctx, accesspolicy.SKUser, ap.ID, 1, 2, accesspolicy.APView))
	a.NoError(m.Update(ctx, ap))
	a.True(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APView))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APChange))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APMove))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APDelete))
	a.False(m.HasRights(ctx, accesspolicy.SKUser, ap.ID, 2, accesspolicy.APCreate))
}

/*
func TestAccessPolicyManagerSetRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// preparing testdata
	//---------------------------------------------------------------------------
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	um, ctx, err := ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)
	a.NotNil(ctx)

	apm := ctx.Value(CKAccessPolicyManager).(*user.AccessPolicyManager)
	a.NotNil(apm)

	gm := ctx.Value(CKGroupManager).(*GroupManager)
	a.NoError(err)
	a.NotNil(gm)

	assignor, err := CreateTestUser(ctx, um, "assignor", "assignor@hometown.local", nil)
	a.NoError(err)
	a.NotNil(assignor)

	// users
	u1, err := CreateTestUser(ctx, um, "testuser1", "testuser1@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u1)

	u2, err := CreateTestUser(ctx, um, "testuser2", "testuser2@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u2)

	u3, err := CreateTestUser(ctx, um, "testuser3", "testuser3@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u3)

	// roles
	role, err := gm.Create(ctx, GKRole, 0, "test_role_group", "Test Role Group")
	a.NoError(err)

	role2, err := gm.Create(ctx, GKRole, 0, "test_role_group2", "Test Role Group 2")
	a.NoError(err)

	// groups
	group, err := gm.Create(ctx, GKGroup, 0, "test_group", "Test Group")
	a.NoError(err)

	group2, err := gm.Create(ctx, GKGroup, 0, "test_group2", "Test Group 2")
	a.NoError(err)

	wantedRights := user.APView | user.APChange | user.APDelete

	//---------------------------------------------------------------------------
	// proceeding with the test
	//---------------------------------------------------------------------------
	// creating new policy
	ap, err := apm.Create(ctx, assignor.ID, 0, "test name", "test_object_type", 1, false, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Equal(assignor.ID, ap.OwnerID)
	a.Zero(ap.ParentID)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	// public
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, nil, wantedRights))

	// users
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, u1, wantedRights))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, u2, wantedRights))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, u3, wantedRights))

	// roles
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, role, wantedRights))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, role2, wantedRights))

	// groups
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, group, wantedRights))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, group2, wantedRights))

	ap, err = apm.Update(ctx, ap)
	a.NoError(err)
}

func TestAccessPolicyManagerBackupAndRestore(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// preparing testdata
	//---------------------------------------------------------------------------
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	um, ctx, err := ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)
	a.NotNil(ctx)

	apm := ctx.Value(CKAccessPolicyManager).(*user.AccessPolicyManager)
	a.NotNil(apm)

	gm := ctx.Value(CKGroupManager).(*GroupManager)
	a.NotNil(gm)

	assignor, err := CreateTestUser(ctx, um, "assignor", "assignor@hometown.local", nil)
	a.NoError(err)
	a.NotNil(assignor)

	// users
	u1, err := CreateTestUser(ctx, um, "testuser1", "testuser1@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u1)

	u2, err := CreateTestUser(ctx, um, "testuser2", "testuser2@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u2)

	u3, err := CreateTestUser(ctx, um, "testuser3", "testuser3@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u3)

	// roles
	role, err := gm.Create(ctx, GKRole, 0, "test_role_group", "Test Role Group")
	a.NoError(err)

	role2, err := gm.Create(ctx, GKRole, 0, "test_role_group2", "Test Role Group 2")
	a.NoError(err)

	// groups
	group, err := gm.Create(ctx, GKGroup, 0, "test_group", "Test Group")
	a.NoError(err)

	group2, err := gm.Create(ctx, GKGroup, 0, "test_group2", "Test Group 2")
	a.NoError(err)

	wantedRights := user.APView | user.APChange | user.APDelete

	//---------------------------------------------------------------------------
	// testing backup restore
	//---------------------------------------------------------------------------
	ap, err := apm.Create(ctx, assignor.ID, 0, "test name (restoring backup)", "test_object_type", 1, false, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Equal(assignor.ID, ap.OwnerID)
	a.Zero(ap.ParentID)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	// making sure that this rosters has no rights set yet
	a.Equal(user.APNoAccess, ap.RightsRoster.Everyone)
	a.Empty(ap.RightsRoster.User)
	a.Empty(ap.RightsRoster.Role)
	a.Empty(ap.RightsRoster.Group)

	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, nil, user.APView))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, u1, user.APView))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, role, user.APView))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, group, user.APView))

	// checking rights that were just set
	a.Equal(user.APView, ap.RightsRoster.Everyone)
	a.Equal(user.APView, ap.RightsRoster.User[u1.ID])
	a.Equal(user.APView, ap.RightsRoster.Role[role.ID])
	a.Equal(user.APView, ap.RightsRoster.Group[group.ID])

	// checking backup and saving policy
	// NOTE: at this point all the changes must be stored,
	// and backup cleared in case of success
	a.NotNil(ap.Backup())

	// backup must be reset after the policy is saved
	ap, err = apm.Update(ctx, ap)
	a.NoError(err)

	// checking whether backup was reset
	backup, err := ap.Backup()
	a.EqualError(ErrNoPolicyBackup, err.Error())
	a.Zero(backup.ID)
	a.Empty(backup.Name)
	a.Empty(backup.ObjectName)
	a.Empty(backup.ObjectID)

	// checking rights before making more policy changes
	a.Equal(user.APView, ap.RightsRoster.Everyone)
	a.Equal(user.APView, ap.RightsRoster.User[u1.ID])
	a.Equal(user.APView, ap.RightsRoster.Role[role.ID])
	a.Equal(user.APView, ap.RightsRoster.Group[group.ID])

	//---------------------------------------------------------------------------
	// more policy changes
	//---------------------------------------------------------------------------
	// public
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, nil, wantedRights))

	// users
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, u1, wantedRights))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, u2, wantedRights))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, u3, wantedRights))

	// roles
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, role, wantedRights))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, role2, wantedRights))

	// groups
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, group, wantedRights))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, group2, wantedRights))

	// restoring backup, policy fields must be replaced with backed up values,
	// and the backup itself must be cleared. must not have any changes applied
	// since its last saving
	a.NoError(ap.RestoreBackup())

	// checking rosters after backup restoration
	a.Equal(user.APView, ap.RightsRoster.Everyone)
	a.Equal(user.APView, ap.RightsRoster.User[u1.ID])
	a.Equal(user.APView, ap.RightsRoster.Role[role.ID])
	a.Equal(user.APView, ap.RightsRoster.Group[group.ID])
	a.Len(ap.RightsRoster.User, 1)
	a.Len(ap.RightsRoster.Role, 1)
	a.Len(ap.RightsRoster.Group, 1)
}

func TestAccessPolicyManagerDelete(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// preparing testdata
	//---------------------------------------------------------------------------
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	um, ctx, err := ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)
	a.NotNil(ctx)

	apm := ctx.Value(CKAccessPolicyManager).(*user.AccessPolicyManager)
	a.NotNil(apm)

	gm := ctx.Value(CKGroupManager).(*GroupManager)
	a.NotNil(gm)

	assignor, err := CreateTestUser(ctx, um, "assignor", "assignor@hometown.local", nil)
	a.NoError(err)
	a.NotNil(assignor)

	// users
	u1, err := CreateTestUser(ctx, um, "testuser1", "testuser1@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u1)

	u2, err := CreateTestUser(ctx, um, "testuser2", "testuser2@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u2)

	u3, err := CreateTestUser(ctx, um, "testuser3", "testuser3@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u3)

	// roles
	role, err := gm.Create(ctx, GKRole, 0, "test_role_group", "Test Role Group")
	a.NoError(err)

	role2, err := gm.Create(ctx, GKRole, 0, "test_role_group2", "Test Role Group 2")
	a.NoError(err)

	// groups
	group, err := gm.Create(ctx, GKGroup, 0, "test_group", "Test Group")
	a.NoError(err)

	group2, err := gm.Create(ctx, GKGroup, 0, "test_group2", "Test Group 2")
	a.NoError(err)

	wantedRights := user.APView | user.APChange | user.APDelete | user.APCopy

	//---------------------------------------------------------------------------
	// creating policy and setting rights
	//---------------------------------------------------------------------------
	ap, err := apm.Create(ctx, assignor.ID, 0, "test name", "test_object_type", 1, false, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Equal(assignor.ID, ap.OwnerID)
	a.Zero(ap.ParentID)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	// public
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, nil, wantedRights))

	// users
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, u1, wantedRights))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, u2, wantedRights))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, u3, wantedRights))

	// roles
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, role, wantedRights))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, role2, wantedRights))

	// groups
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, group, wantedRights))
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, group2, wantedRights))

	// saving policy
	ap, err = apm.Update(ctx, ap)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// making sure it's inside the container
	//---------------------------------------------------------------------------
	fetchedPolicy, err := apm.PolicyByID(ctx, ap.ID)
	a.NoError(err)
	a.True(reflect.DeepEqual(ap, fetchedPolicy))

	fetchedPolicy, err = apm.PolicyByName(ctx, ap.Name)
	a.NoError(err)
	a.NotNil(fetchedPolicy)
	a.True(reflect.DeepEqual(ap, fetchedPolicy))

	fetchedPolicy, err = apm.PolicyByObjectTypeAndID(ctx, ap.ObjectName, ap.ObjectID)
	a.NoError(err)
	a.NotNil(fetchedPolicy)
	a.True(reflect.DeepEqual(ap, fetchedPolicy))

	//---------------------------------------------------------------------------
	// deleting policy
	//---------------------------------------------------------------------------
	a.NoError(apm.DeletePolicy(ctx, ap))

	//---------------------------------------------------------------------------
	// attempting to get policies after their deletion
	//---------------------------------------------------------------------------
	fetchedPolicy, err = apm.PolicyByID(ctx, ap.ID)
	a.Error(err)
	a.EqualError(user.ErrAccessPolicyNotFound, err.Error())
	a.Zero(fetchedPolicy.ID)

	fetchedPolicy, err = apm.PolicyByName(ctx, ap.Name)
	a.Error(err)
	a.EqualError(user.ErrAccessPolicyNotFound, err.Error())
	a.Zero(fetchedPolicy.ID)

	fetchedPolicy, err = apm.PolicyByObjectTypeAndID(ctx, ap.ObjectName, ap.ObjectID)
	a.Error(err)
	a.EqualError(user.ErrAccessPolicyNotFound, err.Error())
	a.Zero(fetchedPolicy.ID)
}
*/
