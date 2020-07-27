package user

import (
	"context"
	"flag"
	"log"

	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/security/accesspolicy"
	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/agubarev/hometown/pkg/token"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/google/uuid"
	"github.com/jackc/pgx"
)

// GroupManagerForTesting initializes a group container for testing
func GroupManagerForTesting(db *pgx.Conn) (*group.Manager, error) {
	s, err := group.NewPostgreSQLStore(db)
	if err != nil {
		return nil, err
	}

	return group.NewManager(context.Background(), s)
}

// ManagerForTesting returns a fully initialized user manager for testing
func ManagerForTesting(db *pgx.Conn) (*Manager, context.Context, error) {
	ctx := context.Background()

	//---------------------------------------------------------------------------
	// initializing stores
	//---------------------------------------------------------------------------
	us, err := NewPostgreSQLStore(db)
	if err != nil {
		return nil, nil, err
	}

	ps, err := password.NewPostgreSQLStore(db)
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
	// initializing group manager
	//---------------------------------------------------------------------------
	gs, err := group.NewPostgreSQLStore(db)
	if err != nil {
		return nil, nil, err
	}

	gm, err := group.NewManager(ctx, gs)
	if err != nil {
		return nil, nil, err
	}

	//---------------------------------------------------------------------------
	// initializing policy manager
	//---------------------------------------------------------------------------
	aps, err := accesspolicy.NewPostgreSQLStore(db)
	if err != nil {
		return nil, nil, err
	}

	apm, err := accesspolicy.NewManager(aps, gm)
	if err != nil {
		return nil, nil, err
	}

	//---------------------------------------------------------------------------
	// initializing token manager
	//---------------------------------------------------------------------------
	tms, err := token.NewStore(db)
	if err != nil {
		return nil, nil, err
	}

	tm, err := token.NewManager(tms)
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

func CreateTestUser(ctx context.Context, m *Manager, username bytearray.ByteString32, email bytearray.ByteString256, pass []byte) (User, error) {
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
				DisplayName: bytearray.NewByteString32(uuid.New().String()),
			},
			ProfileEssential: ProfileEssential{
				Firstname:  bytearray.NewByteString16("John"),
				Lastname:   bytearray.NewByteString16("Smith"),
				Middlename: bytearray.NewByteString16("Jack"),
				Language:   [2]byte{'E', 'N'},
			},
			EmailAddr:   email,
			PhoneNumber: bytearray.NewByteString16(uuid.New().String()[:15]),
			Password:    pass,
		}

		return userObject, nil
	})
}
