package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

func (s *PostgreSQLStore) UpsertProfile(ctx context.Context, profile Profile) (_ Profile, err error) {
	if profile.UserID == uuid.Nil {
		return profile, ErrZeroUserID
	}

	q := `
	INSERT INTO user_profile(user_id, number, is_primary, created_at, confirmed_at, updated_at) 
	VALUES($1, $2, $3, $4, $5)
	ON CONFLICT ON CONSTRAINT user_profile_pk
	DO UPDATE 
		SET is_primary		= EXCLUDED.is_primary,
			is_confirmed	= EXCLUDED.is_confirmed,
			updated_at		= EXCLUDED.updated_at`

	cmd, err := s.db.ExecEx(
		ctx,
		q,
		nil,
	)

	switch err {
	case nil:
		if cmd.RowsAffected() == 0 {
			return profile, ErrNothingChanged
		}

		return profile, nil
	default:
		switch pgerr := err.(pgx.PgError); pgerr.Code {
		case "23505":
			return profile, ErrDuplicatePhoneNumber
		default:
			return profile, errors.Wrap(err, "failed to execute insert phone")
		}
	}
}

func (s *PostgreSQLStore) FetchProfileByUserID(ctx context.Context, userID uuid.UUID) (profile Profile, err error) {
	q := `
	SELECT user_id, firstname, middlename, lastname, language, created_at, updated_at, checksum
	FROM user_profile
		WHERE user_id = $1
	LIMIT 1`

	err = s.db.QueryRowEx(ctx, q, nil, userID).
		Scan(&profile.UserID,
			&profile.Firstname,
			&profile.Middlename,
			&profile.Lastname,
			&profile.Language,
			&profile.CreatedAt,
			&profile.UpdatedAt,
			&profile.Language,
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
