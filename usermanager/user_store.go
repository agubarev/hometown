package usermanager

import (
	"database/sql"
	"log"
	"time"

	"gopkg.in/guregu/null.v3"

	"github.com/go-sql-driver/mysql"
	_ "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
)

// UserStore represents a user storage contract
type UserStore interface {
	Put(u *User) (*User, error)
	Delete(id int64) error
	FetchAll() ([]*User, error)
	FetchByID(id int64) (*User, error)
	FetchByKey(index string, value interface{}) (*User, error)
}

// UserStoreMySQL is a default user store implementation
type UserStoreMySQL struct {
	db *sqlx.DB
}

// NewUserStore initializing new user store
func NewUserStore(db *sqlx.DB) (UserStore, error) {
	if db == nil {
		return nil, ErrNilDB
	}

	s := &UserStoreMySQL{
		db: db,
	}

	return s, nil
}

//? BEGIN ->>>----------------------------------------------------------------
//? unexported utility functions

func (s *UserStoreMySQL) fetchOne(q string, args ...interface{}) (*User, error) {
	u := new(User)

	err := s.db.Unsafe().Get(u, q, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrUserNotFound
		}

		return nil, err
	}

	return u, nil
}

func (s *UserStoreMySQL) fetchMany(q string, args ...interface{}) ([]*User, error) {
	us := make([]*User, 0)

	err := s.db.Unsafe().Select(&us, q, args...)
	if err != nil {
		if err == sql.ErrNoRows {
			return us, nil
		}

		return nil, err
	}

	return us, nil
}

//? unexported utility functions
//? END ---<<<----------------------------------------------------------------

// Put stores a User to the database
// NOTE: creates a user if user's new or updates
func (s *UserStoreMySQL) Put(u *User) (*User, error) {
	if u == nil {
		return nil, ErrNilUser
	}

	// if an object has ID other than 0, then it's considered
	// as being already created, thus requiring an update
	if u.ID != 0 {
		return s.Update(u)
	}

	return s.Create(u)
}

// Create creates a new database record
func (s *UserStoreMySQL) Create(u *User) (*User, error) {
	// if ID is not 0, then it's not considered as new
	if u.ID != 0 {
		return nil, ErrObjectIsNotNew
	}

	// constructing statement
	q := `
	INSERT INTO users(id, username, created_at, created_by_id, updated_at, updated_by_id, firstname, middlename, 
		lastname, user_ref, language, phone, email, ru_id, is_suspended, suspension_reason, is_pass_change_req) 
	VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`

	// executing statement
	result, err := s.db.Exec(q,
		u.ID, u.Username, u.CreatedAt, u.CreatedByID, null.TimeFrom(time.Now()),
		u.UpdatedByID, u.Firstname, u.Middlename, u.Lastname, u.UserReference,
		u.Language, u.Phone, u.Email, u.ReadingUnitID, u.IsSuspended,
		u.SuspensionReason, u.IsPasswordChangeRequested,
	)

	// error handling
	if err != nil {
		switch err := err.(*mysql.MySQLError); err.Number {
		case 1062:
			return nil, ErrDuplicateEntry
		}
	}

	if err != nil {
		return nil, err
	}

	newID, err := result.LastInsertId()
	if err != nil {
		return nil, err
	}

	// setting new ID
	u.ID = newID

	return u, nil
}

// Update updates an existing user in a database
func (s *UserStoreMySQL) Update(u *User) (*User, error) {
	if u.ID == 0 {
		return nil, ErrObjectIsNew
	}

	// update record statement
	q := `
	UPDATE users 
	SET firstname = :firstname, lastname = :lastname, middlename = :middlename, user_ref = :user_ref,
		language = :language, phone = :phone, is_suspended = :is_suspended, suspension_reason = :suspension_reason, 
		is_pass_change_req = :is_pass_change_req, last_login_at = :last_login_at, last_login_ip = :last_login_ip,
		last_login_failed_at = :last_login_failed_at, last_login_failed_ip = :last_login_failed_ip,
		last_login_attempts = :last_login_attempts, ru_id = :ru_id, confirmed_at = :confirmed_at,
		suspended_at = :suspended_at, suspension_expires_at = :suspension_expires_at
	WHERE id = :id
	LIMIT 1
	`

	// just executing query but not refetching the updated version
	res, err := s.db.NamedExec(q, u)
	if err != nil {
		return nil, err
	}

	// checking whether anything was updated at all
	ra, err := res.RowsAffected()
	if err != nil {
		return nil, err
	}

	// if no rows were affected then returning this as a non-critical error
	if ra == 0 {
		return nil, ErrNothingChanged
	}

	return u, nil
}

// Delete a user from the store
func (s *UserStoreMySQL) Delete(id int64) error {
	// checking user for existence just to be consistent,
	// and warn when attempting to delete a user which doesn't exist
	u, err := s.FetchByID(id)
	if err != nil {
		return err
	}

	// beginning transaction
	tx, err := s.db.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			log.Printf("UserStore.Delete(): recovering from panic, transaction rollback: %s", p)
			tx.Rollback()
		}
	}()

	// deleting user from the store
	tx.Exec("DELETE FROM users WHERE id = ? LIMIT 1", u.ID)

	return tx.Commit()
}

// FetchByID returns a User by ID
func (s *UserStoreMySQL) FetchByID(id int64) (*User, error) {
	return s.FetchByKey("id", id)
}

// FetchAll returns all stored users
func (s *UserStoreMySQL) FetchAll() ([]*User, error) {
	return s.fetchMany("SELECT * FROM users")
}

// FetchByKey lookup a user by an index
func (s *UserStoreMySQL) FetchByKey(index string, value interface{}) (*User, error) {
	switch index {
	case "id":
		return s.getByID(value.(int64))
	case "username":
		return s.getByUsername(value.(string))
	case "email":
		return s.getByEmail(value.(string))
	default:
		return nil, ErrUnknownIndex
	}
}

func (s *UserStoreMySQL) getByID(id int64) (*User, error) {
	return s.fetchOne("SELECT * FROM users WHERE id = ? LIMIT 1", id)
}

func (s *UserStoreMySQL) getByUsername(username string) (*User, error) {
	return s.fetchOne("SELECT * FROM users WHERE username = ? LIMIT 1", username)
}

func (s *UserStoreMySQL) getByEmail(email string) (*User, error) {
	return s.fetchOne("SELECT * FROM users WHERE email = ? LIMIT 1", email)
}
