package user

import (
	"context"
	"database/sql"

	"github.com/agubarev/hometown/pkg/util/guard"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

func (s *MySQLStore) fetchPhoneByQuery(ctx context.Context, q string, args ...interface{}) (p Phone, err error) {
	err = s.connection.NewSession(nil).
		SelectBySql(q, args).
		LoadOneContext(ctx, &p)

	if err != nil {
		if err == sql.ErrNoRows {
			return p, ErrPhoneNotFound
		}

		return p, err
	}

	return p, nil
}

func (s *MySQLStore) fetchPhonesByQuery(ctx context.Context, q string, args ...interface{}) (ps []Phone, err error) {
	ps = make([]Phone, 0)

	_, err = s.connection.NewSession(nil).
		SelectBySql(q, args).
		LoadContext(ctx, &ps)

	if err != nil {
		if err == sql.ErrNoRows {
			return ps, nil
		}

		return nil, err
	}

	return ps, nil
}

// CreatePhone creates a new entry in the storage backend
func (s *MySQLStore) CreatePhone(ctx context.Context, p Phone) (_ Phone, err error) {
	// if ObjectID is not 0, then it's not considered as new
	if p.UserID == 0 {
		return p, ErrZeroUserID
	}

	_, err = s.connection.NewSession(nil).
		InsertInto("user_phone").
		Columns(guard.DBColumnsFrom(p)...).
		Record(p).
		ExecContext(ctx)

	if err != nil {
		return p, err
	}

	return p, nil
}

// CreatePhone creates a new entry in the storage backend
func (s *MySQLStore) BulkCreatePhone(ctx context.Context, ps []Phone) (_ []Phone, err error) {
	// there must be something first
	if len(ps) == 0 {
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
	stmt := tx.InsertInto("user_phone").Columns(guard.DBColumnsFrom(ps[0])...)

	// validating each p individually
	for i := range ps {
		if ps[i].UserID != 0 {
			return nil, ErrZeroUserID
		}

		if err := ps[i].Validate(); err != nil {
			return nil, err
		}

		// adding value to the batch
		stmt = stmt.Record(ps[i])
	}

	// executing the batch
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "bulk insert failed")
	}

	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit database transaction")
	}

	return ps, nil
}

func (s *MySQLStore) FetchPrimaryPhoneByUserID(ctx context.Context, userID int64) (p Phone, err error) {
	return s.fetchPhoneByQuery(ctx, "SELECT * FROM `user_phone` WHERE user_id = ? AND is_primary = 1 LIMIT 1", userID)
}

func (s *MySQLStore) FetchPhonesByUserID(ctx context.Context, userID int64) ([]Phone, error) {
	return s.fetchPhonesByQuery(ctx, "SELECT * FROM `user_phone` WHERE user_id = ?", userID)
}

func (s *MySQLStore) FetchPhoneByNumber(ctx context.Context, number string) (p Phone, err error) {
	return s.fetchPhoneByQuery(ctx, "SELECT * FROM `user_phone` WHERE  number = ? LIMIT 1", number)
}

func (s *MySQLStore) FetchPhonesByWhere(ctx context.Context, order string, limit, offset uint64, where string, args ...interface{}) (ps []Phone, err error) {
	_, err = s.connection.NewSession(nil).
		Select("*").
		Where(where, args...).
		Limit(limit).
		Offset(offset).
		OrderBy(order).
		LoadContext(ctx, &ps)

	if err != nil {
		if err == sql.ErrNoRows {
			return ps, nil
		}

		return nil, err
	}

	return ps, nil
}

func (s *MySQLStore) UpdatePhone(ctx context.Context, p Phone, changelog diff.Changelog) (_ Phone, err error) {
	if len(changelog) == 0 {
		return p, ErrNothingChanged
	}

	changes, err := guard.ProcureDBChangesFromChangelog(p, changelog)
	if err != nil {
		return p, errors.Wrap(err, "failed to procure changes from a changelog")
	}

	result, err := s.connection.NewSession(nil).
		Update("user_phone").
		Where("user_id = ? AND number = ?", p.UserID, p.Number).
		SetMap(changes).
		ExecContext(ctx)

	if err != nil {
		return p, err
	}

	// checking whether anything was updated at all
	// if no rows were affected then returning this as a non-critical error
	ra, err := result.RowsAffected()
	if ra == 0 {
		return p, ErrNothingChanged
	}

	return p, nil
}

func (s *MySQLStore) DeletePhoneByNumber(ctx context.Context, userID int64, number string) (err error) {
	if userID == 0 {
		return ErrZeroUserID
	}

	_, err = s.connection.NewSession(nil).
		DeleteFrom("user_phone").
		Where("user_id = ? AND number = ?", userID, number).
		ExecContext(ctx)

	if err != nil {
		return errors.Wrapf(err, "failed to delete phone: user_id=%d, number=%s", userID, number)
	}

	return nil
}

func (s *MySQLStore) DeletePhonesByQuery(ctx context.Context, q string, args ...interface{}) (err error) {
	_, err = s.connection.NewSession(nil).
		DeleteFrom("user_phone").
		Where(q, args...).
		ExecContext(ctx)

	if err != nil {
		return errors.Wrap(err, "failed to delete phones by query")
	}

	return nil
}
