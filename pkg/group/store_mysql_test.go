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
		ID:          uuid.Nil,
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
		ID:          uuid.Nil,
		ParentID:    uuid.Nil,
		Flags:       group.FGroup,
		Key:         group.NewKey("test_key"),
		DisplayName: group.NewName("test name"),
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
		ID:          uuid.Nil,
		ParentID:    uuid.Nil,
		Flags:       group.FGroup,
		Key:         group.NewKey("test_key"),
		DisplayName: group.NewName("test name"),
	}

	g2 := group.Group{
		ID:          uuid.Nil,
		ParentID:    uuid.Nil,
		Flags:       group.FRole,
		Key:         group.NewKey("test_role"),
		DisplayName: group.NewName("test role"),
	}

	g3 := group.Group{
		ID:          uuid.Nil,
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
		ID:          0,
		ParentID:    0,
		Flags:       group.FGroup,
		Key:         group.NewKey("test_group"),
		DisplayName: group.NewName("test group"),
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
		ID:          0,
		ParentID:    0,
		Flags:       group.FGroup,
		Key:         group.NewKey("test_group"),
		DisplayName: group.NewName("test group"),
	}

	// making sure there is no previous relation
	ok, err := s.HasRelation(ctx, g.ID, 1)
	a.NoError(err)
	a.False(ok)

	// creating a relation
	a.NoError(s.CreateRelation(ctx, g.ID, 1))

	// now they must be related
	ok, err = s.HasRelation(ctx, g.ID, 1)
	a.NoError(err)
	a.True(ok)

	// breaking relation
	a.NoError(s.DeleteRelation(ctx, g.ID, 1))

	// making sure the relation is gone
	ok, err = s.HasRelation(ctx, g.ID, 1)
	a.NoError(err)
	a.False(ok)
}
*/
