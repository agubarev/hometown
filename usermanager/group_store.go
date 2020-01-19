package usermanager

import (
	"database/sql"
	"log"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// GroupStore describes a storage contract for groups specifically
type GroupStore interface {
	Put(g *Group) (*Group, error)
	Fetch(groupID int64) (*Group, error)
	FetchAllGroups() ([]*Group, error)
	Delete(groupID int64) error
	FetchAllRelations() (map[int64][]int64, error)
	HasRelation(groupID, userID int64) (bool, error)
	PutRelation(groupID int64, userID int64) error
	DeleteRelation(groupID int64, userID int64) error
}

// GroupStoreMySQL is the default group store implementation
type GroupStoreMySQL struct {
	db *sqlx.DB
}

// NewGroupStore returns a group store with mysql used as a backend
func NewGroupStore(db *sqlx.DB) (GroupStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	return &GroupStoreMySQL{db}, nil
}

//? BEGIN ->>>----------------------------------------------------------------
//? unexported utility functions

func (s *GroupStoreMySQL) get(q string, args ...interface{}) (*Group, error) {
	g := new(Group)

	err := s.db.Unsafe().Get(g, q, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrGroupNotFound
		}

		return nil, err
	}

	return g, nil
}

func (s *GroupStoreMySQL) getMany(q string, args ...interface{}) ([]*Group, error) {
	gs := make([]*Group, 0)

	err := s.db.Unsafe().Select(&gs, q, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return gs, nil
		}

		return nil, err
	}

	return gs, nil
}

//? unexported utility functions
//? END ---<<<----------------------------------------------------------------

// Put storing group
func (s *GroupStoreMySQL) Put(g *Group) (*Group, error) {
	if g == nil {
		return nil, ErrNilGroup
	}

	// if an object has ID other than 0, then it's considered
	// as being already created, thus requiring an update
	if g.ID != 0 {
		return s.Update(g)
	}

	return s.Create(g)
}

// Create creates a new database record
func (s *GroupStoreMySQL) Create(g *Group) (*Group, error) {
	// if ID is not 0, then it's not considered as new
	if g.ID != 0 {
		return nil, ErrObjectIsNotNew
	}

	// constructing statement
	q := "INSERT INTO `groups`(parent_id, kind, `key`, name, description) VALUES(:parent_id, :kind, :key, :name, :description)"

	// executing statement
	result, err := s.db.NamedExec(q, g)

	// error handling
	if err != nil {
		switch err := err.(*mysql.MySQLError); err.Number {
		case 1062:
			return nil, ErrDuplicateEntry
		}
	}

	if err != nil {
		return nil, err
	}

	newID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// setting new ID
	g.ID = newID

	return g, nil
}

// Update updates an existing group
func (s *GroupStoreMySQL) Update(g *Group) (*Group, error) {
	if g.ID == 0 {
		return nil, ErrObjectIsNew
	}

	// update record statement
	q := "UPDATE `groups` SET key = :key, name = :name, description = :description WHERE id = :id LIMIT 1"

	// just executing query but not refetching the updated version
	res, err := s.db.NamedExec(q, &g)
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
		return nil, ErrNothingChanged
	}

	return g, nil
}

// Fetch retrieving a group by ID
func (s *GroupStoreMySQL) Fetch(id int64) (*Group, error) {
	return s.get("SELECT * FROM `groups` WHERE id = ? LIMIT 1", id)
}

// FetchAllGroups retrieving all groups
func (s *GroupStoreMySQL) FetchAllGroups() ([]*Group, error) {
	return s.getMany("SELECT * FROM `groups`")
}

// Delete from the store by group ID
func (s *GroupStoreMySQL) Delete(id int64) error {
	g, err := s.Fetch(id)
	if err != nil {
		return err
	}

	// beginning transaction
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			log.Printf("GroupStore.Delete(): recovering from panic, transaction rollback: %s", p)
			tx.Rollback()
		}
	}()

	//? BEGIN ->>>----------------------------------------------------------------
	//? deleting records which are only relevant to this group

	// group-user relations
	_, err = tx.Exec("DELETE FROM `group_users` WHERE group_id = ? LIMIT 1", g.ID)
	if err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			g.Logger().Warn("rollback failed", zap.Error(err2))
		}

		return err
	}

	// actual groups
	_, err = tx.Exec("DELETE FROM `groups` WHERE id = ? LIMIT 1", g.ID)
	if err != nil {
		if err2 := tx.Rollback(); err2 != nil {
			g.Logger().Warn("rollback failed", zap.Error(err2))
		}

		return err
	}

	//? deleting records which are only relevant to this group
	//? END ---<<<----------------------------------------------------------------

	return tx.Commit()
}

// PutRelation store a relation flagging that user belongs to a group
func (s *GroupStoreMySQL) PutRelation(groupID int64, userID int64) error {
	_, err := s.db.Exec(
		"INSERT IGNORE INTO `group_users`(group_id, user_id) VALUES(?, ?)",
		groupID,
		userID,
	)

	// error handling
	if err != nil {
		switch err := err.(*mysql.MySQLError); err.Number {
		case 1062:
			return ErrDuplicateEntry
		default:
			return err
		}
	}

	return nil
}

// FetchAllRelations retrieving all relations
// NOTE: a map of users IDs -> a slice of group IDs
func (s *GroupStoreMySQL) FetchAllRelations() (map[int64][]int64, error) {
	relations := make(map[int64][]int64)

	// querying for just one column (user_id)
	rows, err := s.db.Queryx("SELECT group_id, user_id FROM `group_users`")
	defer rows.Close()

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrRelationNotFound
		}

		return nil, err
	}

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

		// adding user ID to the resulting slice
		relations[uid] = append(relations[uid], gid)
	}

	return relations, nil
}

// GetGroupRelations retrieving all group-user relations
func (s *GroupStoreMySQL) GetGroupRelations(id int64) ([]int64, error) {
	relations := make([]int64, 0)

	// querying for just one column (user_id)
	rows, err := s.db.Queryx("SELECT user_id FROM `group_users` WHERE group_id = ?", id)
	defer rows.Close()

	if err != nil {
		if err == sql.ErrNoRows {
			return relations, nil
		}

		return nil, err
	}

	// iterating over and scanning found relations
	for rows.Next() {
		var userID int64
		if err := rows.Scan(&userID); err != nil {
			return nil, err
		}

		// adding user ID to the resulting slice
		relations = append(relations, userID)
	}

	return relations, nil
}

// HasRelation checks whether group-user relation exists
func (s *GroupStoreMySQL) HasRelation(groupID, userID int64) (bool, error) {
	// querying for just one column (user_id)
	rows, err := s.db.Queryx(
		"SELECT group_id, user_id FROM `group_users` WHERE group_id = ? AND user_id = ? LIMIT 1",
		groupID,
		userID,
	)

	defer rows.Close()

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
func (s *GroupStoreMySQL) DeleteRelation(groupID int64, userID int64) error {
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
		return ErrNothingChanged
	}

	return nil
}
