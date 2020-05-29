package user_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/user"

	"github.com/stretchr/testify/assert"
)

func TestGroupStorePut(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := user.NewGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g := user.Group{
		ID:          0,
		Kind:        user.GKGroup,
		Key:         "test_key",
		Name:        "test_name",
		Description: "test description",
		ParentID:    0,
	}

	g, err = s.UpsertGroup(ctx, g)
	a.NoError(err)
	a.NotNil(g)
}

func TestGroupStoreGet(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := user.NewGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g := user.Group{
		ID:          0,
		Kind:        user.GKGroup,
		Key:         "test_key",
		Name:        "test_name",
		Description: "test description",
		ParentID:    0,
	}

	g, err = s.UpsertGroup(ctx, g)
	a.NoError(err)
	a.NotNil(g)

	fg, err := s.FetchGroupByID(ctx, g.ID)
	a.NotNil(fg)
	a.NoError(err)
	a.Equal(g.ID, fg.ID)
	a.Equal(g.Kind, fg.Kind)
	a.Equal(g.Key, fg.Key)
	a.Equal(g.Name, fg.Name)
}

func TestGroupStoreGetAll(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := user.NewGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g1 := user.Group{
		ID:          0,
		Kind:        user.GKGroup,
		Key:         "test_group",
		Name:        "test group",
		Description: "test description",
		ParentID:    0,
	}

	g2 := user.Group{
		ID:          0,
		Kind:        user.GKRole,
		Key:         "test_role",
		Name:        "test role",
		Description: "test description",
		ParentID:    0,
	}

	g3 := user.Group{
		ID:          0,
		Kind:        user.GKGroup,
		Key:         "test_group123",
		Name:        "test group 123",
		Description: "test description",
		ParentID:    0,
	}

	g1, err = s.UpsertGroup(ctx, g1)
	a.NoError(err)

	g2, err = s.UpsertGroup(ctx, g2)
	a.NoError(err)

	g3, err = s.UpsertGroup(ctx, g3)
	a.NoError(err)

	gs, err := s.FetchAllGroups(ctx)
	a.NoError(err)
	a.Len(gs, 3)

	a.Equal(g1.ID, gs[0].ID)
	a.Equal(g1.Kind, gs[0].Kind)
	a.Equal(g1.Key, gs[0].Key)
	a.Equal(g1.Name, gs[0].Name)

	a.Equal(g2.ID, gs[1].ID)
	a.Equal(g2.Kind, gs[1].Kind)
	a.Equal(g2.Key, gs[1].Key)
	a.Equal(g2.Name, gs[1].Name)

	a.Equal(g3.ID, gs[2].ID)
	a.Equal(g3.Kind, gs[2].Kind)
	a.Equal(g3.Key, gs[2].Key)
	a.Equal(g3.Name, gs[2].Name)
}

func TestGroupStoreDelete(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := user.NewGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g := user.Group{
		ID:          0,
		Kind:        user.GKGroup,
		Key:         "test_group",
		Name:        "test group",
		Description: "test description",
		ParentID:    0,
	}

	g, err = s.UpsertGroup(ctx, g)
	a.NoError(err)

	fg, err := s.FetchGroupByID(ctx, g.ID)
	a.NotNil(fg)
	a.NoError(err)

	err = s.DeleteByID(ctx, g.ID)
	a.NoError(err)

	fg, err = s.FetchGroupByID(ctx, g.ID)
	a.Error(err)
	a.EqualError(user.ErrGroupNotFound, err.Error())
}

func TestGroupStoreRelations(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	um, _, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	s, err := user.NewGroupStore(db)
	a.NoError(err)
	a.NotNil(s)

	g := user.Group{
		ID:          0,
		Kind:        user.GKGroup,
		Key:         "test_group",
		Name:        "test group",
		Description: "test description",
		ParentID:    0,
	}

	u, err := user.CreateTestUser(ctx, um, "testuser", "testuser@hometown.local")
	a.NoError(err)
	a.NotNil(u)

	// making sure there is no previous relation
	ok, err := s.HasRelation(ctx, g.ID, u.ID)
	a.NoError(err)
	a.False(ok)

	// creating a relation
	a.NoError(s.CreateRelation(ctx, g.ID, u.ID))

	// now they must be related
	ok, err = s.HasRelation(ctx, g.ID, u.ID)
	a.NoError(err)
	a.True(ok)

	// breaking relation
	a.NoError(s.DeleteRelation(ctx, g.ID, u.ID))

	// making sure the relation is gone
	ok, err = s.HasRelation(ctx, g.ID, u.ID)
	a.NoError(err)
	a.False(ok)
}
