package password_test

import (
	"context"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/google/uuid"

	"github.com/stretchr/testify/assert"
)

func TestPasswordStorePut(t *testing.T) {
	a := assert.New(t)

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := password.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	p, err := password.New(password.OKUser, uuid.New(), []byte("namelimilenivonalimalovili"), nil)
	a.NoError(err)
	a.NotNil(p)

	a.NoError(s.Upsert(context.Background(), p))
	a.NoError(s.Delete(context.Background(), password.OKUser, p.OwnerID))
}

func TestPasswordStoreGet(t *testing.T) {
	a := assert.New(t)

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := password.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	ownerID := uuid.New()
	pass := []byte("namelimilenivonalimalovili")

	p, err := password.New(password.OKUser, ownerID, pass, nil)
	a.NoError(err)
	a.NotNil(p)

	err = s.Upsert(context.Background(), p)
	a.NoError(err)

	p2, err := s.Get(context.Background(), password.OKUser, ownerID)
	a.NoError(err)
	a.Len(p.Hash, len(p2.Hash))
	a.Equal(p.Hash, p2.Hash)
}

func TestPasswordStoreDelete(t *testing.T) {
	a := assert.New(t)

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := password.NewPostgreSQLStore(db)
	a.NoError(err)
	a.NotNil(s)

	ownerID := uuid.New()
	pass := []byte("namelimilenivonalimalovili")

	original, err := password.New(password.OKUser, ownerID, pass, nil)
	a.NoError(err)
	a.NotNil(original)

	err = s.Upsert(context.Background(), original)
	a.NoError(err)

	p, err := s.Get(context.Background(), password.OKUser, ownerID)
	a.NoError(err)
	a.Len(p.Hash, len(original.Hash))
	a.Equal(p.Hash, original.Hash)

	err = s.Delete(context.Background(), password.OKUser, ownerID)
	a.NoError(err)

	_, err = s.Get(context.Background(), password.OKUser, ownerID)
	a.Error(err)
}
