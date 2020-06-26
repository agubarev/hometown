package user

import (
	"context"
	"log"

	"github.com/agubarev/hometown/pkg/util/guard"
	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
)

// Store is a storage contract interface for the AccessPolicy objects
// TODO: keep rights separate and segregated by it's kind i.e. Public, AccessPolicy, Role, User etc.
type Store interface {
	CreatePolicy(ctx context.Context, ap AccessPolicy) (AccessPolicy, error)
	UpdatePolicy(ctx context.Context, ap AccessPolicy) error
	FetchPolicyByID(ctx context.Context, id uint32) (AccessPolicy, error)
	FetchPolicyByName(ctx context.Context, name TKey) (ap AccessPolicy, err error)
	FetchPolicyByObjectTypeAndID(ctx context.Context, id uint32, objectType TObjectType) (ap AccessPolicy, err error)
	DeletePolicy(ctx context.Context, ap AccessPolicy) error
	CreateRoster(ctx context.Context, policyID uint32, r *Roster) (err error)
	FetchRosterByID(ctx context.Context, policyID uint32) (r *Roster, err error)
	UpdateRoster(ctx context.Context, policyID uint32, r *Roster) (err error)
	DeleteRoster(ctx context.Context, policyID uint32) (err error)
}

// RosterDatabaseRecord represents a single database row
type RosterDatabaseRecord struct {
	PolicyID        uint32      `db:"policy_id"`
	SubjectKind     SubjectKind `db:"subject_kind"`
	SubjectID       uint32      `db:"subject_id"`
	AccessRight     Right       `db:"access"`
	AccessExplained string      `db:"access_explained"`
}

// DefaultMySQLStore is a default access policy store implementation
type DefaultMySQLStore struct {
	db *dbr.Connection
}

// NewDefaultMySQLStore returns an initialized default domain store
func NewDefaultMySQLStore(db *dbr.Connection) (Store, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}

	s := &DefaultMySQLStore{db}

	return s, nil
}

func (s *DefaultMySQLStore) get(ctx context.Context, q string, args ...interface{}) (ap AccessPolicy, err error) {
	sess := s.db.NewSession(nil)

	// fetching policy itself
	if err := sess.SelectBySql(q, args...).LoadOneContext(ctx, &ap); err != nil {
		if err == dbr.ErrNotFound {
			return ap, ErrAccessPolicyNotFound
		}

		return ap, err
	}

	return ap, nil
}

func (s *DefaultMySQLStore) getMany(ctx context.Context, q string, args ...interface{}) (aps []AccessPolicy, err error) {
	count, err := s.db.NewSession(nil).SelectBySql(q, args...).LoadContext(ctx, &aps)
	switch true {
	case err != nil:
		return nil, err
	case count == 0:
		return nil, ErrAccessPolicyNotFound
	}

	return aps, nil
}

// Upsert creating access policy
func (s *DefaultMySQLStore) CreatePolicy(ctx context.Context, ap AccessPolicy) (_ AccessPolicy, err error) {
	if ap.ID != 0 {
		return ap, ErrNonZeroID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return ap, errors.Wrap(err, "failed to begin transaction")
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

	if err != nil {
		return ap, errors.Wrap(err, "failed to insert access policy")
	}

	// obtaining new ObjectID
	id64, err := result.LastInsertId()
	if err != nil {
		return ap, err
	}

	ap.ID = uint32(id64)

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err := tx.Commit(); err != nil {
		return ap, errors.Wrap(err, "failed to commit database transaction")
	}

	return ap, nil
}

// UpdatePolicy updates an existing access policy along with its rights rosters
// NOTE: rights rosters keeps track of its changes, thus, update will
// only affect changes mentioned by the respective Roster object
func (s *DefaultMySQLStore) UpdatePolicy(ctx context.Context, ap AccessPolicy) (err error) {
	if ap.ID == 0 {
		return ErrZeroPolicyID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin a transaction")
	}
	defer tx.RollbackUnlessCommitted()

	//---------------------------------------------------------------------------
	// updating access policy and its rights rosters (only the changes)
	//---------------------------------------------------------------------------
	// listing fields to be updated
	updates := map[string]interface{}{
		"parent_id":   ap.ParentID,
		"owner_id":    ap.OwnerID,
		"name":        ap.Key,
		"object_type": ap.ObjectType,
		"object_id":   ap.ObjectID,
		"flags":       ap.Flags,
	}

	// executing statement
	_, err = tx.Update("accesspolicy").SetMap(updates).Where("id = ?", ap.ID).ExecContext(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to update access policy")
	}

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit database transaction")
	}

	return nil
}

// FetchPolicyByID fetches access policy by ID
func (s *DefaultMySQLStore) FetchPolicyByID(ctx context.Context, policyID uint32) (AccessPolicy, error) {
	return s.get(ctx, "SELECT * FROM accesspolicy WHERE id = ? LIMIT 1", policyID)
}

// PolicyByName retrieving a access policy by a key
func (s *DefaultMySQLStore) FetchPolicyByName(ctx context.Context, name TKey) (AccessPolicy, error) {
	return s.get(ctx, "SELECT * FROM accesspolicy WHERE `name` = ? LIMIT 1", name)
}

// PolicyByObjectTypeAndID retrieving a access policy by a kind and its respective id
func (s *DefaultMySQLStore) FetchPolicyByObjectTypeAndID(ctx context.Context, id uint32, objectType TObjectType) (AccessPolicy, error) {
	return s.get(ctx, "SELECT * FROM accesspolicy WHERE object_type = ? AND object_id = ? LIMIT 1", objectType, id)
}

// DeletePolicy deletes access policy and its corresponding roster
func (s *DefaultMySQLStore) DeletePolicy(ctx context.Context, ap AccessPolicy) (err error) {
	if ap.ID == 0 {
		return ErrZeroPolicyID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin database transaction")
	}
	defer tx.RollbackUnlessCommitted()

	// deleting access policy
	if _, err = tx.ExecContext(ctx, "DELETE FROM accesspolicy WHERE id = ?", ap.ID); err != nil {
		return errors.Wrapf(err, "failed to delete access policy: policy_id=%d", ap.ID)
	}

	// deleting roster
	if _, err = tx.ExecContext(ctx, "DELETE FROM accesspolicy_roster WHERE policy_id = ?", ap.ID); err != nil {
		return errors.Wrapf(err, "failed to delete rights rosters: policy_id=%d", ap.ID)
	}

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit database transaction")
	}

	return nil
}

func (s *DefaultMySQLStore) CreateRoster(ctx context.Context, policyID uint32, r *Roster) (err error) {
	if policyID != 0 {
		return ErrZeroPolicyID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.RollbackUnlessCommitted()

	//---------------------------------------------------------------------------
	// preparing new rosters records
	//---------------------------------------------------------------------------
	rrs := make([]RosterDatabaseRecord, 0)

	// for everyone
	rrs = append(rrs, RosterDatabaseRecord{
		PolicyID:        policyID,
		SubjectKind:     SKEveryone,
		AccessRight:     r.Everyone,
		AccessExplained: r.Everyone.String(),
	})

	r.registryLock.RLock()
	for _, _r := range r.Registry {
		switch _r.Kind {
		case SKRoleGroup, SKGroup, SKUser:
			rrs = append(rrs, RosterDatabaseRecord{
				PolicyID:        policyID,
				SubjectKind:     _r.Kind,
				SubjectID:       _r.ID,
				AccessRight:     _r.Rights,
				AccessExplained: _r.Rights.String(),
			})
		default:
			log.Printf(
				"unrecognized subject kind for access policy (subject_kind=%d, subject_id=%d, access_right=%d)",
				_r.Kind,
				_r.ID,
				_r.Rights,
			)
		}
	}
	r.registryLock.RUnlock()

	//---------------------------------------------------------------------------
	// creating rosters records
	//---------------------------------------------------------------------------
	// looping over rights rosters to be created

	// TODO: squash into a single insert statement
	for _, _r := range rrs {
		_, err = tx.InsertInto("accesspolicy_roster").
			Columns(guard.DBColumnsFrom(&_r)...).
			Record(&_r).
			ExecContext(ctx)

		if err != nil {
			return err
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

// FetchRosterByID fetches access policy roster by policy ID
func (s *DefaultMySQLStore) FetchRosterByID(ctx context.Context, policyID uint32) (r *Roster, err error) {
	sess := s.db.NewSession(nil)

	rrs := make([]RosterDatabaseRecord, 0)

	count, err := sess.
		SelectBySql("SELECT * FROM accesspolicy_roster WHERE policy_id = ?", policyID).
		LoadContext(ctx, &rrs)

	switch true {
	case err != nil:
		return r, err
	case count == 0:
		return r, ErrEmptyRoster
	}

	// initializing roster
	r = NewRoster()

	// transforming database records into the roster object
	for _, _r := range rrs {
		switch _r.SubjectKind {
		case SKEveryone:
			r.Everyone = _r.AccessRight
		case SKRoleGroup, SKGroup, SKUser:
			r.register(_r.SubjectKind, _r.SubjectID, _r.AccessRight)
		default:
			log.Printf(
				"unrecognized subject kind for access policy (subject_kind=%d, subject_id=%d, access_right=%d)",
				_r.SubjectKind,
				_r.SubjectID,
				_r.AccessRight,
			)
		}
	}

	return r, nil
}

func (s *DefaultMySQLStore) UpdateRoster(ctx context.Context, policyID uint32, r *Roster) (err error) {
	if policyID == 0 {
		return ErrZeroPolicyID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin a transaction")
	}
	defer tx.RollbackUnlessCommitted()

	// checking whether the rights rosters has any changes
	// TODO: optimize by squashing inserts and deletes into single queries
	for _, c := range r.changes {
		switch c.action {
		case RSet:
			//---------------------------------------------------------------------------
			// creating
			//---------------------------------------------------------------------------
			_, err = tx.Exec(
				"INSERT INTO accesspolicy_roster (policy_id, subject_kind, subject_id, access, access_explained) VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE access = ?",
				policyID,
				c.subjectKind,
				c.subjectID,
				c.accessRight,
				c.accessRight.String(),
				c.accessRight,
			)

			if err != nil {
				return errors.Wrap(err, "failed to upsert rights rosters record")
			}
		case RUnset:
			//---------------------------------------------------------------------------
			// deleting
			//---------------------------------------------------------------------------
			_, err = tx.Exec(
				"DELETE FROM accesspolicy_roster WHERE policy_id = ? AND subject_kind = ? AND subject_id = ?",
				policyID,
				c.subjectKind,
				c.subjectID,
			)

			if err != nil {
				return errors.Wrap(err, "failed to delete rights rosters record")
			}
		}
	}

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit database transaction")
	}

	return nil
}

func (s *DefaultMySQLStore) DeleteRoster(ctx context.Context, policyID uint32) (err error) {
	if policyID == 0 {
		return ErrZeroPolicyID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin database transaction")
	}
	defer tx.RollbackUnlessCommitted()

	// deleting the roster
	if _, err = tx.ExecContext(ctx, "DELETE FROM accesspolicy_roster WHERE policy_id = ?", policyID); err != nil {
		return errors.Wrapf(err, "failed to delete access policy roster: %d", policyID)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit database transaction")
	}

	return nil
}
