package accesspolicy_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	ap "gitlab.com/agubarev/hometown/accesspolicy"
)

func TestNewAccessPolicy(t *testing.T) {
	a := assert.New(t)

	p := ap.NewAccessPolicy(nil, nil, false, false)
	a.NotNil(p)
	a.Nil(p.Owner)
	a.Nil(p.Parent)
	a.False(p.IsInherited)
	a.False(p.IsExtended)

	// with owner
	user, err := user.NewUser("testuser")
	a.NoError(err)

	p = ap.NewAccessPolicy(user, nil, false, false)
	a.NotNil(p)
	a.Equal(user, p.Owner)
	a.Nil(p.Parent)
	a.False(p.IsInherited)
	a.False(p.IsExtended)

	// with parent
	parent := ap.NewAccessPolicy(nil, nil, false, false)
	a.NoError(err)

	p = ap.NewAccessPolicy(nil, parent, false, false)
	a.NotNil(p)
	a.Nil(p.Owner)
	a.Equal(parent, p.Parent)
	a.False(p.IsInherited)
	a.False(p.IsExtended)

	// with inheritance
	p = ap.NewAccessPolicy(nil, nil, true, false)
	a.NotNil(p)
	a.Nil(p.Owner)
	a.Nil(p.Parent)
	a.True(p.IsInherited)
	a.False(p.IsExtended)

	// with extension
	p = ap.NewAccessPolicy(nil, nil, false, true)
	a.NotNil(p)
	a.Nil(p.Owner)
	a.Nil(p.Parent)
	a.False(p.IsInherited)
	a.True(p.IsExtended)
}

func TestSetPublicRights(t *testing.T) {
	a := assert.New(t)

	assignor, err := user.NewUser("assignor")
	a.NoError(err)

	testuser, err := user.NewUser("testuser")
	a.NoError(err)

	wantedRights := ap.APView | ap.APChange

	// no parent, not inheriting and not extending
	p := ap.NewAccessPolicy(assignor, nil, false, false)
	err = p.SetPublicRights(assignor, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.Everyone)
	a.True(p.HasRights(testuser, wantedRights))

	// with parent, using legacy only
	pWithInheritance := ap.NewAccessPolicy(assignor, p, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)
	a.Equal(wantedRights, pWithInheritance.Parent.RightsRoster.Everyone)
	a.True(pWithInheritance.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn := ap.NewAccessPolicy(assignor, p, false, true)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedNoOwn.Parent.RightsRoster.Everyone)
	a.Equal(ap.APNoAccess, pExtendedNoOwn.RightsRoster.Everyone)
	a.True(pExtendedNoOwn.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added APMove to own rights
	pExtendedWithOwn := ap.NewAccessPolicy(assignor, p, false, true)
	pExtendedWithOwn.SetPublicRights(assignor, ap.APMove)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedWithOwn.Parent.RightsRoster.Everyone)
	a.Equal(ap.APMove, pExtendedWithOwn.RightsRoster.Everyone)
	a.True(pExtendedWithOwn.HasRights(testuser, wantedRights|ap.APMove))
}

func TestSetGroupRights(t *testing.T) {
	a := assert.New(t)

	assignor, err := user.NewUser("assignor")
	a.NoError(err)

	testuser, err := user.NewUser("testuser")
	a.NoError(err)

	// adding the user to 2 groups but setting rights to only one
	group1 := user.NewGroup("test group 1")
	err = group1.AddUser(testuser)
	a.NoError(err)

	group2 := user.NewGroup("test group 2")
	err = group2.AddUser(testuser)
	a.NoError(err)

	wantedRights := ap.APView | ap.APChange

	// no parent, not inheriting and not extending
	// WARNING: [p] will be reused and inherited below in this function
	p := ap.NewAccessPolicy(assignor, nil, false, false)
	err = p.SetGroupRights(assignor, group1, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.Group[group1.Name])
	a.True(p.HasRights(testuser, wantedRights))

	// with parent, using legacy only
	pWithInheritance := ap.NewAccessPolicy(assignor, p, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)
	a.Equal(wantedRights, pWithInheritance.Parent.RightsRoster.Group[group1.Name])
	a.True(pWithInheritance.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn := ap.NewAccessPolicy(assignor, p, false, true)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedNoOwn.Parent.RightsRoster.Group[group1.Name])
	a.Equal(ap.APNoAccess, pExtendedNoOwn.RightsRoster.Group[group1.Name])
	a.True(pExtendedNoOwn.HasRights(testuser, wantedRights))

	// with parent, inheritance false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added APMove to own rights
	pExtendedWithOwn := ap.NewAccessPolicy(assignor, p, false, true)
	pExtendedWithOwn.SetGroupRights(assignor, group1, ap.APMove)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedWithOwn.Parent.RightsRoster.Group[group1.Name])
	a.Equal(ap.APMove, pExtendedWithOwn.RightsRoster.Group[group1.Name])
	a.True(pExtendedWithOwn.HasRights(testuser, wantedRights|ap.APMove))
}

func TestSetRoleRights(t *testing.T) {
	a := assert.New(t)

	assignor, err := user.NewUser("assignor")
	a.NoError(err)

	testuser, err := user.NewUser("testuser")
	a.NoError(err)

	// adding the user to 2 groups but setting rights to only one
	role1 := user.NewRole("test role 1")
	err = role1.AddUser(testuser)
	a.NoError(err)

	role2 := user.NewRole("test role 2")
	err = role2.AddUser(testuser)
	a.NoError(err)

	wantedRights := ap.APView | ap.APChange

	// no parent, not inheriting and not extending
	// WARNING: [p] will be reused and inherited below in this function
	p := ap.NewAccessPolicy(assignor, nil, false, false)
	err = p.SetRoleRights(assignor, role1, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.Role[role1.Name])
	a.True(p.HasRights(testuser, wantedRights))

	// with parent, using legacy only
	pWithInheritance := ap.NewAccessPolicy(assignor, p, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)
	a.Equal(wantedRights, pWithInheritance.Parent.RightsRoster.Role[role1.Name])
	a.True(pWithInheritance.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn := ap.NewAccessPolicy(assignor, p, false, true)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedNoOwn.Parent.RightsRoster.Role[role1.Name])
	a.Equal(ap.APNoAccess, pExtendedNoOwn.RightsRoster.Role[role1.Name])
	a.True(pExtendedNoOwn.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added APMove to own rights
	pExtendedWithOwn := ap.NewAccessPolicy(assignor, p, false, true)
	pExtendedWithOwn.SetRoleRights(assignor, role1, ap.APMove)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedWithOwn.Parent.RightsRoster.Role[role1.Name])
	a.Equal(ap.APMove, pExtendedWithOwn.RightsRoster.Role[role1.Name])
	a.True(pExtendedWithOwn.HasRights(testuser, wantedRights|ap.APMove))
}

func TestSetUserRights(t *testing.T) {
	a := assert.New(t)

	assignor, err := user.NewUser("assignor")
	a.NoError(err)

	testuser, err := user.NewUser("testuser")
	a.NoError(err)

	wantedRights := ap.APView | ap.APChange

	// no parent, not inheriting and not extending
	// WARNING: [p] will be reused and inherited below in this function
	p := ap.NewAccessPolicy(assignor, nil, false, false)
	err = p.SetUserRights(assignor, testuser, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.User[testuser.Username])
	a.True(p.HasRights(testuser, wantedRights))

	// with parent, using legacy only
	pWithInheritance := ap.NewAccessPolicy(assignor, p, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)
	a.Equal(wantedRights, pWithInheritance.Parent.RightsRoster.User[testuser.Username])
	a.True(pWithInheritance.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn := ap.NewAccessPolicy(assignor, p, false, true)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedNoOwn.Parent.RightsRoster.User[testuser.Username])
	a.Equal(ap.APNoAccess, pExtendedNoOwn.RightsRoster.User[testuser.Username])
	a.True(pExtendedNoOwn.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added APMove to own rights
	pExtendedWithOwn := ap.NewAccessPolicy(assignor, p, false, true)
	pExtendedWithOwn.SetUserRights(assignor, testuser, ap.APMove)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedWithOwn.Parent.RightsRoster.User[testuser.Username])
	a.Equal(ap.APMove, pExtendedWithOwn.RightsRoster.User[testuser.Username])
	a.True(pExtendedWithOwn.HasRights(testuser, wantedRights|ap.APMove))
}

func TestIsOwner(t *testing.T) {

}
