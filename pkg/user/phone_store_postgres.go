package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

func (s *PostgreSQLStore) onePhone(ctx context.Context, q string, args ...interface{}) (phone Phone, err error) {
	err = s.db.QueryRowEx(ctx, q, nil, args...).
		Scan(&phone.UserID,
			&phone.Number,
			&phone.IsPrimary,
			&phone.CreatedAt,
			&phone.ConfirmedAt,
			&phone.UpdatedAt)

	switch err {
	case nil:
		return phone, nil
	case pgx.ErrNoRows:
		return phone, ErrPhoneNotFound
	default:
		return phone, errors.Wrap(err, "failed to scan phone")
	}
}

func (s *PostgreSQLStore) manyPhones(ctx context.Context, q string, args ...interface{}) (phones []Phone, err error) {
	phones = make([]Phone, 0)

	rows, err := s.db.QueryEx(ctx, q, nil, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch phones")
	}

	for rows.Next() {
		var phone Phone

		err := rows.Scan(
			&phone.UserID,
			&phone.Number,
			&phone.IsPrimary,
			&phone.CreatedAt,
			&phone.ConfirmedAt,
			&phone.UpdatedAt)

		if err != nil {
			return phones, errors.Wrap(err, "failed to scan phones")
		}

		phones = append(phones, phone)
	}

	return phones, nil
}

func (s *PostgreSQLStore) UpsertPhone(ctx context.Context, p Phone) (_ Phone, err error) {
	if p.UserID == uuid.Nil {
		return p, ErrZeroUserID
	}

	q := `
	INSERT INTO user_phone(user_id, number, is_primary, created_at, confirmed_at, updated_at) 
	VALUES($1, $2, $3, $4, $5, $6)
	ON CONFLICT ON CONSTRAINT user_phone_pk
	DO UPDATE 
		SET is_primary 		= EXCLUDED.is_primary,
			updated_at 		= EXCLUDED.updated_at,
			confirmed_at	= EXCLUDED.confirmed_at`

	_, err = s.db.ExecEx(
		ctx,
		q,
		nil,
		p.UserID,
		p.Number,
		p.IsPrimary,
		p.CreatedAt,
		p.ConfirmedAt,
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

func (s *PostgreSQLStore) FetchPrimaryPhoneByUserID(ctx context.Context, userID uuid.UUID) (e Phone, err error) {
	q := `
	SELECT user_id, number, is_primary, created_at, confirmed_at, updated_at
	FROM user_phone 
		WHERE user_id = $1 AND is_primary = 1 
	LIMIT 1`

	return s.onePhone(ctx, q, userID)
}

func (s *PostgreSQLStore) FetchPhonesByUserID(ctx context.Context, userID uuid.UUID) ([]Phone, error) {
	q := `
	SELECT user_id, number, is_primary, created_at, confirmed_at, updated_at
	FROM user_phone 
		WHERE user_id = $1`

	return s.manyPhones(ctx, q, userID)
}

func (s *PostgreSQLStore) FetchPhoneByNumber(ctx context.Context, number string) (e Phone, err error) {
	q := `
	SELECT user_id, number, is_primary, created_at, confirmed_at, updated_at
	FROM user_phone 
		WHERE number = $1
	LIMIT 1`

	return s.onePhone(ctx, q, number)
}

func (s *PostgreSQLStore) DeletePhoneByNumber(ctx context.Context, userID uuid.UUID, number string) (err error) {
	if userID == uuid.Nil {
		return ErrZeroUserID
	}

	cmd, err := s.db.ExecEx(
		ctx,
		`DELETE FROM user_phone WHERE user_id = $1 AND number = $2`,
		nil,
		userID, number,
	)

	if err != nil {
		return errors.Wrap(err, "failed to delete phone")
	}

	if cmd.RowsAffected() == 0 {
		return ErrNothingChanged
	}

	return nil
}
