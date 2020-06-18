package group_test

import (
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/stretchr/testify/assert"
)

func TestManager_Create(t *testing.T) {
	a := assert.New(t)

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	um, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	gm := um.GroupManager()

	//---------------------------------------------------------------------------
	// creating and storing test users
	//---------------------------------------------------------------------------
	u1, err := user.CreateTestUser(ctx, um, "testuser1", "testuser1@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u1)

	u2, err := user.CreateTestUser(ctx, um, "testuser2", "testuser2@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u2)

	u3, err := user.CreateTestUser(ctx, um, "testuser3", "testuser3@hometown.local", nil)
	a.NoError(err)
	a.NotNil(u1)

	//---------------------------------------------------------------------------
	// creating test groups and roles
	//---------------------------------------------------------------------------

	g1, err := gm.Create(ctx, group.GKGroup, 0, "group_1", "Group 1")
	a.NoError(err)
	a.NotNil(g1)

	g2, err := gm.Create(ctx, group.GKGroup, 0, "group_2", "Group 2")
	a.NoError(err)
	a.NotNil(g1)

	g3, err := gm.Create(ctx, group.GKGroup, g2.ID, "group_3", "Group 3 (sub-group of Group 2)")
	a.NoError(err)
	a.NotNil(g1)

	r1, err := gm.Create(ctx, group.GKRole, 0, "role_1", "Role 1")
	a.NoError(err)
	a.NotNil(g1)

	r2, err := gm.Create(ctx, group.GKRole, 0, "role_2", "Role 2")
	a.NoError(err)
	a.NotNil(g1)

	//---------------------------------------------------------------------------
	// assigning users to groups and roles
	//---------------------------------------------------------------------------
	// user 1 to group 1
	a.NoError(g1.AddMember(ctx, u1.ID))

	// user 2 and 3 to group 2
	a.NoError(g2.AddMember(ctx, u2.ID))
	a.NoError(g2.AddMember(ctx, u3.ID))

	// user 3 to group 3
	a.NoError(g3.AddMember(ctx, u3.ID))

	// user 1 and 2 to role 1
	a.NoError(r1.AddMember(ctx, u1.ID))
	a.NoError(r1.AddMember(ctx, u2.ID))

	// user 3 to role 2
	a.NoError(r2.AddMember(ctx, u3.ID))

	//---------------------------------------------------------------------------
	// reinitializing the user manager and expecting all of its required,
	// previously created users, groups, roles, tokens, policies etc. to be loaded
	//---------------------------------------------------------------------------
	um, _, err = user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	//---------------------------------------------------------------------------
	// checking groups
	//---------------------------------------------------------------------------
	gm = um.GroupManager()
	a.NoError(err)
	a.NotNil(gm)

	// validating user 1
	u1, err = um.UserByID(ctx, u1.ID)
	a.NoError(err)
	a.NotNil(u1)
	a.True(g1.IsMember(ctx, u1.ID))
	a.False(g2.IsMember(ctx, u1.ID))
	a.False(g3.IsMember(ctx, u1.ID))
	a.True(r1.IsMember(ctx, u1.ID))
	a.False(r2.IsMember(ctx, u1.ID))

	// validating user 2
	u2, err = um.UserByID(ctx, u2.ID)
	a.NoError(err)
	a.NotNil(u2)
	a.False(g1.IsMember(ctx, u2.ID))
	a.True(g2.IsMember(ctx, u2.ID))
	a.False(g3.IsMember(ctx, u2.ID))
	a.True(r1.IsMember(ctx, u2.ID))
	a.False(r2.IsMember(ctx, u2.ID))

	// validating user 3
	u3, err = um.UserByID(ctx, u3.ID)
	a.NoError(err)
	a.NotNil(u3)
	a.False(g1.IsMember(ctx, u3.ID))
	a.True(g2.IsMember(ctx, u3.ID))
	a.True(g3.IsMember(ctx, u3.ID))
	a.False(r1.IsMember(ctx, u3.ID))
	a.True(r2.IsMember(ctx, u3.ID))
}
