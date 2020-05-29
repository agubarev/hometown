package token_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/token"
	"github.com/agubarev/hometown/pkg/util"

	"github.com/stretchr/testify/assert"
)

func TestTokenContainerNewContainer(t *testing.T) {
	a := assert.New(t)

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := token.NewTokenStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := token.NewTokenManager(s)
	a.NoError(err)
	a.NotNil(c)
}

func TestTokenContainerNewToken(t *testing.T) {
	a := assert.New(t)

	id := util.NewULID()

	tok, err := token.NewToken(token.TkUserEmailConfirmation, id, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TkUserEmailConfirmation, tok.Kind)
	a.True(len(tok.Payload) > 0)
	a.False(tok.ExpireAt.IsZero())
}

func TestTokenContainerCreateGetAndDelete(t *testing.T) {
	a := assert.New(t)

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := token.NewTokenStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := token.NewTokenManager(s)
	a.NoError(err)
	a.NotNil(c)
	id := util.NewULID()

	// creating new token
	tok, err := c.Create(context.Background(), token.TkUserEmailConfirmation, id, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TkUserEmailConfirmation, tok.Kind)
	a.True(len(tok.Payload) > 0)
	a.False(tok.ExpireAt.IsZero())

	// obtaining token from the container
	tok2, err := c.Get(context.Background(), tok.Token)
	a.NoError(err)
	a.NotNil(tok2)
	a.Equal(tok.Token, tok2.Token)
	a.True(tok.ExpireAt.Equal(tok2.ExpireAt))
	a.Equal(tok.Payload, tok2.Payload)
	a.Equal(tok.CheckinRemainder, tok2.CheckinRemainder)

	// trying to get nonexistent token
	nonexistentToken, err := c.Get(context.Background(), "nonexistent token")
	a.EqualError(token.ErrTokenNotFound, err.Error())
	a.Nil(nonexistentToken)

	// deleting token
	a.NoError(c.Delete(context.Background(), tok2))

	// attempting to get it back from the container
	tok3, err := c.Get(context.Background(), tok.Token)
	a.EqualError(token.ErrTokenNotFound, err.Error())
	a.Nil(tok3)
}

func TestTokenContainerCheckin(t *testing.T) {
	a := assert.New(t)

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := token.NewTokenStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := token.NewTokenManager(s)
	a.NoError(err)
	a.NotNil(c)

	payload := util.NewULID()

	//---------------------------------------------------------------------------
	// checking in the token (1 checkin, must be removed after successful checkin)
	// this call must fail now, because there's no callback registered
	//---------------------------------------------------------------------------
	// creating new token; 10 sec expiration and one-time use
	tok, err := c.Create(context.Background(), token.TkUserEmailConfirmation, payload, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TkUserEmailConfirmation, tok.Kind)
	a.True(len(tok.Payload) > 0)
	a.False(tok.ExpireAt.IsZero())
	a.Equal(1, tok.CheckinRemainder)

	callbackID := "test callback id"

	// registering a proper callback
	flagSwitch := false
	err = c.AddCallback(tok.Kind, callbackID, func(ctx context.Context, t *token.Token) error {
		flagSwitch = true
		return nil
	})
	a.NoError(err)

	// must pass
	err = c.Checkin(context.Background(), tok.Token)
	a.NoError(err)
	a.True(flagSwitch)

	// token must be void and missing now
	tok, err = c.Get(context.Background(), tok.Token)
	a.EqualError(token.ErrTokenNotFound, err.Error())
	a.Nil(tok)

	//---------------------------------------------------------------------------
	// now everything is the same as above, except that this token has 2 uses
	// token must still be intact after one checkin
	//---------------------------------------------------------------------------
	// removing previous callback
	err = c.RemoveCallback(callbackID)
	a.NoError(err)

	// creating new token; 10 sec expiration and two-time use
	tok, err = c.Create(context.Background(), token.TkUserEmailConfirmation, payload, 10*time.Second, 2)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TkUserEmailConfirmation, tok.Kind)
	a.True(len(tok.Payload) > 0)
	a.False(tok.ExpireAt.IsZero())
	a.Equal(2, tok.CheckinRemainder)

	// registering a proper callback
	flagSwitch = false
	err = c.AddCallback(tok.Kind, callbackID, func(ctx context.Context, t *token.Token) error {
		flagSwitch = true
		return nil
	})
	a.NoError(err)

	// must pass
	err = c.Checkin(context.Background(), tok.Token)
	a.NoError(err)
	a.True(flagSwitch)

	// token must exist but have just 1 checkin remaining
	tok, err = c.Get(context.Background(), tok.Token)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TkUserEmailConfirmation, tok.Kind)
	a.True(len(tok.Payload) > 0)
	a.False(tok.ExpireAt.IsZero())
	a.Equal(1, tok.CheckinRemainder)

	//---------------------------------------------------------------------------
	// now testing failing mechanism
	//---------------------------------------------------------------------------
	// removing previous callback
	err = c.RemoveCallback(callbackID)
	a.NoError(err)

	// creating new token
	tok, err = c.Create(context.Background(), token.TkUserEmailConfirmation, payload, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TkUserEmailConfirmation, tok.Kind)
	a.True(len(tok.Payload) > 0)
	a.False(tok.ExpireAt.IsZero())
	a.Equal(1, tok.CheckinRemainder)

	// checkin must fail due to not having a preregistered callback for this token's kind
	err = c.Checkin(context.Background(), tok.Token)
	a.EqualError(token.ErrTokenCallbackNotFound, err.Error())

	// token must be still present
	tok, err = c.Get(context.Background(), tok.Token)
	a.NoError(err)
	a.NotNil(tok)

	// registering a proper callback but will return an error
	flagSwitch = false
	err = c.AddCallback(tok.Kind, callbackID, func(ctx context.Context, t *token.Token) error {
		// flipping this flag only to make sure that this callback has been called
		flagSwitch = true
		return errors.New("some error")
	})
	a.NoError(err)

	// this checkin must fail due to callback returning an err
	err = c.Checkin(context.Background(), tok.Token)
	a.Error(err)
	a.True(flagSwitch)

	// token must still exist
	tok, err = c.Get(context.Background(), tok.Token)
	a.NoError(err)
	a.NotNil(tok)
}

func TestTokenContainerAddCallback(t *testing.T) {
	a := assert.New(t)

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := token.NewTokenStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := token.NewTokenManager(s)
	a.NoError(err)
	a.NotNil(c)

	payload := util.NewULID()

	tok, err := c.Create(context.Background(), token.TkUserEmailConfirmation, payload, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TkUserEmailConfirmation, tok.Kind)
	a.True(len(tok.Payload) > 0)
	a.False(tok.ExpireAt.IsZero())
	a.Equal(1, tok.CheckinRemainder)

	validID, wrongID := "test callback id", "wrong callback id"

	// registering a proper callback
	err = c.AddCallback(tok.Kind, validID, func(ctx context.Context, t *token.Token) error {
		return nil
	})
	a.NoError(err)

	// attempting to register again with the same id
	err = c.AddCallback(tok.Kind, validID, func(ctx context.Context, t *token.Token) error {
		return nil
	})
	a.EqualError(token.ErrTokenDuplicateCallbackID, err.Error())

	cb, err := c.GetCallback(validID)
	a.NoError(err)
	a.NotNil(cb)
	a.Equal(validID, cb.ID)
	a.Equal(tok.Kind, cb.Kind)
	a.NotNil(cb.Function)

	// slice length
	a.Len(c.GetCallbacks(tok.Kind), 1)

	cb, err = c.GetCallback(wrongID)
	a.Error(token.ErrTokenCallbackNotFound, err.Error())
	a.Nil(cb)
}

func TestTokenContainerRemoveCallback(t *testing.T) {
	a := assert.New(t)

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := token.NewTokenStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := token.NewTokenManager(s)
	a.NoError(err)
	a.NotNil(c)

	payload := util.NewULID()

	tok, err := c.Create(context.Background(), token.TkUserEmailConfirmation, payload, 10*time.Second, 1)
	a.NoError(err)
	a.NotNil(tok)
	a.Equal(token.TkUserEmailConfirmation, tok.Kind)
	a.True(len(tok.Payload) > 0)
	a.False(tok.ExpireAt.IsZero())
	a.Equal(1, tok.CheckinRemainder)

	callbackID := "test callback id"

	// registering callback
	err = c.AddCallback(tok.Kind, callbackID, func(ctx context.Context, t *token.Token) error {
		return nil
	})
	a.NoError(err)

	cb, err := c.GetCallback(callbackID)
	a.NoError(err)
	a.NotNil(cb)
	a.Equal(callbackID, cb.ID)
	a.Equal(tok.Kind, cb.Kind)
	a.NotNil(cb.Function)

	// removing callback
	a.NoError(c.RemoveCallback(callbackID))

	// slice length
	a.Len(c.GetCallbacks(tok.Kind), 0)

	// callback must not exist now
	cb, err = c.GetCallback(callbackID)
	a.Error(token.ErrTokenCallbackNotFound, err.Error())
	a.Nil(cb)
}

func TestTokenContainerCleanup(t *testing.T) {
	a := assert.New(t)

	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	s, err := token.NewTokenStore(db)
	a.NoError(err)
	a.NotNil(s)

	c, err := token.NewTokenManager(s)
	a.NoError(err)
	a.NotNil(c)

	c.Create(context.Background(), token.TkUserEmailConfirmation, util.NewULID(), 4*time.Second, 1)
	c.Create(context.Background(), token.TkUserEmailConfirmation, util.NewULID(), 4*time.Second, 1)
	c.Create(context.Background(), token.TkUserEmailConfirmation, util.NewULID(), 4*time.Second, 1)
	c.Create(context.Background(), token.TkUserEmailConfirmation, util.NewULID(), 6*time.Second, 1)
	c.Create(context.Background(), token.TkUserEmailConfirmation, util.NewULID(), 7*time.Second, 1)
	c.Create(context.Background(), token.TkUserEmailConfirmation, util.NewULID(), 8*time.Second, 1)
	c.Create(context.Background(), token.TkUserEmailConfirmation, util.NewULID(), 16*time.Second, 1)

	a.Len(c.List(token.TkAllTokens), 7)

	// TODO: time travel
}
