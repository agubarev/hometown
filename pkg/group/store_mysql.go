package group

/*
// MySQLStore is the default group store implementation
type MySQLStore struct {
	db *dbr.Connection
}

// NewMySQLStore returns a group store with mysql used as a backend
func NewMySQLStore(db *dbr.Connection) (Store, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}

	return &MySQLStore{db}, nil
}

//? BEGIN ->>>----------------------------------------------------------------
//? unexported utility functions

func (s *MySQLStore) get(ctx context.Context, q string, args ...interface{}) (g Group, err error) {
	err = s.db.NewSession(nil).
		SelectBySql(q, args...).
		LoadOneContext(ctx, &g)

	if err != nil {
		if err == dbr.ErrNotFound {
			return g, ErrGroupNotFound
		}

		return g, err
	}

	return g, nil
}

func (s *MySQLStore) getMany(ctx context.Context, q string, args ...interface{}) (gs []Group, err error) {
	if _, err := s.db.NewSession(nil).SelectBySql(q, args...).LoadContext(ctx, &gs); err != nil {
		return nil, err
	}

	return gs, nil
}

//? unexported utility functions
//? END ---<<<----------------------------------------------------------------

// UpdatePolicy storing group
func (s *MySQLStore) UpsertGroup(ctx context.Context, g Group) (Group, error) {
	// if an object has object id other than nil, then it's considered
	// as being already created, thus requiring an update
	if g.ID != uuid.Nil {
		return s.Update(ctx, g)
	}

	return s.Create(ctx, g)
}

// Upsert creates a new database record
func (s *MySQLStore) Create(ctx context.Context, g Group) (Group, error) {
	// if object id is not nil, then it's not considered as new
	if g.ID != uuid.Nil {
		return g, ErrNonZeroID
	}

	res, err := s.db.NewSession(nil).
		InsertInto("group").
		Columns(guard.DBColumnsFrom(&g)...).
		Record(&g).
		ExecContext(ctx)

	// error handling
	if err != nil {
		return g, err
	}

	return g, nil
}

// UpdatePolicy updates an existing group
func (s *MySQLStore) Update(ctx context.Context, g Group) (Group, error) {
	if g.ID == uuid.Nil {
		return g, ErrZeroID
	}

	updates := map[string]interface{}{
		"key":  g.Key,
		"name": g.DisplayName,
	}

	// just executing query but not refetching the updated version
	res, err := s.db.NewSession(nil).Update("group").SetMap(updates).Where("id = ?", g.ID).ExecContext(ctx)
	if err != nil {
		return g, err
	}

	// checking whether anything was updated at all
	ra, err := res.RowsAffected()
	if err != nil {
		return g, err
	}

	// if no rows were affected then returning this as a non-critical error
	if ra == 0 {
		return g, ErrNothingChanged
	}

	return g, nil
}

func (s *MySQLStore) FetchGroupByID(ctx context.Context, id uuid.UUID) (Group, error) {
	return s.get(ctx, "SELECT * FROM `group` WHERE id = ? LIMIT 1", id)
}

func (s *MySQLStore) FetchGroupByKey(ctx context.Context, key TKey) (Group, error) {
	return s.get(ctx, "SELECT * FROM `group` WHERE `key` = ? LIMIT 1", key)
}

// FetchGroupByName retrieves a single group by a direct name match
func (s *MySQLStore) FetchGroupByName(ctx context.Context, name TName) (Group, error) {
	return s.get(ctx, "SELECT * FROM `group` WHERE `name` = ? LIMIT 1", name)
}

// FetchGroupsByName retrieves groups by their name
// NOTE: optionally search by partial prefix
func (s *MySQLStore) FetchGroupsByName(ctx context.Context, isPartial bool, name TName) ([]Group, error) {
	if isPartial {
		return s.getMany(ctx, "SELECT * FROM `group` WHERE `name` LIKE ?", fmt.Sprintf("%s%%", name))
	}

	return s.getMany(ctx, "SELECT * FROM `group` WHERE `name` = ?", name)
}

// TODO: 1. decide whether I want to load all existing groups
// TODO: 2. if (1), then develop a safer way to load all existing groups
func (s *MySQLStore) FetchAllGroups(ctx context.Context) ([]Group, error) {
	return s.getMany(ctx, "SELECT * FROM `group`")
}

// DeletePolicy from the store by group ObjectID
func (s *MySQLStore) DeleteByID(ctx context.Context, id uuid.UUID) (err error) {
	g, err := s.FetchGroupByID(ctx, id)
	if err != nil {
		return err
	}

	sess := s.db.NewSession(nil)

	// beginning transaction
	tx, err := sess.Begin()
	if err != nil {
		return err
	}
	defer tx.RollbackUnlessCommitted()

	// deleting the actual group
	_, err = sess.DeleteFrom("group").Where("id = ?", g.ID).ExecContext(ctx)
	if err != nil {
		return err
	}

	// deleting group relations
	// TODO: delete group relations

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit database transaction")
	}

	return nil
}

// CreateRelation store a relation flagging that user belongs to a group
func (s *MySQLStore) CreateRelation(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) (err error) {
	_, err = s.db.ExecContext(
		ctx,
		"INSERT IGNORE INTO `group_users`(group_id, user_id) VALUES(?, ?)",
		groupID,
		userID,
	)

	return err
}

// FetchAllRelations retrieving all relations
// NOTE: a map of users IDs -> a slice of group IDs
func (s *MySQLStore) FetchAllRelations(ctx context.Context) (relations map[uuid.UUID][]uuid.UUID, err error) {
	relations = make(map[uuid.UUID][]uuid.UUID, 0)

	// querying for just one column (user_id)
	rows, err := s.db.NewSession(nil).QueryContext(ctx, "SELECT group_id, user_id FROM `group_users`")
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRelationNotFound
		}

		return nil, err
	}

	defer func(c io.Closer) {
		if rerr := c.Close(); rerr != nil {
			err = rerr
		}
	}(rows)

	// iterating over and scanning found relations
	for rows.Next() {
		var groupID, userID uuid.UUID
		if err := rows.Scan(&groupID, &userID); err != nil {
			return nil, err
		}

		// initializing a nested slice if it's nil
		if relations[userID] == nil {
			relations[userID] = make([]uuid.UUID, 0)
		}

		// adding user ObjectID to the resulting slice
		relations[userID] = append(relations[userID], groupID)
	}

	return relations, nil
}

// FetchGroupRelations retrieving all group-user relations
func (s *MySQLStore) FetchGroupRelations(ctx context.Context, groupID uuid.UUID) ([]uuid.UUID, error) {
	relations := make([]uuid.UUID, 0)

	sess := s.db.NewSession(nil)

	// querying for just one column (user_id)
	rows, err := sess.QueryContext(ctx, "SELECT user_id FROM `group_users` WHERE group_id = ?", groupID)
	if err != nil {
		if err == sql.ErrNoRows {
			return relations, nil
		}

		return nil, err
	}

	defer func(c io.Closer) {
		if xerr := c.Close(); xerr != nil {
			err = xerr
		}
	}(rows)

	// iterating over and scanning found relations
	for rows.Next() {
		var userID uuid.UUID

		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}

		// adding user ObjectID to the resulting slice
		relations = append(relations, userID)
	}

	return relations, nil
}

// DeleteRelation deletes a group-user relation
func (s *MySQLStore) DeleteRelation(ctx context.Context, groupID uuid.UUID, userID uuid.UUID) error {
	res, err := s.db.ExecContext(
		ctx,
		"DELETE FROM `group_users` WHERE group_id = ? AND user_id = ? LIMIT 1",
		groupID,
		userID,
	)

	// error handling
	if err != nil {
		switch err := err.(*mysql.MySQLError); err.Number {
		//---------------------------------------------------------------------------
		// reserved for possible error code handling
		//---------------------------------------------------------------------------

		default:
			return err
		}
	}

	// checking whether anything was updated at all
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}

	// if no rows were affected then returning this as a non-critical error
	if ra == 0 {
		return ErrNothingChanged
	}

	return nil
}
*/
