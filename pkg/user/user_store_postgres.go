package user

import (
	"context"

	"github.com/agubarev/hometown/pkg/util/guard"
	"github.com/gocraft/dbr/v2"
	"github.com/jackc/pgx"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
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

func (s *PostgreSQLStore) fetchUserByQuery(ctx context.Context, q string, args ...interface{}) (u User, err error) {
	err = s.db.NewSession(nil).
		SelectBySql(q, args).
		LoadOneContext(ctx, &u)

	if err != nil {
		if err == dbr.ErrNotFound {
			return u, ErrUserNotFound
		}

		return u, err
	}

	return u, nil
}

func (s *PostgreSQLStore) fetchUsersByQuery(ctx context.Context, q string, args ...interface{}) (us []User, err error) {
	us = make([]User, 0)

	_, err = s.db.NewSession(nil).
		SelectBySql(q, args).
		LoadContext(ctx, &us)

	if err != nil {
		if err == dbr.ErrNotFound {
			return us, nil
		}

		return nil, err
	}

	return us, nil
}

// CreateUser creates a new entry in the storage backend
func (s *PostgreSQLStore) CreateUser(ctx context.Context, u User) (_ User, err error) {
	// if object ActorID is not 0, then it's not considered as new
	if u.ID != 0 {
		return u, ErrNonZeroID
	}

	result, err := s.db.NewSession(nil).
		InsertInto("user").
		Columns(guard.DBColumnsFrom(u)...).
		Record(u).
		ExecContext(ctx)

	if err != nil {
		return u, err
	}

	newID, err := result.LastInsertId()
	if err != nil {
		return u, err
	}

	// setting newly generated ActorID
	u.ID = uint32(newID)

	return u, nil
}

// CreateUser creates a new entry in the storage backend
func (s *PostgreSQLStore) BulkCreateUser(ctx context.Context, us []User) (_ []User, err error) {
	uLen := len(us)

	// there must be something first
	if uLen == 0 {
		return nil, ErrNoInputData
	}

	tx, err := s.db.NewSession(nil).Begin()
	if err != nil {
		return nil, errors.Wrap(err, "failed to initialize database transaction")
	}
	defer tx.RollbackUnlessCommitted()

	//---------------------------------------------------------------------------
	// building the bulk statement
	//---------------------------------------------------------------------------
	stmt := tx.InsertInto("user").Columns(guard.DBColumnsFrom(us[0])...)

	// validating each user individually
	for i := range us {
		if us[i].ID != 0 {
			return nil, ErrNonZeroID
		}

		if err := us[i].Validate(); err != nil {
			return nil, err
		}

		// adding value to the batch
		stmt = stmt.Record(us[i])
	}

	// executing the batch
	result, err := stmt.ExecContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "bulk insert failed")
	}

	// returned ObjectID belongs to the first user created
	firstNewID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit database transaction")
	}

	// distributing new IDs in their sequential order
	for i := range us {
		us[i].ID = uint32(firstNewID)
		firstNewID++
	}

	return us, nil
}

func (s *PostgreSQLStore) FetchUserByID(ctx context.Context, id uint32) (u User, err error) {
	return s.fetchUserByQuery(ctx, "SELECT * FROM `user` WHERE id = ? LIMIT 1", id)
}

func (s *PostgreSQLStore) FetchUserByUsername(ctx context.Context, username string) (u User, err error) {
	return s.fetchUserByQuery(ctx, "SELECT * FROM `user` WHERE username = ? LIMIT 1", username)
}

func (s *PostgreSQLStore) FetchUserByEmailAddr(ctx context.Context, addr string) (u User, err error) {
	return s.fetchUserByQuery(ctx, "SELECT * FROM `user` u LEFT JOIN `user_email` e ON u.id=e.user_id WHERE e.addr = ? LIMIT 1", addr)
}

func (s *PostgreSQLStore) FetchUserByPhoneNumber(ctx context.Context, number string) (u User, err error) {
	return s.fetchUserByQuery(ctx, "SELECT * FROM `user` u LEFT JOIN `user_phone` e ON u.id=e.user_id WHERE e.number = ? LIMIT 1", number)
}

func (s *PostgreSQLStore) UpdateUser(ctx context.Context, u User, changelog diff.Changelog) (_ User, err error) {
	if len(changelog) == 0 {
		return u, ErrNothingChanged
	}

	changes, err := guard.ProcureDBChangesFromChangelog(u, changelog)
	if err != nil {
		return u, errors.Wrap(err, "failed to procure changes from a changelog")
	}

	//spew.Dump(changes)
	//spew.Dump(changelog)

	result, err := s.db.NewSession(nil).
		Update("user").
		Where("id = ?", u.ID).
		SetMap(changes).
		ExecContext(ctx)

	if err != nil {
		return u, err
	}

	// checking whether anything was updated at all
	// if no rows were affected then returning this as a non-critical error
	ra, err := result.RowsAffected()
	if ra == 0 {
		return u, ErrNothingChanged
	}

	return u, nil
}

func (s *PostgreSQLStore) DeleteUserByID(ctx context.Context, id uint32) (err error) {
	if id == 0 {
		return ErrZeroID
	}

	_, err = s.db.NewSession(nil).
		DeleteFrom("user").
		Where("id = ?", id).
		ExecContext(ctx)

	if err != nil {
		return errors.Wrapf(err, "failed to delete user: id=%d", id)
	}

	return nil
}

func (s *PostgreSQLStore) DeleteUsersByQuery(ctx context.Context, q string, args ...interface{}) (err error) {
	_, err = s.db.NewSession(nil).
		DeleteFrom("user").
		Where(q, args...).
		ExecContext(ctx)

	if err != nil {
		return errors.Wrap(err, "failed to delete users by query")
	}

	return nil
}
