package user_test

import (
	"testing"

	"github.com/agubarev/hometown/internal/core"
	"github.com/stretchr/testify/assert"
)

func TestNewUserContainer(t *testing.T) {
	a := assert.New(t)

	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := core.NewUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := core.NewUserContainer()
	a.NoError(err)
	a.NotNil(c)

	a.NoError(c.Validate())
}

func TestUserContainerAdd(t *testing.T) {
	a := assert.New(t)

	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(core.TruncateDatabaseForTesting(db))

	userManager, err := core.NewUserManagerForTesting(db)
	a.NoError(err)
	a.NotNil(userManager)

	//---------------------------------------------------------------------------
	// make sure the container is fresh and empty
	// NOTE: at this point the store is newly created and must be empty
	// ! WARNING: NOT USING userManager's default container
	//---------------------------------------------------------------------------
	// initializing new container
	userContainer, err := core.NewUserContainer()
	a.NoError(err)
	a.NotNil(userContainer)

	a.Len(userContainer.List(nil), 0)

	// creating test users
	u1, err := userManager.Create("testuser1")
	a.NoError(err)
	a.NotNil(u1)

	u2, err := userManager.Create("testuser2")
	a.NoError(err)
	a.NotNil(u2)

	u3, err := userManager.Create("testuser3")
	a.NoError(err)
	a.NotNil(u3)

	// setting user IDs manually
	u1.ID = 1
	u2.ID = 2
	u3.ID = 3

	a.NoError(userContainer.Add(u1))
	a.NoError(userContainer.Add(u2))
	a.NoError(userContainer.Add(u3))
}
