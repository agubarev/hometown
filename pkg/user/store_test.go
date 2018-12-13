package user_test

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"gitlab.com/agubarev/hometown/pkg/user"
	"gitlab.com/agubarev/hometown/pkg/util"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/bbolt"
)

func TestNewDefaultStore(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NotNil(db)
	a.NoError(err)
	defer db.Close()

	s, err := user.NewDefaultStore(db)
	a.NotNil(s)
	a.NoError(err)
}

func TestStoreCreate(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := user.NewDefaultStore(db)
	a.NoError(err)
	a.NotNil(s)

	m := user.NewDefaultManager(s)
	a.NoError(err)
	a.NotNil(m)

	unsavedUser := user.NewUser("testuser", "test@example.com")
	a.NotNil(unsavedUser)

	newUser, err := m.Create(context.Background(), unsavedUser)
	a.NoError(err)
	a.NotNil(newUser)
	a.Equal("testuser", newUser.Username)
	a.Equal("test@example.com", newUser.Email)
	a.True(reflect.DeepEqual(unsavedUser, newUser))
}

func TestStorePut(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := user.NewDefaultStore(db)
	a.NoError(err)
	a.NotNil(s)

	newuser := user.NewUser("testuser", "test@example.com")
	a.NotNil(newuser)

	err = s.Put(context.Background(), newuser)
	a.NoError(err)
}

func TestStoreGetters(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := user.NewDefaultStore(db)
	a.NoError(err)
	a.NotNil(s)

	newuser := user.NewUser("testuser", "test@example.com")
	a.NotNil(newuser)

	err = s.Put(context.Background(), newuser)
	a.NoError(err)

	u, err := s.GetByID(context.Background(), newuser.ID)
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	u, err = s.GetByIndex(context.Background(), "username", "testuser")
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	u, err = s.GetByIndex(context.Background(), "email", "test@example.com")
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)
}
