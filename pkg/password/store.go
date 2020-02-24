package password

import (
	"context"
	"database/sql"

	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
)

// Store interface
// NOTE: ownerID represents the ObjectID of whoever owns a given password
type Store interface {
	Upsert(ctx context.Context, p Password) error
	Update(ctx context.Context, k Kind, ownerID uint32, newpass []byte) error
	Get(ctx context.Context, k Kind, ownerID uint32) (Password, error)
	Delete(ctx context.Context, k Kind, ownerID uint32) error
}

type passwordStore struct {
	db *dbr.Connection
}

// NewPasswordStore initializes a default password store
func NewPasswordStore(db *dbr.Connection) (Store, error) {
	// reserving error return for the future, just in case
	return &passwordStore{db}, nil
}

// Upsert stores password
// ObjectID must be equal to the user's ObjectID
func (s *passwordStore) Upsert(ctx context.Context, p Password) (err error) {
	if p.OwnerID == 0 {
		return ErrZeroOwnerID
	}

	if p.Hash[0] == 0 {
		return ErrEmptyPassword
	}

	if err = p.Validate(); err != nil {
		return errors.Wrap(err, "password validation failed")
	}

	_, err = s.db.NewSession(nil).
		ExecContext(
			ctx,
			"INSERT INTO `password`(`kind`, `owner_id`, `hash`, `is_change_required`, `created_at`, `expire_at`) VALUES(?, ?, ?, ?, ?, ?) ON DUPLICATE KEY UPDATE `hash`=?, `updated_at`=?, `expire_at`=?",
			p.Kind, p.OwnerID, p.Hash, p.IsChangeRequired, p.CreatedAt, p.ExpireAt, p.Hash, p.UpdatedAt, p.ExpireAt,
		)

	return nil
}

// UpdatePolicy updates an existing password record
func (s *passwordStore) Update(ctx context.Context, k Kind, ownerID uint32, newpass []byte) (err error) {
	if len(newpass) == 0 {
		return ErrEmptyPassword
	}

	updates := map[string]interface{}{
		"hash": newpass,
	}

	// updating the password
	_, err = s.db.NewSession(nil).Update("password").
		SetMap(updates).Where("kind = ? AND owner_id = ?", k, ownerID).ExecContext(ctx)

	if err != nil {
		return err
	}

	return nil
}

// Get retrieves a stored password
func (s *passwordStore) Get(ctx context.Context, k Kind, userID uint32) (p Password, err error) {
	// retrieving password
	err = s.db.NewSession(nil).
		Select("*").
		From("password").
		Where("kind = ? AND owner_id = ?", k, userID).
		LoadOneContext(ctx, &p)

	if err != nil {
		if err == sql.ErrNoRows {
			return p, ErrPasswordNotFound
		}

		return p, err
	}

	return p, nil
}

// DeletePolicy deletes a stored password
func (s *passwordStore) Delete(ctx context.Context, k Kind, ownerID uint32) (err error) {
	_, err = s.db.NewSession(nil).DeleteFrom("password").Where("kind = ? AND id = ?", k, ownerID).ExecContext(ctx)
	if err != nil {
		return err
	}

	return nil
}
