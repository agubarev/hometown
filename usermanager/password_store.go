package usermanager

import (
	"database/sql"

	"github.com/go-sql-driver/mysql"

	"github.com/jmoiron/sqlx"
)

// PasswordStore interface
// NOTE: ownerID represents the ID of whoever owns a given password
type PasswordStore interface {
	Create(p *Password) error
	Update(ownerID int64, newpass []byte) error
	Get(ownerID int64) (*Password, error)
	Delete(ownerID int64) error
}

type passwordStore struct {
	db *sqlx.DB
}

// NewPasswordStore initializes a default password store
func NewPasswordStore(db *sqlx.DB) (PasswordStore, error) {
	// reserving error return for the future, just in case
	return &passwordStore{db}, nil
}

// Create stores password
// ID must be equal to the user's ID
func (s *passwordStore) Create(p *Password) error {
	if p == nil {
		return ErrNilPassword
	}

	// creating password record
	q := `INSERT INTO passwords(owner_id, hash, created_at, is_change_req) 
			VALUES(:owner_id, :hash, :created_at, :is_change_req)`

	_, err := s.db.NamedExec(q, &p)

	// error handling
	if err != nil {
		switch err := err.(*mysql.MySQLError); err.Number {
		case 1062:
			return ErrDuplicateEntry
		default:
			return err
		}
	}

	return nil
}

// Update updates an existing password record
func (s *passwordStore) Update(ownerID int64, newpass []byte) error {
	// just executing query but not refetching the updated version
	res, err := s.db.Exec("UPDATE passwords SET hash = ? WHERE owner_id = ? LIMIT 1", ownerID, newpass)
	if err != nil {
		return err
	}

	// checking whether anything was updated at all
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}

	// if no rows were affected then returning this as a non-critical error
	if ra == 0 {
		return ErrNothingChanged
	}

	return nil
}

// Get retrieves a stored password
func (s *passwordStore) Get(ownerID int64) (*Password, error) {
	p := new(Password)

	// retrieving password
	err := s.db.Get(p, "SELECT * FROM passwords WHERE owner_id = ? LIMIT 1", ownerID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrPasswordNotFound
		}

		return nil, err
	}

	return p, nil
}

// Delete deletes a stored password
func (s *passwordStore) Delete(ownerID int64) error {
	res, err := s.db.Exec("DELETE FROM passwords WHERE owner_id = ? LIMIT 1", ownerID)
	if err != nil {
		return err
	}

	// checking whether anything was updated at all
	ra, err := res.RowsAffected()
	if err != nil {
		return err
	}

	// if no rows were affected then returning this as a non-critical error
	if ra == 0 {
		return ErrNothingChanged
	}

	return nil
}
