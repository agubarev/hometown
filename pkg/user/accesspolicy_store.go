package user

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/go-sql-driver/mysql"
	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
)

// AccessPolicyStore is a storage contract interface for the AccessPolicy objects
// TODO: keep rights separate and segregated by it's kind i.e. Public, AccessPolicy, Role, User etc.
type AccessPolicyStore interface {
	CreatePolicy(ctx context.Context, ap AccessPolicy) (AccessPolicy, error)
	UpdatePolicy(ctx context.Context, ap AccessPolicy) error
	FetchPolicyByID(ctx context.Context, id uint32) (AccessPolicy, error)
	FetchPolicyByName(ctx context.Context, name TPolicyName) (ap AccessPolicy, err error)
	FetchPolicyByObjectTypeAndID(ctx context.Context, objectType TAPObjectType, id uint32) (ap AccessPolicy, err error)
	DeletePolicy(ctx context.Context, ap AccessPolicy) error
}

// RightsRosterDatabaseRecord represents a single database row
type RightsRosterDatabaseRecord struct {
	PolicyID        uint32      `db:"policy_id"`
	SubjectKind     SubjectKind `db:"subject_kind"`
	SubjectID       uint32      `db:"subject_id"`
	AccessRight     AccessRight `db:"access_right"`
	AccessExplained string      `db:"access_explained"`
}

// DefaultAccessPolicyStore is a default access policy store implementation
type DefaultAccessPolicyStore struct {
	db *dbr.Connection
}

// NewDefaultAccessPolicyStore returns an initialized default domain store
func NewDefaultAccessPolicyStore(db *dbr.Connection) (AccessPolicyStore, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}

	s := &DefaultAccessPolicyStore{db}

	return s, nil
}

//? BEGIN ->>>----------------------------------------------------------------
//? unexported utility functions

func (s *DefaultAccessPolicyStore) get(ctx context.Context, q string, args ...interface{}) (ap AccessPolicy, err error) {
	sess := s.db.NewSession(nil)

	// fetching policy itself
	if err := sess.SelectBySql(q, args).LoadOneContext(ctx, &ap); err != nil {
		if err == dbr.ErrNotFound {
			return ap, ErrAccessPolicyNotFound
		}

		return ap, err
	}

	// fetching rights roster
	rrs := make([]RightsRosterDatabaseRecord, 0)

	count, err := sess.
		SelectBySql("SELECT * FROM accesspolicy_rights_roster WHERE policy_id = ?", ap.ID).
		LoadContext(ctx, &rrs)

	switch true {
	case err != nil:
		return ap, err
	case count == 0:
		return ap, ErrEmptyRightsRoster
	}

	// initializing rights roster
	rr := &RightsRoster{
		Group: make(map[uint32]AccessRight),
		Role:  make(map[uint32]AccessRight),
		User:  make(map[uint32]AccessRight),
	}

	// transforming database records into rights roster
	for _, r := range rrs {
		switch r.SubjectKind {
		case SKEveryone:
			rr.Everyone = r.AccessRight
		case SKRoleGroup:
			rr.Role[r.SubjectID] = r.AccessRight
		case SKGroup:
			rr.Group[r.SubjectID] = r.AccessRight
		case SKUser:
			rr.User[r.SubjectID] = r.AccessRight
		default:
			log.Printf(
				"unrecognized subject kind for access policy (subject_kind=%s, subject_id=%d, access_right=%d)",
				r.SubjectKind,
				r.SubjectID,
				r.AccessRight,
			)
		}
	}

	// attaching its rights roster
	ap.RightsRoster = rr

	return ap, nil
}

func (s *DefaultAccessPolicyStore) getMany(ctx context.Context, q string, args ...interface{}) (aps []AccessPolicy, err error) {
	count, err := s.db.NewSession(nil).SelectBySql(q, args).LoadContext(ctx, &aps)
	switch true {
	case err != nil:
		return nil, err
	case count == 0:
		return nil, ErrAccessPolicyNotFound
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return aps, nil
		}

		return nil, err
	}

	return aps, nil
}

//? unexported utility functions
//? END ---<<<----------------------------------------------------------------

// Upsert creating access policy
func (s *DefaultAccessPolicyStore) CreatePolicy(ctx context.Context, ap AccessPolicy) (_ AccessPolicy, err error) {
	// basic validations
	if ap.ID != 0 {
		return ap, ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return ap, ErrNilRightsRoster
	}

	if ap.ID != 0 {
		return ap, ErrNonZeroID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return ap, fmt.Errorf("failed to begin a transaction: %s", err)
	}

	// panic handler
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(err, "recovered from panic")

			// rolling back transaction
			if xerr := tx.Rollback(); xerr != nil { // oshi-
				err = errors.Wrapf(err, "rollback failed: %s", xerr)
			}
		}
	}()

	//---------------------------------------------------------------------------
	// creating access policy
	//---------------------------------------------------------------------------
	// executing statement
	result, err := tx.InsertInto("accesspolicy").Record(&ap).ExecContext(ctx)

	// error handling
	if err != nil {
		switch err := err.(*mysql.MySQLError); err.Number {
		case 1062:
			panic(err)
		}
	}

	if err != nil {
		panic(err)
	}

	// obtaining new ObjectID
	newID, err := result.LastInsertId()
	if err != nil {
		panic(err)
	}

	//---------------------------------------------------------------------------
	// preparing new roster records
	//---------------------------------------------------------------------------
	rrs := make([]RightsRosterDatabaseRecord, 0)

	// for everyone
	rrs = append(rrs, RightsRosterDatabaseRecord{
		PolicyID:        uint32(newID),
		SubjectKind:     SKEveryone,
		AccessRight:     ap.RightsRoster.Everyone,
		AccessExplained: ap.RightsRoster.Everyone.Explain(),
	})

	// for roles
	for id, right := range ap.RightsRoster.Role {
		rrs = append(rrs, RightsRosterDatabaseRecord{
			PolicyID:        uint32(newID),
			SubjectKind:     SKRoleGroup,
			SubjectID:       id,
			AccessRight:     right,
			AccessExplained: right.Explain(),
		})
	}

	// for groups
	for id, right := range ap.RightsRoster.Group {
		rrs = append(rrs, RightsRosterDatabaseRecord{
			PolicyID:        uint32(newID),
			SubjectKind:     SKGroup,
			SubjectID:       id,
			AccessRight:     right,
			AccessExplained: right.Explain(),
		})
	}

	// for users
	for id, right := range ap.RightsRoster.User {
		rrs = append(rrs, RightsRosterDatabaseRecord{
			PolicyID:        uint32(newID),
			SubjectKind:     SKUser,
			SubjectID:       id,
			AccessRight:     right,
			AccessExplained: right.Explain(),
		})
	}

	//---------------------------------------------------------------------------
	// creating roster records
	//---------------------------------------------------------------------------
	// looping over rights roster to be created
	for _, r := range rrs {
		_, err := tx.InsertInto("accesspolicy").Record(&r).ExecContext(ctx)
		if err != nil {
			panic(err)
		}
	}

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err := tx.Commit(); err != nil {
		panic(err)
	}

	// setting its new ObjectID
	ap.ID = uint32(newID)

	return ap, nil
}

// UpdatePolicy updates an existing access policy along with its rights roster
// NOTE: rights roster keeps track of its changes, thus, update will
// only affect changes mentioned by the respective RightsRoster object
func (s *DefaultAccessPolicyStore) UpdatePolicy(ctx context.Context, ap AccessPolicy) (err error) {
	// basic validations
	if ap.ID == 0 {
		return ErrZeroID
	}

	if ap.RightsRoster == nil {
		return ErrNilRightsRoster
	}

	if ap.ID == 0 {
		return ErrZeroID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin a transaction")
	}

	// panic handler
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(err, "recovered from panic")

			// rolling back the transaction
			if xerr := tx.Rollback(); xerr != nil { // oshi-
				err = errors.Wrap(xerr, "rollback failed")
			}
		}
	}()

	//---------------------------------------------------------------------------
	// updating access policy and its rights roster (only the changes)
	//---------------------------------------------------------------------------
	// listing fields to be updated
	updates := map[string]interface{}{
		"parent_id":    ap.ParentID,
		"owner_id":     ap.OwnerID,
		"key":          ap.Key,
		"object_kind":  ap.ObjectType,
		"object_id":    ap.ObjectID,
		"is_extended":  ap.IsExtended,
		"is_inherited": ap.IsInherited,
	}

	// executing statement
	_, err = tx.Update("accesspolicy").SetMap(updates).Where("id = ?", ap.ID).ExecContext(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to update access policy")
	}

	// checking whether the rights roster has any changes
	if ap.RightsRoster != nil {
		for _, c := range ap.RightsRoster.changes {
			switch c.action {
			case 0:
				//---------------------------------------------------------------------------
				// deleting
				//---------------------------------------------------------------------------
				_, err = tx.Exec(
					"DELETE FROM accesspolicy_rights_roster WHERE policy_id = ? AND subject_kind = ? AND subject_id = ?",
					ap.ID,
					c.subjectKind,
					c.subjectID,
				)

				if err != nil {
					return errors.Wrap(err, "failed to delete rights roster record")
				}
			case 1:
				//---------------------------------------------------------------------------
				// creating
				//---------------------------------------------------------------------------
				_, err = tx.Exec(
					"INSERT INTO accesspolicy_rights_roster (policy_id, subject_kind, subject_id, access, access_explained) VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE access = ?",
					ap.ID,
					c.subjectKind,
					c.subjectID,
					c.accessRight,
					c.accessRight.Explain(),
					c.accessRight,
				)

				if err != nil {
					return errors.Wrap(err, "failed to create rights roster record")
				}
			}
		}
	}

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit database transaction")
	}

	return nil
}

// GroupByID retrieving a access policy by ObjectID
func (s *DefaultAccessPolicyStore) FetchPolicyByID(ctx context.Context, policyID uint32) (AccessPolicy, error) {
	return s.get(ctx, "SELECT * FROM accesspolicy WHERE id = ? LIMIT 1", policyID)
}

// PolicyByName retrieving a access policy by a key
func (s *DefaultAccessPolicyStore) FetchPolicyByName(ctx context.Context, name TPolicyName) (AccessPolicy, error) {
	return s.get(ctx, "SELECT * FROM accesspolicy WHERE `name` = ? LIMIT 1", name)
}

// PolicyByObjectTypeAndID retrieving a access policy by a kind and its respective id
func (s *DefaultAccessPolicyStore) FetchPolicyByObjectTypeAndID(ctx context.Context, objectType TAPObjectType, id uint32) (AccessPolicy, error) {
	return s.get(ctx, "SELECT * FROM accesspolicy WHERE object_type = ? AND object_id = ? LIMIT 1", objectType, id)
}

// DeletePolicy access policy
func (s *DefaultAccessPolicyStore) DeletePolicy(ctx context.Context, ap AccessPolicy) (err error) {
	// basic validations
	if ap.ID == 0 {
		return ErrZeroID
	}

	if ap.RightsRoster == nil {
		return ErrNilRightsRoster
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin database transaction")
	}
	tx.RollbackUnlessCommitted()

	// panic handler
	defer func() {
		if r := recover(); r != nil {
			err = errors.Wrap(err, "recovered from panic")

			// rolling back the transaction
			if xerr := tx.Rollback(); xerr != nil { // oshi-
				err = errors.Wrap(xerr, "rollback failed")
			}
		}
	}()

	//---------------------------------------------------------------------------
	// deleting policy along with it's respective rights roster
	//---------------------------------------------------------------------------
	// deleting access policy
	if _, err = tx.ExecContext(ctx, "DELETE FROM accesspolicy WHERE id = ?", ap.ID); err != nil {
		return errors.Wrap(err, "failed to delete access policy")
	}

	// deleting rights roster
	if _, err = tx.ExecContext(ctx, "DELETE FROM accesspolicy_rights_roster WHERE policy_id = ?", ap.ID); err != nil {
		return errors.Wrap(err, "failed to delete rights roster")
	}

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit database transaction")
	}

	return nil
}
