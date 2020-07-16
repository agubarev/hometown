package group_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/stretchr/testify/assert"
)

func TestManager_Create(t *testing.T) {
	a := assert.New(t)

	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := group.NewMySQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	m, err := group.NewManager(context.Background(), s)
	a.NoError(err)
	a.NotNil(m)

	// blank context
	ctx := context.Background()

	//---------------------------------------------------------------------------
	// creating test groups and roles
	//---------------------------------------------------------------------------

	g1, err := m.Create(ctx, group.FGroup, 0, "group_1", "Group 1")
	a.NoError(err)
	a.NotNil(g1)

	g2, err := m.Create(ctx, group.FGroup, 0, "group_2", "Group 2")
	a.NoError(err)
	a.NotNil(g1)

	g3, err := m.Create(ctx, group.FGroup, g2.ID, "group_3", "Group 3 (sub-group of Group 2)")
	a.NoError(err)
	a.NotNil(g1)

	r1, err := m.Create(ctx, group.FRole, 0, "role_1", "Role 1")
	a.NoError(err)
	a.NotNil(g1)

	r2, err := m.Create(ctx, group.FRole, 0, "role_2", "Role 2")
	a.NoError(err)
	a.NotNil(g1)

	//---------------------------------------------------------------------------
	// assigning users to groups and roles
	//---------------------------------------------------------------------------
	// user 1 to group 1
	a.NoError(m.CreateRelation(ctx, g1.ID, 1))

	// user 2 and 3 to group 2
	a.NoError(m.CreateRelation(ctx, g2.ID, 2))
	a.NoError(m.CreateRelation(ctx, g2.ID, 3))

	// user 3 to group 3
	a.NoError(m.CreateRelation(ctx, g3.ID, 3))

	// user 1 and 2 to role 1
	a.NoError(m.CreateRelation(ctx, r1.ID, 1))
	a.NoError(m.CreateRelation(ctx, r1.ID, 2))

	// user 3 to role 2
	a.NoError(m.CreateRelation(ctx, r2.ID, 3))

	//---------------------------------------------------------------------------
	// checking groups
	//---------------------------------------------------------------------------
	// re-initializing group manager and expecting all assignments to be restored
	m, err = group.NewManager(context.Background(), s)
	a.NoError(err)
	a.NotNil(m)

	// validating user 1
	a.True(m.IsAsset(ctx, g1.ID, 1))
	a.False(m.IsAsset(ctx, g2.ID, 1))
	a.False(m.IsAsset(ctx, g3.ID, 1))
	a.True(m.IsAsset(ctx, r1.ID, 1))
	a.False(m.IsAsset(ctx, r2.ID, 1))

	// validating user 2
	a.False(m.IsAsset(ctx, g1.ID, 2))
	a.True(m.IsAsset(ctx, g2.ID, 2))
	a.False(m.IsAsset(ctx, g3.ID, 2))
	a.True(m.IsAsset(ctx, r1.ID, 2))
	a.False(m.IsAsset(ctx, r2.ID, 2))

	// validating user 3
	a.False(m.IsAsset(ctx, g1.ID, 3))
	a.True(m.IsAsset(ctx, g2.ID, 3))
	a.True(m.IsAsset(ctx, g3.ID, 3))
	a.False(m.IsAsset(ctx, r1.ID, 3))
	a.True(m.IsAsset(ctx, r2.ID, 3))
}
