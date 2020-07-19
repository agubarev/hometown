package group_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestManager_Create(t *testing.T) {
	a := assert.New(t)

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := group.NewPostgreSQLStore(db)
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

	g1, err := m.Create(ctx, group.FGroup, uuid.Nil, group.NewKey("group_1"), group.NewName("Group 1"))
	a.NoError(err)
	a.NotNil(g1)

	g2, err := m.Create(ctx, group.FGroup, uuid.Nil, group.NewKey("group_2"), group.NewName("Group 2"))
	a.NoError(err)
	a.NotNil(g1)

	g3, err := m.Create(ctx, group.FGroup, g2.ID, group.NewKey("group_3"), group.NewName("Group 3 (sub-group of Group 2)"))
	a.NoError(err)
	a.NotNil(g1)

	r1, err := m.Create(ctx, group.FRole, uuid.Nil, group.NewKey("role_1"), group.NewName("Role 1"))
	a.NoError(err)
	a.NotNil(g1)

	r2, err := m.Create(ctx, group.FRole, uuid.Nil, group.NewKey("role_2"), group.NewName("Role 2"))
	a.NoError(err)
	a.NotNil(g1)

	uid1 := uuid.New()
	uid2 := uuid.New()
	uid3 := uuid.New()

	//---------------------------------------------------------------------------
	// assigning users to groups and roles
	//---------------------------------------------------------------------------
	// user 1 to group 1
	a.NoError(m.CreateRelation(ctx, group.NewRelation(g1.ID, group.AKUser, uid1)))

	// user 2 and 3 to group 2
	a.NoError(m.CreateRelation(ctx, group.NewRelation(g2.ID, group.AKUser, uid2)))
	a.NoError(m.CreateRelation(ctx, group.NewRelation(g2.ID, group.AKUser, uid3)))

	// user 3 to group 3
	a.NoError(m.CreateRelation(ctx, group.NewRelation(g3.ID, group.AKUser, uid3)))

	// user 1 and 2 to role 1
	a.NoError(m.CreateRelation(ctx, group.NewRelation(r1.ID, group.AKUser, uid1)))
	a.NoError(m.CreateRelation(ctx, group.NewRelation(r1.ID, group.AKUser, uid2)))

	// user 3 to role 2
	a.NoError(m.CreateRelation(ctx, group.NewRelation(r2.ID, group.AKUser, uid3)))

	//---------------------------------------------------------------------------
	// checking groups
	//---------------------------------------------------------------------------
	// re-initializing group manager and expecting all assignments to be restored
	m, err = group.NewManager(context.Background(), s)
	a.NoError(err)
	a.NotNil(m)

	// validating user 1
	a.True(m.IsAsset(ctx, g1.ID, group.NewAsset(group.AKUser, uid1)))
	a.False(m.IsAsset(ctx, g2.ID, group.NewAsset(group.AKUser, uid1)))
	a.False(m.IsAsset(ctx, g3.ID, group.NewAsset(group.AKUser, uid1)))
	a.True(m.IsAsset(ctx, r1.ID, group.NewAsset(group.AKUser, uid1)))
	a.False(m.IsAsset(ctx, r2.ID, group.NewAsset(group.AKUser, uid1)))

	// validating user 2
	a.False(m.IsAsset(ctx, g1.ID, group.NewAsset(group.AKUser, uid2)))
	a.True(m.IsAsset(ctx, g2.ID, group.NewAsset(group.AKUser, uid2)))
	a.False(m.IsAsset(ctx, g3.ID, group.NewAsset(group.AKUser, uid2)))
	a.True(m.IsAsset(ctx, r1.ID, group.NewAsset(group.AKUser, uid2)))
	a.False(m.IsAsset(ctx, r2.ID, group.NewAsset(group.AKUser, uid2)))

	// validating user 3
	a.False(m.IsAsset(ctx, g1.ID, group.NewAsset(group.AKUser, uid3)))
	a.True(m.IsAsset(ctx, g2.ID, group.NewAsset(group.AKUser, uid3)))
	a.True(m.IsAsset(ctx, g3.ID, group.NewAsset(group.AKUser, uid3)))
	a.False(m.IsAsset(ctx, r1.ID, group.NewAsset(group.AKUser, uid3)))
	a.True(m.IsAsset(ctx, r2.ID, group.NewAsset(group.AKUser, uid3)))
}
