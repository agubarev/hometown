package user_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/stretchr/testify/assert"
)

func TestUserManagerNew(t *testing.T) {
	a := assert.New(t)

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	userStore, err := user.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	passwordStore, err := password.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(passwordStore)

	passwordManager, err := password.NewManager(passwordStore)
	a.NoError(err)
	a.NotNil(passwordManager)
}

func TestUserManagerCreate(t *testing.T) {
	a := assert.New(t)

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	userStore, err := user.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	passwordStore, err := password.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(passwordStore)

	passwordManager, err := password.NewManager(passwordStore)
	a.NoError(err)
	a.NotNil(passwordManager)

	userManager, _, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(userManager)

	u1, err := userManager.CreateUser(context.Background(), func(ctx context.Context) (object user.NewUserObject, err error) {
		object = user.NewUserObject{
			Essential: user.Essential{
				Username:    bytearray.NewByteString32("testuser"),
				DisplayName: bytearray.NewByteString32("test display name"),
			},
			ProfileEssential: user.ProfileEssential{
				Firstname:  bytearray.NewByteString16("Andrejs"),
				Lastname:   bytearray.NewByteString16("Gubarevs"),
				Middlename: string{},
			},
			EmailAddr:   bytearray.NewByteString256("testuser@hometown.local"),
			PhoneNumber: bytearray.NewByteString16("12398543292"),
			Password:    util.NewULID().Entropy(),
		}

		return object, nil
	})
	a.NoError(err)
	a.NotNil(u1)
}
