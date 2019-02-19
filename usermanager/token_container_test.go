package usermanager_test

import (
	"os"
	"testing"
	"time"

	"gitlab.com/agubarev/hometown/util"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/usermanager"
)

func TestTokenContainerNewContainer(t *testing.T) {
	a := assert.New(t)

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)
	a.NoError(err)
	a.NotNil(db)

	s, err := usermanager.NewDefaultTokenStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := usermanager.NewTokenContainer(s)
	a.NoError(err)
	a.NotNil(c)
}

func TestTokenContainerNewToken(t *testing.T) {
	a := assert.New(t)

	id := util.NewULID()

	tok, err := usermanager.NewToken(usermanager.TkUserConfirmation, id, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(usermanager.TkUserConfirmation, tok.Kind)
	a.True(len(tok.Payload) > 0)
	a.False(tok.ExpireAt.IsZero())
}

func TestTokenContainerCreateGetAndDelete(t *testing.T) {
	a := assert.New(t)

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)
	a.NoError(err)
	a.NotNil(db)

	s, err := usermanager.NewDefaultTokenStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := usermanager.NewTokenContainer(s)
	a.NoError(err)
	a.NotNil(c)
	id := util.NewULID()

	// creating new token
	tok, err := c.Create(usermanager.TkUserConfirmation, id, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(usermanager.TkUserConfirmation, tok.Kind)
	a.True(len(tok.Payload) > 0)
	a.False(tok.ExpireAt.IsZero())

	// obtaining token from the container
	tok2, err := c.Get(tok.Token)
	a.NoError(err)
	a.NotNil(tok2)
	a.Equal(tok.Token, tok2.Token)
	a.True(tok.ExpireAt.Equal(tok2.ExpireAt))
	a.Equal(tok.Payload, tok2.Payload)
	a.Equal(tok.CheckinsLeft, tok2.CheckinsLeft)

	// trying to get nonexistent token
	nonexistentToken, err := c.Get("nonexistent token")
	a.EqualError(usermanager.ErrTokenNotFound, err.Error())
	a.Nil(nonexistentToken)

	// deleting token
	a.NoError(c.Delete(tok2.Token))

	// attempting to get it back from the container
	tok3, err := c.Get(tok.Token)
	a.EqualError(usermanager.ErrTokenNotFound, err.Error())
	a.Nil(tok3)
}
