package password_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/password"

	"github.com/stretchr/testify/assert"
)

func TestPasswordStorePut(t *testing.T) {
	a := assert.New(t)

	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := password.NewPasswordStore(db)
	a.NoError(err)
	a.NotNil(s)

	p, err := password.New(1, "namelimilenivonalimalovili", nil)
	a.NoError(err)
	a.NotNil(p)

	a.NoError(s.Upsert(context.TODO(), p))
	a.NoError(s.Delete(context.TODO(), p.OwnerID))
}

func TestPasswordStoreGet(t *testing.T) {
	a := assert.New(t)

	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(core.TruncateDatabaseForTesting(db))

	s, err := password.NewPasswordStore(db)
	a.NoError(err)
	a.NotNil(s)

	ownerID := int64(1)
	pass := "namelimilenivonalimalovili"

	p, err := password.New(ownerID, pass, nil)
	a.NoError(err)
	a.NotNil(p)

	err = s.Upsert(context.TODO(), p)
	a.NoError(err)

	p2, err := s.Get(context.TODO(), ownerID)
	a.NoError(err)
	a.Len(p.Hash, len(p2.Hash))
	a.Equal(p.Hash, p2.Hash)
}

func TestPasswordStoreDelete(t *testing.T) {
	a := assert.New(t)

	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(core.TruncateDatabaseForTesting(db))

	s, err := password.NewPasswordStore(db)
	a.NoError(err)
	a.NotNil(s)

	ownerID := int64(1)
	pass := "namelimilenivonalimalovili"

	original, err := password.New(ownerID, pass, nil)
	a.NoError(err)
	a.NotNil(original)

	err = s.Upsert(context.TODO(), original)
	a.NoError(err)

	p, err := s.Get(context.TODO(), ownerID)
	a.NoError(err)
	a.Len(p.Hash, len(original.Hash))
	a.Equal(p.Hash, original.Hash)

	err = s.Delete(context.TODO(), ownerID)
	a.NoError(err)

	p2, err := s.Get(context.TODO(), ownerID)
	a.Error(err)
	a.Nil(p2)
}
