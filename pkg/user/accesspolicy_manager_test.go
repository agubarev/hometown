package user_test

import (
	"reflect"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewAccessPolicyManager(t *testing.T) {
	a := assert.New(t)

	conn, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(conn)

	s, err := user.NewMySQLStore(conn)
	a.NoError(err)
	a.NotNil(s)

	aps, err := user.NewDefaultAccessPolicyStore(conn)
	a.NoError(err)
	a.NotNil(aps)

	c, err := user.NewAccessPolicyManager(aps)
	a.NoError(err)
	a.NotNil(c)
}

func TestAccessPolicyManagerCreate(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// preparing testdata
	//---------------------------------------------------------------------------
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	accessPolicyStore, err := user.NewDefaultAccessPolicyStore(db)
	a.NoError(err)
	a.NotNil(accessPolicyStore)

	um, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	apm := um.AccessPolicyManager()
	a.NotNil(apm)

	u, err := user.CreateTestUser(ctx, um, "testuser", "testuser@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u)

	//---------------------------------------------------------------------------
	// proceeding with the test
	//---------------------------------------------------------------------------
	// creating policy with just an object type name
	ap, err := apm.Create(ctx, 0, 0, "test policy name", "", 0, false, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Zero(ap.ParentID)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	// creating a policy with only its object type set, without object id
	ap, err = apm.Create(ctx, 0, 0, "", "test type name", 0, false, false)
	a.Error(err)

	// creating policy with object type and id
	ap, err = apm.Create(ctx, 0, 0, "test name", "test_object_type", 1, false, false)
	a.NoError(err)

	// creating the same policy with the same object type and id
	ap, err = apm.Create(ctx, 0, 0, "test name", "test_object_type", 1, false, false)
	a.Error(err)

	// creating the same policy with the same name
	ap, err = apm.Create(ctx, 0, 0, "test name", "", 0, false, false)
	a.Error(err)
	a.EqualError(user.ErrAccessPolicyNameTaken, err.Error())

	// creating a policy without a name and object type and id
	ap, err = apm.Create(ctx, 0, 0, "", "", 0, false, false)
	a.EqualError(user.ErrAccessPolicyEmptyDesignators, errors.Cause(err).Error())

	// with owner
	ap, err = apm.Create(ctx, u.ID, 0, "test name 2", "", 0, false, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Equal(u.ID, ap.OwnerID)
	a.Zero(ap.ParentID)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	//---------------------------------------------------------------------------
	// with parent
	//---------------------------------------------------------------------------
	// initializing a parent
	parent, err := apm.Create(ctx, 0, 0, "parent name", "", 0, false, false)
	a.NoError(err)
	a.NotNil(parent)

	// creating normally
	ap, err = apm.Create(ctx, 0, parent.ID, "test with parent name", "", 0, false, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Zero(ap.OwnerID)
	a.Equal(parent.ID, ap.ParentID)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	// inheritance without a parent set
	ap, err = apm.Create(ctx, 0, 0, "inheritance without a parent set", "", 0, true, false)
	a.Error(err)

	// extension without a parent set
	ap, err = apm.Create(ctx, 0, 0, "extension without a parent set", "", 0, false, true)
	a.Error(err)

	// extension and inheritance without a parent set
	ap, err = apm.Create(ctx, 0, 0, "extension and inheritance without a parent set", "", 0, true, true)
	a.Error(err)

	// proper creation with inheritance
	ap, err = apm.Create(ctx, 0, parent.ID, "proper creation with inheritance", "", 0, true, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Zero(ap.OwnerID)
	a.NotZero(ap.ParentID)
	a.Equal(parent.ID, ap.ParentID)
	a.True(ap.IsInherited)
	a.False(ap.IsExtended)

	// proper creation with extension
	ap, err = apm.Create(ctx, 0, parent.ID, "proper creation with extension", "", 0, false, true)
	a.NoError(err)
	a.NotNil(ap)
	a.Zero(ap.OwnerID)
	a.NotZero(ap.ParentID)
	a.Equal(parent.ID, ap.ParentID)
	a.False(ap.IsInherited)
	a.True(ap.IsExtended)

	// attempting to create with inheritance and extension
	ap, err = apm.Create(ctx, 0, parent.ID, "test name with inheritance and extension", "", 0, true, true)
	a.Error(err)
}

func TestAccessPolicyManagerUpdate(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// preparing testdata
	//---------------------------------------------------------------------------
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	um, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)
	a.NotNil(ctx)

	apm := ctx.Value(user.CKAccessPolicyManager).(*user.AccessPolicyManager)
	a.NotNil(apm)

	assignor, err := user.CreateTestUser(ctx, um, "assignor", "assignor@hometown.local", nil)
	a.NoError(err)
	a.NotNil(assignor)

	assignee, err := user.CreateTestUser(ctx, um, "assignee", "assignee@hometown.local", nil)
	a.NoError(err)
	a.NotNil(assignee)

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

	// creating parent policy
	parent, err := apm.Create(ctx, 0, 0, "parent policy", "", 0, false, false)
	a.NoError(err)
	a.NotNil(parent)
	a.Zero(parent.OwnerID)
	a.Zero(parent.ParentID)
	a.False(parent.IsInherited)
	a.False(parent.IsExtended)

	// setting parent
	a.NoError(ap.SetParentID(parent.ID))
	a.Equal(parent.ID, ap.ParentID)
	a.Equal(parent.ID, ap.ParentID)

	// unsetting parent
	a.NoError(ap.SetParentID(0))
	a.Zero(ap.ParentID)
	a.Equal(ap.ParentID, int64(0))

	// changing policy name
	newName := "updated name"
	ap.Name = newName

	// saving policy
	ap, err = apm.Update(ctx, ap)
	a.NoError(err)
	a.Equal(newName, ap.Name)

	// set assignor rights
	a.NoError(apm.SetRights(ctx, &ap, assignor.ID, assignee.ID, user.APView))
	ap, err = apm.Update(ctx, ap)
	a.NoError(err)
}

func TestAccessPolicyManagerSetRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// preparing testdata
	//---------------------------------------------------------------------------
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	um, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)
	a.NotNil(ctx)

	apm := ctx.Value(user.CKAccessPolicyManager).(*user.AccessPolicyManager)
	a.NotNil(apm)

	gm := ctx.Value(user.CKGroupManager).(*user.GroupManager)
	a.NoError(err)
	a.NotNil(gm)

	assignor, err := user.CreateTestUser(ctx, um, "assignor", "assignor@hometown.local", nil)
	a.NoError(err)
	a.NotNil(assignor)

	// users
	u1, err := user.CreateTestUser(ctx, um, "testuser1", "testuser1@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u1)

	u2, err := user.CreateTestUser(ctx, um, "testuser2", "testuser2@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u2)

	u3, err := user.CreateTestUser(ctx, um, "testuser3", "testuser3@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u3)

	// roles
	role, err := gm.Create(ctx, 0, user.GKRole, "test_role_group", "Test Role Group")
	a.NoError(err)

	role2, err := gm.Create(ctx, 0, user.GKRole, "test_role_group2", "Test Role Group 2")
	a.NoError(err)

	// groups
	group, err := gm.Create(ctx, 0, user.GKGroup, "test_group", "Test Group")
	a.NoError(err)

	group2, err := gm.Create(ctx, 0, user.GKGroup, "test_group2", "Test Group 2")
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

	um, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)
	a.NotNil(ctx)

	apm := ctx.Value(user.CKAccessPolicyManager).(*user.AccessPolicyManager)
	a.NotNil(apm)

	gm := ctx.Value(user.CKGroupManager).(*user.GroupManager)
	a.NotNil(gm)

	assignor, err := user.CreateTestUser(ctx, um, "assignor", "assignor@hometown.local", nil)
	a.NoError(err)
	a.NotNil(assignor)

	// users
	u1, err := user.CreateTestUser(ctx, um, "testuser1", "testuser1@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u1)

	u2, err := user.CreateTestUser(ctx, um, "testuser2", "testuser2@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u2)

	u3, err := user.CreateTestUser(ctx, um, "testuser3", "testuser3@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u3)

	// roles
	role, err := gm.Create(ctx, 0, user.GKRole, "test_role_group", "Test Role Group")
	a.NoError(err)

	role2, err := gm.Create(ctx, 0, user.GKRole, "test_role_group2", "Test Role Group 2")
	a.NoError(err)

	// groups
	group, err := gm.Create(ctx, 0, user.GKGroup, "test_group", "Test Group")
	a.NoError(err)

	group2, err := gm.Create(ctx, 0, user.GKGroup, "test_group2", "Test Group 2")
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

	// making sure that this roster has no rights set yet
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
	a.EqualError(user.ErrNoPolicyBackup, err.Error())
	a.Zero(backup.ID)
	a.Empty(backup.Name)
	a.Empty(backup.ObjectType)
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

	// checking roster after backup restoration
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

	um, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)
	a.NotNil(ctx)

	apm := ctx.Value(user.CKAccessPolicyManager).(*user.AccessPolicyManager)
	a.NotNil(apm)

	gm := ctx.Value(user.CKGroupManager).(*user.GroupManager)
	a.NotNil(gm)

	assignor, err := user.CreateTestUser(ctx, um, "assignor", "assignor@hometown.local", nil)
	a.NoError(err)
	a.NotNil(assignor)

	// users
	u1, err := user.CreateTestUser(ctx, um, "testuser1", "testuser1@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u1)

	u2, err := user.CreateTestUser(ctx, um, "testuser2", "testuser2@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u2)

	u3, err := user.CreateTestUser(ctx, um, "testuser3", "testuser3@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u3)

	// roles
	role, err := gm.Create(ctx, 0, user.GKRole, "test_role_group", "Test Role Group")
	a.NoError(err)

	role2, err := gm.Create(ctx, 0, user.GKRole, "test_role_group2", "Test Role Group 2")
	a.NoError(err)

	// groups
	group, err := gm.Create(ctx, 0, user.GKGroup, "test_group", "Test Group")
	a.NoError(err)

	group2, err := gm.Create(ctx, 0, user.GKGroup, "test_group2", "Test Group 2")
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

	fetchedPolicy, err = apm.PolicyByObjectTypeAndID(ctx, ap.ObjectType, ap.ObjectID)
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

	fetchedPolicy, err = apm.PolicyByObjectTypeAndID(ctx, ap.ObjectType, ap.ObjectID)
	a.Error(err)
	a.EqualError(user.ErrAccessPolicyNotFound, err.Error())
	a.Zero(fetchedPolicy.ID)
}
