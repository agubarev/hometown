package usermanager_test

import (
	"testing"

	"github.com/agubarev/hometown/usermanager"
	"github.com/stretchr/testify/assert"
)

func TestNewUserContainer(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := usermanager.NewUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := usermanager.NewUserContainer()
	a.NoError(err)
	a.NotNil(c)

	a.NoError(c.Validate())
}

func TestUserContainerAdd(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	userManager, err := usermanager.NewUserManagerForTesting(db)
	a.NoError(err)
	a.NotNil(userManager)

	//---------------------------------------------------------------------------
	// make sure the container is fresh and empty
	// NOTE: at this point the store is newly created and must be empty
	// ! WARNING: NOT USING userManager's default container
	//---------------------------------------------------------------------------
	// initializing new container
	userContainer, err := usermanager.NewUserContainer()
	a.NoError(err)
	a.NotNil(userContainer)

	a.Len(userContainer.List(nil), 0)

	// creating test users
	u1, err := userManager.Create("testuser1", "testuser1@example.com", map[string]string{})
	a.NoError(err)
	a.NotNil(u1)

	u2, err := userManager.Create("testuser2", "testuser2@example.com", map[string]string{})
	a.NoError(err)
	a.NotNil(u2)

	u3, err := userManager.Create("testuser3", "testuser3@example.com", map[string]string{})
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
