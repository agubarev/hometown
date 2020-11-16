package accesspolicy

/*
// RosterEntry represents a single database row
type RosterEntry struct {
	PolicyID        uint32      `db:"policy_id"`
	ActorID       uint32      `db:"subject_id"`
	ActorKind     ActorKind `db:"subject_kind"`
	Access     Right       `db:"accesspolicy"`
	AccessExplained string      `db:"access_explained"`
}

// DefaultMySQLStore is a default accesspolicy policy store implementation
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
func (s *DefaultMySQLStore) breakdownRoster(policyID uint32, r *Roster) (records []RosterEntry) {
	records = make([]RosterEntry, len(r.Registry))

	// for everyone
	records = append(records, RosterEntry{
		PolicyID:        policyID,
		ActorKind:     AKEveryone,
		Access:     r.Everyone,
		AccessExplained: r.Everyone.String(),
	})

	// breakdown
	r.registryLock.RLock()
	for _, _r := range r.Registry {
		switch _r.ActorKind {
		case AKRoleGroup, AKGroup, AKUser:
			records = append(records, RosterEntry{
				PolicyID:        policyID,
				ActorKind:     _r.ActorKind,
				ActorID:       _r.ActorID,
				Access:     _r.Rights,
				AccessExplained: _r.Rights.String(),
			})
		default:
			log.Printf(
				"unrecognized subject kind for accesspolicy policy (subject_kind=%d, subject_id=%d, access_right=%d)",
				_r.ActorKind,
				_r.ActorID,
				_r.Rights,
			)
		}
	}
	r.registryLock.RUnlock()

	return records
}

func (s *DefaultMySQLStore) buildRoster(records []RosterEntry) (r *Roster) {
	r = NewRoster(len(records))

	// transforming database records into the roster object
	for _, _r := range records {
		switch _r.ActorKind {
		case AKEveryone:
			r.Everyone = _r.Access
		case AKRoleGroup, AKGroup, AKUser:
			r.put(_r.ActorKind, _r.ActorID, _r.Access)
		default:
			log.Printf(
				"unrecognized subject kind for accesspolicy policy (subject_kind=%d, subject_id=%d, access_right=%d)",
				_r.ActorKind,
				_r.ActorID,
				_r.Access,
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
				"INSERT INTO accesspolicy_roster (policy_id, subject_kind, subject_id, accesspolicy, access_explained) VALUES (?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE accesspolicy = ?",
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

// Upsert creating accesspolicy policy
func (s *DefaultMySQLStore) CreatePolicy(ctx context.Context, ap Policy, r *Roster) (Policy, *Roster, error) {
	if ap.ActorID != 0 {
		return ap, r, ErrNonZeroID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return ap, r, errors.Wrap(err, "failed to begin transaction")
	}
	defer tx.RollbackUnlessCommitted()

	//---------------------------------------------------------------------------
	// creating accesspolicy policy
	//---------------------------------------------------------------------------
	result, err := tx.InsertInto("accesspolicy").
		Columns(guard.DBColumnsFrom(&ap)...).
		Record(&ap).
		ExecContext(ctx)

	if err != nil {
		return ap, r, errors.Wrap(err, "failed to insert accesspolicy policy")
	}

	// obtaining newly generated ActorID
	id64, err := result.LastInsertId()
	if err != nil {
		return ap, r, err
	}

	ap.ActorID = uint32(id64)

	//---------------------------------------------------------------------------
	// creating accesspolicy roster
	//---------------------------------------------------------------------------
	if r == nil {
		r = NewRoster(0)
	}

	for _, _r := range s.breakdownRoster(ap.ActorID, r) {
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

// UpdatePolicy updates an existing accesspolicy policy along with its rights rosters
// NOTE: rights rosters keeps track of its changes, thus, update will
// only affect changes mentioned by the respective Roster object
func (s *DefaultMySQLStore) UpdatePolicy(ctx context.Context, ap Policy, r *Roster) (err error) {
	if ap.ActorID == 0 {
		return ErrZeroPolicyID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin a transaction")
	}
	defer tx.RollbackUnlessCommitted()

	//---------------------------------------------------------------------------
	// updating accesspolicy policy and its rights rosters (only the changes)
	//---------------------------------------------------------------------------
	// listing fields to be updated
	updates := map[string]interface{}{
		"parent_id":   ap.ParentID,
		"owner_id":    ap.OwnerID,
		"key":         ap.TKey,
		"object_type": ap.TObjectName,
		"object_id":   ap.ObjectID,
		"flags":       ap.Flags,
	}

	// updating accesspolicy policy
	_, err = tx.Update("accesspolicy").SetMap(updates).Where("id = ?", ap.ActorID).ExecContext(ctx)
	if err != nil {
		return errors.Wrap(err, "failed to update accesspolicy policy")
	}

	// applying roster changes to the database
	if err = s.applyRosterChanges(tx, ap.ActorID, r); err != nil {
		return errors.Wrap(err, "failed to apply accesspolicy policy roster changes during policy update")
	}

	//---------------------------------------------------------------------------
	// commiting transaction
	//---------------------------------------------------------------------------
	if err = tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit database transaction")
	}

	return nil
}

// FetchPolicyByID fetches accesspolicy policy by ActorID
func (s *DefaultMySQLStore) FetchPolicyByID(ctx context.Context, policyID uint32) (Policy, error) {
	return s.get(ctx, "SELECT * FROM accesspolicy WHERE id = ? LIMIT 1", policyID)
}

// PolicyByKey retrieving a accesspolicy policy by a key
func (s *DefaultMySQLStore) FetchPolicyByKey(ctx context.Context, key TKey) (Policy, error) {
	return s.get(ctx, "SELECT * FROM accesspolicy WHERE `key` = ? LIMIT 1", key)
}

// PolicyByObject retrieving a accesspolicy policy by a kind and its respective id
func (s *DefaultMySQLStore) FetchPolicyByObject(ctx context.Context, id uint32, objectType TObjectName) (Policy, error) {
	return s.get(ctx, "SELECT * FROM accesspolicy WHERE object_type = ? AND object_id = ? LIMIT 1", objectType, id)
}

// DeletePolicy deletes accesspolicy policy and its corresponding roster
func (s *DefaultMySQLStore) DeletePolicy(ctx context.Context, ap Policy) (err error) {
	if ap.ActorID == 0 {
		return ErrZeroPolicyID
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return errors.Wrap(err, "failed to begin database transaction")
	}
	defer tx.RollbackUnlessCommitted()

	// deleting accesspolicy policy
	if _, err = tx.ExecContext(ctx, "DELETE FROM accesspolicy WHERE id = ?", ap.ActorID); err != nil {
		return errors.Wrapf(err, "failed to delete accesspolicy policy: policy_id=%d", ap.ActorID)
	}

	// deleting roster
	if _, err = tx.ExecContext(ctx, "DELETE FROM accesspolicy_roster WHERE policy_id = ?", ap.ActorID); err != nil {
		return errors.Wrapf(err, "failed to delete rights rosters: policy_id=%d", ap.ActorID)
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

// FetchRosterByPolicyID fetches accesspolicy policy roster by policy ActorID
func (s *DefaultMySQLStore) FetchRosterByPolicyID(ctx context.Context, policyID uint32) (_ *Roster, err error) {
	records := make([]RosterEntry, 0)

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
		return errors.Wrap(err, "failed to apply accesspolicy policy roster changes during roster update")
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
		return errors.Wrapf(err, "failed to delete accesspolicy policy roster: %d", policyID)
	}

	if err := tx.Commit(); err != nil {
		return errors.Wrap(err, "failed to commit database transaction")
	}

	return nil
}
*/
