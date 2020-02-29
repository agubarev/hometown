package user

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/agubarev/hometown/pkg/util/guard"
	"github.com/go-sql-driver/mysql"
	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
)

// AccessPolicyStore is a storage contract interface for the AccessPolicy objects
// TODO: keep rights separate and segregated by it's kind i.e. Public, AccessPolicy, Role, User etc.
type AccessPolicyStore interface {
	CreatePolicy(ctx context.Context, ap AccessPolicy) (AccessPolicy, error)
	UpdatePolicy(ctx context.Context, ap AccessPolicy) error
	FetchPolicyByID(ctx context.Context, id int64) (AccessPolicy, error)
	FetchPolicyByName(ctx context.Context, name string) (ap AccessPolicy, err error)
	FetchPolicyByObjectTypeAndID(ctx context.Context, objectType string, id int64) (ap AccessPolicy, err error)
	DeletePolicy(ctx context.Context, ap AccessPolicy) error
}

// RightsRosterDatabaseRecord represents a single database row
type RightsRosterDatabaseRecord struct {
	PolicyID        int64       `db:"policy_id"`
	SubjectKind     SubjectKind `db:"subject_kind"`
	SubjectID       int64       `db:"subject_id"`
	AccessRight     AccessRight `db:"access"`
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
	if err := sess.SelectBySql(q, args...).LoadOneContext(ctx, &ap); err != nil {
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
	rr := RightsRoster{
		Group: make(map[int64]AccessRight),
		Role:  make(map[int64]AccessRight),
		User:  make(map[int64]AccessRight),
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
				"unrecognized subject kind for access policy (subject_kind=%d, subject_id=%d, access_right=%d)",
				r.SubjectKind,
				r.SubjectID,
				r.AccessRight,
			)
		}
	}

	// attaching its rights roster
	ap.RightsRoster = &rr

	return ap, nil
}

func (s *DefaultAccessPolicyStore) getMany(ctx context.Context, q string, args ...interface{}) (aps []AccessPolicy, err error) {
	count, err := s.db.NewSession(nil).SelectBySql(q, args...).LoadContext(ctx, &aps)
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
	defer tx.RollbackUnlessCommitted()

	//---------------------------------------------------------------------------
	// creating access policy
	//---------------------------------------------------------------------------
	// executing statement

	result, err := tx.InsertInto("accesspolicy").
		Columns(guard.DBColumnsFrom(&ap)...).
		Record(&ap).
		ExecContext(ctx)

	// error handling
	if err != nil {
		switch err := err.(*mysql.MySQLError); err.Number {
		case 1062:
			return ap, err
		}
	}

	if err != nil {
		return ap, err
	}

	// obtaining new ObjectID
	newID, err := result.LastInsertId()
	if err != nil {
		return ap, err
	}

	//---------------------------------------------------------------------------
	// preparing new roster records
	//---------------------------------------------------------------------------
	rrs := make([]RightsRosterDatabaseRecord, 0)

	// for everyone
	rrs = append(rrs, RightsRosterDatabaseRecord{
		PolicyID:        newID,
		SubjectKind:     SKEveryone,
		AccessRight:     ap.RightsRoster.Everyone,
		AccessExplained: ap.RightsRoster.Everyone.String(),
	})

	// for roles
	for id, right := range ap.RightsRoster.Role {
		rrs = append(rrs, RightsRosterDatabaseRecord{
			PolicyID:        newID,
			SubjectKind:     SKRoleGroup,
			SubjectID:       id,
			AccessRight:     right,
			AccessExplained: right.String(),
		})
	}

	// for groups
	for id, right := range ap.RightsRoster.Group {
		rrs = append(rrs, RightsRosterDatabaseRecord{
			PolicyID:        newID,
			SubjectKind:     SKGroup,
			SubjectID:       id,
			AccessRight:     right,
			AccessExplained: right.String(),
		})
	}

	// for users
	for id, right := range ap.RightsRoster.User {
		rrs = append(rrs, RightsRosterDatabaseRecord{
			PolicyID:        newID,
			SubjectKind:     SKUser,
			SubjectID:       id,
			AccessRight:     right,
			AccessExplained: right.String(),
		})
	}

	//---------------------------------------------------------------------------
	// creating roster records
	//---------------------------------------------------------------------------
	// looping over rights roster to be created

	// TODO: squash into a single insert statement
	for _, r := range rrs {
		_, err := tx.InsertInto("accesspolicy_rights_roster").
			Columns(guard.DBColumnsFrom(&r)...).
			Record(&r).
			ExecContext(ctx)

		if err != nil {
			return ap, err
		}
	}

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err := tx.Commit(); err != nil {
		return ap, errors.Wrap(err, "failed to commit database transaction")
	}

	// setting its new ObjectID
	ap.ID = newID

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
	defer tx.RollbackUnlessCommitted()

	//---------------------------------------------------------------------------
	// updating access policy and its rights roster (only the changes)
	//---------------------------------------------------------------------------
	// listing fields to be updated
	updates := map[string]interface{}{
		"parent_id":    ap.ParentID,
		"owner_id":     ap.OwnerID,
		"name":         ap.Name,
		"object_type":  ap.ObjectType,
		"object_id":    ap.ObjectID,
		"is_extended":  ap.IsExtended,
		"is_inherited": ap.IsInherited,
		"checksum":     ap.Checksum,
	}

	// executing statement
	_, err = tx.Update("accesspolicy").SetMap(updates).Where("id = ?", ap.ID).ExecContext(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to update access policy")
	}

	// checking whether the rights roster has any changes
	// TODO: optimize by squashing inserts and deletes into single queries
	if ap.RightsRoster != nil {
		for _, c := range ap.RightsRoster.changes {
			switch c.action {
			case RRSet:
				//---------------------------------------------------------------------------
				// creating
				//---------------------------------------------------------------------------
				_, err = tx.Exec(
					"INSERT INTO accesspolicy_rights_roster (policy_id, subject_kind, subject_id, access, access_explained) VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE access = ?",
					ap.ID,
					c.subjectKind,
					c.subjectID,
					c.accessRight,
					c.accessRight.String(),
					c.accessRight,
				)

				if err != nil {
					return errors.Wrap(err, "failed to upsert rights roster record")
				}
			case RRUnset:
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
func (s *DefaultAccessPolicyStore) FetchPolicyByID(ctx context.Context, policyID int64) (AccessPolicy, error) {
	return s.get(ctx, "SELECT * FROM accesspolicy WHERE id = ? LIMIT 1", policyID)
}

// PolicyByName retrieving a access policy by a key
func (s *DefaultAccessPolicyStore) FetchPolicyByName(ctx context.Context, name string) (AccessPolicy, error) {
	return s.get(ctx, "SELECT * FROM accesspolicy WHERE `name` = ? LIMIT 1", name)
}

// PolicyByObjectTypeAndID retrieving a access policy by a kind and its respective id
func (s *DefaultAccessPolicyStore) FetchPolicyByObjectTypeAndID(ctx context.Context, objectType string, id int64) (AccessPolicy, error) {
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
	defer tx.RollbackUnlessCommitted()

	//---------------------------------------------------------------------------
	// deleting policy along with it's respective rights roster
	//---------------------------------------------------------------------------
	// deleting access policy
	if _, err = tx.ExecContext(ctx, "DELETE FROM accesspolicy WHERE id = ?", ap.ID); err != nil {
		return errors.Wrapf(err, "failed to delete access policy: policy_id=%d", ap.ID)
	}

	// deleting rights roster
	if _, err = tx.ExecContext(ctx, "DELETE FROM accesspolicy_rights_roster WHERE policy_id = ?", ap.ID); err != nil {
		return errors.Wrapf(err, "failed to delete rights roster: policy_id=%d", ap.ID)
	}

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit database transaction")
	}

	return nil
}
