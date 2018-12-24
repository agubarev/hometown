package user

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewAccessPolicy(t *testing.T) {
	a := assert.New(t)

	p := NewAccessPolicy(nil, nil, false, false)
	a.NotNil(p)
	a.Nil(p.Owner)
	a.Nil(p.Parent)
	a.False(p.IsInherited)
	a.False(p.IsExtended)

	// with owner
	user, err := NewUser("testuser")
	a.NoError(err)

	p = NewAccessPolicy(user, nil, false, false)
	a.NotNil(p)
	a.Equal(user, p.Owner)
	a.Nil(p.Parent)
	a.False(p.IsInherited)
	a.False(p.IsExtended)

	// with parent
	parent := NewAccessPolicy(nil, nil, false, false)
	a.NoError(err)

	p = NewAccessPolicy(nil, parent, false, false)
	a.NotNil(p)
	a.Nil(p.Owner)
	a.Equal(parent, p.Parent)
	a.False(p.IsInherited)
	a.False(p.IsExtended)

	// with inheritance
	p = NewAccessPolicy(nil, nil, true, false)
	a.NotNil(p)
	a.Nil(p.Owner)
	a.Nil(p.Parent)
	a.True(p.IsInherited)
	a.False(p.IsExtended)

	// with extension
	p = NewAccessPolicy(nil, nil, false, true)
	a.NotNil(p)
	a.Nil(p.Owner)
	a.Nil(p.Parent)
	a.False(p.IsInherited)
	a.True(p.IsExtended)
}

func TestSetPublicRights(t *testing.T) {
	a := assert.New(t)

	assignor, err := NewUser("assignor")
	a.NoError(err)

	testuser, err := NewUser("testuser")
	a.NoError(err)

	wantedRights := APView | APChange

	// no parent, not inheriting and not extending
	p := NewAccessPolicy(assignor, nil, false, false)
	err = p.SetPublicRights(assignor, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.Everyone)
	a.True(p.HasRights(testuser, wantedRights))

	// with parent, using legacy only
	pWithInheritance := NewAccessPolicy(assignor, p, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)
	a.Equal(wantedRights, pWithInheritance.Parent.RightsRoster.Everyone)
	a.True(pWithInheritance.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn := NewAccessPolicy(assignor, p, false, true)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedNoOwn.Parent.RightsRoster.Everyone)
	a.Equal(APNoAccess, pExtendedNoOwn.RightsRoster.Everyone)
	a.True(pExtendedNoOwn.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added APMove to own rights
	pExtendedWithOwn := NewAccessPolicy(assignor, p, false, true)
	pExtendedWithOwn.SetPublicRights(assignor, APMove)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedWithOwn.Parent.RightsRoster.Everyone)
	a.Equal(APMove, pExtendedWithOwn.RightsRoster.Everyone)
	a.True(pExtendedWithOwn.HasRights(testuser, wantedRights|APMove))
}

func TestSetGroupRights(t *testing.T) {
	a := assert.New(t)

	assignor, err := NewUser("assignor")
	a.NoError(err)

	testuser, err := NewUser("testuser")
	a.NoError(err)

	// adding the user to 2 groups but setting rights to only one
	group1 := NewGroup("test group 1")
	err = group1.AddUser(testuser)
	a.NoError(err)

	group2 := NewGroup("test group 2")
	err = group2.AddUser(testuser)
	a.NoError(err)

	wantedRights := APView | APChange

	// no parent, not inheriting and not extending
	// WARNING: [p] will be reused and inherited below in this function
	p := NewAccessPolicy(assignor, nil, false, false)
	err = p.SetGroupRights(assignor, group1, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.Group[group1.Name])
	a.True(p.HasRights(testuser, wantedRights))

	// with parent, using legacy only
	pWithInheritance := NewAccessPolicy(assignor, p, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)
	a.Equal(wantedRights, pWithInheritance.Parent.RightsRoster.Group[group1.Name])
	a.True(pWithInheritance.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn := NewAccessPolicy(assignor, p, false, true)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedNoOwn.Parent.RightsRoster.Group[group1.Name])
	a.Equal(APNoAccess, pExtendedNoOwn.RightsRoster.Group[group1.Name])
	a.True(pExtendedNoOwn.HasRights(testuser, wantedRights))

	// with parent, inheritance false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added APMove to own rights
	pExtendedWithOwn := NewAccessPolicy(assignor, p, false, true)
	pExtendedWithOwn.SetGroupRights(assignor, group1, APMove)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedWithOwn.Parent.RightsRoster.Group[group1.Name])
	a.Equal(APMove, pExtendedWithOwn.RightsRoster.Group[group1.Name])
	a.True(pExtendedWithOwn.HasRights(testuser, wantedRights|APMove))
}

func TestSetRoleRights(t *testing.T) {
	a := assert.New(t)

	assignor, err := NewUser("assignor")
	a.NoError(err)

	testuser, err := NewUser("testuser")
	a.NoError(err)

	// adding the user to 2 groups but setting rights to only one
	role1 := NewRole("test role 1")
	err = role1.AddUser(testuser)
	a.NoError(err)

	role2 := NewRole("test role 2")
	err = role2.AddUser(testuser)
	a.NoError(err)

	wantedRights := APView | APChange

	// no parent, not inheriting and not extending
	// WARNING: [p] will be reused and inherited below in this function
	p := NewAccessPolicy(assignor, nil, false, false)
	err = p.SetRoleRights(assignor, role1, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.Role[role1.Name])
	a.True(p.HasRights(testuser, wantedRights))

	// with parent, using legacy only
	pWithInheritance := NewAccessPolicy(assignor, p, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)
	a.Equal(wantedRights, pWithInheritance.Parent.RightsRoster.Role[role1.Name])
	a.True(pWithInheritance.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn := NewAccessPolicy(assignor, p, false, true)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedNoOwn.Parent.RightsRoster.Role[role1.Name])
	a.Equal(APNoAccess, pExtendedNoOwn.RightsRoster.Role[role1.Name])
	a.True(pExtendedNoOwn.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added APMove to own rights
	pExtendedWithOwn := NewAccessPolicy(assignor, p, false, true)
	pExtendedWithOwn.SetRoleRights(assignor, role1, APMove)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedWithOwn.Parent.RightsRoster.Role[role1.Name])
	a.Equal(APMove, pExtendedWithOwn.RightsRoster.Role[role1.Name])
	a.True(pExtendedWithOwn.HasRights(testuser, wantedRights|APMove))
}

func TestSetUserRights(t *testing.T) {
	a := assert.New(t)

	assignor, err := NewUser("assignor")
	a.NoError(err)

	testuser, err := NewUser("testuser")
	a.NoError(err)

	wantedRights := APView | APChange

	// no parent, not inheriting and not extending
	// WARNING: [p] will be reused and inherited below in this function
	p := NewAccessPolicy(assignor, nil, false, false)
	err = p.SetUserRights(assignor, testuser, wantedRights)
	a.NoError(err)
	a.Equal(wantedRights, p.RightsRoster.User[testUsername])
	a.True(p.HasRights(testuser, wantedRights))

	// with parent, using legacy only
	pWithInheritance := NewAccessPolicy(assignor, p, true, false)
	// not setting it's own rights as it must inherit them from a parent
	a.NoError(err)
	a.Equal(wantedRights, pWithInheritance.Parent.RightsRoster.User[testUsername])
	a.True(pWithInheritance.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights
	// no own rights
	pExtendedNoOwn := NewAccessPolicy(assignor, p, false, true)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedNoOwn.Parent.RightsRoster.User[testUsername])
	a.Equal(APNoAccess, pExtendedNoOwn.RightsRoster.User[testUsername])
	a.True(pExtendedNoOwn.HasRights(testuser, wantedRights))

	// with parent, legacy false, extend true; using parent's rights with it's own
	// adding one more right to itself
	// NOTE: added APMove to own rights
	pExtendedWithOwn := NewAccessPolicy(assignor, p, false, true)
	pExtendedWithOwn.SetUserRights(assignor, testuser, APMove)
	a.NoError(err)
	a.Equal(wantedRights, pExtendedWithOwn.Parent.RightsRoster.User[testUsername])
	a.Equal(APMove, pExtendedWithOwn.RightsRoster.User[testUsername])
	a.True(pExtendedWithOwn.HasRights(testuser, wantedRights|APMove))
}

func TestIsOwner(t *testing.T) {

}
