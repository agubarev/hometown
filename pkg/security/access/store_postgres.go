package access

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

type RosterDatabaseRecord struct {
	PolicyID        uuid.UUID `db:"policy_id"`
	ID              uuid.UUID `db:"subject_id"`
	Kind            ActorKind `db:"kind"`
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
