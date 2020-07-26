package token

import (
	"context"
	"time"

	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

type defaultTokenStore struct {
	db *pgx.Conn
}

// NewStore initializes and returns a new default token store
func NewStore(db *pgx.Conn) (Store, error) {
	if db == nil {
		return nil, ErrNilDatabase
	}

	return &defaultTokenStore{db}, nil
}

// UpdatePolicy puts token into a store
func (s *defaultTokenStore) Put(ctx context.Context, t Token) (err error) {
	if t.Hash[0] == 0 {
		return ErrEmptyTokenHash
	}

	q := `
	INSERT INTO token(
		kind, 
		hash, 
		checkin_total,
		checkin_remainder,
		created_at,
		expire_at) 
	VALUES($1, $2, $3, $4, $5, $6)
	ON CONFLICT ON CONSTRAINT token_pk
	DO UPDATE 
		SET checkin_total		= EXCLUDED.checkin_total,
			checkin_remainder	= EXCLUDED.checkin_remainder,
			expire_at			= EXCLUDED.expire_at`

	cmd, err := s.db.ExecEx(
		ctx,
		q,
		nil,
		t.Kind,
		t.Hash,
		t.CheckinTotal,
		t.CheckinRemainder,
		time.Unix(t.CreatedAt, 0),
		time.Unix(t.ExpireAt, 0),
	)

	if err != nil {
		return errors.Wrap(err, "failed to upsert token")
	}

	if cmd.RowsAffected() == 0 {
		return ErrNothingChanged
	}

	return nil
}

// Get retrieves token from a store
func (s *defaultTokenStore) Get(ctx context.Context, hash THash) (t Token, err error) {
	q := `
	SELECT 
		kind, 
		hash, 
		checkin_total,
		checkin_remainder,
		created_at,
		expire_at
	FROM token
	WHERE hash = $1
	LIMIT 1`

	err = s.db.QueryRowEx(ctx, q, nil, hash).
		Scan(
			&t.Kind,
			&t.Hash, &t.CheckinTotal,
			&t.CheckinRemainder,
			&t.CreatedAt,
			&t.ExpireAt,
		)

	switch err {
	case nil:
		return t, nil
	case pgx.ErrNoRows:
		return t, ErrTokenNotFound
	default:
		return t, errors.Wrap(err, "failed to scan token")
	}
}

// DeletePolicy deletes token from a store
func (s *defaultTokenStore) Delete(ctx context.Context, hash THash) error {
	cmd, err := s.db.ExecEx(ctx, `DELETE FROM token WHERE hash = $1`, nil, hash)
	if err != nil {
		return errors.Wrap(err, "failed to delete token")
	}

	if cmd.RowsAffected() == 0 {
		return ErrNothingChanged
	}

	return nil
}
