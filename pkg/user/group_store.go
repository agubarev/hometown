package user

import (
	"context"
	"database/sql"
	"io"
	"log"

	"github.com/agubarev/hometown/pkg/util/guard"
	"github.com/go-sql-driver/mysql"
	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
)

// GroupStore describes a storage contract for groups specifically
type GroupStore interface {
	UpsertGroup(ctx context.Context, g Group) (Group, error)
	PutRelation(ctx context.Context, groupID int64, userID int64) error
	FetchByID(ctx context.Context, groupID int64) (Group, error)
	FetchAllGroups(ctx context.Context) ([]Group, error)
	FetchAllRelations(ctx context.Context) (map[int64][]int64, error)
	HasRelation(ctx context.Context, groupID int64, userID int64) (bool, error)
	DeleteByID(ctx context.Context, groupID int64) error
	DeleteRelation(ctx context.Context, groupID int64, userID int64) error
}

// GroupMySQLStore is the default group store implementation
type GroupMySQLStore struct {
	db *dbr.Connection
}

// NewGroupStore returns a group store with mysql used as a backend
func NewGroupStore(db *dbr.Connection) (GroupStore, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}

	return &GroupMySQLStore{db}, nil
}

//? BEGIN ->>>----------------------------------------------------------------
//? unexported utility functions

func (s *GroupMySQLStore) get(ctx context.Context, q string, args ...interface{}) (g Group, err error) {
	err = s.db.NewSession(nil).
		SelectBySql(q, args).
		LoadOneContext(ctx, &g)

	if err != nil {
		if err == dbr.ErrNotFound {
			return g, ErrGroupNotFound
		}

		return g, err
	}

	return g, nil
}

func (s *GroupMySQLStore) getMany(ctx context.Context, q string, args ...interface{}) (gs []Group, err error) {
	if _, err := s.db.NewSession(nil).SelectBySql(q, args).LoadContext(ctx, &gs); err != nil {
		return nil, err
	}

	return gs, nil
}

//? unexported utility functions
//? END ---<<<----------------------------------------------------------------

// UpdatePolicy storing group
func (s *GroupMySQLStore) UpsertGroup(ctx context.Context, g Group) (Group, error) {
	// if an object has ObjectID other than 0, then it's considered
	// as being already created, thus requiring an update
	if g.ID != 0 {
		return s.Update(ctx, g)
	}

	return s.Create(ctx, g)
}

// Upsert creates a new database record
func (s *GroupMySQLStore) Create(ctx context.Context, g Group) (Group, error) {
	// if ObjectID is not 0, then it's not considered as new
	if g.ID != 0 {
		return g, ErrNonZeroID
	}

	_, err := s.db.NewSession(nil).
		InsertInto("group").
		Columns(guard.DBColumnsFrom(&g)...).
		Record(&g).
		ExecContext(ctx)

	// error handling
	if err != nil {
		switch err := err.(*mysql.MySQLError); err.Number {
		case 1062:
			return g, ErrDuplicateGroup
		}
	}

	if err != nil {
		return g, err
	}

	return g, nil
}

// UpdatePolicy updates an existing group
func (s *GroupMySQLStore) Update(ctx context.Context, g Group) (Group, error) {
	if g.ID == 0 {
		return g, ErrZeroID
	}

	updates := map[string]interface{}{
		"key":         g.Key,
		"name":        g.Name,
		"description": g.Description,
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

// UserByID retrieving a group by ObjectID
func (s *GroupMySQLStore) FetchByID(ctx context.Context, id int64) (Group, error) {
	return s.get(ctx, "SELECT * FROM `group` WHERE id = ? LIMIT 1", id)
}

// FetchAllGroups retrieving all groups
func (s *GroupMySQLStore) FetchAllGroups(ctx context.Context) ([]Group, error) {
	return s.getMany(ctx, "SELECT * FROM `group`")
}

// DeletePolicy from the store by group ObjectID
func (s *GroupMySQLStore) DeleteByID(ctx context.Context, id int64) (err error) {
	g, err := s.FetchByID(ctx, id)
	if err != nil {
		return err
	}

	sess := s.db.NewSession(nil)

	// beginning transaction
	tx, err := sess.Begin()
	if err != nil {
		return err
	}

	_, err = sess.DeleteFrom("group").Where("id = ?", g.ID).ExecContext(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			err = p.(error)

			log.Printf("GroupStore.DeletePolicy(): recovering from panic, transaction rollback: %s", p)

			if xerr := tx.Rollback(); xerr != nil {
				panic(errors.Wrapf(err, "rollback failed: %s", xerr))
			}
		}
	}()

	if err := tx.Commit(); err != nil {
		return errors.Wrapf(err, "failed to commit transaction")
	}

	return nil
}

// PutRelation store a relation flagging that user belongs to a group
func (s *GroupMySQLStore) PutRelation(ctx context.Context, groupID int64, userID int64) (err error) {
	_, err = s.db.ExecContext(
		ctx,
		"INSERT IGNORE INTO `group_users`(group_id, user_id) VALUES(?, ?)",
		groupID,
		userID,
	)

	// error handling
	if err != nil {
		switch err := err.(*mysql.MySQLError); err.Number {
		case 1062:
			return ErrDuplicateRelation
		default:
			return err
		}
	}

	return nil
}

// FetchAllRelations retrieving all relations
// NOTE: a map of users IDs -> a slice of group IDs
func (s *GroupMySQLStore) FetchAllRelations(ctx context.Context) (relations map[int64][]int64, err error) {
	relations = make(map[int64][]int64, 0)

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
		var gid, uid int64
		if err := rows.Scan(&gid, &uid); err != nil {
			return nil, err
		}

		// initializing a nested slice if it's nil
		if relations[uid] == nil {
			relations[uid] = make([]int64, 0)
		}

		// adding user ObjectID to the resulting slice
		relations[uid] = append(relations[uid], gid)
	}

	return relations, nil
}

// GetGroupRelations retrieving all group-user relations
func (s *GroupMySQLStore) GetGroupRelations(ctx context.Context, id int64) ([]int64, error) {
	relations := make([]int64, 0)

	sess := s.db.NewSession(nil)

	// querying for just one column (user_id)
	rows, err := sess.QueryContext(ctx, "SELECT user_id FROM `group_users` WHERE group_id = ?", id)
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
		var userID int64

		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}

		// adding user ObjectID to the resulting slice
		relations = append(relations, userID)
	}

	return relations, nil
}

// HasRelation checks whether group-user relation exists
func (s *GroupMySQLStore) HasRelation(ctx context.Context, groupID int64, userID int64) (bool, error) {
	sess := s.db.NewSession(nil)

	// querying for just one column (user_id)
	rows, err := sess.QueryContext(
		ctx,
		"SELECT group_id, user_id FROM `group_users` WHERE group_id = ? AND user_id = ? LIMIT 1",
		groupID,
		userID,
	)

	// deferring a Close() on io.Closer (standard procedure)
	defer func(c io.Closer) {
		if err == nil {
			return
		}

		if xerr := c.Close(); xerr != nil {
			err = xerr
		}
	}(rows)

	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}

		return false, err
	}

	// iterating over and scanning found relations
	for rows.Next() {
		var foundGroupID, foundUserID int64
		if err := rows.Scan(&foundGroupID, &foundUserID); err != nil {
			return false, err
		}

		// paranoid check
		if foundGroupID == groupID && foundUserID == userID {
			return true, nil
		}
	}

	return false, nil
}

// DeleteRelation deletes a group-user relation
func (s *GroupMySQLStore) DeleteRelation(ctx context.Context, groupID int64, userID int64) error {
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
