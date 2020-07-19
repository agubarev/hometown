package access

import (
	"context"
	"log"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

type RosterDatabaseRecord struct {
	PolicyID        uuid.UUID `db:"policy_id"`
	ID              uuid.UUID `db:"actor_id"`
	Kind            ActorKind `db:"actor_kind"`
	AccessRight     Right     `db:"access"`
	AccessExplained string    `db:"access_explained"`
}

type PostgreSQLStore struct {
	db *pgx.Conn
}

func NewPostgreSQLStore(db *pgx.Conn) (Store, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}

	return &PostgreSQLStore{db}, nil
}

func (s *PostgreSQLStore) withTransaction(ctx context.Context, fn func(tx *pgx.Tx) error) (err error) {
	tx, err := s.db.BeginEx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "failed to begin transaction")
	}

	// deferring rollback unless there was a successful commit
	defer func(tx *pgx.Tx) {
		if tx.Status() != pgx.TxStatusCommitSuccess {
			// rolling back transaction if it hasn't been committed
			if tx.Status() != pgx.TxStatusCommitSuccess {
				if txerr := tx.RollbackEx(ctx); txerr != nil {
					err = errors.Wrapf(err, "failed to rollback transaction: %s", txerr)
				}
			}
		}
	}(tx)

	// applying function
	if err = fn(tx); err != nil {
		return errors.Wrap(err, "transaction failed")
	}

	// committing transaction
	if err = tx.CommitEx(ctx); err != nil {
		return errors.Wrap(err, "failed to commit transaction")
	}

	return nil
}

// breakdownRoster decomposes roster entries into usable database records
func (s *PostgreSQLStore) breakdownRoster(pid uuid.UUID, r *Roster) (records []RosterDatabaseRecord) {
	records = make([]RosterDatabaseRecord, len(r.Registry))

	// for everyone
	records = append(records, RosterDatabaseRecord{
		PolicyID:        pid,
		Kind:            SKEveryone,
		AccessRight:     r.Everyone,
		AccessExplained: r.Everyone.String(),
	})

	// breakdown
	r.registryLock.RLock()
	for _, _r := range r.Registry {
		switch _r.Key.Kind {
		case SKRoleGroup, SKGroup, SKUser:
			records = append(records, RosterDatabaseRecord{
				PolicyID:        pid,
				Kind:            _r.Key.Kind,
				ID:              _r.Key.ID,
				AccessRight:     _r.Rights,
				AccessExplained: _r.Rights.String(),
			})
		default:
			log.Printf(
				"unrecognized actor kind for access policy: actor(kind=%s, id=%s), access=(%s; %s)",
				_r.Key.Kind,
				_r.Key.ID,
				_r.Rights,
				_r.Rights.Translate(),
			)
		}
	}
	r.registryLock.RUnlock()

	return records
}

func (s *PostgreSQLStore) buildRoster(records []RosterDatabaseRecord) (r *Roster) {
	r = NewRoster(len(records))

	// transforming database records into the roster object
	for _, _r := range records {
		switch _r.Kind {
		case SKEveryone:
			r.Everyone = _r.AccessRight
		case SKRoleGroup, SKGroup, SKUser:
			r.put(NewActor(_r.Kind, _r.ID), _r.AccessRight)
		default:
			log.Printf(
				"unrecognized actor kind for access policy (actor_kind=%d, actor_id=%d, access_right=%d)",
				_r.Kind,
				_r.ID,
				_r.AccessRight,
			)
		}
	}

	return r
}

func (s *PostgreSQLStore) applyRosterChanges(tx *pgx.Tx, pid uuid.UUID, r *Roster) (err error) {
	// checking whether the rights rosters has any changes
	// TODO: optimize by squashing inserts and deletes into single queries
	for _, c := range r.changes {
		switch c.action {
		case RSet:
			//---------------------------------------------------------------------------
			// creating
			//---------------------------------------------------------------------------
			q := `
			INSERT INTO policy_roster(policy_id, actor_kind, actor_id, access, access_explained) 
			VALUES ($1, $2, $3, $4, $5) 
			ON CONFLICT ON CONSTRAINT policy_roster_policy_id_subject_kind_subject_id_uindex
			DO UPDATE access = $6
			`

			_, err = tx.Exec(
				q,
				pid,
				c.key.Kind,
				c.key.ID,
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
				"DELETE FROM policy_roster WHERE policy_id = ? AND actor_kind = ? AND actor_id = ?",
				pid,
				c.key.Kind,
				c.key.ID,
			)

			if err != nil {
				return errors.Wrap(err, "failed to delete rights rosters record")
			}
		}
	}

	return nil
}

func (s *PostgreSQLStore) onePolicy(ctx context.Context, q string, args ...interface{}) (p Policy, err error) {
	row := s.db.QueryRowEx(ctx, q, nil, args...)

	switch row.Scan(&p.ID, &p.ParentID, &p.OwnerID, &p.Key, &p.ObjectName, &p.ObjectID, &p.Flags) {
	case nil:
		return p, nil
	case pgx.ErrNoRows:
		return p, ErrPolicyNotFound
	default:
		return p, errors.Wrap(err, "failed to scan policy")
	}
}

func (s *PostgreSQLStore) manyPolicies(ctx context.Context, q string, args ...interface{}) (gs []Policy, err error) {
	gs = make([]Policy, 0)

	rows, err := s.db.QueryEx(ctx, q, nil, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch policies")
	}

	for rows.Next() {
		var p Policy

		if err = rows.Scan(&p.ID, &p.ParentID, &p.OwnerID, &p.Key, &p.ObjectName, &p.ObjectID, &p.Flags); err != nil {
			return gs, errors.Wrap(err, "failed to scan policies")
		}

		gs = append(gs, p)
	}

	return gs, nil
}

func (s *PostgreSQLStore) CreatePolicy(ctx context.Context, p Policy, r *Roster) (Policy, *Roster, error) {
	if p.ID == uuid.Nil {
		return p, r, ErrNilPolicyID
	}

	err := s.withTransaction(ctx, func(tx *pgx.Tx) error {
		//---------------------------------------------------------------------------
		// creating policy
		//---------------------------------------------------------------------------
		q := `
		INSERT INTO "policy"(id, parent_id, owner_id, key, object_name, object_id, flags) 
		VALUES($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (id)
		DO NOTHING
		`

		cmd, err := tx.ExecEx(
			ctx,
			q,
			nil,
			p.ID, p.ParentID, p.OwnerID, p.Key, p.ObjectName, p.ObjectID, p.Flags,
		)

		if err != nil {
			return errors.Wrap(err, "failed to execute insert policy")
		}

		if cmd.RowsAffected() == 0 {
			return ErrNothingChanged
		}

		//---------------------------------------------------------------------------
		// creating access roster
		//---------------------------------------------------------------------------
		if r == nil {
			r = NewRoster(0)
		}

		for _, _r := range s.breakdownRoster(p.ID, r) {
			q := `
			INSERT INTO "policy_roster"(policy_id, actor_kind, actor_id, access, access_explained) 
			VALUES($1, $2, $3, $4, $5)
			ON CONFLICT ON CONSTRAINT policy_roster_policy_id_subject_kind_subject_id_uindex
			DO NOTHING
			`

			_, err := tx.ExecEx(
				ctx,
				q,
				nil,
				_r.ID, _r.PolicyID, _r.Kind, _r.ID, _r.AccessRight, _r.AccessExplained,
			)

			if err != nil {
				return errors.Wrap(err, "failed to execute insert roster entry")
			}
		}

		return nil
	})

	return p, r, err
}

//-???-[ NOTE ]--------------------------------------------------------------
// ??? rights rosters keeps track of its changes, thus, update will
// ??? only affect changes mentioned by the respective Roster object
//-???-----------------------------------------------------------------------
func (s *PostgreSQLStore) UpdatePolicy(ctx context.Context, p Policy, r *Roster) (err error) {
	if p.ID == uuid.Nil {
		return ErrNilPolicyID
	}

	err = s.withTransaction(ctx, func(tx *pgx.Tx) error {
		//---------------------------------------------------------------------------
		// updating access policy and its rights rosters (only the changes)
		//---------------------------------------------------------------------------
		// listing fields to be updated
		updates := map[string]interface{}{
			"parent_id":   p.ParentID,
			"owner_id":    p.OwnerID,
			"key":         p.Key,
			"object_type": p.ObjectName,
			"object_id":   p.ObjectID,
			"flags":       p.Flags,
		}

		// applying roster changes to the database
		if err = s.applyRosterChanges(tx, p.ID, r); err != nil {
			return errors.Wrap(err, "failed to apply access policy roster changes during policy update")
		}

		// updating access policy
		_, err = tx.Update("access").SetMap(updates).Where("id = ?", p.ID).ExecContext(ctx)
		if err != nil {
			return errors.Wrap(err, "failed to update access policy")
		}

		// applying roster changes to the database
		if err = s.applyRosterChanges(tx, p.ID, r); err != nil {
			return errors.Wrap(err, "failed to apply access policy roster changes during policy update")
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "failed to update policy")
	}

	return nil
}

func (s *PostgreSQLStore) FetchPolicyByID(ctx context.Context, id uuid.UUID) (Policy, error) {
	panic("implement me")
}

func (s *PostgreSQLStore) FetchPolicyByKey(ctx context.Context, key Key) (p Policy, err error) {
	panic("implement me")
}

func (s *PostgreSQLStore) FetchPolicyByObject(ctx context.Context, obj Object) (p Policy, err error) {
	panic("implement me")
}

func (s *PostgreSQLStore) DeletePolicy(ctx context.Context, p Policy) error {
	panic("implement me")
}

func (s *PostgreSQLStore) CreateRoster(ctx context.Context, policyID uuid.UUID, r *Roster) (err error) {
	panic("implement me")
}

func (s *PostgreSQLStore) FetchRosterByPolicyID(ctx context.Context, pid uuid.UUID) (r *Roster, err error) {
	panic("implement me")
}

func (s *PostgreSQLStore) UpdateRoster(ctx context.Context, pid uuid.UUID, r *Roster) (err error) {
	panic("implement me")
}

func (s *PostgreSQLStore) DeleteRoster(ctx context.Context, pid uuid.UUID) (err error) {
	panic("implement me")
}
