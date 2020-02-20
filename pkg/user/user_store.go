package user

import (
	"context"
	"database/sql"

	"github.com/agubarev/hometown/pkg/util/guard"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

func (s *MySQLStore) fetchUserByQuery(ctx context.Context, q string, args ...interface{}) (u *User, err error) {
	err = s.connection.NewSession(nil).
		SelectBySql(q, args).
		LoadOneContext(ctx, &u)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return u, nil
}

func (s *MySQLStore) fetchUsersByQuery(ctx context.Context, q string, args ...interface{}) (us []*User, err error) {
	us = make([]*User, 0)

	_, err = s.connection.NewSession(nil).
		SelectBySql(q, args).
		LoadContext(ctx, &us)

	if err != nil {
		if err == sql.ErrNoRows {
			return us, nil
		}

		return nil, err
	}

	return us, nil
}

// CreateUser creates a new entry in the storage backend
func (s *MySQLStore) CreateUser(ctx context.Context, u *User) (_ *User, err error) {
	if u == nil {
		return nil, ErrNilUser
	}

	// if ID is not 0, then it's not considered as new
	if u.ID != 0 {
		return nil, ErrNonZeroID
	}

	result, err := s.connection.NewSession(nil).
		InsertInto("user").
		Columns(guard.DBColumnsFrom(u)...).
		Record(u).
		ExecContext(ctx)

	if err != nil {
		return nil, err
	}

	newID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// setting new ID
	u.ID = int(newID)

	return u, nil
}

// CreateUser creates a new entry in the storage backend
func (s *MySQLStore) BulkCreateUser(ctx context.Context, us []*User) (_ []*User, err error) {
	uLen := len(us)

	// there must be something first
	if uLen == 0 {
		return nil, ErrNoInputData
	}

	tx, err := s.connection.NewSession(nil).Begin()
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

	// returned ID belongs to the first u created
	firstNewID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit database transaction")
	}

	// distributing new IDs in their sequential order
	for i := range us {
		us[i].ID = int(firstNewID)
		firstNewID++
	}

	return us, nil
}

func (s *MySQLStore) FetchUserByID(ctx context.Context, id int) (u *User, err error) {
	return s.fetchUserByQuery(ctx, "SELECT * FROM `user` WHERE id = ? LIMIT 1", id)
}

func (s *MySQLStore) FetchUserByEmailAddr(ctx context.Context, addr TEmailAddr) (u *User, err error) {
	return s.fetchUserByQuery(ctx, "SELECT * FROM `user` u LEFT JOIN `user_email` e ON u.id=e.user_id WHERE e.email = ? LIMIT 1", addr)
}

func (s *MySQLStore) FetchUserByPhoneNumber(ctx context.Context, number TPhoneNumber) (u *User, err error) {
	return s.fetchUserByQuery(ctx, "SELECT * FROM `user` u LEFT JOIN `user_phone` e ON u.id=e.user_id WHERE e.number = ? LIMIT 1", number)
}

func (s *MySQLStore) UpdateUser(ctx context.Context, u *User, changelog diff.Changelog) (_ *User, err error) {
	if len(changelog) == 0 {
		return u, ErrNothingChanged
	}

	changes, err := guard.ProcureDBChangesFromChangelog(u, changelog)
	if err != nil {
		return nil, errors.Wrap(err, "failed to procure changes from a changelog")
	}

	result, err := s.connection.NewSession(nil).
		Update("user").
		Where("id = ?", u.ID).
		SetMap(changes).
		ExecContext(ctx)

	if err != nil {
		return nil, err
	}

	// checking whether anything was updated at all
	// if no rows were affected then returning this as a non-critical error
	ra, err := result.RowsAffected()
	if ra == 0 {
		return nil, ErrNothingChanged
	}

	return u, nil
}

func (s *MySQLStore) DeleteUserByID(ctx context.Context, id int) (err error) {
	if id == 0 {
		return ErrZeroID
	}

	_, err = s.connection.NewSession(nil).
		DeleteFrom("user").
		Where("id = ?", id).
		ExecContext(ctx)

	if err != nil {
		return errors.Wrapf(err, "failed to delete user: id=%d", id)
	}

	return nil
}

func (s *MySQLStore) DeleteUsersByQuery(ctx context.Context, q string, args ...interface{}) (err error) {
	_, err = s.connection.NewSession(nil).
		DeleteFrom("user").
		Where(q, args...).
		ExecContext(ctx)

	if err != nil {
		return errors.Wrap(err, "failed to delete users by query")
	}

	return nil
}
