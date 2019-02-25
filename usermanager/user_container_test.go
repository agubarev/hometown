package usermanager_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/oklog/ulid"

	"github.com/blevesearch/bleve"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/usermanager"
)

func TestNewUserContainer(t *testing.T) {
	a := assert.New(t)

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)
	a.NoError(err)
	a.NotNil(db)

	s, err := usermanager.NewDefaultUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	index, err := bleve.New(filepath.Join(dbPath, "index"), bleve.NewIndexMapping())
	a.NoError(err)
	a.NotNil(index)

	c, err := usermanager.NewUserContainer(s, index)
	a.NoError(err)
	a.NotNil(c)

	a.NoError(c.Validate())
}

func TestUserContainerCreate(t *testing.T) {
	a := assert.New(t)

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)
	a.NoError(err)
	a.NotNil(db)

	s, err := usermanager.NewDefaultUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	index, err := bleve.New(filepath.Join(dbPath, "index"), bleve.NewIndexMapping())
	a.NoError(err)
	a.NotNil(index)

	c, err := usermanager.NewUserContainer(s, index)
	a.NoError(err)
	a.NotNil(c)

	a.NoError(c.Validate())

	u1, err := c.Create("testuser1", "testuser1@example.com", map[string]string{
		"firstname": "Andrei",
	})
	a.NoError(err)
	a.NotNil(u1)

	u2, err := c.Create("testuser2", "testuser2@example.com", map[string]string{
		"lastname": "Gubarev",
	})
	a.NoError(err)
	a.NotNil(u2)

	u3, err := c.Create("testuser3", "testuser3@example.com", map[string]string{
		"firstname":  "Андрей",
		"lastname":   "Губарев",
		"middlename": "Анатолиевич",
	})
	a.NoError(err)
	a.NotNil(u3)

	u4, err := c.Create("testuser4", "testuser4@example.com", map[string]string{
		"firstname": "Andrei 123",
		"lastname":  "456 Gubarev",
	})
	a.Error(err)
	a.Nil(u4)

	// container must contain only 3 runtime user objects at this point
	a.Len(c.List(), 3)

	//---------------------------------------------------------------------------
	// checking store
	//---------------------------------------------------------------------------
	testUsers := map[ulid.ULID]*usermanager.User{
		u1.ID: u1,
		u2.ID: u2,
		u3.ID: u3,
	}

	storedUsers, err := s.GetAll()
	a.NoError(err)
	a.Len(storedUsers, 3)

	for _, su := range storedUsers {
		a.Equal(testUsers[su.ID].Username, su.Username)
		a.Equal(testUsers[su.ID].Email, su.Email)
		a.Equal(testUsers[su.ID].Firstname, su.Firstname)
		a.Equal(testUsers[su.ID].Lastname, su.Lastname)
		a.Equal(testUsers[su.ID].Middlename, su.Middlename)
		a.Equal(testUsers[su.ID].EmailConfirmedAt, su.EmailConfirmedAt)
	}
}

func TestUserContainerInit(t *testing.T) {
	a := assert.New(t)

	//---------------------------------------------------------------------------
	// initializing store and container
	//---------------------------------------------------------------------------

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)
	a.NoError(err)
	a.NotNil(db)

	s, err := usermanager.NewDefaultUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	index, err := bleve.New(filepath.Join(dbPath, "index"), bleve.NewIndexMapping())
	a.NoError(err)
	a.NotNil(index)

	c, err := usermanager.NewUserContainer(s, index)
	a.NoError(err)
	a.NotNil(c)

	a.NoError(c.Validate())

	//---------------------------------------------------------------------------
	// make sure the container is fresh and empty
	// NOTE: at this point the store is newly created and must be empty
	//---------------------------------------------------------------------------
	a.Len(c.List(), 0)
	storedUsers, err := s.GetAll()
	a.NoError(err)
	a.Len(storedUsers, 0)

	//---------------------------------------------------------------------------
	// creating and storing test users
	//---------------------------------------------------------------------------

	// will just reuse the same user info data for each new user
	validUserinfo := map[string]string{
		"firstname": "Андрей",
		"lastname":  "Губарев",
	}

	u1, err := c.Create("testuser1", "testuser1@example.com", validUserinfo)
	a.NoError(err)
	a.NotNil(u1)

	u2, err := c.Create("testuser2", "testuser2@example.com", validUserinfo)
	a.NoError(err)
	a.NotNil(u2)

	u3, err := c.Create("testuser3", "testuser3@example.com", validUserinfo)
	a.NoError(err)
	a.NotNil(u3)

	//---------------------------------------------------------------------------
	// checking store
	//---------------------------------------------------------------------------
	storedUsers, err = s.GetAll()
	a.NoError(err)
	a.Len(storedUsers, 3)
}

func TestUserContainerAdd(t *testing.T) {
	a := assert.New(t)

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)
	a.NoError(err)
	a.NotNil(db)

	s, err := usermanager.NewDefaultUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	index, err := bleve.New(filepath.Join(dbPath, "index"), bleve.NewIndexMapping())
	a.NoError(err)
	a.NotNil(index)

	c, err := usermanager.NewUserContainer(s, index)
	a.NoError(err)
	a.NotNil(c)

	a.NoError(c.Validate())

	u1, err := usermanager.NewUser("testuser1", "testuser1@example.com", map[string]string{})
	a.NoError(err)
	a.NotNil(u1)

	u2, err := usermanager.NewUser("testuser2", "testuser2@example.com", map[string]string{})
	a.NoError(err)
	a.NotNil(u2)

	u3, err := usermanager.NewUser("testuser3", "testuser3@example.com", map[string]string{})
	a.NoError(err)
	a.NotNil(u3)

	a.NoError(c.Add(u1))
	a.NoError(c.Add(u2))
	a.NoError(c.Add(u3))
}
