package group_test

import (
	"testing"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/group"
)

func TestManager_Create(t *testing.T) {
	//---------------------------------------------------------------------------
	// creating test groups and roles
	//---------------------------------------------------------------------------
	gm, err := group.NewManager()
	a.NoError(err)
	a.NotNil(gm)

	g1, err := gm.Create(group.GKGroup, "group_1", "Group 1", nil)
	a.NoError(err)
	a.NotNil(g1)

	g2, err := gm.Create(group.GKGroup, "group_2", "Group 2", nil)
	a.NoError(err)
	a.NotNil(g1)

	g3, err := gm.Create(group.GKGroup, "group_3", "Group 3 (sub-group of Group 2)", g2)
	a.NoError(err)
	a.NotNil(g1)

	r1, err := gm.Create(group.GKRole, "role_1", "Role 1", nil)
	a.NoError(err)
	a.NotNil(g1)

	r2, err := gm.Create(group.GKRole, "role_2", "Role 2", nil)
	a.NoError(err)
	a.NotNil(g1)

	//---------------------------------------------------------------------------
	// reinitializing the user manager and expecting all of its required,
	// previously created users, groups, roles, tokens, policies etc. to be loaded
	//---------------------------------------------------------------------------
	um, err = core.NewUserManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	//---------------------------------------------------------------------------
	// checking groups
	//---------------------------------------------------------------------------
	gm, err = um.GroupManager()
	a.NoError(err)
	a.NotNil(gm)

	// validating user 1
	u1, err = um.GetByKey("id", u1.ID)
	a.NoError(err)
	a.NotNil(u1)
	a.True(g1.IsMember(u1))
	a.False(g2.IsMember(u1))
	a.False(g3.IsMember(u1))
	a.True(r1.IsMember(u1))
	a.False(r2.IsMember(u1))

	// validating user 2
	u2, err = um.GetByKey("id", u2.ID)
	a.NoError(err)
	a.NotNil(u2)
	a.False(g1.IsMember(u2))
	a.True(g2.IsMember(u2))
	a.False(g3.IsMember(u2))
	a.True(r1.IsMember(u2))
	a.False(r2.IsMember(u2))

	// validating user 3
	u3, err = um.GetByKey("id", u3.ID)
	a.NoError(err)
	a.NotNil(u3)
	a.False(g1.IsMember(u3))
	a.True(g2.IsMember(u3))
	a.True(g3.IsMember(u3))
	a.False(r1.IsMember(u3))
	a.True(r2.IsMember(u3))
}
