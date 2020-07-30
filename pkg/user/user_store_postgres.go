package user

import (
	"context"
	"fmt"

	"github.com/agubarev/hometown/pkg/util/bytearray"
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
		INSERT INTO  accesspolicy(id, parent_id, owner_id, key, object_name, object_id, flags) 
		VALUES($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT ON CONSTRAINT accesspolicy_pk
		DO NOTHING`

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

	return u, nil
}

func (s *PostgreSQLStore) FetchUserByID(ctx context.Context, id uuid.UUID) (u User, err error) {
	return s.oneUserByWhere(ctx, "id = $1", id)
}

func (s *PostgreSQLStore) FetchUserByUsername(ctx context.Context, username bytearray.ByteString32) (u User, err error) {
	return s.oneUserByWhere(ctx, "SELECT * FROM user WHERE username = ? LIMIT 1", username)
}

func (s *PostgreSQLStore) FetchUserByEmailAddr(ctx context.Context, addr bytearray.ByteString256) (u User, err error) {
	return s.oneUserByWhere(ctx, "SELECT * FROM user u LEFT JOIN user_email e ON u.id=e.user_id WHERE e.addr = ? LIMIT 1", addr)
}

func (s *PostgreSQLStore) FetchUserByPhoneNumber(ctx context.Context, number bytearray.ByteString16) (u User, err error) {
	return s.oneUserByWhere(ctx, "SELECT * FROM user u LEFT JOIN user_phone e ON u.id=e.user_id WHERE e.number = ? LIMIT 1", number)
}

func (s *PostgreSQLStore) DeleteUserByID(ctx context.Context, id uuid.UUID) (err error) {
	if id == uuid.Nil {
		return ErrZeroID
	}

	panic("not implemented")

	return nil
}
