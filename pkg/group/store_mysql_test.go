package group_test

/*
func TestGroupStorePut(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	g := group.Group{
		ActorID:          uuid.Nil,
		ParentID:    uuid.Nil,
		Flags:       group.FGroup,
		Key:         group.NewKey("test_key"),
		DisplayName: group.NewName("test name"),
	}

	g, err = s.UpsertGroup(ctx, g)
	a.NoError(err)
	a.NotNil(g)
}

func TestGroupStoreGet(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	g := group.Group{
		ActorID:          uuid.Nil,
		ParentID:    uuid.Nil,
		Flags:       group.FGroup,
		Key:         group.NewKey("test_key"),
		DisplayName: group.NewName("test name"),
	}

	g, err = s.UpsertGroup(ctx, g)
	a.NoError(err)
	a.NotNil(g)

	fg, err := s.FetchGroupByID(ctx, g.ActorID)
	a.NotNil(fg)
	a.NoError(err)
	a.Equal(g.ActorID, fg.ActorID)
	a.Equal(g.Flags, fg.Flags)
	a.Equal(g.Key, fg.Key)
	a.Equal(g.DisplayName, fg.DisplayName)
}

func TestGroupStoreGetAll(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	g1 := group.Group{
		ActorID:          uuid.Nil,
		ParentID:    uuid.Nil,
		Flags:       group.FGroup,
		Key:         group.NewKey("test_key"),
		DisplayName: group.NewName("test name"),
	}

	g2 := group.Group{
		ActorID:          uuid.Nil,
		ParentID:    uuid.Nil,
		Flags:       group.FRole,
		Key:         group.NewKey("test_role"),
		DisplayName: group.NewName("test role"),
	}

	g3 := group.Group{
		ActorID:          uuid.Nil,
		ParentID:    uuid.Nil,
		Flags:       group.FGroup,
		Key:         group.NewKey("test_group123"),
		DisplayName: group.NewName("test group 123"),
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

	a.Equal(g1.ActorID, gs[0].ActorID)
	a.Equal(g1.Flags, gs[0].Flags)
	a.Equal(g1.Key, gs[0].Key)
	a.Equal(g1.DisplayName, gs[0].DisplayName)

	a.Equal(g2.ActorID, gs[1].ActorID)
	a.Equal(g2.Flags, gs[1].Flags)
	a.Equal(g2.Key, gs[1].Key)
	a.Equal(g2.DisplayName, gs[1].DisplayName)

	a.Equal(g3.ActorID, gs[2].ActorID)
	a.Equal(g3.Flags, gs[2].Flags)
	a.Equal(g3.Key, gs[2].Key)
	a.Equal(g3.DisplayName, gs[2].DisplayName)
}

func TestGroupStoreDelete(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	g := group.Group{
		ActorID:          0,
		ParentID:    0,
		Flags:       group.FGroup,
		Key:         group.NewKey("test_group"),
		DisplayName: group.NewName("test group"),
	}

	g, err = s.UpsertGroup(ctx, g)
	a.NoError(err)

	fg, err := s.FetchGroupByID(ctx, g.ActorID)
	a.NotNil(fg)
	a.NoError(err)

	err = s.DeleteByID(ctx, g.ActorID)
	a.NoError(err)

	fg, err = s.FetchGroupByID(ctx, g.ActorID)
	a.Error(err)
	a.EqualError(group.ErrGroupNotFound, err.Error())
}

func TestGroupStoreRelations(t *testing.T) {
	a := assert.New(t)

	ctx := context.Background()

	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := group.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	g := group.Group{
		ActorID:          0,
		ParentID:    0,
		Flags:       group.FGroup,
		Key:         group.NewKey("test_group"),
		DisplayName: group.NewName("test group"),
	}

	// making sure there is no previous relation
	ok, err := s.HasRelation(ctx, g.ActorID, 1)
	a.NoError(err)
	a.False(ok)

	// creating a relation
	a.NoError(s.CreateRelation(ctx, g.ActorID, 1))

	// now they must be related
	ok, err = s.HasRelation(ctx, g.ActorID, 1)
	a.NoError(err)
	a.True(ok)

	// breaking relation
	a.NoError(s.DeleteRelation(ctx, g.ActorID, 1))

	// making sure the relation is gone
	ok, err = s.HasRelation(ctx, g.ActorID, 1)
	a.NoError(err)
	a.False(ok)
}
*/
