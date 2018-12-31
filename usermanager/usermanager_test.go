package usermanager_test

import (
	"context"
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

	s := usermanager.NewStore(
		usermanager.NewDefaultDomainStore(db),
		usermanager.NewDefaultUserStore(db),
		usermanager.NewDefaultGroupStore(db),
		usermanager.NewDefaultAccessPolicyStore(db),
	)

	c := usermanager.NewConfig(
		context.Background(),
		s,
	)

	m, err := usermanager.NewUserManager(c)
	a.NoError(err)
	a.NotNil(m)
}
