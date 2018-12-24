package user_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/user"
	"gitlab.com/agubarev/hometown/util"
	"go.etcd.io/bbolt"
)

func TestNewDefaultStoreCache(t *testing.T) {
	a := assert.New(t)

	c := user.NewDefaultStoreCache(1000)
	a.NotNil(c)
}

func TestCacheStorePut(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := NewDefaultStore(db, nil)
	a.NoError(err)
	a.NotNil(s)

	newuser := NewUser("testuser", "test@example.com")
	a.NotNil(newuser)

	err = s.Put(context.Background(), newuser)
	a.NoError(err)
}
