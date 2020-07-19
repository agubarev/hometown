package access

import (
	"context"

	"github.com/google/uuid"
)

// Store is a storage contract interface for the Policy objects
// TODO: keep rights separate and segregated by it's kind i.e. Public, Policy, Role, User etc.
type Store interface {
	CreatePolicy(ctx context.Context, p Policy, r *Roster) (Policy, *Roster, error)
	UpdatePolicy(ctx context.Context, p Policy, r *Roster) error
	FetchPolicyByID(ctx context.Context, id uuid.UUID) (Policy, error)
	FetchPolicyByKey(ctx context.Context, key Key) (p Policy, err error)
	FetchPolicyByObject(ctx context.Context, obj Object) (p Policy, err error)
	DeletePolicy(ctx context.Context, p Policy) error
	CreateRoster(ctx context.Context, policyID uuid.UUID, r *Roster) (err error)
	FetchRosterByPolicyID(ctx context.Context, pid uuid.UUID) (r *Roster, err error)
	UpdateRoster(ctx context.Context, pid uuid.UUID, r *Roster) (err error)
	DeleteRoster(ctx context.Context, pid uuid.UUID) (err error)
}

/*
// RosterDatabaseRecord represents a single database row
type RosterDatabaseRecord struct {
	PolicyID        uint32      `db:"policy_id"`
	ID       uint32      `db:"subject_id"`
	Kind     Kind `db:"subject_kind"`
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

func (s *DefaultMySQLStore) get(ctx context.Context, q string, args ...interface{}) (ap Policy, err error) {
	sess := s.db.NewSession(nil)

	// fetching policy itself
	if err = sess.SelectBySql(q, args...).LoadOneContext(ctx, &ap); err != nil {
		if err == dbr.ErrNotFound {
			return ap, ErrPolicyNotFound
		}

		return ap, err
	}

	return ap, nil
}

func (s *DefaultMySQLStore) getMany(ctx context.Context, q string, args ...interface{}) (aps []Policy, err error) {
	count, err := s.db.NewSession(nil).SelectBySql(q, args...).LoadContext(ctx, &aps)
	switch true {
	case err != nil:
		return nil, err
	case count == 0:
		return nil, ErrPolicyNotFound
	}

	return aps, nil
}

// breakdownRoster decomposes roster entries into usable database records
func (s *DefaultMySQLStore) breakdownRoster(policyID uint32, r *Roster) (records []RosterDatabaseRecord) {
	records = make([]RosterDatabaseRecord, len(r.Registry))

	// for everyone
	records = append(records, RosterDatabaseRecord{
		PolicyID:        policyID,
		Kind:     SKEveryone,
		AccessRight:     r.Everyone,
		AccessExplained: r.Everyone.String(),
	})

	// breakdown
	r.registryLock.RLock()
	for _, _r := range r.Registry {
		switch _r.Kind {
		case SKRoleGroup, SKGroup, SKUser:
			records = append(records, RosterDatabaseRecord{
				PolicyID:        policyID,
				Kind:     _r.Kind,
				ID:       _r.ID,
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

	return records
}

func (s *DefaultMySQLStore) buildRoster(records []RosterDatabaseRecord) (r *Roster) {
	r = NewRoster(len(records))

	// transforming database records into the roster object
	for _, _r := range records {
		switch _r.Kind {
		case SKEveryone:
			r.Everyone = _r.AccessRight
		case SKRoleGroup, SKGroup, SKUser:
			r.put(_r.Kind, _r.ID, _r.AccessRight)
		default:
			log.Printf(
				"unrecognized subject kind for access policy (subject_kind=%d, subject_id=%d, access_right=%d)",
				_r.Kind,
				_r.ID,
				_r.AccessRight,
			)
		}
	}

	return r
}

func (s *DefaultMySQLStore) applyRosterChanges(tx *dbr.Tx, policyID uint32, r *Roster) (err error) {
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

	return nil
}

// Upsert creating access policy
func (s *DefaultMySQLStore) CreatePolicy(ctx context.Context, ap Policy, r *Roster) (Policy, *Roster, error) {
	if ap.ID != 0 {
		return ap, r, ErrNonZeroID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return ap, r, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.RollbackUnlessCommitted()

	//---------------------------------------------------------------------------
	// creating access policy
	//---------------------------------------------------------------------------
	result, err := tx.InsertInto("access").
		Columns(guard.DBColumnsFrom(&ap)...).
		Record(&ap).
		ExecContext(ctx)

	if err != nil {
		return ap, r, errors.Wrap(err, "failed to insert access policy")
	}

	// obtaining newly generated ID
	id64, err := result.LastInsertId()
	if err != nil {
		return ap, r, err
	}

	ap.ID = uint32(id64)

	//---------------------------------------------------------------------------
	// creating access roster
	//---------------------------------------------------------------------------
	if r == nil {
		r = NewRoster(0)
	}

	for _, _r := range s.breakdownRoster(ap.ID, r) {
		_, err = tx.InsertInto("accesspolicy_roster").
			Columns(guard.DBColumnsFrom(&_r)...).
			Record(&_r).
			ExecContext(ctx)

		if err != nil {
			return ap, nil, err
		}
	}

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err := tx.Commit(); err != nil {
		return ap, r, errors.Wrap(err, "failed to commit transaction")
	}

	return ap, r, nil
}

// UpdatePolicy updates an existing access policy along with its rights rosters
// NOTE: rights rosters keeps track of its changes, thus, update will
// only affect changes mentioned by the respective Roster object
func (s *DefaultMySQLStore) UpdatePolicy(ctx context.Context, ap Policy, r *Roster) (err error) {
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
		"key":         ap.Key,
		"object_type": ap.ObjectName,
		"object_id":   ap.ObjectID,
		"flags":       ap.Flags,
	}

	// updating access policy
	_, err = tx.Update("access").SetMap(updates).Where("id = ?", ap.ID).ExecContext(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to update access policy")
	}

	// applying roster changes to the database
	if err = s.applyRosterChanges(tx, ap.ID, r); err != nil {
		return errors.Wrap(err, "failed to apply access policy roster changes during policy update")
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
func (s *DefaultMySQLStore) FetchPolicyByID(ctx context.Context, policyID uint32) (Policy, error) {
	return s.get(ctx, "SELECT * FROM access WHERE id = ? LIMIT 1", policyID)
}

// PolicyByKey retrieving a access policy by a key
func (s *DefaultMySQLStore) FetchPolicyByKey(ctx context.Context, key Key) (Policy, error) {
	return s.get(ctx, "SELECT * FROM access WHERE `key` = ? LIMIT 1", key)
}

// PolicyByObject retrieving a access policy by a kind and its respective id
func (s *DefaultMySQLStore) FetchPolicyByObject(ctx context.Context, id uint32, objectType ObjectName) (Policy, error) {
	return s.get(ctx, "SELECT * FROM access WHERE object_type = ? AND object_id = ? LIMIT 1", objectType, id)
}

// DeletePolicy deletes access policy and its corresponding roster
func (s *DefaultMySQLStore) DeletePolicy(ctx context.Context, ap Policy) (err error) {
	if ap.ID == 0 {
		return ErrZeroPolicyID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin database transaction")
	}
	defer tx.RollbackUnlessCommitted()

	// deleting access policy
	if _, err = tx.ExecContext(ctx, "DELETE FROM access WHERE id = ?", ap.ID); err != nil {
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

	// looping over rights rosters to be created
	// TODO: squash into a single insert statement
	for _, _r := range s.breakdownRoster(policyID, r) {
		_, err = tx.InsertInto("accesspolicy_roster").
			Columns(guard.DBColumnsFrom(&_r)...).
			Record(&_r).
			ExecContext(ctx)

		if err != nil {
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit database transaction")
	}

	return nil
}

// FetchRosterByPolicyID fetches access policy roster by policy ID
func (s *DefaultMySQLStore) FetchRosterByPolicyID(ctx context.Context, policyID uint32) (_ *Roster, err error) {
	records := make([]RosterDatabaseRecord, 0)

	count, err := s.db.NewSession(nil).
		SelectBySql("SELECT * FROM accesspolicy_roster WHERE policy_id = ?", policyID).
		LoadContext(ctx, &records)

	switch true {
	case err != nil:
		return nil, err
	case count == 0:
		return nil, ErrEmptyRoster
	}

	return s.buildRoster(records), nil
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

	// applying roster changes to the database
	if err = s.applyRosterChanges(tx, policyID, r); err != nil {
		return errors.Wrap(err, "failed to apply access policy roster changes during roster update")
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
*/
