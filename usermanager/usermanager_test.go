package usermanager_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/usermanager"
	"gitlab.com/agubarev/hometown/util"
	"go.etcd.io/bbolt"
)

func TestUserManagerTestNew(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer os.Remove(db.Path())

	rootUser, err := usermanager.NewUser("root", "root@example.com")
	a.NoError(err)
	a.NotNil(rootUser)

	domain, err := usermanager.NewDomain(rootUser)
	a.NoError(err)
	a.NotNil(domain)

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

	m, err := usermanager.New(usermanager.NewStore(ds, us, gs, aps))
	a.NoError(err)
	a.NotNil(m)
}
