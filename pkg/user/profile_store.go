package user

import (
	"context"
	"database/sql"

	"github.com/agubarev/hometown/pkg/util/guard"
	"github.com/pkg/errors"
	"github.com/r3labs/diff"
)

func (s *MySQLStore) fetchProfileByQuery(ctx context.Context, q string, args ...interface{}) (profile Profile, err error) {
	err = s.connection.NewSession(nil).
		SelectBySql(q, args).
		LoadOneContext(ctx, &profile)

	if err != nil {
		if err == sql.ErrNoRows {
			return profile, ErrProfileNotFound
		}

		return profile, err
	}

	return profile, nil
}

func (s *MySQLStore) fetchProfilesByQuery(ctx context.Context, q string, args ...interface{}) (profiles []Profile, err error) {
	profiles = make([]Profile, 0)

	_, err = s.connection.NewSession(nil).
		SelectBySql(q, args).
		LoadContext(ctx, &profiles)

	if err != nil {
		if err == sql.ErrNoRows {
			return profiles, nil
		}

		return nil, err
	}

	return profiles, nil
}

// CreateProfile creates a new entry in the storage backend
func (s *MySQLStore) CreateProfile(ctx context.Context, profile Profile) (_ Profile, err error) {
	// if ObjectID is not 0, then it's not considered as new
	if profile.UserID == 0 {
		return profile, ErrZeroUserID
	}

	_, err = s.connection.NewSession(nil).
		InsertInto("user_profile").
		Columns(guard.DBColumnsFrom(profile)...).
		Record(profile).
		ExecContext(ctx)

	if err != nil {
		return profile, err
	}

	return profile, nil
}

// CreateProfile creates a new entry in the storage backend
func (s *MySQLStore) BulkCreateProfile(ctx context.Context, profiles []Profile) (_ []Profile, err error) {
	// there must be something first
	if len(profiles) == 0 {
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
	stmt := tx.InsertInto("user_profile").Columns(guard.DBColumnsFrom(profiles[0])...)

	// validating each profile individually
	for i := range profiles {
		if profiles[i].UserID != 0 {
			return nil, ErrZeroUserID
		}

		if err := profiles[i].Validate(); err != nil {
			return nil, err
		}

		// adding value to the batch
		stmt = stmt.Record(profiles[i])
	}

	// executing the batch
	_, err = stmt.ExecContext(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "bulk insert failed")
	}

	if err = tx.Commit(); err != nil {
		return nil, errors.Wrap(err, "failed to commit database transaction")
	}

	return profiles, nil
}

func (s *MySQLStore) FetchProfileByUserID(ctx context.Context, id uint32) (profile Profile, err error) {
	return s.fetchProfileByQuery(ctx, "SELECT * FROM `user_profile` WHERE user_id = ? LIMIT 1", id)
}

func (s *MySQLStore) FetchProfilesByWhere(ctx context.Context, order string, limit, offset uint64, where string, args ...interface{}) (profiles []Profile, err error) {
	_, err = s.connection.NewSession(nil).
		Select("*").
		Where(where, args...).
		Limit(limit).
		Offset(offset).
		OrderBy(order).
		LoadContext(ctx, &profiles)

	if err != nil {
		if err == sql.ErrNoRows {
			return profiles, nil
		}

		return profiles, err
	}

	return profiles, nil
}

func (s *MySQLStore) UpdateProfile(ctx context.Context, profile Profile, changelog diff.Changelog) (_ Profile, err error) {
	if len(changelog) == 0 {
		return profile, ErrNothingChanged
	}

	changes, err := guard.ProcureDBChangesFromChangelog(profile, changelog)
	if err != nil {
		return profile, errors.Wrap(err, "failed to procure changes from a changelog")
	}

	_, err = s.connection.NewSession(nil).
		Update("user_profile").
		Where("user_id = ?", profile.UserID).
		SetMap(changes).
		ExecContext(ctx)

	if err != nil {
		return profile, err
	}

	return profile, nil
}

func (s *MySQLStore) DeleteProfileByUserID(ctx context.Context, userID uint32) (err error) {
	if userID == 0 {
		return ErrZeroID
	}

	_, err = s.connection.NewSession(nil).
		DeleteFrom("user_profile").
		Where("user_id = ?", userID).
		ExecContext(ctx)

	if err != nil {
		return errors.Wrapf(err, "failed to delete profile: userID=%d", userID)
	}

	return nil
}

func (s *MySQLStore) DeleteProfilesByQuery(ctx context.Context, q string, args ...interface{}) (err error) {
	_, err = s.connection.NewSession(nil).
		DeleteFrom("user_profile").
		Where(q, args...).
		ExecContext(ctx)

	if err != nil {
		return errors.Wrap(err, "failed to delete profiles by query")
	}

	return nil
}
