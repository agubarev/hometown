package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

func (s *PostgreSQLStore) oneEmail(ctx context.Context, q string, args ...interface{}) (email Email, err error) {
	err = s.db.QueryRowEx(ctx, q, nil, args...).
		Scan(&email.UserID,
			&email.Addr,
			&email.IsPrimary,
			&email.CreatedAt,
			&email.ConfirmedAt,
			&email.UpdatedAt)

	switch err {
	case nil:
		return email, nil
	case pgx.ErrNoRows:
		return email, ErrEmailNotFound
	default:
		return email, errors.Wrap(err, "failed to scan email")
	}
}

func (s *PostgreSQLStore) manyEmails(ctx context.Context, q string, args ...interface{}) (emails []Email, err error) {
	emails = make([]Email, 0)

	rows, err := s.db.QueryEx(ctx, q, nil, args...)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch emails")
	}

	for rows.Next() {
		var email Email

		err := rows.Scan(&email.UserID,
			&email.Addr,
			&email.IsPrimary,
			&email.CreatedAt,
			&email.ConfirmedAt,
			&email.UpdatedAt)

		if err != nil {
			return emails, errors.Wrap(err, "failed to scan emails")
		}

		emails = append(emails, email)
	}

	return emails, nil
}

// UpsertEmail creates a new entry in the storage backend
func (s *PostgreSQLStore) UpsertEmail(ctx context.Context, e Email) (_ Email, err error) {
	if e.UserID == uuid.Nil {
		return e, ErrZeroUserID
	}

	q := `
	INSERT INTO user_email(user_id, addr, is_primary, created_at, confirmed_at, updated_at) 
	VALUES($1, $2, $3, $4, $5)
	ON CONFLICT ON CONSTRAINT user_email_pk
	DO UPDATE 
		SET is_primary = EXCLUDED.is_primary,
			updated_at = EXCLUDED.updated_at`

	cmd, err := s.db.ExecEx(
		ctx,
		q,
		nil,
	)

	switch err {
	case nil:
		if cmd.RowsAffected() == 0 {
			return e, ErrNothingChanged
		}

		return e, nil
	default:
		switch pgerr := err.(pgx.PgError); pgerr.Code {
		case "23505":
			return e, ErrDuplicateEmailAddr
		default:
			return e, errors.Wrap(err, "failed to execute insert email")
		}
	}
}

func (s *PostgreSQLStore) FetchPrimaryEmailByUserID(ctx context.Context, userID uuid.UUID) (e Email, err error) {
	q := `
	SELECT user_id, addr, is_primary, created_at, confirmed_at, updated_at
	FROM user_email 
		WHERE user_id = $1 AND is_primary = 1 
	LIMIT 1`

	return s.oneEmail(ctx, q, userID)
}

func (s *PostgreSQLStore) FetchEmailsByUserID(ctx context.Context, userID uuid.UUID) ([]Email, error) {
	q := `
	SELECT user_id, addr, is_primary, created_at, confirmed_at, updated_at
	FROM user_email 
		WHERE user_id = $1`

	return s.manyEmails(ctx, q, userID)
}

func (s *PostgreSQLStore) FetchEmailByAddr(ctx context.Context, addr string) (e Email, err error) {
	q := `
	SELECT user_id, addr, is_primary, created_at, confirmed_at, updated_at
	FROM user_email 
		WHERE addr = $1
	LIMIT 1`

	return s.oneEmail(ctx, q, addr)
}

func (s *PostgreSQLStore) DeleteEmailByAddr(ctx context.Context, userID uuid.UUID, addr string) (err error) {
	if userID == uuid.Nil {
		return ErrZeroUserID
	}

	cmd, err := s.db.ExecEx(
		ctx,
		`DELETE FROM user_email WHERE user_id = $1 AND addr = $2`,
		nil,
		userID, addr,
	)

	if err != nil {
		return errors.Wrap(err, "failed to delete email")
	}

	if cmd.RowsAffected() == 0 {
		return ErrNothingChanged
	}

	return nil
}
