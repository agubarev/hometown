package auth_test

import (
	"net"
	"reflect"
	"testing"

	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/security/auth"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestAuthenticate(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// initializing test user manager
	um, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	// initializing accesspolicy manager
	am, err := auth.NewAuthenticator(nil, um, auth.NewDefaultRegistryBackend(), auth.DefaultOptions())
	a.NoError(err)
	a.NotNil(am)

	// setting authenticator logger
	al, err := util.DefaultLogger(true, "")
	a.NoError(err)
	a.NotNil(al)
	a.NoError(am.SetLogger(al))

	// using ULID as a random password
	testpass := util.NewULID().Entropy()

	// creating test user
	testuser, err := user.CreateTestUser(ctx, um, "testuser", "testuser@hometown.local", testpass)
	a.NoError(err)
	a.NotNil(testuser)

	// ====================================================================================
	// normal case
	// ====================================================================================
	u, err := am.AuthenticateByCredentials(ctx, testuser.Username, testpass, auth.NewRequestMetadata(nil))
	a.NoError(err)
	a.NotNil(u)
	a.Equal(testuser.ID, u.ID)
	a.True(reflect.DeepEqual(testuser.Essential, u.Essential))

	// ====================================================================================
	// wrong username
	// ====================================================================================
	u, err = am.AuthenticateByCredentials(ctx, "wrongusername", testpass, auth.NewRequestMetadata(nil))
	a.EqualError(user.ErrUserNotFound, errors.Cause(err).Error())

	// ====================================================================================
	// wrong password
	// ====================================================================================
	u, err = am.AuthenticateByCredentials(ctx, testuser.Username, []byte("wrongpass"), auth.NewRequestMetadata(nil))
	a.EqualError(auth.ErrAuthenticationFailed, errors.Cause(err).Error())
}

func TestAuthenticateByRefreshToken(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// initializing test user manager
	um, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	// initializing accesspolicy manager
	am, err := auth.NewAuthenticator(nil, um, auth.NewDefaultRegistryBackend(), auth.DefaultOptions())
	a.NoError(err)
	a.NotNil(am)

	// setting authenticator logger
	al, err := util.DefaultLogger(true, "")
	a.NoError(err)
	a.NotNil(al)
	a.NoError(am.SetLogger(al))

	// using ULID as a random password
	testpass := util.NewULID().Entropy()

	// creating test u
	testuser, err := user.CreateTestUser(ctx, um, "testuser", "testuser@hometown.local", testpass)
	a.NoError(err)
	a.NotNil(testuser)

	// ====================================================================================
	// normal case
	// ====================================================================================
	u, err := am.AuthenticateByCredentials(ctx, testuser.Username, testpass, auth.NewRequestMetadata(nil))
	a.NoError(err)
	a.NotZero(u.ID)
	a.Equal(testuser.ID, u.ID)
	a.True(reflect.DeepEqual(testuser.Essential, u.Essential))

	// ====================================================================================
	// wrong username
	// ====================================================================================
	u, err = am.AuthenticateByCredentials(ctx, "wrongusername", testpass, auth.NewRequestMetadata(nil))
	a.EqualError(user.ErrUserNotFound, errors.Cause(err).Error())

	// ====================================================================================
	// wrong password
	// ====================================================================================
	u, err = am.AuthenticateByCredentials(ctx, testuser.Username, []byte("wrongpass"), auth.NewRequestMetadata(nil))
	a.EqualError(auth.ErrAuthenticationFailed, errors.Cause(err).Error())
}

func TestDestroySession(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// initializing test user manager
	um, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	// initializing accesspolicy manager
	am, err := auth.NewAuthenticator(nil, um, auth.NewDefaultRegistryBackend(), auth.DefaultOptions())
	a.NoError(err)
	a.NotNil(am)

	// setting authenticator logger
	al, err := util.DefaultLogger(true, "")
	a.NoError(err)
	a.NotNil(al)
	a.NoError(am.SetLogger(al))

	// using ULID as a random password
	testpass := util.NewULID().Entropy()

	// creating user
	testuser, err := user.CreateTestUser(ctx, um, "testuser", "testuser@hometown.local", testpass)
	a.NoError(err)
	a.NotNil(testuser)

	// creating another user
	testuser2, err := user.CreateTestUser(ctx, um, "testuser2", "testuser2@hometown.local", testpass)
	a.NoError(err)
	a.NotNil(testuser)

	// different RequestInfos
	correctMD := &auth.RequestMetadata{
		IP:        net.IPv4(127, 0, 0, 1),
		UserAgent: "correct u-agent",
	}

	wrongIP := &auth.RequestMetadata{
		IP:        net.IPv4(127, 0, 0, 2),
		UserAgent: "correct u-agent",
	}

	wrongUserAgent := &auth.RequestMetadata{
		IP:        net.IPv4(127, 0, 0, 1),
		UserAgent: "wrong u-agent",
	}

	// authentication is not necessary for this test, just keeps things consistent
	u, err := am.AuthenticateByCredentials(ctx, testuser.Username, testpass, correctMD)
	a.NoError(err)
	a.NotNil(u)
	a.True(reflect.DeepEqual(testuser, u))

	// generating token trinity for the user with correct request metadata
	tokenTrinity, err := am.GenerateTokenTrinity(ctx, u, correctMD)
	a.NoError(err)
	a.NotNil(tokenTrinity)

	// fetching session from the backend
	bs, err := am.SessionByID(tokenTrinity.SessionToken)
	a.NoError(err)
	a.NotNil(bs)
	a.Equal(u.ID, bs.UserID)

	// ====================================================================================
	// testing the actual session destruction
	// ====================================================================================
	// first attempting to destroy session with a wrong u
	// this session was created for "testuser", attempting to destroy
	// by "testuser2"
	a.EqualError(
		am.DestroySession(ctx, testuser2.ID, bs.Token, correctMD),
		auth.ErrIdentityMismatch.Error(),
	)

	// correct user but wrong IPAddr
	a.EqualError(
		am.DestroySession(ctx, testuser.ID, bs.Token, wrongIP),
		auth.ErrIPAddrMismatch.Error(),
	)

	// correct user but wrong user agent
	a.EqualError(
		am.DestroySession(ctx, testuser.ID, bs.Token, wrongUserAgent),
		auth.ErrUserAgentMismatch.Error(),
	)

	spew.Dump(tokenTrinity)
	spew.Dump(bs)

	// and finally, everything should be correct
	a.NoError(am.DestroySession(ctx, testuser.ID, bs.Token, correctMD))

	// ====================================================================================
	// making sure that session doesn't exist anymore and its
	// corresponding accesspolicy token is revoked properly
	// ====================================================================================
	s, err := am.SessionByID(bs.Token)
	a.EqualError(err, auth.ErrSessionNotFound.Error())
	a.Nil(s)

	// checking whether the accesspolicy token is revoked
	a.True(am.IsRevoked(s.JTI))
}
