package core

import (
	"fmt"
	"os"

	"github.com/agubarev/hometown/pkg/accesspolicy"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/password"
	"github.com/agubarev/hometown/pkg/token"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/gocraft/dbr/v2"
	"github.com/pkg/errors"
)

// DatabaseForTesting simply returns a database connection
func DatabaseForTesting() (*dbr.Connection, error) {
	conn, err := dbr.Open("mysql", os.Getenv("HOMETOWN_DATABASE_DSN"), nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to connect to test database")
	}

	return conn, nil
}

// TruncateDatabaseForTesting truncates whole database for testing
func TruncateDatabaseForTesting(db *dbr.Connection) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer func() {
		if p := recover(); p != nil {
			err = fmt.Errorf("recovering from panic after TruncateDatabaseForTesting: %s", err)
		}
	}()

	tables := []string{
		"user",
		"user_email",
		"user_phone",
		"user_profile",
		"password",
		"token",
		"group",
		"group_users",
		"accesspolicy",
		"accesspolicy_rights_roster",
	}

	// =========================================================================
	// truncating tables
	// =========================================================================
	for _, tableName := range tables {
		_, err := tx.Exec(fmt.Sprintf("TRUNCATE TABLE `hometown_test`.%s", tableName))
		if err != nil {
			return errors.Wrap(err, tx.Rollback().Error())
		}
	}

	return tx.Commit()
}

// NewUserContainerForTesting initializes a user container for tests only
func NewUserContainerForTesting() (*UserContainer, error) {
	return NewUserContainer()
}

// NewGroupContainerForTesting initializes a group container for testing
func NewGroupContainerForTesting(db *dbr.Connection) (*group.Manager, error) {
	s, err := group.NewGroupStore(db)
	if err != nil {
		return nil, err
	}

	return group.NewGroupManager(s)
}

// NewUserManagerForTesting returns a fully initialized user manager for testing
func NewUserManagerForTesting(db *dbr.Connection) (*UserManager, error) {
	//---------------------------------------------------------------------------
	// initializing stores
	//---------------------------------------------------------------------------
	us, err := NewUserStore(db)
	if err != nil {
		return nil, err
	}

	gs, err := group.NewGroupStore(db)
	if err != nil {
		return nil, err
	}

	ts, err := token.NewTokenStore(db)
	if err != nil {
		return nil, err
	}

	aps, err := accesspolicy.NewDefaultAccessPolicyStore(db)
	if err != nil {
		return nil, err
	}

	ps, err := password.NewPasswordStore(db)
	if err != nil {
		return nil, err
	}

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	pm, err := password.NewDefaultManager(ps)
	if err != nil {
		return nil, err
	}

	gm, err := group.NewGroupManager(gs)
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

	tc, err := token.NewTokenManager(ts)
	if err != nil {
		return nil, err
	}

	apc, err := accesspolicy.NewManager(aps)
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
