package usermanager_test

import (
	"testing"

	"github.com/agubarev/hometown/usermanager"

	"github.com/stretchr/testify/assert"
)

var testReusableUserinfo = map[string]string{
	"firstname": "Andrei",
	"lastname":  "Gubarev",
}

func TestNewAccessPolicy(t *testing.T) {
	a := assert.New(t)

	p := usermanager.NewAccessPolicy(nil, nil, false, false)
	a.NotNil(p)
	a.Nil(p.Owner)
	a.Nil(p.Parent)
	a.False(p.IsInherited)
	a.False(p.IsExtended)

	// with owner
	user, err := usermanager.NewUser("testuser", "testuser@example.com", testReusableUserinfo)
	a.NoError(err)

	p = usermanager.NewAccessPolicy(user, nil, false, false)
	a.NotNil(p)
	a.Equal(user, p.Owner)
	a.Nil(p.Parent)
	a.False(p.IsInherited)
	a.False(p.IsExtended)

	// with parent
	parent := usermanager.NewAccessPolicy(nil, nil, false, false)
	a.NoError(err)

	p = usermanager.NewAccessPolicy(nil, parent, false, false)
	a.NotNil(p)
	a.Nil(p.Owner)
	a.Equal(parent, p.Parent)
	a.False(p.IsInherited)
	a.False(p.IsExtended)

	// with inheritance
	p = usermanager.NewAccessPolicy(nil, nil, true, false)
	a.NotNil(p)
	a.Nil(p.Owner)
	a.Nil(p.Parent)
	a.True(p.IsInherited)
	a.False(p.IsExtended)

	// with extension
	p = usermanager.NewAccessPolicy(nil, nil, false, true)
	a.NotNil(p)
	a.Nil(p.Owner)
	a.Nil(p.Parent)
	a.False(p.IsInherited)
	a.True(p.IsExtended)
}

func TestSetPublicRights(t *testing.T) {
	a := assert.New(t)

	assignor, err := usermanager.NewUser("assignor", "testuser@example.com", testReusableUserinfo)
	a.NoError(err)

	testuser, err := usermanager.NewUser("testuser", "testuser@example.com", testReusableUserinfo)
	a.NoError(err)

	wantedRights := usermanager.APView | usermanager.APChange

	// no parent, not inheriting and not extending
	p := usermanager.NewAccessPolicy(assignor, nil, false, false)
	err = p.SetPublicRights(assignor, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.Everyone)
	a.True(p.HasRights(testuser, wantedRights))

	// with parent, using legacy only
	pWithInheritance := usermanager.NewAccessPolicy(assignor, p, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)
	a.Equal(wantedRights, pWithInheritance.Parent.RightsRoster.Everyone)
	a.True(pWithInheritance.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn := usermanager.NewAccessPolicy(assignor, p, false, true)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedNoOwn.Parent.RightsRoster.Everyone)
	a.Equal(usermanager.APNoAccess, pExtendedNoOwn.RightsRoster.Everyone)
	a.True(pExtendedNoOwn.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added usermanager.APMove to own rights
	pExtendedWithOwn := usermanager.NewAccessPolicy(assignor, p, false, true)
	pExtendedWithOwn.SetPublicRights(assignor, usermanager.APMove)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedWithOwn.Parent.RightsRoster.Everyone)
	a.Equal(usermanager.APMove, pExtendedWithOwn.RightsRoster.Everyone)
	a.True(pExtendedWithOwn.HasRights(testuser, wantedRights|usermanager.APMove))
}

func TestSetGroupRights(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)
	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	userManager, err := usermanager.NewUserManagerForTesting(db)
	a.NoError(err)
	a.NotNil(userManager)

	groupManager, err := userManager.GroupManager()
	a.NoError(err)
	a.NotNil(groupManager)

	// creating test users via container
	assignor, err := userManager.Create("assignor", "test.assignor@example.com", testReusableUserinfo)
	a.NoError(err)

	testuser, err := userManager.Create("testuser", "testuser@example.com", testReusableUserinfo)
	a.NoError(err)

	// adding the user to 2 groups but setting rights to only one
	group1, err := groupManager.Create(usermanager.GKGroup, "test_group_1", "test group 1", nil)
	a.NoError(err)

	err = group1.AddMember(testuser)
	a.NoError(err)

	group2, err := groupManager.Create(usermanager.GKGroup, "test_group_2", "test group 2", nil)
	a.NoError(err)

	err = group2.AddMember(testuser)
	a.NoError(err)

	wantedRights := usermanager.APView | usermanager.APChange

	// no parent, not inheriting and not extending
	// WARNING: "p" will be reused and inherited below in this function
	p := usermanager.NewAccessPolicy(assignor, nil, false, false)
	err = p.SetGroupRights(assignor, group1, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.Group[group1.ID])
	a.True(p.HasRights(testuser, wantedRights))

	// with parent, using legacy only
	pWithInheritance := usermanager.NewAccessPolicy(assignor, p, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)
	a.Equal(wantedRights, pWithInheritance.Parent.RightsRoster.Group[group1.ID])
	a.True(pWithInheritance.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn := usermanager.NewAccessPolicy(assignor, p, false, true)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedNoOwn.Parent.RightsRoster.Group[group1.ID])
	a.Equal(usermanager.APNoAccess, pExtendedNoOwn.RightsRoster.Group[group1.ID])
	a.True(pExtendedNoOwn.HasRights(testuser, wantedRights))

	// with parent, inheritance false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added usermanager.APMove to own rights
	pExtendedWithOwn := usermanager.NewAccessPolicy(assignor, p, false, true)
	pExtendedWithOwn.SetGroupRights(assignor, group1, usermanager.APMove)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedWithOwn.Parent.RightsRoster.Group[group1.ID])
	a.Equal(usermanager.APMove, pExtendedWithOwn.RightsRoster.Group[group1.ID])
	a.True(pExtendedWithOwn.HasRights(testuser, wantedRights|usermanager.APMove))
}

func TestSetRoleRights(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	// truncating test database
	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	userStore, err := usermanager.NewUserStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	userManager, err := usermanager.NewUserManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	// creating test users via container
	assignor, err := userManager.Create("assignor", "test.assignor@example.com", testReusableUserinfo)
	a.NoError(err)

	testuser, err := userManager.Create("testuser", "testuser@example.com", testReusableUserinfo)
	a.NoError(err)

	gc, err := usermanager.NewGroupContainerForTesting(db)
	a.NoError(err)
	a.NotNil(gc)

	// adding the user to 2 groups but setting rights to only one
	role1, err := gc.Create(usermanager.GKRole, "test_role_1", "test role 1", nil)
	a.NoError(err)

	err = role1.AddMember(testuser)
	a.NoError(err)

	role2, err := gc.Create(usermanager.GKRole, "test_role_2", "test role 2", nil)
	a.NoError(err)

	err = role2.AddMember(testuser)
	a.NoError(err)

	wantedRights := usermanager.APView | usermanager.APChange

	// no parent, not inheriting and not extending
	// NOTE: "p" will be reused and inherited below in this function
	p := usermanager.NewAccessPolicy(assignor, nil, false, false)
	err = p.SetRoleRights(assignor, role1, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.Role[role1.ID])
	a.True(p.HasRights(testuser, wantedRights))

	// with parent, using legacy only
	pWithInheritance := usermanager.NewAccessPolicy(assignor, p, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)
	a.Equal(wantedRights, pWithInheritance.Parent.RightsRoster.Role[role1.ID])
	a.True(pWithInheritance.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn := usermanager.NewAccessPolicy(assignor, p, false, true)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedNoOwn.Parent.RightsRoster.Role[role1.ID])
	a.Equal(usermanager.APNoAccess, pExtendedNoOwn.RightsRoster.Role[role1.ID])
	a.True(pExtendedNoOwn.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added usermanager.APMove to own rights
	pExtendedWithOwn := usermanager.NewAccessPolicy(assignor, p, false, true)
	pExtendedWithOwn.SetRoleRights(assignor, role1, usermanager.APMove)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedWithOwn.Parent.RightsRoster.Role[role1.ID])
	a.Equal(usermanager.APMove, pExtendedWithOwn.RightsRoster.Role[role1.ID])
	a.True(pExtendedWithOwn.HasRights(testuser, wantedRights|usermanager.APMove))
}

func TestSetUserRights(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	// truncating test database
	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	userStore, err := usermanager.NewUserStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	userManager, err := usermanager.NewUserManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	// creating test users via container
	assignor, err := userManager.Create("assignor", "test.assignor@example.com", testReusableUserinfo)
	a.NoError(err)

	testuser, err := userManager.Create("testuser", "testuser@example.com", testReusableUserinfo)
	a.NoError(err)

	wantedRights := usermanager.APView | usermanager.APChange

	// no parent, not inheriting and not extending
	// WARNING: [p] will be reused and inherited below in this function
	p := usermanager.NewAccessPolicy(assignor, nil, false, false)
	err = p.SetUserRights(assignor, testuser, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.User[testuser.ID])
	a.True(p.HasRights(testuser, wantedRights))

	// with parent, using legacy only
	pWithInheritance := usermanager.NewAccessPolicy(assignor, p, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)
	a.Equal(wantedRights, pWithInheritance.Parent.RightsRoster.User[testuser.ID])
	a.True(pWithInheritance.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn := usermanager.NewAccessPolicy(assignor, p, false, true)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedNoOwn.Parent.RightsRoster.User[testuser.ID])
	a.Equal(usermanager.APNoAccess, pExtendedNoOwn.RightsRoster.User[testuser.ID])
	a.True(pExtendedNoOwn.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added usermanager.APMove to own rights
	pExtendedWithOwn := usermanager.NewAccessPolicy(assignor, p, false, true)
	pExtendedWithOwn.SetUserRights(assignor, testuser, usermanager.APMove)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedWithOwn.Parent.RightsRoster.User[testuser.ID])
	a.Equal(usermanager.APMove, pExtendedWithOwn.RightsRoster.User[testuser.ID])
	a.True(pExtendedWithOwn.HasRights(testuser, wantedRights|usermanager.APMove))
}

func TestIsOwner(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	// truncating test database
	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	userStore, err := usermanager.NewUserStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	userManager, err := usermanager.NewUserManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	testuser, err := userManager.Create("testuser", "testuser@example.com", testReusableUserinfo)
	a.NoError(err)

	testuser2, err := userManager.Create("testuser2", "testuser2@example.com", testReusableUserinfo)
	a.NoError(err)

	p := usermanager.NewAccessPolicy(testuser, nil, false, false)
	err = p.SetUserRights(testuser, testuser, usermanager.APView)
	a.NoError(err)
	a.Equal(usermanager.APView, p.RightsRoster.User[testuser.ID])
	a.Equal(usermanager.APNoAccess, p.RightsRoster.User[testuser2.ID])
	a.True(p.HasRights(testuser, usermanager.APFullAccess))
	a.False(p.HasRights(testuser2, usermanager.APView))
	a.False(p.HasRights(testuser2, usermanager.APFullAccess))
	a.True(p.IsOwner(testuser))
	a.False(p.IsOwner(testuser2))
}

func TestAccessPolicyClone(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	// truncating test database
	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	userStore, err := usermanager.NewUserStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	userManager, err := usermanager.NewUserManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	owner, err := userManager.Create("owner", "owner@example.com", testReusableUserinfo)
	a.NoError(err)

	testuser, err := userManager.Create("testuser", "testuser@example.com", testReusableUserinfo)
	a.NoError(err)

	p := usermanager.NewAccessPolicy(owner, nil, false, false)
	err = p.SetUserRights(owner, owner, usermanager.APView)
	a.NoError(err)
	a.Equal(usermanager.APView, p.RightsRoster.User[owner.ID])
	a.Equal(usermanager.APNoAccess, p.RightsRoster.User[testuser.ID])
	a.True(p.HasRights(owner, usermanager.APFullAccess))
	a.False(p.HasRights(testuser, usermanager.APView))
	a.False(p.HasRights(testuser, usermanager.APFullAccess))
	a.True(p.IsOwner(owner))
	a.False(p.IsOwner(testuser))

	// cloning
	clone, err := p.Clone()
	a.NoError(err)
	a.NotNil(clone)
	a.Equal(p.ID, clone.ID)
	a.Equal(p.ParentID, clone.ParentID)
	a.Equal(p.OwnerID, clone.OwnerID)
	a.Equal(p.Key, clone.Key)
	a.Equal(p.ObjectKind, clone.ObjectKind)
	a.Equal(p.ObjectID, clone.ObjectID)
	a.Equal(p.IsInherited, clone.IsInherited)
	a.Equal(p.IsExtended, clone.IsExtended)
}

func TestAccessPolicyUnsetRights(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	// truncating test database
	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	userStore, err := usermanager.NewUserStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	userManager, err := usermanager.NewUserManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	groupContainer, err := usermanager.NewGroupContainerForTesting(db)
	a.NoError(err)
	a.NotNil(groupContainer)

	// creating test users via container
	assignor, err := userManager.Create("assignor", "test.assignor@example.com", testReusableUserinfo)
	a.NoError(err)

	user, err := userManager.Create("user", "user@example.com", testReusableUserinfo)
	a.NoError(err)

	role, err := groupContainer.Create(usermanager.GKRole, "test_role_group", "Test Role Group", nil)
	a.NoError(err)

	group, err := groupContainer.Create(usermanager.GKGroup, "test_group", "Test Group", nil)
	a.NoError(err)

	wantedRights := usermanager.APView | usermanager.APChange

	//---------------------------------------------------------------------------
	// user rights
	//---------------------------------------------------------------------------
	// setting
	ap := usermanager.NewAccessPolicy(assignor, nil, false, false)
	a.NoError(ap.SetUserRights(assignor, user, wantedRights))
	a.Equal(wantedRights, ap.RightsRoster.User[user.ID])
	a.True(ap.HasRights(user, wantedRights))

	// unsetting
	a.NoError(ap.UnsetRights(assignor, user))
	a.NotContains(ap.RightsRoster.User, user.ID)
	a.False(ap.HasRights(user, wantedRights))

	//---------------------------------------------------------------------------
	// role rights
	//---------------------------------------------------------------------------
	// setting
	ap = usermanager.NewAccessPolicy(assignor, nil, false, false)
	a.NoError(ap.SetRoleRights(assignor, role, wantedRights))
	a.Equal(wantedRights, ap.RightsRoster.Role[role.ID])

	// unsetting
	a.NoError(ap.UnsetRights(assignor, role))
	a.NotContains(ap.RightsRoster.Role, role.ID)

	//---------------------------------------------------------------------------
	// group rights
	//---------------------------------------------------------------------------
	// setting
	ap = usermanager.NewAccessPolicy(assignor, nil, false, false)
	a.NoError(ap.SetGroupRights(assignor, group, wantedRights))
	a.Equal(wantedRights, ap.RightsRoster.Group[group.ID])

	// unsetting
	a.NoError(ap.UnsetRights(assignor, group))
	a.NotContains(ap.RightsRoster.Group, group.ID)
}

func TestHasGroupRights(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	// truncating test database
	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	userStore, err := usermanager.NewUserStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	userManager, err := usermanager.NewUserManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	// creating test users via container
	assignor, err := userManager.Create("assignor", "test.assignor@example.com", testReusableUserinfo)
	a.NoError(err)

	groupContainer, err := usermanager.NewGroupContainerForTesting(db)
	a.NoError(err)
	a.NotNil(groupContainer)

	// adding the user to 2 groups but setting rights to only one
	group1, err := groupContainer.Create(usermanager.GKGroup, "test_group_1", "test group 1", nil)
	a.NoError(err)

	group2, err := groupContainer.Create(usermanager.GKGroup, "test_group_2", "test group 2", group1)
	a.NoError(err)

	group3, err := groupContainer.Create(usermanager.GKGroup, "test_group_3", "test group 3", group2)
	a.NoError(err)

	//---------------------------------------------------------------------------
	// setting rights only for the first group, thus group3 must inherit its
	// rights only from group 1
	//---------------------------------------------------------------------------
	ap := usermanager.NewAccessPolicy(assignor, nil, false, false)
	a.NotNil(ap)

	wantedRights := usermanager.APCreate | usermanager.APView

	// setting rights for group 1
	a.NoError(ap.SetGroupRights(assignor, group1, wantedRights))
	a.True(ap.HasGroupRights(group1, wantedRights))
	a.True(ap.HasGroupRights(group2, wantedRights))
	a.True(ap.HasGroupRights(group3, wantedRights))

	//---------------------------------------------------------------------------
	// setting rights only for the second group, thus group3 must inherit its
	// rights only from group 2, and group 1 must not have the rights of group 2
	//---------------------------------------------------------------------------
	ap = usermanager.NewAccessPolicy(assignor, nil, false, false)
	a.NotNil(ap)

	wantedRights = usermanager.APCreate | usermanager.APView

	// setting rights for group 2
	a.NoError(ap.SetGroupRights(assignor, group2, wantedRights))
	a.False(ap.HasGroupRights(group1, wantedRights))
	a.True(ap.HasGroupRights(group2, wantedRights))
	a.True(ap.HasGroupRights(group3, wantedRights))

	//---------------------------------------------------------------------------
	// setting rights only for the third group, group 1 and group 2 must not have
	// these rights
	//---------------------------------------------------------------------------
	ap = usermanager.NewAccessPolicy(assignor, nil, false, false)
	a.NotNil(ap)

	wantedRights = usermanager.APCreate | usermanager.APView

	// setting rights for group 2
	a.NoError(ap.SetGroupRights(assignor, group3, wantedRights))
	a.False(ap.HasGroupRights(group1, wantedRights))
	a.False(ap.HasGroupRights(group2, wantedRights))
	a.True(ap.HasGroupRights(group3, wantedRights))

	//---------------------------------------------------------------------------
	// setting rights only for group 1 & 2, group 3 must inherit the rights
	// from its direct ancestor that has its own rights (group 2)
	//---------------------------------------------------------------------------
	ap = usermanager.NewAccessPolicy(assignor, nil, false, false)
	a.NotNil(ap)

	group1Rights := usermanager.APView | usermanager.APCreate
	wantedRights = usermanager.APDelete | usermanager.APCopy

	// setting rights for group 1 & 2
	a.NoError(ap.SetGroupRights(assignor, group1, group1Rights))
	a.NoError(ap.SetGroupRights(assignor, group2, wantedRights))

	a.True(ap.HasGroupRights(group1, group1Rights))
	a.False(ap.HasGroupRights(group1, wantedRights))

	a.True(ap.HasGroupRights(group2, wantedRights))
	a.False(ap.HasGroupRights(group2, group1Rights))

	a.True(ap.HasGroupRights(group3, wantedRights))
	a.False(ap.HasGroupRights(group3, group1Rights))

	//---------------------------------------------------------------------------
	// not setting any rights, only checking
	//---------------------------------------------------------------------------
	ap = usermanager.NewAccessPolicy(assignor, nil, false, false)
	a.NotNil(ap)

	wantedRights = usermanager.APCreate | usermanager.APView

	// setting rights for group 2
	a.False(ap.HasGroupRights(group1, wantedRights))
	a.False(ap.HasGroupRights(group2, wantedRights))
	a.False(ap.HasGroupRights(group3, wantedRights))
}
