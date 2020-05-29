package user

import (
	"context"
	"flag"
	"log"

	"github.com/agubarev/hometown/pkg/password"
	"github.com/agubarev/hometown/pkg/token"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/gocraft/dbr/v2"
)

// GroupManagerForTesting initializes a group container for testing
func GroupManagerForTesting(db *dbr.Connection) (*GroupManager, error) {
	s, err := NewGroupStore(db)
	if err != nil {
		return nil, err
	}

	return NewGroupManager(s)
}

// ManagerForTesting returns a fully initialized user manager for testing
func ManagerForTesting(db *dbr.Connection) (*Manager, context.Context, error) {
	ctx := context.Background()

	//---------------------------------------------------------------------------
	// initializing stores
	//---------------------------------------------------------------------------
	us, err := NewMySQLStore(db)
	if err != nil {
		return nil, nil, err
	}

	ps, err := password.NewPasswordStore(db)
	if err != nil {
		return nil, nil, err
	}

	//---------------------------------------------------------------------------
	// initializing dependencies
	//---------------------------------------------------------------------------
	pm, err := password.NewManager(ps)
	if err != nil {
		return nil, nil, err
	}

	//---------------------------------------------------------------------------
	// initializing policy manager
	//---------------------------------------------------------------------------
	aps, err := NewDefaultAccessPolicyStore(db)
	if err != nil {
		return nil, nil, err
	}

	apm, err := NewAccessPolicyManager(aps)
	if err != nil {
		return nil, nil, err
	}

	//---------------------------------------------------------------------------
	// initializing group manager
	//---------------------------------------------------------------------------
	gs, err := NewGroupStore(db)
	if err != nil {
		return nil, nil, err
	}

	gm, err := NewGroupManager(gs)
	if err != nil {
		return nil, nil, err
	}

	//---------------------------------------------------------------------------
	// initializing token manager
	//---------------------------------------------------------------------------
	tms, err := token.NewTokenStore(db)
	if err != nil {
		return nil, nil, err
	}

	tm, err := token.NewTokenManager(tms)
	if err != nil {
		return nil, nil, err
	}

	//---------------------------------------------------------------------------
	// initializing user manager
	//---------------------------------------------------------------------------
	um, err := NewManager(us)
	if err != nil {
		return nil, nil, err
	}

	err = um.SetPasswordManager(pm)
	if err != nil {
		return nil, nil, err
	}

	err = um.SetAccessPolicyManager(apm)
	if err != nil {
		return nil, nil, err
	}

	err = um.SetTokenManager(tm)
	if err != nil {
		return nil, nil, err
	}

	err = um.SetGroupManager(gm)
	if err != nil {
		return nil, nil, err
	}

	userLogger, err := util.DefaultLogger(false, "")
	if err != nil {
		return nil, nil, err
	}

	err = um.SetLogger(userLogger)
	if err != nil {
		return nil, nil, err
	}

	// configuring context
	ctx = context.WithValue(ctx, CKUserManager, um)
	ctx = context.WithValue(ctx, CKGroupManager, gm)
	ctx = context.WithValue(ctx, CKAccessPolicyManager, apm)

	return um, ctx, nil
}

func CreateTestUser(ctx context.Context, m *Manager, username string, email string, pass []byte) (User, error) {
	if flag.Lookup("test.v") == nil {
		log.Fatal("can only be called during testing")
	}

	return m.CreateUser(ctx, func(ctx context.Context) (userObject NewUserObject, err error) {
		if pass == nil {
			pass = []byte("9dcni22lqadffa9h")
		}

		userObject = NewUserObject{
			Essential: Essential{
				Username:    username,
				DisplayName: util.NewULID().String(),
			},
			ProfileEssential: ProfileEssential{
				Firstname:  "John",
				Lastname:   "Smith",
				Middlename: "Jack",
				Language:   "EN",
			},
			EmailAddr:   email,
			PhoneNumber: util.NewULID().String()[15:26],
			Password:    pass,
		}

		return userObject, nil
	})
}
