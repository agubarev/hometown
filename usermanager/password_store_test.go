package usermanager_test

import (
	"fmt"
	"os"
	"testing"

	"gitlab.com/agubarev/hometown/usermanager"

	"github.com/dgraph-io/badger"
	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/util"
)

func TestPasswordStorePut(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	dbDir := fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID())
	opts := badger.DefaultOptions
	opts.Dir = dbDir
	opts.ValueDir = dbDir

	db, err := badger.Open(opts)
	a.NoError(err)
	a.NotNil(db)
	defer os.RemoveAll(opts.Dir)

	s, err := usermanager.NewDefaultPasswordStore(db)
	a.NoError(err)
	a.NotNil(s)

	id := util.NewULID()
	pass := []byte("test123")

	err = s.Put(id, pass)
	a.NoError(err)
}

func TestPasswordStoreGet(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	dbDir := fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID())
	opts := badger.DefaultOptions
	opts.Dir = dbDir
	opts.ValueDir = dbDir

	db, err := badger.Open(opts)
	a.NoError(err)
	a.NotNil(db)
	defer os.RemoveAll(opts.Dir)

	s, err := usermanager.NewDefaultPasswordStore(db)
	a.NoError(err)
	a.NotNil(s)

	id := util.NewULID()
	pass := []byte("test123")

	err = s.Put(id, pass)
	a.NoError(err)

	p, err := s.Get(id)
	a.NoError(err)
	a.Len(p, len(pass))
	a.Equal(pass, p)
}

func TestPasswordStoreDelete(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	dbDir := fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID())
	opts := badger.DefaultOptions
	opts.Dir = dbDir
	opts.ValueDir = dbDir

	db, err := badger.Open(opts)
	a.NoError(err)
	a.NotNil(db)
	defer os.RemoveAll(opts.Dir)

	s, err := usermanager.NewDefaultPasswordStore(db)
	a.NoError(err)
	a.NotNil(s)

	id := util.NewULID()
	pass := []byte("test123")

	err = s.Put(id, pass)
	a.NoError(err)

	p, err := s.Get(id)
	a.NoError(err)
	a.Len(p, len(pass))
	a.Equal(pass, p)

	err = s.Delete(id)
	a.NoError(err)

	p, err = s.Get(id)
	a.Error(err)
}
