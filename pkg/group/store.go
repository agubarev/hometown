package group

import (
	"context"
	"database/sql"
	"io"
	"log"

	"github.com/agubarev/hometown/internal/core"
	"github.com/go-sql-driver/mysql"
	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
)

// Store describes a storage contract for groups specifically
type Store interface {
	Put(g *Group) (*Group, error)
	FetchByID(ctx context.Context, groupID int) (*Group, error)
	FetchAllGroups(ctx context.Context) ([]*Group, error)
	DeleteByID(ctx context.Context, groupID int) error
	FetchAllRelations(ctx context.Context) (map[int][]int, error)
	HasRelation(ctx context.Context, groupID int, userID int) (bool, error)
	PutRelation(groupID int, userID int) error
	DeleteRelation(groupID int, userID int) error
}

// GroupStoreMySQL is the default group store implementation
type GroupStoreMySQL struct {
	db *dbr.Connection
}

// NewGroupStore returns a group store with mysql used as a backend
func NewGroupStore(db *dbr.Connection) (Store, error) {
	if db == nil {
		return nil, core.ErrNilDB
	}

	return &GroupStoreMySQL{db}, nil
}

//? BEGIN ->>>----------------------------------------------------------------
//? unexported utility functions

func (s *GroupStoreMySQL) get(ctx context.Context, q string, args ...interface{}) (*Group, error) {
	g := new(Group)

	err := s.db.NewSession(nil).SelectBySql(q, args).LoadOneContext(ctx, g)

	if err != nil {
		if err == dbr.ErrNotFound {
			return nil, core.ErrGroupNotFound
		}

		return nil, err
	}

	return g, nil
}

func (s *GroupStoreMySQL) getMany(ctx context.Context, q string, args ...interface{}) ([]*Group, error) {
	gs := make([]*Group, 0)

	if _, err := s.db.NewSession(nil).SelectBySql(q, args).LoadContext(ctx, gs); err != nil {
		return nil, err
	}

	return gs, nil
}

//? unexported utility functions
//? END ---<<<----------------------------------------------------------------

// UpdateAccessPolicy storing group
func (s *GroupStoreMySQL) Put(g *Group) (*Group, error) {
	if g == nil {
		return nil, core.ErrNilGroup
	}

	// if an object has ID other than 0, then it's considered
	// as being already created, thus requiring an update
	if g.ID != 0 {
		return s.Update(context.TODO(), g)
	}

	return s.Create(context.TODO(), g)
}

// Create creates a new database record
func (s *GroupStoreMySQL) Create(ctx context.Context, g *Group) (*Group, error) {
	// if ID is not 0, then it's not considered as new
	if g.ID != 0 {
		return nil, core.ErrObjectIsNotNew
	}

	sess := s.db.NewSession(nil)

	_, err := sess.InsertInto("group").Record(&g).ExecContext(ctx)

	// error handling
	if err != nil {
		switch err := err.(*mysql.MySQLError); err.Number {
		case 1062:
			return nil, core.ErrDuplicateEntry
		}
	}

	if err != nil {
		return nil, err
	}

	return g, nil
}

// UpdateAccessPolicy updates an existing group
func (s *GroupStoreMySQL) Update(ctx context.Context, g *Group) (*Group, error) {
	if g.ID == 0 {
		return nil, core.ErrObjectIsNew
	}

	sess := s.db.NewSession(nil)

	updates := map[string]interface{}{
		"key":         g.Key,
		"name":        g.Name,
		"description": g.Description,
	}

	// just executing query but not refetching the updated version
	res, err := sess.Update("group").SetMap(updates).Where("id = ?", g.ID).ExecContext(ctx)
	if err != nil {
		return nil, err
	}

	// checking whether anything was updated at all
	ra, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	// if no rows were affected then returning this as a non-critical error
	if ra == 0 {
		return nil, core.ErrNothingChanged
	}

	return g, nil
}

// GetUserByID retrieving a group by ID
func (s *GroupStoreMySQL) FetchByID(ctx context.Context, id int) (*Group, error) {
	return s.get(ctx, "SELECT * FROM `groups` WHERE id = ? LIMIT 1", id)
}

// FetchAllGroups retrieving all groups
func (s *GroupStoreMySQL) FetchAllGroups(ctx context.Context) ([]*Group, error) {
	return s.getMany(ctx, "SELECT * FROM `group`")
}

// Delete from the store by group ID
func (s *GroupStoreMySQL) DeleteByID(ctx context.Context, id int) (err error) {
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

			log.Printf("Store.Delete(): recovering from panic, transaction rollback: %s", p)

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
func (s *GroupStoreMySQL) PutRelation(groupID int, userID int) error {
	_, err := s.db.Exec(
		"INSERT IGNORE INTO `group_users`(group_id, user_id) VALUES(?, ?)",
		groupID,
		userID,
	)

	// error handling
	if err != nil {
		switch err := err.(*mysql.MySQLError); err.Number {
		case 1062:
			return core.ErrDuplicateEntry
		default:
			return err
		}
	}

	return nil
}

// FetchAllRelations retrieving all relations
// NOTE: a map of users IDs -> a slice of group IDs
func (s *GroupStoreMySQL) FetchAllRelations(ctx context.Context) (_ map[int][]int, err error) {
	relations := make(map[int][]int)

	sess := s.db.NewSession(nil)

	// querying for just one column (user_id)
	rows, err := sess.QueryContext(ctx, "SELECT group_id, user_id FROM `group_users`")
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, core.ErrRelationNotFound
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
		var gid, uid int
		if err := rows.Scan(&gid, &uid); err != nil {
			return nil, err
		}

		// initializing a nested slice if it's nil
		if relations[uid] == nil {
			relations[uid] = make([]int, 0)
		}

		// adding user ID to the resulting slice
		relations[uid] = append(relations[uid], gid)
	}

	return relations, nil
}

// GetGroupRelations retrieving all group-user relations
func (s *GroupStoreMySQL) GetGroupRelations(ctx context.Context, id int) ([]int, error) {
	relations := make([]int, 0)

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
		var userID int

		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}

		// adding user ID to the resulting slice
		relations = append(relations, userID)
	}

	return relations, nil
}

// HasRelation checks whether group-user relation exists
func (s *GroupStoreMySQL) HasRelation(ctx context.Context, groupID int, userID int) (bool, error) {
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
		var foundGroupID, foundUserID int
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
func (s *GroupStoreMySQL) DeleteRelation(groupID int, userID int) error {
	res, err := s.db.Exec(
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
		return core.ErrNothingChanged
	}

	return nil
}
