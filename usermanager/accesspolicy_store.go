package usermanager

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// AccessPolicyStore is a storage contract interface for the AccessPolicy objects
// TODO: keep rights separate and segregated by it's kind i.e. Public, AccessPolicy, Role, User etc.
type AccessPolicyStore interface {
	Create(ap *AccessPolicy) (*AccessPolicy, error)
	Update(ap *AccessPolicy) error
	GetByID(id int64) (*AccessPolicy, error)
	GetByKey(key string) (*AccessPolicy, error)
	GetByKindAndID(kind string, id int64) (*AccessPolicy, error)
	Delete(ap *AccessPolicy) error
}

// RightsRosterDatabaseRecord represents a single database row
type RightsRosterDatabaseRecord struct {
	PolicyID    int64       `db:"policy_id"`
	SubjectKind string      `db:"subject_kind"`
	SubjectID   int64       `db:"subject_id"`
	AccessRight AccessRight `db:"access_right"`
	Human       string      `db:"human"`
}

// DefaultAccessPolicyStore is a default access policy store implementation
type DefaultAccessPolicyStore struct {
	db *sqlx.DB
}

// NewDefaultAccessPolicyStore returns an initialized default domain store
func NewDefaultAccessPolicyStore(db *sqlx.DB) (AccessPolicyStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	s := &DefaultAccessPolicyStore{db}

	return s, nil
}

//? BEGIN ->>>----------------------------------------------------------------
//? unexported utility functions

func (s *DefaultAccessPolicyStore) get(q string, args ...interface{}) (*AccessPolicy, error) {
	ap := new(AccessPolicy)

	// fetching policy itself
	err := s.db.Unsafe().Get(ap, q, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrAccessPolicyNotFound
		}

		return nil, err
	}

	// fetching rights roster
	rrs := make([]*RightsRosterDatabaseRecord, 0)
	err = s.db.Unsafe().Select(&rrs, "SELECT * FROM access_policy_rights_roster WHERE policy_id = ?", ap.ID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrEmptyRightsRoster
		}

		return nil, err
	}

	// initializing rights roster
	rr := &RightsRoster{
		Group: make(map[int64]AccessRight),
		Role:  make(map[int64]AccessRight),
		User:  make(map[int64]AccessRight),
	}

	// transforming database records into rights roster
	for _, r := range rrs {
		switch r.SubjectKind {
		case "everyone":
			rr.Everyone = r.AccessRight
		case "role":
			rr.Role[r.SubjectID] = r.AccessRight
		case "group":
			rr.Group[r.SubjectID] = r.AccessRight
		case "user":
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

func (s *DefaultAccessPolicyStore) getMany(q string, args ...interface{}) ([]*AccessPolicy, error) {
	aps := make([]*AccessPolicy, 0)

	err := s.db.Unsafe().Select(&aps, q, args...)
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
func (s *DefaultAccessPolicyStore) Create(ap *AccessPolicy) (retap *AccessPolicy, err error) {
	// basic validations
	if ap == nil {
		return nil, ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return nil, ErrNilRightsRoster
	}

	if ap.ID != 0 {
		return nil, ErrObjectIsNotNew
	}

	tx, err := s.db.Beginx()
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
	q := "INSERT INTO access_policy(parent_id, owner_id, `key`, object_kind, object_id, is_extended, is_inherited) VALUES(:parent_id, :owner_id, :key, :object_kind, :object_id, :is_extended, :is_inherited)"

	// executing statement
	result, err := tx.NamedExec(q, ap)

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
		PolicyID:    newID,
		SubjectKind: "everyone",
		AccessRight: ap.RightsRoster.Everyone,
		Human:       ap.RightsRoster.Everyone.Human(),
	})

	// for roles
	for id, right := range ap.RightsRoster.Role {
		rrs = append(rrs, RightsRosterDatabaseRecord{
			PolicyID:    newID,
			SubjectKind: "role",
			SubjectID:   id,
			AccessRight: right,
			Human:       right.Human(),
		})
	}

	// for groups
	for id, right := range ap.RightsRoster.Group {
		rrs = append(rrs, RightsRosterDatabaseRecord{
			PolicyID:    newID,
			SubjectKind: "group",
			SubjectID:   id,
			AccessRight: right,
			Human:       right.Human(),
		})
	}

	// for users
	for id, right := range ap.RightsRoster.User {
		rrs = append(rrs, RightsRosterDatabaseRecord{
			PolicyID:    newID,
			SubjectKind: "user",
			SubjectID:   id,
			AccessRight: right,
			Human:       right.Human(),
		})
	}

	//---------------------------------------------------------------------------
	// creating roster records
	//---------------------------------------------------------------------------
	q = `
	INSERT INTO access_policy_rights_roster (policy_id, subject_kind, subject_id, access_right, human) 
	VALUES (:policy_id, :subject_kind, :subject_id, :access_right, :human)
	`

	// looping over rights roster to be created
	for _, r := range rrs {
		_, err := tx.NamedExec(q, r)
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
	ap.ID = newID

	return ap, nil
}

// Update updates an existing access policy along with its rights roster
// NOTE: rights roster keeps track of its changes, thus, update will
// only affect changes mentioned by the respective RightsRoster object
func (s *DefaultAccessPolicyStore) Update(ap *AccessPolicy) (err error) {
	// basic validations
	if ap == nil {
		return ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return ErrNilRightsRoster
	}

	if ap.ID == 0 {
		return ErrObjectIsNew
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin a transaction: %s", err)
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
	// updating access policy and its rights roster (only the changes)
	//---------------------------------------------------------------------------
	q := `
	UPDATE access_policy
	SET parent_id = :parent_id,
		owner_id = :owner_id,
		` + "`key`" + `= :key,
		object_kind = :object_kind,
		object_id = :object_id,
		is_extended = :is_extended,
		is_inherited = :is_inherited
	WHERE id = :id
	`

	// executing statement
	_, err = tx.NamedExec(q, ap)
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
					"DELETE FROM access_policy_rights_roster WHERE policy_id = ? AND object_kind = ? AND object_id = ?",
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
					"INSERT INTO access_policy_rights_roster (policy_id, subject_kind, subject_id, access_right, human) VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE access_right = ?",
					ap.ID,
					c.subjectKind,
					c.subjectID,
					c.accessRight,
					c.accessRight.Human(),
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
func (s *DefaultAccessPolicyStore) GetByID(id int64) (*AccessPolicy, error) {
	return s.get("SELECT * FROM access_policy WHERE id = ? LIMIT 1", id)
}

// GetByKey retrieving a access policy by a key
func (s *DefaultAccessPolicyStore) GetByKey(key string) (*AccessPolicy, error) {
	return s.get("SELECT * FROM access_policy WHERE `key` = ? LIMIT 1", key)
}

// GetByKindAndID retrieving a access policy by a kind and its respective id
func (s *DefaultAccessPolicyStore) GetByKindAndID(kind string, id int64) (*AccessPolicy, error) {
	return s.get("SELECT * FROM access_policy WHERE object_kind = ? AND object_id = ? LIMIT 1", kind, id)
}

// Delete access policy
func (s *DefaultAccessPolicyStore) Delete(ap *AccessPolicy) (err error) {
	// basic validations
	if ap == nil {
		return ErrNilAccessPolicy
	}

	if ap.RightsRoster == nil {
		return ErrNilRightsRoster
	}

	if ap.ID == 0 {
		return ErrObjectIsNew
	}

	tx, err := s.db.Beginx()
	if err != nil {
		return fmt.Errorf("failed to begin a transaction: %s", err)
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
	// deleting policy along with it's respective rights roster
	//---------------------------------------------------------------------------
	// deleting access policy
	_, err = tx.Exec("DELETE FROM access_policy WHERE id = ?", ap.ID)
	if err != nil {
		panic(err)
	}

	// deleting rights roster
	_, err = tx.Exec("DELETE FROM access_policy_rights_roster WHERE policy_id = ?", ap.ID)
	if err != nil {
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
