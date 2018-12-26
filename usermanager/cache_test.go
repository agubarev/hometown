package usermanager_test

import (
	"context"
	"fmt"
	"testing"

	"gitlab.com/agubarev/hometown/usermanager"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/util"
	"go.etcd.io/bbolt"
)

func TestNewDefaultStoreCache(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	c := usermanager.NewUserStoreCache(1000)
	a.NotNil(c)
}

func TestCacheStorePut(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := usermanager.NewBoltStore(db, nil)
	a.NoError(err)
	a.NotNil(s)

	newuser, err := usermanager.NewUser("testuser", "test@example.com")
	a.NoError(err)
	a.NotNil(newuser)

	err = s.PutUser(context.Background(), newuser)
	a.NoError(err)
}
