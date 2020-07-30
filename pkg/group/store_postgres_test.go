package group_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestPostgreSQLStore_UpsertGroup(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	g := group.Group{
		ID:          uuid.New(),
		ParentID:    uuid.Nil,
		Flags:       group.FGroup,
		Key:         bytearray.NewByteString32("test_key"),
		DisplayName: bytearray.NewByteString128("test name"),
	}

	g, err = s.UpsertGroup(ctx, g)
	a.NoError(err)
	a.NotNil(g)
}

func TestPostgreSQLStore_FetchGroupByID(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	g := group.Group{
		ID:          uuid.New(),
		ParentID:    uuid.Nil,
		Flags:       group.FGroup,
		Key:         bytearray.NewByteString32("test_key"),
		DisplayName: bytearray.NewByteString128("test name"),
	}

	g, err = s.UpsertGroup(ctx, g)
	a.NoError(err)
	a.NotNil(g)

	fg, err := s.FetchGroupByID(ctx, g.ID)
	a.NotNil(fg)
	a.NoError(err)
	a.Equal(g.ID, fg.ID)
	a.Equal(g.Flags, fg.Flags)
	a.Equal(g.Key, fg.Key)
	a.Equal(g.DisplayName, fg.DisplayName)
}

func TestPostgreSQLStore_FetchAllGroups(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	g1 := group.Group{
		ID:          uuid.New(),
		ParentID:    uuid.Nil,
		Flags:       group.FGroup,
		Key:         bytearray.NewByteString32("test_key"),
		DisplayName: bytearray.NewByteString128("test name"),
	}

	g2 := group.Group{
		ID:          uuid.New(),
		ParentID:    uuid.Nil,
		Flags:       group.FRole,
		Key:         bytearray.NewByteString32("test_role"),
		DisplayName: bytearray.NewByteString128("test role"),
	}

	g3 := group.Group{
		ID:          uuid.New(),
		ParentID:    uuid.Nil,
		Flags:       group.FGroup,
		Key:         bytearray.NewByteString32("test_group123"),
		DisplayName: bytearray.NewByteString128("test group 123"),
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
	a.Equal(g1.Flags, gs[0].Flags)
	a.Equal(g1.Key, gs[0].Key)
	a.Equal(g1.DisplayName, gs[0].DisplayName)

	a.Equal(g2.ID, gs[1].ID)
	a.Equal(g2.Flags, gs[1].Flags)
	a.Equal(g2.Key, gs[1].Key)
	a.Equal(g2.DisplayName, gs[1].DisplayName)

	a.Equal(g3.ID, gs[2].ID)
	a.Equal(g3.Flags, gs[2].Flags)
	a.Equal(g3.Key, gs[2].Key)
	a.Equal(g3.DisplayName, gs[2].DisplayName)
}

func TestPostgreSQLStore_DeleteByID(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	g := group.Group{
		ID:          uuid.New(),
		ParentID:    uuid.Nil,
		Flags:       group.FGroup,
		Key:         bytearray.NewByteString32("test_group"),
		DisplayName: bytearray.NewByteString128("test group"),
	}

	// creating test group
	g, err = s.UpsertGroup(ctx, g)
	a.NoError(err)

	fg, err := s.FetchGroupByID(ctx, g.ID)
	a.NotNil(fg)
	a.NoError(err)

	err = s.DeleteByID(ctx, g.ID)
	a.NoError(err)

	fg, err = s.FetchGroupByID(ctx, g.ID)
	a.Error(err)
	a.EqualError(group.ErrGroupNotFound, err.Error())
}

func TestPostgreSQLStore_DeleteRelation(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	g := group.Group{
		ID:          uuid.New(),
		ParentID:    uuid.Nil,
		Flags:       group.FGroup,
		Key:         bytearray.NewByteString32("test_group"),
		DisplayName: bytearray.NewByteString128("test group"),
	}

	// creating test group
	g, err = s.UpsertGroup(ctx, g)
	a.NoError(err)

	uid1 := uuid.New()

	// making sure there is no previous relation
	ok, err := s.HasRelation(ctx, group.NewRelation(g.ID, group.AKUser, uid1))
	a.NoError(err)
	a.False(ok)

	// creating a relation
	a.NoError(s.CreateRelation(ctx, group.NewRelation(g.ID, group.AKUser, uid1)))

	// now they must be related
	ok, err = s.HasRelation(ctx, group.NewRelation(g.ID, group.AKUser, uid1))
	a.NoError(err)
	a.True(ok)

	// breaking relation
	a.NoError(s.DeleteRelation(ctx, group.NewRelation(g.ID, group.AKUser, uid1)))

	// making sure the relation is gone
	ok, err = s.HasRelation(ctx, group.NewRelation(g.ID, group.AKUser, uid1))
	a.NoError(err)
	a.False(ok)
}
