package accesspolicy

import (
	"context"
	"database/sql"
	"fmt"
	"log"

	"github.com/agubarev/hometown/internal/core"
	"github.com/go-sql-driver/mysql"
	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
)

// Store is a storage contract interface for the AccessPolicy objects
// TODO: keep rights separate and segregated by it's kind i.e. Public, AccessPolicy, Role, User etc.
type Store interface {
	Create(ctx context.Context, ap *AccessPolicy) (*AccessPolicy, error)
	UpdateAccessPolicy(ctx context.Context, ap *AccessPolicy) error
	GetByID(id int) (*AccessPolicy, error)
	GetByName(name TAPName) (*AccessPolicy, error)
	GetByObjectAndID(objectType TAPObjectType, id int) (*AccessPolicy, error)
	Delete(ap *AccessPolicy) error
}

// RightsRosterDatabaseRecord represents a single database row
type RightsRosterDatabaseRecord struct {
	PolicyID        int         `db:"policy_id"`
	SubjectKind     SubjectKind `db:"subject_kind"`
	SubjectID       int         `db:"subject_id"`
	AccessRight     AccessRight `db:"access_right"`
	AccessExplained string      `db:"access_explained"`
}

// DefaultAccessPolicyStore is a default access policy store implementation
type DefaultAccessPolicyStore struct {
	db *dbr.Connection
}

// NewDefaultAccessPolicyStore returns an initialized default domain store
func NewDefaultAccessPolicyStore(db *dbr.Connection) (Store, error) {
	if db == nil {
		return nil, core.ErrNilDB
	}

	s := &DefaultAccessPolicyStore{db}

	return s, nil
}

//? BEGIN ->>>----------------------------------------------------------------
//? unexported utility functions

func (s *DefaultAccessPolicyStore) get(ctx context.Context, q string, args ...interface{}) (*AccessPolicy, error) {
	ap := new(AccessPolicy)

	sess := s.db.NewSession(nil)

	// fetching policy itself
	if err := sess.SelectBySql(q, args).LoadOneContext(ctx, &ap); err != nil {
		if err == dbr.ErrNotFound {
			return nil, core.ErrAccessPolicyNotFound
		}

		return nil, err
	}

	// fetching rights roster
	rrs := make([]*RightsRosterDatabaseRecord, 0)

	count, err := sess.
		SelectBySql("SELECT * FROM access_policy_rights_roster WHERE policy_id = ?", ap.ID).
		LoadContext(ctx, &rrs)

	switch true {
	case err != nil:
		return nil, err
	case count == 0:
		return nil, core.ErrEmptyRightsRoster
	}

	// initializing rights roster
	rr := &RightsRoster{
		Group: make(map[int]AccessRight),
		Role:  make(map[int]AccessRight),
		User:  make(map[int]AccessRight),
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

func (s *DefaultAccessPolicyStore) getMany(ctx context.Context, q string, args ...interface{}) ([]*AccessPolicy, error) {
	aps := make([]*AccessPolicy, 0)

	count, err := s.db.NewSession(nil).SelectBySql(q, args).LoadContext(ctx, &aps)
	switch true {
	case err != nil:
		return nil, err
	case count == 0:
		return nil, core.ErrAccessPolicyNotFound
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

// Create creating access policy
func (s *DefaultAccessPolicyStore) Create(ctx context.Context, ap *AccessPolicy) (retap *AccessPolicy, err error) {
	// basic validations
	if ap == nil {
		return nil, core.ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return nil, core.ErrNilRightsRoster
	}

	if ap.ID != 0 {
		return nil, core.ErrObjectIsNotNew
	}

	sess := s.db.NewSession(nil)

	tx, err := sess.Begin()
	if err != nil {
		return nil, fmt.Errorf("failed to begin a transaction: %s", err)
	}

	// panic handler
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %s", r)

			// rolling back the transaction
			if err := tx.Rollback(); err != nil { // oshi-
				err = fmt.Errorf("recovered from panic: %s [but rollback failed: %s]", r, err)
			}
		}
	}()

	//---------------------------------------------------------------------------
	// creating access policy
	//---------------------------------------------------------------------------
	// executing statement
	result, err := tx.InsertInto("access_policy").Record(&ap).ExecContext(ctx)

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

	// obtaining new ID
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
		PolicyID:        int(newID),
		SubjectKind:     SKEveryone,
		AccessRight:     ap.RightsRoster.Everyone,
		AccessExplained: ap.RightsRoster.Everyone.Explain(),
	})

	// for roles
	for id, right := range ap.RightsRoster.Role {
		rrs = append(rrs, RightsRosterDatabaseRecord{
			PolicyID:        int(newID),
			SubjectKind:     SKRoleGroup,
			SubjectID:       id,
			AccessRight:     right,
			AccessExplained: right.Explain(),
		})
	}

	// for groups
	for id, right := range ap.RightsRoster.Group {
		rrs = append(rrs, RightsRosterDatabaseRecord{
			PolicyID:        int(newID),
			SubjectKind:     SKGroup,
			SubjectID:       id,
			AccessRight:     right,
			AccessExplained: right.Explain(),
		})
	}

	// for users
	for id, right := range ap.RightsRoster.User {
		rrs = append(rrs, RightsRosterDatabaseRecord{
			PolicyID:        int(newID),
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
		_, err := tx.InsertInto("access_policy").Record(&r).ExecContext(ctx)
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

	// setting its new ID
	ap.ID = int(newID)

	return ap, nil
}

// UpdateAccessPolicy updates an existing access policy along with its rights roster
// NOTE: rights roster keeps track of its changes, thus, update will
// only affect changes mentioned by the respective RightsRoster object
func (s *DefaultAccessPolicyStore) Update(ctx context.Context, ap *AccessPolicy) (err error) {
	// basic validations
	if ap == nil {
		return core.ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return core.ErrNilRightsRoster
	}

	if ap.ID == 0 {
		return core.ErrObjectIsNew
	}

	sess := s.db.NewSession(nil)

	tx, err := sess.Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin a transaction")
	}

	// panic handler
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %s", r)

			// rolling back the transaction
			if err := tx.Rollback(); err != nil { // oshi-
				err = errors.Wrapf(err, "recovered from panic, but rollback failed: %s", r)
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
	_, err = tx.Update("access_policy").SetMap(updates).Where("id = ?", ap.ID).ExecContext(ctx)
	if err != nil {
		panic(err)
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
					panic(err)
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
					panic(err)
				}
			}
		}
	}

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err := tx.Commit(); err != nil {
		panic(err)
	}

	return nil
}

// GetByID retrieving a access policy by ID
func (s *DefaultAccessPolicyStore) GetByID(id int) (*AccessPolicy, error) {
	return s.get(context.TODO(), "SELECT * FROM access_policy WHERE id = ? LIMIT 1", id)
}

// GetByName retrieving a access policy by a key
func (s *DefaultAccessPolicyStore) GetByName(name []byte) (*AccessPolicy, error) {
	return s.get(context.TODO(), "SELECT * FROM access_policy WHERE `name` = ? LIMIT 1", name)
}

// GetByObjectTypeAndID retrieving a access policy by a kind and its respective id
func (s *DefaultAccessPolicyStore) GetByKindAndID(kind string, id int) (*AccessPolicy, error) {
	return s.get(context.TODO(), "SELECT * FROM access_policy WHERE object_kind = ? AND object_id = ? LIMIT 1", kind, id)
}

// Delete access policy
func (s *DefaultAccessPolicyStore) Delete(ap *AccessPolicy) (err error) {
	// basic validations
	if ap == nil {
		return core.ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return core.ErrNilRightsRoster
	}

	if ap.ID == 0 {
		return core.ErrObjectIsNew
	}

	sess := s.db.NewSession(nil)

	tx, err := sess.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin a transaction: %s", err)
	}

	// panic handler
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from panic: %s", r)

			// rolling back the transaction
			if err := tx.Rollback(); err != nil { // oshi-
				err = fmt.Errorf("recovered from panic, but rollback failed %s", r)
			}
		}
	}()

	//---------------------------------------------------------------------------
	// deleting policy along with it's respective rights roster
	//---------------------------------------------------------------------------
	// deleting access policy
	if _, err = tx.Exec("DELETE FROM accesspolicy WHERE id = ?", ap.ID); err != nil {
		panic(err)
	}

	// deleting rights roster
	if _, err = tx.Exec("DELETE FROM accesspolicy_rights_roster WHERE policy_id = ?", ap.ID); err != nil {
		panic(err)
	}

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err := tx.Commit(); err != nil {
		panic(err)
	}

	return nil
}
