package user_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/stretchr/testify/assert"
)

func TestUserManagerNew(t *testing.T) {
	a := assert.New(t)

	db := database.PostgreSQLForTesting(nil)
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

	db := database.PostgreSQLForTesting(nil)
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
				Username:    "testuser",
				DisplayName: "test display name",
			},
			ProfileEssential: user.ProfileEssential{
				Firstname:  "Andrejs",
				Lastname:   "Gubarevs",
				Middlename: "",
			},
			EmailAddr:   "testuser@hometown.local",
			PhoneNumber: "12398543292",
			Password:    password.NewRaw(32, 3, password.GFDefault),
		}

		return object, nil
	})
	a.NoError(err)
	a.NotNil(u1)
}
