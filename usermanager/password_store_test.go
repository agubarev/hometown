package usermanager_test

import (
	"testing"

	"github.com/agubarev/hometown/usermanager"

	"github.com/stretchr/testify/assert"
)

func TestPasswordStorePut(t *testing.T) {

	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := usermanager.NewPasswordStore(db)
	a.NoError(err)
	a.NotNil(s)

	p, err := usermanager.NewPassword(1, "namelimilenivonalimalovili", nil)
	a.NoError(err)
	a.NotNil(p)

	a.NoError(s.Create(p))
	a.NoError(s.Delete(p.OwnerID))
}

func TestPasswordStoreGet(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	s, err := usermanager.NewPasswordStore(db)
	a.NoError(err)
	a.NotNil(s)

	ownerID := int64(1)
	pass := "namelimilenivonalimalovili"

	p, err := usermanager.NewPassword(ownerID, pass, nil)
	a.NoError(err)
	a.NotNil(p)

	err = s.Create(p)
	a.NoError(err)

	p2, err := s.Get(ownerID)
	a.NoError(err)
	a.Len(p.Hash, len(p2.Hash))
	a.Equal(p.Hash, p2.Hash)
}

func TestPasswordStoreDelete(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	s, err := usermanager.NewPasswordStore(db)
	a.NoError(err)
	a.NotNil(s)

	ownerID := int64(1)
	pass := "namelimilenivonalimalovili"

	original, err := usermanager.NewPassword(ownerID, pass, nil)
	a.NoError(err)
	a.NotNil(original)

	err = s.Create(original)
	a.NoError(err)

	p, err := s.Get(ownerID)
	a.NoError(err)
	a.Len(p.Hash, len(original.Hash))
	a.Equal(p.Hash, original.Hash)

	err = s.Delete(ownerID)
	a.NoError(err)

	p2, err := s.Get(ownerID)
	a.Error(err)
	a.Nil(p2)
}
