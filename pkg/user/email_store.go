package user

import (
	"context"
	"database/sql"

	"github.com/agubarev/hometown/pkg/util/guard"
	"github.com/go-sql-driver/mysql"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

func (s *MySQLStore) fetchEmailByQuery(ctx context.Context, q string, args ...interface{}) (e Email, err error) {
	err = s.connection.NewSession(nil).
		SelectBySql(q, args).
		LoadOneContext(ctx, &e)

	if err != nil {
		if err == sql.ErrNoRows {
			return e, ErrEmailNotFound
		}

		return e, err
	}

	return e, nil
}

func (s *MySQLStore) fetchEmailsByQuery(ctx context.Context, q string, args ...interface{}) (es []Email, err error) {
	es = make([]Email, 0)

	_, err = s.connection.NewSession(nil).
		SelectBySql(q, args).
		LoadContext(ctx, &es)

	if err != nil {
		if err == sql.ErrNoRows {
			return es, nil
		}

		return nil, err
	}

	return es, nil
}

// CreateEmail creates a new entry in the storage backend
func (s *MySQLStore) CreateEmail(ctx context.Context, e Email) (_ Email, err error) {
	// if ObjectID is not 0, then it's not considered as new
	if e.UserID == 0 {
		return e, ErrZeroUserID
	}

	_, err = s.connection.NewSession(nil).
		InsertInto("user_email").
		Columns(guard.DBColumnsFrom(e)...).
		Record(e).
		ExecContext(ctx)

	if err != nil {
		if myerr, ok := err.(*mysql.MySQLError); ok {
			switch myerr.Number {
			case 1062:
				return e, ErrDuplicateEmailAddr
			}
		}

		return e, err
	}

	return e, nil
}

// CreateEmail creates a new entry in the storage backend
func (s *MySQLStore) BulkCreateEmail(ctx context.Context, es []Email) (_ []Email, err error) {
	// there must be something first
	if len(es) == 0 {
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
	stmt := tx.InsertInto("user_email").Columns(guard.DBColumnsFrom(es[0])...)

	// validating each e individually
	for i := range es {
		if es[i].UserID != 0 {
			return nil, ErrZeroUserID
		}

		if err := es[i].Validate(); err != nil {
			return nil, err
		}

		// adding value to the batch
		stmt = stmt.Record(es[i])
	}

	// executing the batch
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "bulk insert failed")
	}

	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit database transaction")
	}

	return es, nil
}

func (s *MySQLStore) FetchPrimaryEmailByUserID(ctx context.Context, userID uint32) (e Email, err error) {
	return s.fetchEmailByQuery(ctx, "SELECT * FROM `user_email` WHERE user_id = ? AND is_primary = 1 LIMIT 1", userID)
}

func (s *MySQLStore) FetchEmailsByUserID(ctx context.Context, userID uint32) ([]Email, error) {
	return s.fetchEmailsByQuery(ctx, "SELECT * FROM `user_email` WHERE user_id = ?", userID)
}

func (s *MySQLStore) FetchEmailByAddr(ctx context.Context, addr string) (e Email, err error) {
	return s.fetchEmailByQuery(ctx, "SELECT * FROM `user_email` WHERE addr = ? LIMIT 1", addr)
}

func (s *MySQLStore) FetchEmailsByWhere(ctx context.Context, order string, limit, offset uint64, where string, args ...interface{}) (es []Email, err error) {
	_, err = s.connection.NewSession(nil).
		Select("*").
		Where(where, args...).
		Limit(limit).
		Offset(offset).
		OrderBy(order).
		LoadContext(ctx, &es)

	if err != nil {
		if err == sql.ErrNoRows {
			return es, nil
		}

		return nil, err
	}

	return es, nil
}

func (s *MySQLStore) UpdateEmail(ctx context.Context, e Email, changelog diff.Changelog) (_ Email, err error) {
	if len(changelog) == 0 {
		return e, ErrNothingChanged
	}

	changes, err := guard.ProcureDBChangesFromChangelog(e, changelog)
	if err != nil {
		return e, errors.Wrap(err, "failed to procure changes from a changelog")
	}

	result, err := s.connection.NewSession(nil).
		Update("user_email").
		Where("user_id = ? AND addr = ?", e.UserID, e.Addr).
		SetMap(changes).
		ExecContext(ctx)

	if err != nil {
		return e, err
	}

	// checking whether anything was updated at all
	// if no rows were affected then returning this as a non-critical error
	ra, err := result.RowsAffected()
	if ra == 0 {
		return e, ErrNothingChanged
	}

	return e, nil
}

func (s *MySQLStore) DeleteEmailByAddr(ctx context.Context, userID uint32, addr string) (err error) {
	if userID == 0 {
		return ErrZeroUserID
	}

	_, err = s.connection.NewSession(nil).
		DeleteFrom("user_email").
		Where("user_id = ? AND addr = ?", userID, addr).
		ExecContext(ctx)

	if err != nil {
		return errors.Wrapf(err, "failed to delete email: user_id=%d, addr=%s", userID, addr)
	}

	return nil
}

func (s *MySQLStore) DeleteEmailsByQuery(ctx context.Context, q string, args ...interface{}) (err error) {
	_, err = s.connection.NewSession(nil).
		DeleteFrom("user_email").
		Where(q, args...).
		ExecContext(ctx)

	if err != nil {
		return errors.Wrap(err, "failed to delete emails by query")
	}

	return nil
}
