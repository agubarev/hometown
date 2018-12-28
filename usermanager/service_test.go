package usermanager

import (
	"context"
	"fmt"
	"testing"

	"gitlab.com/agubarev/hometown/util"
	"go.etcd.io/bbolt"

	"github.com/stretchr/testify/assert"
)

func TestNewService(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	store, err := NewBoltStore(db, nil)
	a.NotNil(store)
	a.NoError(err)

	m, err := NewUserService(store)
	a.NoError(err)
	a.NotNil(m)
}

func TestServiceCreate(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	store, err := NewBoltStore(db, nil)
	a.NotNil(store)
	a.NoError(err)

	m, err := NewUserService(store)
	a.NoError(err)
	a.NotNil(m)

	u, err := NewUser("testuser", "testuser@example.com")
	a.NoError(err)
	a.NotNil(u)

	// creating a new user
	uu, err := m.Create(context.Background(), u)
	a.NotNil(uu)
	a.NoError(err)
}
