package token_test

import (
	"context"
	"testing"
	"time"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/token"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestTokenContainerNewContainer(t *testing.T) {
	a := assert.New(t)

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := token.NewStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := token.NewManager(s)
	a.NoError(err)
	a.NotNil(c)
}

func TestTokenContainerNewToken(t *testing.T) {
	a := assert.New(t)

	tok, err := token.NewToken(token.TEmailConfirmation, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TEmailConfirmation, tok.Kind)
	a.NotZero(tok.ExpireAt)
}

func TestTokenContainerCreateGetAndDelete(t *testing.T) {
	a := assert.New(t)

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := token.NewStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := token.NewManager(s)
	a.NoError(err)
	a.NotNil(c)

	// creating new token
	tok, err := c.Create(context.Background(), token.TEmailConfirmation, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TEmailConfirmation, tok.Kind)
	a.NotZero(tok.ExpireAt)

	// obtaining token from the container
	tok2, err := c.Get(context.Background(), tok.Hash)
	a.NoError(err)
	a.NotNil(tok2)
	a.Equal(tok.Hash, tok2.Hash)
	a.True(tok.ExpireAt == tok2.ExpireAt)
	a.Equal(tok.CheckinRemainder, tok2.CheckinRemainder)

	// trying to get nonexistent token
	_, err = c.Get(context.Background(), token.NewHash())
	a.EqualError(token.ErrTokenNotFound, errors.Cause(err).Error())

	// deleting token
	a.NoError(c.Delete(context.Background(), tok2))

	// attempting to get it back from the container
	_, err = c.Get(context.Background(), tok.Hash)
	a.EqualError(token.ErrTokenNotFound, errors.Cause(err).Error())
}

func TestTokenContainerCheckin(t *testing.T) {
	a := assert.New(t)

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := token.NewStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := token.NewManager(s)
	a.NoError(err)
	a.NotNil(c)

	//---------------------------------------------------------------------------
	// checking in the token (1 checkin, must be removed after successful checkin)
	// this call must fail now, because there's no callback registered
	//---------------------------------------------------------------------------
	// creating new token; 10 sec expiration and one-time use
	tok, err := c.Create(context.Background(), token.TEmailConfirmation, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TEmailConfirmation, tok.Kind)
	a.NotZero(tok.ExpireAt)
	a.Equal(tok.CheckinRemainder, int32(1))

	callbackName := token.CallbackName("test callback name")

	// registering a proper callback
	flagSwitch := false
	a.NoError(c.AddCallback(tok.Kind, callbackName, func(ctx context.Context, t token.Token) error {
		flagSwitch = true
		return nil
	}))

	// must pass
	err = c.Checkin(context.Background(), tok.Hash)
	a.NoError(err)
	a.True(flagSwitch)

	// token must be void and missing now
	tok, err = c.Get(context.Background(), tok.Hash)
	a.Error(err)
	a.EqualError(token.ErrTokenNotFound, errors.Cause(err).Error())

	//---------------------------------------------------------------------------
	// now everything is the same as above, except that this token has 2 uses
	// token must still be intact after one checkin
	//---------------------------------------------------------------------------
	// removing previous callback
	err = c.RemoveCallback(callbackName)
	a.NoError(err)

	// creating new token; 10 sec expiration and two-time use
	tok, err = c.Create(context.Background(), token.TEmailConfirmation, 10*time.Second, 2)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TEmailConfirmation, tok.Kind)
	a.Equal(tok.CheckinRemainder, int32(2))

	// registering a proper callback
	flagSwitch = false
	a.NoError(c.AddCallback(tok.Kind, callbackName, func(ctx context.Context, t token.Token) error {
		flagSwitch = true
		return nil
	}))

	// must pass
	err = c.Checkin(context.Background(), tok.Hash)
	a.NoError(err)
	a.True(flagSwitch)

	// token must exist but have just 1 checkin remaining
	tok, err = c.Get(context.Background(), tok.Hash)
	a.NoError(err)
	a.Equal(token.TEmailConfirmation, tok.Kind)
	a.NotZero(tok.ExpireAt)
	a.Equal(tok.CheckinRemainder, int32(1))

	//---------------------------------------------------------------------------
	// now testing failing mechanism
	//---------------------------------------------------------------------------
	// removing previous callback
	err = c.RemoveCallback(callbackName)
	a.NoError(err)

	// creating new token
	tok, err = c.Create(context.Background(), token.TEmailConfirmation, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TEmailConfirmation, tok.Kind)
	a.NotZero(tok.ExpireAt)
	a.Equal(tok.CheckinRemainder, int32(1))

	// checkin must fail due to not having a preregistered callback for this token's kind
	err = c.Checkin(context.Background(), tok.Hash)
	a.EqualError(token.ErrTokenCallbackNotFound, err.Error())

	// token must be still present
	tok, err = c.Get(context.Background(), tok.Hash)
	a.NoError(err)
	a.NotNil(tok)

	// registering a proper callback but will return an error
	flagSwitch = false
	err = c.AddCallback(tok.Kind, callbackName, func(ctx context.Context, t token.Token) error {
		// flipping this flag only to make sure that this callback has been called
		flagSwitch = true
		return errors.New("some error")
	})
	a.NoError(err)

	// this checkin must fail due to callback returning an err
	err = c.Checkin(context.Background(), tok.Hash)
	a.Error(err)
	a.True(flagSwitch)

	// token must still exist
	tok, err = c.Get(context.Background(), tok.Hash)
	a.NoError(err)
	a.NotNil(tok)
}

func TestTokenContainerAddCallback(t *testing.T) {
	a := assert.New(t)

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := token.NewStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := token.NewManager(s)
	a.NoError(err)
	a.NotNil(c)

	tok, err := c.Create(context.Background(), token.TEmailConfirmation, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TEmailConfirmation, tok.Kind)
	a.NotZero(tok.ExpireAt)
	a.Equal(tok.CheckinRemainder, int32(1))

	validName := token.CallbackName("test callback name")
	wrongName := token.CallbackName("wrong callback name")

	// registering a proper callback
	err = c.AddCallback(tok.Kind, validName, func(ctx context.Context, t token.Token) error {
		return nil
	})
	a.NoError(err)

	// attempting to register again with the same id
	err = c.AddCallback(tok.Kind, validName, func(ctx context.Context, t token.Token) error {
		return nil
	})
	a.Error(err)
	a.EqualError(token.ErrTokenDuplicateCallbackID, err.Error())

	cb, err := c.GetCallback(validName)
	a.NoError(err)
	a.NotNil(cb)
	a.Equal(validName, cb.Name)
	a.Equal(tok.Kind, cb.Kind)
	a.NotNil(cb.Function)

	// slice length
	a.Len(c.GetCallbacks(tok.Kind), 1)

	cb, err = c.GetCallback(wrongName)
	a.Error(err)
	a.EqualError(token.ErrTokenCallbackNotFound, err.Error())
	a.Nil(cb)
}

func TestTokenContainerRemoveCallback(t *testing.T) {
	a := assert.New(t)

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := token.NewStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := token.NewManager(s)
	a.NoError(err)
	a.NotNil(c)

	tok, err := c.Create(context.Background(), token.TEmailConfirmation, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TEmailConfirmation, tok.Kind)
	a.NotZero(tok.ExpireAt)
	a.Equal(tok.CheckinRemainder, int32(1))

	callbackName := token.CallbackName("test callback name")

	// registering callback
	err = c.AddCallback(tok.Kind, callbackName, func(ctx context.Context, t token.Token) error {
		return nil
	})
	a.NoError(err)

	cb, err := c.GetCallback(callbackName)
	a.NoError(err)
	a.NotNil(cb)
	a.Equal(callbackName, cb.Name)
	a.Equal(tok.Kind, cb.Kind)
	a.NotNil(cb.Function)

	// removing callback
	a.NoError(c.RemoveCallback(callbackName))

	// slice length
	a.Len(c.GetCallbacks(tok.Kind), 0)

	// callback must not exist now
	cb, err = c.GetCallback(callbackName)
	a.Error(err)
	a.EqualError(token.ErrTokenCallbackNotFound, err.Error())
	a.Nil(cb)
}

func TestTokenContainerCleanup(t *testing.T) {
	a := assert.New(t)

	db, err := database.PostgreSQLForTesting(nil)
	a.NoError(err)
	a.NotNil(db)

	s, err := token.NewStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := token.NewManager(s)
	a.NoError(err)
	a.NotNil(c)

	c.Create(context.Background(), token.TEmailConfirmation, 4*time.Second, 1)
	c.Create(context.Background(), token.TEmailConfirmation, 4*time.Second, 1)
	c.Create(context.Background(), token.TEmailConfirmation, 4*time.Second, 1)
	c.Create(context.Background(), token.TEmailConfirmation, 6*time.Second, 1)
	c.Create(context.Background(), token.TEmailConfirmation, 7*time.Second, 1)
	c.Create(context.Background(), token.TEmailConfirmation, 8*time.Second, 1)
	c.Create(context.Background(), token.TEmailConfirmation, 16*time.Second, 1)

	a.Len(c.List(token.TAll), 7)

	// TODO: time travel
}
