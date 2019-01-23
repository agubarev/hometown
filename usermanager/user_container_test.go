package usermanager_test

import (
	"fmt"
	"os"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/usermanager"
	"gitlab.com/agubarev/hometown/util"
)

func TestNewUserContainer(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	s, err := usermanager.NewDefaultUserStore(db, usermanager.NewUserStoreCache(1000))
	a.NoError(err)
	a.NotNil(s)

	c, err := usermanager.NewUserContainer(s)
	a.NoError(err)
	a.NotNil(c)

	a.NoError(c.Validate())
}

func TestUserContainerAdd(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	s, err := usermanager.NewDefaultUserStore(db, usermanager.NewUserStoreCache(1000))
	a.NoError(err)
	a.NotNil(s)

	c, err := usermanager.NewUserContainer(s)
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
