package user_test

import (
	"context"
	"fmt"
	"testing"

	"go.etcd.io/bbolt"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/pkg/user"
	"gitlab.com/agubarev/hometown/pkg/util"
)

func TestNewManager(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	store, err := user.NewDefaultStore(db)
	a.NotNil(store)
	a.NoError(err)

	m := user.NewDefaultManager(store)
	a.NotNil(m)
}

func TestManagerCreate(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	store, err := user.NewDefaultStore(db)
	a.NotNil(store)
	a.NoError(err)

	m := user.NewDefaultManager(store)
	a.NotNil(m)

	u := user.NewUser("testuser", "testuser@example.com")
	a.NotNil(u)

	// creating a new user
	uu, err := m.Create(context.Background(), u)
	a.NotNil(uu)
	a.NoError(err)
}
