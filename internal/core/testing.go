package core

import (
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/password"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/gocraft/dbr/v2"
)

// NewGroupContainerForTesting initializes a group container for testing
func NewGroupContainerForTesting(db *dbr.Connection) (*group.Manager, error) {
	s, err := group.NewGroupStore(db)
	if err != nil {
		return nil, err
	}

	return group.NewGroupManager(s)
}

// NewUserManagerForTesting returns a fully initialized user manager for testing
func NewUserManagerForTesting(db *dbr.Connection) (*user.Manager, error) {
	//---------------------------------------------------------------------------
	// initializing stores
	//---------------------------------------------------------------------------
	us, err := user.NewMySQLStore(db)
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
	pm, err := password.NewManager(ps)
	if err != nil {
		return nil, err
	}

	//---------------------------------------------------------------------------
	// initializing user manager
	//---------------------------------------------------------------------------

	um, err := user.NewManager(us, nil)
	if err != nil {
		return nil, err
	}

	err = um.SetPasswordManager(pm)
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

	return um, nil
}
