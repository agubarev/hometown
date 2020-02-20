package password

import (
	"context"
	"database/sql"

	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
)

// Store interface
// NOTE: ownerID represents the ID of whoever owns a given password
type Store interface {
	Create(ctx context.Context, p *Password) error
	Update(ctx context.Context, k Kind, ownerID int, newpass []byte) error
	Get(ctx context.Context, k Kind, ownerID int) (*Password, error)
	Delete(ctx context.Context, k Kind, ownerID int) error
}

type passwordStore struct {
	db *dbr.Connection
}

// NewPasswordStore initializes a default password store
func NewPasswordStore(db *dbr.Connection) (Store, error) {
	// reserving error return for the future, just in case
	return &passwordStore{db}, nil
}

// Create stores password
// ID must be equal to the user's ID
func (s *passwordStore) Create(ctx context.Context, p *Password) (err error) {
	if p == nil {
		return ErrNilPassword
	}

	if err = p.Validate(); err != nil {
		return errors.Wrap(err, "password validation failed")
	}

	_, err = s.db.NewSession(nil).
		InsertInto("password").
		Record(p).
		ExecContext(ctx)

	return nil
}

// UpdateAccessPolicy updates an existing password record
func (s *passwordStore) Update(ctx context.Context, k Kind, ownerID int, newpass []byte) (err error) {
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
func (s *passwordStore) Get(ctx context.Context, k Kind, userID int) (p *Password, err error) {
	// retrieving password
	err = s.db.NewSession(nil).
		Select("*").
		From("password").
		Where("kind = ? AND owner_id = ?", k, userID).
		LoadOneContext(ctx, &p)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPasswordNotFound
		}

		return nil, err
	}

	return p, nil
}

// Delete deletes a stored password
func (s *passwordStore) Delete(ctx context.Context, k Kind, ownerID int) (err error) {
	_, err = s.db.NewSession(nil).DeleteFrom("password").Where("kind = ? AND id = ?", k, ownerID).ExecContext(ctx)
	if err != nil {
		return err
	}

	return nil
}
