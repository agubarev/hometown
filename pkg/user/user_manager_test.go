package user_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/password"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/stretchr/testify/assert"
)

func TestUserManagerNew(t *testing.T) {
	a := assert.New(t)

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	userStore, err := user.NewMySQLStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	passwordStore, err := password.NewPasswordStore(db)
	a.NoError(err)
	a.NotNil(passwordStore)

	passwordManager, err := password.NewManager(passwordStore)
	a.NoError(err)
	a.NotNil(passwordManager)
}

func TestUserManagerCreate(t *testing.T) {
	a := assert.New(t)

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	userStore, err := user.NewMySQLStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	passwordStore, err := password.NewPasswordStore(db)
	a.NoError(err)
	a.NotNil(passwordStore)

	passwordManager, err := password.NewManager(passwordStore)
	a.NoError(err)
	a.NotNil(passwordManager)

	userManager, err := user.NewManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	u1, err := userManager.CreateUser(context.Background(), func(ctx context.Context) (object user.NewUserObject, err error) {
		object = user.NewUserObject{
			Essential: user.Essential{
				Username:    user.TUsername{},
				DisplayName: user.TDisplayName{},
			},
			ProfileEssential: user.ProfileEssential{},
			EmailAddr:        user.TEmailAddr{},
			PhoneNumber:      user.TPhoneNumber{},
			Password:         nil,
		}

		return object, nil
	})
	a.NoError(err)
	a.NotNil(u1)
}
