package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

func (s *PostgreSQLStore) UpsertProfile(ctx context.Context, p Profile) (_ Profile, err error) {
	if p.UserID == uuid.Nil {
		return p, ErrZeroUserID
	}

	q := `
	INSERT INTO user_profile(user_id, firstname, middlename, lastname, checksum, created_at, updated_at) 
	VALUES($1, $2, $3, $4, $5, $6, $7)
	ON CONFLICT ON CONSTRAINT user_profile_pk
	DO UPDATE 
		SET firstname	= EXCLUDED.firstname,
			middlename	= EXCLUDED.middlename,
			lastname	= EXCLUDED.lastname,
			checksum	= EXCLUDED.checksum,
			updated_at	= EXCLUDED.updated_at`

	_, err = s.db.ExecEx(
		ctx,
		q,
		nil,
		p.UserID,
		p.Firstname,
		p.Middlename,
		p.Lastname,
		p.Checksum,
		p.CreatedAt,
		p.UpdatedAt,
	)

	switch err {
	case nil:
		return p, nil
	default:
		if pgerr, ok := err.(pgx.PgError); ok {
			switch pgerr.Code {
			case "23505":
				return p, ErrDuplicatePhoneNumber
			default:
				return p, errors.Wrap(err, "failed to execute insert phone")
			}
		}

		return p, err
	}
}

func (s *PostgreSQLStore) FetchProfileByUserID(ctx context.Context, userID uuid.UUID) (profile Profile, err error) {
	q := `
	SELECT user_id, firstname, middlename, lastname, created_at, updated_at, checksum
	FROM user_profile
		WHERE user_id = $1
	LIMIT 1`

	err = s.db.QueryRowEx(ctx, q, nil, userID).
		Scan(&profile.UserID,
			&profile.Firstname,
			&profile.Middlename,
			&profile.Lastname,
			&profile.CreatedAt,
			&profile.UpdatedAt,
		)

	switch err {
	case nil:
		return profile, nil
	case pgx.ErrNoRows:
		return profile, ErrProfileNotFound
	default:
		return profile, errors.Wrap(err, "failed to scan profile")
	}
}

func (s *PostgreSQLStore) DeleteProfileByUserID(ctx context.Context, userID uuid.UUID) (err error) {
	if userID == uuid.Nil {
		return ErrZeroUserID
	}

	cmd, err := s.db.ExecEx(
		ctx,
		`DELETE FROM user_profile WHERE user_id = $1`,
		nil,
		userID,
	)

	if err != nil {
		return errors.Wrap(err, "failed to delete profile")
	}

	if cmd.RowsAffected() == 0 {
		return ErrNothingChanged
	}

	return nil
}
