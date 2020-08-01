package password

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

type PostgreSQLStore struct {
	db *pgx.Conn
}

func NewPostgreSQLStore(db *pgx.Conn) (Store, error) {
	// reserving error return for the future, just in case
	return &PostgreSQLStore{db}, nil
}

// Upsert stores password
// ObjectID must be equal to the user's ObjectID
func (s *PostgreSQLStore) Upsert(ctx context.Context, p Password) (err error) {
	if p.OwnerID == uuid.Nil {
		return ErrNilOwnerID
	}

	if len(p.Hash) == 0 {
		return ErrEmptyPassword
	}

	if err = p.Validate(); err != nil {
		return errors.Wrap(err, "password validation failed")
	}

	q := `
	INSERT INTO password(kind, owner_id, hash, is_change_required, created_at, updated_at, expire_at)
	VALUES($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT ON CONSTRAINT password_pk
	DO UPDATE
		SET hash				= EXCLUDED.hash,
			is_change_required 	= EXCLUDED.is_change_required,
			updated_at			= EXCLUDED.updated_at,
			expire_at			= EXCLUDED.expire_at`

	_, err = s.db.ExecEx(
		ctx,
		q,
		nil,
		p.Kind, p.OwnerID, p.Hash, p.IsChangeRequired, p.CreatedAt, p.UpdatedAt, p.ExpireAt,
	)

	if err != nil {
		return errors.Wrap(err, "failed to upsert password")
	}

	return nil
}

// Get retrieves a stored password
func (s *PostgreSQLStore) Get(ctx context.Context, kind OwnerKind, ownerID uuid.UUID) (p Password, err error) {
	q := `
	SELECT kind, owner_id, hash, is_change_required, created_at, updated_at, expire_at
	FROM password
	WHERE kind = $1 AND owner_id = $2
	LIMIT 1`

	err = s.db.QueryRowEx(ctx, q, nil, kind, ownerID).
		Scan(&p.Kind, &p.OwnerID, &p.Hash, &p.IsChangeRequired,
			&p.CreatedAt, &p.UpdatedAt, &p.ExpireAt)

	switch err {
	case nil:
		return p, nil
	case pgx.ErrNoRows:
		return p, ErrPasswordNotFound
	default:
		return p, errors.Wrap(err, "failed to scan password")
	}
}

// DeletePolicy deletes a stored password
func (s *PostgreSQLStore) Delete(ctx context.Context, kind OwnerKind, ownerID uuid.UUID) (err error) {
	q := `DELETE FROM password WHERE kind = $1 AND owner_id = $2`

	_, err = s.db.ExecEx(ctx, q, nil, kind, ownerID)
	if err != nil {
		return errors.Wrap(err, "failed to delete password")
	}

	return err
}
