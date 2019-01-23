package usermanager_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/dgraph-io/badger"
	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/usermanager"
	"gitlab.com/agubarev/hometown/util"
	"go.etcd.io/bbolt"
)

func TestUserManagerTestNew(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	// bbolt db
	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	// badger db for passwords
	dbDir := fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID())
	opts := badger.DefaultOptions
	opts.Dir = dbDir
	opts.ValueDir = dbDir

	pdb, err := badger.Open(opts)
	a.NoError(err)
	a.NotNil(db)
	defer os.RemoveAll(opts.Dir)

	// proceeding with the test
	superuser, err := usermanager.NewUser("superuser", "superuser@example.com")
	a.NoError(err)
	a.NotNil(superuser)

	ds, err := usermanager.NewDefaultDomainStore(db)
	a.NoError(err)
	a.NotNil(ds)

	us, err := usermanager.NewDefaultUserStore(db, usermanager.NewUserStoreCache(1000))
	a.NoError(err)
	a.NotNil(us)

	gs, err := usermanager.NewDefaultGroupStore(db)
	a.NoError(err)
	a.NotNil(gs)

	aps, err := usermanager.NewDefaultAccessPolicyStore(db)
	a.NoError(err)
	a.NotNil(aps)

	ps, err := usermanager.NewDefaultPasswordStore(pdb)
	a.NoError(err)
	a.NotNil(aps)

	m, err := usermanager.New(usermanager.NewStore(ds, us, gs, aps, ps))
	a.NoError(err)
	a.NotNil(m)

	err = m.Init()
	a.Error(usermanager.ErrEmptyDominion, err)
}
