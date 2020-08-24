package user

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
)

type PostgreSQLStore struct {
	db *pgx.Conn
}

func NewPostgreSQLStore(conn *pgx.Conn) (Store, error) {
	if conn == nil {
		return nil, ErrNilUserStore
	}

	s := &PostgreSQLStore{
		db: conn,
	}

	return s, nil
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

func (s *PostgreSQLStore) oneUserByWhere(ctx context.Context, where string, args ...interface{}) (u User, err error) {
	q := fmt.Sprintf(`
	SELECT
		id,	username, display_name, last_login_at, last_login_ip, last_login_failed_at, last_login_failed_ip,
		last_login_attempts, is_suspended, suspension_reason, suspension_expires_at, suspended_by_id, checksum,
		confirmed_at, created_at, created_by_id, updated_at, updated_by_id, deleted_at, deleted_by_id
	FROM "user"
	WHERE %s
	LIMIT 1`, where)

	err = s.db.QueryRowEx(ctx, q, nil, args...).
		Scan(&u.ID, &u.Username, &u.DisplayName, &u.LastLoginAt, &u.LastLoginIP,
			&u.LastLoginFailedIP, &u.LastLoginAttempts, &u.IsSuspended, &u.SuspensionReason,
			&u.SuspensionExpiresAt, &u.SuspendedByID, &u.Checksum, &u.ConfirmedAt,
			&u.CreatedAt, &u.CreatedByID, &u.UpdatedAt, &u.UpdatedByID, &u.DeletedAt, &u.DeletedByID)

	switch err {
	case nil:
		return u, nil
	case pgx.ErrNoRows:
		return u, ErrUserNotFound
	default:
		return u, errors.Wrap(err, "failed to scan user")
	}
}

func (s *PostgreSQLStore) manyUsersByWhere(ctx context.Context, where string, args ...interface{}) (us []User, err error) {
	us = make([]User, 0)

	q := fmt.Sprintf(`
	SELECT
		id,	username, display_name, last_login_at, last_login_ip, last_login_failed_at, last_login_failed_ip,
		last_login_attempts, is_suspended, suspension_reason, suspension_expires_at, suspended_by_id, checksum,
		confirmed_at, created_at, created_by_id, updated_at, updated_by_id, deleted_at, deleted_by_id
	FROM "user"
	WHERE %s
	LIMIT 1`, where)

	rows, err := s.db.QueryEx(ctx, q, nil, args...)
	if err != nil {
		return us, errors.Wrap(err, "failed to fetch relations")
	}
	defer rows.Close()

	for rows.Next() {
		var u User

		err = rows.Scan(&u.ID, &u.Username, &u.DisplayName, &u.LastLoginAt, &u.LastLoginIP,
			&u.LastLoginFailedIP, &u.LastLoginAttempts, &u.IsSuspended, &u.SuspensionReason,
			&u.SuspensionExpiresAt, &u.SuspendedByID, &u.Checksum, &u.ConfirmedAt,
			&u.CreatedAt, &u.CreatedByID, &u.UpdatedAt, &u.UpdatedByID, &u.DeletedAt, &u.DeletedByID)

		if err != nil {
			return nil, errors.Wrap(err, "failed to scan user")
		}

		us = append(us, u)
	}

	return us, nil
}

// CreateUser creates a new entry in the storage backend
func (s *PostgreSQLStore) UpsertUser(ctx context.Context, u User) (_ User, err error) {
	q := `
		INSERT INTO "user"(
			id,	username, display_name, last_login_at, last_login_ip, 
			last_login_failed_at, last_login_failed_ip,	last_login_attempts, 
			is_suspended, suspension_reason, suspension_expires_at, 
			suspended_by_id, checksum, confirmed_at, created_at, 
			created_by_id, updated_at, updated_by_id, deleted_at, deleted_by_id) 
		VALUES($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20)
		ON CONFLICT ON CONSTRAINT user_pk
		DO UPDATE
			SET display_name 			= EXCLUDED.display_name,
				last_login_at			= EXCLUDED.last_login_at,
				last_login_ip			= EXCLUDED.last_login_ip,
				last_login_failed_at	= EXCLUDED.last_login_failed_at,
				last_login_failed_ip	= EXCLUDED.last_login_failed_ip,
				last_login_attempts		= EXCLUDED.last_login_attempts,
				is_suspended			= EXCLUDED.is_suspended,
				suspension_reason		= EXCLUDED.suspension_reason,
				suspension_expires_at	= EXCLUDED.suspension_expires_at,
				suspended_by_id			= EXCLUDED.suspended_by_id,
				checksum				= EXCLUDED.checksum,
				confirmed_at			= EXCLUDED.confirmed_at,
				created_at				= EXCLUDED.created_at,
				created_by_id			= EXCLUDED.created_by_id,
				updated_at				= EXCLUDED.updated_at,
				updated_by_id			= EXCLUDED.updated_by_id,
				deleted_at				= EXCLUDED.deleted_at,
				deleted_by_id			= EXCLUDED.deleted_by_id`

	_, err = s.db.ExecEx(
		ctx,
		q,
		nil,
		u.ID, u.Username, u.DisplayName, u.LastLoginAt, u.LastLoginIP,
		u.LastLoginFailedAt, u.LastLoginFailedIP, u.LastLoginAttempts,
		u.IsSuspended, u.SuspensionReason, u.SuspensionExpiresAt, u.SuspendedByID,
		u.Checksum, u.ConfirmedAt, u.CreatedAt, u.CreatedByID, u.UpdatedAt,
		u.UpdatedByID, u.DeletedAt, u.DeletedByID,
	)

	if err != nil {
		return u, errors.Wrap(err, "failed to execute upsert user")
	}

	return u, nil
}

func (s *PostgreSQLStore) FetchUserByID(ctx context.Context, id uuid.UUID) (u User, err error) {
	return s.oneUserByWhere(ctx, "id = $1", id)
}

func (s *PostgreSQLStore) FetchUserByUsername(ctx context.Context, username string) (u User, err error) {
	return s.oneUserByWhere(ctx, "SELECT * FROM user WHERE username = ? LIMIT 1", username)
}

func (s *PostgreSQLStore) FetchUserByEmailAddr(ctx context.Context, addr string) (u User, err error) {
	return s.oneUserByWhere(ctx, "SELECT * FROM user u LEFT JOIN user_email e ON u.id=e.user_id WHERE e.addr = ? LIMIT 1", addr)
}

func (s *PostgreSQLStore) FetchUserByPhoneNumber(ctx context.Context, number string) (u User, err error) {
	return s.oneUserByWhere(ctx, "SELECT * FROM user u LEFT JOIN user_phone e ON u.id=e.user_id WHERE e.number = ? LIMIT 1", number)
}

func (s *PostgreSQLStore) DeleteUserByID(ctx context.Context, id uuid.UUID) (err error) {
	if id == uuid.Nil {
		return ErrZeroID
	}

	err = s.withTransaction(ctx, func(tx *pgx.Tx) (err error) {
		// deleting phones
		_, err = tx.ExecEx(ctx, `DELETE FROM "user_phone" WHERE user_id = $1`, nil, id)
		if err != nil {
			return errors.Wrap(err, "failed to delete user phones")
		}

		// deleting emails
		_, err = tx.ExecEx(ctx, `DELETE FROM "user_email" WHERE user_id = $1`, nil, id)
		if err != nil {
			return errors.Wrap(err, "failed to delete user emails")
		}

		// deleting profile
		_, err = tx.ExecEx(ctx, `DELETE FROM "user_profile" WHERE user_id = $1`, nil, id)
		if err != nil {
			return errors.Wrap(err, "failed to delete user profile")
		}

		// deleting the main user record
		cmd, err := tx.ExecEx(ctx, `DELETE FROM "user" WHERE id = $1`, nil, id)
		if err != nil {
			return errors.Wrap(err, "failed to delete user record")
		}

		if cmd.RowsAffected() == 0 {
			return ErrNothingChanged
		}

		return nil
	})

	if err != nil {
		return errors.Wrap(err, "failed to delete user by id")
	}

	return nil
}
