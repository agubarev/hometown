package usermanager_test

import (
	"os"
	"path/filepath"
	"testing"

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

func TestUserContainerRegister(t *testing.T) {
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

	u1, err := usermanager.NewUser("testuser1", "testuser1@example.com")
	a.NoError(err)
	a.NotNil(u1)

	u2, err := usermanager.NewUser("testuser2", "testuser2@example.com")
	a.NoError(err)
	a.NotNil(u2)

	u3, err := usermanager.NewUser("testuser3", "testuser3@example.com")
	a.NoError(err)
	a.NotNil(u3)

	a.NoError(c.Register(u1))
	a.NoError(c.Register(u2))
	a.NoError(c.Register(u3))
}
