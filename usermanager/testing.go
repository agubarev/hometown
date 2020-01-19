package usermanager

import (
	"fmt"
	"os"

	"github.com/agubarev/hometown/util"
	"github.com/dgraph-io/badger"
	"github.com/jmoiron/sqlx"
)

// CreateRandomBadgerDB creates a "throwaway" Badger database for testing
func CreateRandomBadgerDB() (*badger.DB, string, error) {
	dbDir := fmt.Sprintf("/tmp/testdb-%s", util.NewULID())
	db, err := badger.Open(badger.DefaultOptions(dbDir))
	if err != nil {
		return nil, "", err
	}

	return db, dbDir, nil
}

// DatabaseForTesting simply returns a database connection
func DatabaseForTesting() (*sqlx.DB, error) {
	return sqlx.Open("mysql", os.Getenv("DB_TEST_DSN"))
}

// TruncateDatabaseForTesting truncates whole database for testing
func TruncateDatabaseForTesting(db *sqlx.DB) (err error) {
	tx, err := db.Beginx()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("recovering from panic after TruncateDatabaseForTesting: %s", err)
		}
	}()

	tables := []string{
		"users",
		"passwords",
		"tokens",
		"groups",
		"group_users",
		"access_policy",
		"access_policy_rights_roster",
	}

	//? BEGIN ->>>----------------------------------------------------------------
	//? truncating database tables

	for _, tableName := range tables {
		_, err := tx.Exec(fmt.Sprintf("TRUNCATE TABLE `hometown_test`.%s", tableName))
		if err != nil {
			tx.Rollback()
			return err
		}
	}

	//? truncating database tables
	//? END ---<<<----------------------------------------------------------------

	return tx.Commit()
}

// NewUserContainerForTesting initializes a user container for tests only
func NewUserContainerForTesting(db *sqlx.DB) (*UserContainer, error) {
	return NewUserContainer()
}

// NewGroupContainerForTesting initializes a group container for testing
func NewGroupContainerForTesting(db *sqlx.DB) (*GroupManager, error) {
	s, err := NewGroupStore(db)
	if err != nil {
		return nil, err
	}

	return NewGroupManager(s)
}

// NewUserManagerForTesting returns a fully initialized user manager for testing
func NewUserManagerForTesting(db *sqlx.DB) (*UserManager, error) {
	//---------------------------------------------------------------------------
	// initializing stores
	//---------------------------------------------------------------------------
	us, err := NewUserStore(db)
	if err != nil {
		return nil, err
	}

	gs, err := NewGroupStore(db)
	if err != nil {
		return nil, err
	}

	ts, err := NewTokenStore(db)
	if err != nil {
		return nil, err
	}

	aps, err := NewDefaultAccessPolicyStore(db)
	if err != nil {
		return nil, err
	}

	ps, err := NewPasswordStore(db)
	if err != nil {
		return nil, err
	}

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	pm, err := NewDefaultPasswordManager(ps)
	if err != nil {
		return nil, err
	}

	gm, err := NewGroupManager(gs)
	if err != nil {
		return nil, err
	}

	gl, err := util.DefaultLogger(false, "")
	if err != nil {
		return nil, err
	}

	err = gm.SetLogger(gl)
	if err != nil {
		return nil, err
	}

	tc, err := NewTokenManager(ts)
	if err != nil {
		return nil, err
	}

	apc, err := NewAccessPolicyContainer(aps)
	if err != nil {
		return nil, err
	}

	//---------------------------------------------------------------------------
	// initializing user manager
	//---------------------------------------------------------------------------

	um, err := NewUserManager(us, nil)
	if err != nil {
		return nil, err
	}

	err = um.SetGroupManager(gm)
	if err != nil {
		return nil, err
	}

	err = um.SetTokenManager(tc)
	if err != nil {
		return nil, err
	}

	err = um.SetPasswordManager(pm)
	if err != nil {
		return nil, err
	}

	err = um.SetAccessPolicyContainer(apc)
	if err != nil {
		return nil, err
	}

	userLogger, err := util.DefaultLogger(false, "")
	if err != nil {
		return nil, err
	}

	err = um.SetLogger(userLogger)
	if err != nil {
		return nil, err
	}

	err = um.Init()
	if err != nil {
		return nil, err
	}

	return um, nil
}
