package user_test

import (
	"reflect"
	"testing"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/stretchr/testify/assert"
)

func TestNewAccessPolicyContainer(t *testing.T) {
	a := assert.New(t)

	s, err := user.NewMemoryStore()
	a.NoError(err)
	a.NotNil(s)

	c, err := user.NewAccessPolicyManager(s)
	a.NoError(err)
	a.NotNil(c)
}

func TestAccessPolicyContainerCreate(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// preparing testdata
	//---------------------------------------------------------------------------
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)
	a.NoError(core.TruncateDatabaseForTesting(db))

	accessPolicyStore, err := user.NewDefaultAccessPolicyStore(db)
	a.NoError(err)
	a.NotNil(accessPolicyStore)

	accessPolicyContainer, err := user.NewAccessPolicyManager(accessPolicyStore)
	a.NoError(err)
	a.NotNil(accessPolicyContainer)

	userStore, err := core.NewUserStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	userManager, err := core.NewUserManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	user, err := userManager.Create("testuser")
	a.NoError(err)
	a.NotNil(user)

	//---------------------------------------------------------------------------
	// proceeding with the test
	//---------------------------------------------------------------------------
	// creating policy with just a key
	ap, err := accessPolicyContainer.Create(context.Background(), nil, nil, "test key", "test_kind", 1, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Nil(ap.Owner)
	a.Nil(ap.Parent)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	// creating a policy with only its kind set, without an ObjectID
	ap, err = accessPolicyContainer.Create(context.Background(), nil, nil, "", "test_kind", 0, false)
	a.Error(err)

	// creating the same policy with the same kind and id
	ap, err = accessPolicyContainer.Create(context.Background(), nil, nil, "", "test_kind", 1, false)
	a.Error(err)

	// creating the same policy with the same key
	ap, err = accessPolicyContainer.Create(context.Background(), nil, nil, "test key", "test_kind", 1, false)
	a.EqualError(core.ErrAccessPolicyNameTaken, err.Error())

	// creating a policy without a key and kind+id
	ap, err = accessPolicyContainer.Create(context.Background(), nil, nil, "", "", 0, false)
	a.EqualError(core.ErrAccessPolicyEmptyDesignators, err.Error())

	// with owner
	ap, err = accessPolicyContainer.Create(context.Background(), user, nil, "test key 2", "", 0, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Equal(user, ap.Owner)
	a.Nil(ap.Parent)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	//---------------------------------------------------------------------------
	// with parent
	//---------------------------------------------------------------------------
	// initializing a parent
	parent, err := accessPolicyContainer.Create(context.Background(), nil, nil, "parent key", "", 0, false)
	a.NoError(err)
	a.NotNil(parent)

	// creating normally
	ap, err = accessPolicyContainer.Create(context.Background(), nil, parent, "test with parent key", "", 0, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Nil(ap.Owner)
	a.Equal(parent, ap.Parent)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	// inheritance without a parent set
	ap, err = accessPolicyContainer.Create(context.Background(), nil, nil, "test key with inheritance", "", 0, true)
	a.Error(err)

	// extension without a parent set
	ap, err = accessPolicyContainer.Create(context.Background(), nil, nil, "test key with inheritance", "", 0, false)
	a.Error(err)

	// extension and inheritance without a parent set
	ap, err = accessPolicyContainer.Create(context.Background(), nil, nil, "test key with inheritance", "", 0, true)
	a.Error(err)

	// proper creation with inheritance
	ap, err = accessPolicyContainer.Create(context.Background(), nil, parent, "test key with inheritance", "", 0, true)
	a.NoError(err)
	a.NotNil(ap)
	a.Nil(ap.Owner)
	a.NotNil(ap.Parent)
	a.Equal(parent, ap.Parent)
	a.True(ap.IsInherited)
	a.False(ap.IsExtended)

	// proper creation with extension
	ap, err = accessPolicyContainer.Create(context.Background(), nil, parent, "test key with extension", "", 0, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Nil(ap.Owner)
	a.NotNil(ap.Parent)
	a.Equal(parent, ap.Parent)
	a.False(ap.IsInherited)
	a.True(ap.IsExtended)

	// attempting to create with inheritance and extension
	ap, err = accessPolicyContainer.Create(context.Background(), nil, parent, "test key with inheritance and extension", "", 0, true)
	a.Error(err)
}

func TestAccessPolicyContainerUpdate(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// preparing testdata
	//---------------------------------------------------------------------------
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)
	a.NoError(core.TruncateDatabaseForTesting(db))

	accessPolicyStore, err := user.NewDefaultAccessPolicyStore(db)
	a.NoError(err)
	a.NotNil(accessPolicyStore)

	accessPolicyContainer, err := user.NewAccessPolicyManager(accessPolicyStore)
	a.NoError(err)
	a.NotNil(accessPolicyContainer)

	userStore, err := core.NewUserStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	userManager, err := core.NewUserManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	user, err := userManager.Create("testuser")
	a.NoError(err)
	a.NotNil(user)

	assignee, err := userManager.Create("assignee")
	a.NoError(err)
	a.NotNil(user)

	//---------------------------------------------------------------------------
	// proceeding with the test
	//---------------------------------------------------------------------------
	// creating new policy
	ap, err := accessPolicyContainer.Create(context.Background(), user, nil, "test key", "test_kind", 1, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Equal(user, ap.Owner)
	a.Nil(ap.Parent)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	// creating parent policy
	parent, err := accessPolicyContainer.Create(context.Background(), nil, nil, "parent policy", "", 0, false)
	a.NoError(err)
	a.NotNil(parent)
	a.Nil(parent.Owner)
	a.Nil(parent.Parent)
	a.False(parent.IsInherited)
	a.False(parent.IsExtended)

	// setting parent
	a.NoError(ap.SetParent(parent))
	a.Equal(parent, ap.Parent)
	a.Equal(parent.ID, ap.ParentID)
	a.NoError(accessPolicyContainer.Save(ap))

	// unsetting parent
	a.NoError(ap.SetParent(nil))
	a.Nil(ap.Parent)
	a.Equal(int64(0), ap.ParentID)
	a.NoError(accessPolicyContainer.Save(ap))

	// changing key
	ap.Key = "updated key"
	a.NoError(accessPolicyContainer.Save(ap))
	a.Equal("updated key", ap.Key)

	// set user rights
	a.NoError(accessPolicyContainer.SetRights(ap, user, assignee, user.APView))
	a.NoError(accessPolicyContainer.Save(ap))
}

func TestAccessPolicyContainerSetRights(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// preparing testdata
	//---------------------------------------------------------------------------
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)
	a.NoError(core.TruncateDatabaseForTesting(db))

	accessPolicyStore, err := user.NewDefaultAccessPolicyStore(db)
	a.NoError(err)
	a.NotNil(accessPolicyStore)

	accessPolicyContainer, err := user.NewAccessPolicyManager(accessPolicyStore)
	a.NoError(err)
	a.NotNil(accessPolicyContainer)

	userStore, err := core.NewUserStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	userManager, err := core.NewUserManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	groupContainer, err := core.NewGroupContainerForTesting(db)
	a.NoError(err)
	a.NotNil(groupContainer)

	assignor, err := userManager.Create("assignor")
	a.NoError(err)
	a.NotNil(assignor)

	// users
	user, err := userManager.Create("testuser")
	a.NoError(err)
	a.NotNil(assignor)

	user2, err := userManager.Create("testuser2")
	a.NoError(err)
	a.NotNil(assignor)

	user3, err := userManager.Create("testuser3")
	a.NoError(err)
	a.NotNil(assignor)

	// roles
	role, err := groupContainer.Create(user.GKRole, "test_role_group", "Test Role Group", nil)
	a.NoError(err)

	role2, err := groupContainer.Create(user.GKRole, "test_role_group2", "Test Role Group 2", nil)
	a.NoError(err)

	// groups
	group, err := groupContainer.Create(user.GKGroup, "test_group", "Test Group", nil)
	a.NoError(err)

	group2, err := groupContainer.Create(group.GKGroup, "test_group2", "Test Group 2", nil)
	a.NoError(err)

	wantedRights := user.APView | user.APChange | user.APDelete

	//---------------------------------------------------------------------------
	// proceeding with the test
	//---------------------------------------------------------------------------
	// creating new policy
	ap, err := accessPolicyContainer.Create(context.Background(), assignor, nil, "test key", "test_kind", 1, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Equal(assignor, ap.Owner)
	a.Nil(ap.Parent)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	// public
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, nil, wantedRights))

	// users
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, user, wantedRights))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, user2, wantedRights))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, user3, wantedRights))

	// roles
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, role, wantedRights))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, role2, wantedRights))

	// groups
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, group, wantedRights))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, group2, wantedRights))

	a.NoError(accessPolicyContainer.Save(ap))
}

func TestAccessPolicyContainerBackupAndRestore(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// preparing testdata
	//---------------------------------------------------------------------------
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)
	a.NoError(core.TruncateDatabaseForTesting(db))

	accessPolicyStore, err := user.NewDefaultAccessPolicyStore(db)
	a.NoError(err)
	a.NotNil(accessPolicyStore)

	accessPolicyContainer, err := user.NewAccessPolicyManager(accessPolicyStore)
	a.NoError(err)
	a.NotNil(accessPolicyContainer)

	userStore, err := core.NewUserStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	userManager, err := core.NewUserManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	groupContainer, err := core.NewGroupContainerForTesting(db)
	a.NoError(err)
	a.NotNil(groupContainer)

	assignor, err := userManager.Create("assignor")
	a.NoError(err)
	a.NotNil(assignor)

	// users
	user, err := userManager.Create("testuser")
	a.NoError(err)
	a.NotNil(assignor)

	user2, err := userManager.Create("testuser2")
	a.NoError(err)
	a.NotNil(assignor)

	user3, err := userManager.Create("testuser3")
	a.NoError(err)
	a.NotNil(assignor)

	// roles
	role, err := groupContainer.Create(user.GKRole, "test_role_group", "Test Role Group", nil)
	a.NoError(err)

	role2, err := groupContainer.Create(user.GKRole, "test_role_group2", "Test Role Group 2", nil)
	a.NoError(err)

	// groups
	group, err := groupContainer.Create(user.GKGroup, "test_group", "Test Group", nil)
	a.NoError(err)

	group2, err := groupContainer.Create(group.GKGroup, "test_group2", "Test Group 2", nil)
	a.NoError(err)

	wantedRights := user.APView | user.APChange | user.APDelete

	//---------------------------------------------------------------------------
	// testing backup restore
	//---------------------------------------------------------------------------
	ap, err := accessPolicyContainer.Create(context.Background(), assignor, nil, "test key (restoring backup)", "test_kind", 1, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Equal(assignor, ap.Owner)
	a.Nil(ap.Parent)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	// making sure that this roster has no rights set yet
	a.Equal(user.APNoAccess, ap.RightsRoster.Everyone)
	a.Empty(ap.RightsRoster.User)
	a.Empty(ap.RightsRoster.Role)
	a.Empty(ap.RightsRoster.Group)

	a.NoError(accessPolicyContainer.SetRights(ap, assignor, nil, user.APView))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, user, user.APView))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, role, user.APView))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, group, user.APView))

	// checking rights that were just set
	a.Equal(user.APView, ap.RightsRoster.Everyone)
	a.Equal(user.APView, ap.RightsRoster.User[user.ID])
	a.Equal(user.APView, ap.RightsRoster.Role[role.ID])
	a.Equal(user.APView, ap.RightsRoster.Group[group.ID])

	// checking backup and saving policy
	// NOTE: at this point all the changes must be stored,
	// and backup cleared in case of success
	a.NotNil(ap.Backup())
	a.NoError(accessPolicyContainer.Save(ap))
	a.Nil(ap.Backup())

	// checking rights before making more policy changes
	a.Equal(user.APView, ap.RightsRoster.Everyone)
	a.Equal(user.APView, ap.RightsRoster.User[user.ID])
	a.Equal(user.APView, ap.RightsRoster.Role[role.ID])
	a.Equal(user.APView, ap.RightsRoster.Group[group.ID])

	//---------------------------------------------------------------------------
	// more policy changes
	//---------------------------------------------------------------------------
	// public
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, nil, wantedRights))

	// users
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, user, wantedRights))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, user2, wantedRights))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, user3, wantedRights))

	// roles
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, role, wantedRights))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, role2, wantedRights))

	// groups
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, group, wantedRights))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, group2, wantedRights))

	// restoring backup, policy fields must be replaced with backed up values,
	// and the backup itself must be cleared. must not have any changes applied
	// since its last saving
	a.NoError(ap.RestoreBackup())

	// checking roster after backup restoration
	a.Equal(user.APView, ap.RightsRoster.Everyone)
	a.Equal(user.APView, ap.RightsRoster.User[user.ID])
	a.Equal(user.APView, ap.RightsRoster.Role[role.ID])
	a.Equal(user.APView, ap.RightsRoster.Group[group.ID])
	a.Len(ap.RightsRoster.User, 1)
	a.Len(ap.RightsRoster.Role, 1)
	a.Len(ap.RightsRoster.Group, 1)
}

func TestAccessPolicyContainerDelete(t *testing.T) {
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

	accessPolicyContainer, err := user.NewAccessPolicyManager(accessPolicyStore)
	a.NoError(err)
	a.NotNil(accessPolicyContainer)

	userStore, err := NewMySQLStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	userManager, err := NewManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	groupContainer, err := core.NewGroupContainerForTesting(db)
	a.NoError(err)
	a.NotNil(groupContainer)

	assignor, err := userManager.CreateUser("assignor")
	a.NoError(err)
	a.NotNil(assignor)

	// users
	user, err := userManager.CreateUser("testuser")
	a.NoError(err)
	a.NotNil(assignor)

	user2, err := userManager.CreateUser("testuser2")
	a.NoError(err)
	a.NotNil(assignor)

	user3, err := userManager.CreateUser("testuser3")
	a.NoError(err)
	a.NotNil(assignor)

	// roles
	role, err := groupContainer.Create(user.GKRole, "test_role_group", "Test Role Group", nil)
	a.NoError(err)

	role2, err := groupContainer.Create(user.GKRole, "test_role_group2", "Test Role Group 2", nil)
	a.NoError(err)

	// groups
	group, err := groupContainer.Create(user.GKGroup, "test_group", "Test Group", nil)
	a.NoError(err)

	group2, err := groupContainer.Create(group.GKGroup, "test_group2", "Test Group 2", nil)
	a.NoError(err)

	wantedRights := user.APView | user.APChange | user.APDelete | user.APCopy

	//---------------------------------------------------------------------------
	// creating policy and setting rights
	//---------------------------------------------------------------------------
	ap, err := accessPolicyContainer.Create(context.Background(), assignor, nil, "test key", "test_kind", 1, false)
	a.NoError(err)
	a.NotNil(ap)
	a.Equal(assignor, ap.Owner)
	a.Equal(assignor.ID, ap.OwnerID)
	a.Nil(ap.Parent)
	a.False(ap.IsInherited)
	a.False(ap.IsExtended)

	// public
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, nil, wantedRights))

	// users
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, user, wantedRights))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, user2, wantedRights))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, user3, wantedRights))

	// roles
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, role, wantedRights))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, role2, wantedRights))

	// groups
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, group, wantedRights))
	a.NoError(accessPolicyContainer.SetRights(ap, assignor, group2, wantedRights))

	// saving policy (through itself)
	a.NoError(ap.Save())

	//---------------------------------------------------------------------------
	// making sure it's inside the container
	//---------------------------------------------------------------------------
	newPolicy, err := accessPolicyContainer.PolicyByID(ap.ID)
	a.NoError(err)
	a.NotNil(newPolicy)
	a.True(reflect.DeepEqual(ap, newPolicy))

	newPolicy, err = accessPolicyContainer.PolicyByName(ap.Key)
	a.NoError(err)
	a.NotNil(newPolicy)
	a.True(reflect.DeepEqual(ap, newPolicy))

	newPolicy, err = accessPolicyContainer.PolicyByObjectTypeAndID(ap.ObjectType, ap.ObjectID)
	a.NoError(err)
	a.NotNil(newPolicy)
	a.True(reflect.DeepEqual(ap, newPolicy))

	//---------------------------------------------------------------------------
	// deleting policy
	//---------------------------------------------------------------------------
	a.NoError(accessPolicyContainer.DeletePolicy(ap))

	//---------------------------------------------------------------------------
	// attempting to get policies after their deletion
	//---------------------------------------------------------------------------
	newPolicy, err = accessPolicyContainer.PolicyByID(ap.ID)
	a.EqualError(core.ErrAccessPolicyNotFound, err.Error())
	a.Nil(newPolicy)

	newPolicy, err = accessPolicyContainer.PolicyByName(ap.Key)
	a.EqualError(core.ErrAccessPolicyNotFound, err.Error())
	a.Nil(newPolicy)

	newPolicy, err = accessPolicyContainer.PolicyByObjectTypeAndID(ap.ObjectType, ap.ObjectID)
	a.EqualError(core.ErrAccessPolicyNotFound, err.Error())
	a.Nil(newPolicy)
}
