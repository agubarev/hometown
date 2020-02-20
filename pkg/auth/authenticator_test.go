package auth_test

import (
	"net"
	"reflect"
	"testing"

	"github.com/agubarev/hometown/pkg/auth"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestAuthenticate(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)
	a.NoError(core.TruncateDatabaseForTesting(db))

	// initializing test user manager
	um, err := core.NewUserManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	// initializing access manager
	am, err := auth.NewAuthenticator(nil, um, nil)
	a.NoError(err)
	a.NotNil(am)

	// setting authenticator logger
	al, err := util.DefaultLogger(true, "")
	a.NoError(err)
	a.NotNil(al)
	a.NoError(am.SetLogger(al))

	// using ULID as a random password
	testpass := util.NewULID().String()

	// creating test user
	testuser, err := um.CreateWithPassword(
		"testuser",
		"testuser@example.com",
		testpass,
		map[string]string{},
	)
	a.NoError(err)
	a.NotNil(testuser)

	// ====================================================================================
	// normal case
	// ====================================================================================
	user, err := am.Authenticate(testuser.Username, testpass, auth.NewRequestInfo(nil))
	a.NoError(err)
	a.NotNil(user)
	a.True(reflect.DeepEqual(testuser, user))

	// ====================================================================================
	// wrong username
	// ====================================================================================
	user, err = am.Authenticate("wrongusername", testpass, auth.NewRequestInfo(nil))
	a.EqualError(core.ErrUserNotFound, err.Error())
	a.Nil(user)

	// ====================================================================================
	// wrong password
	// ====================================================================================
	user, err = am.Authenticate(testuser.Username, "wrongpass", auth.NewRequestInfo(nil))
	a.EqualError(auth.ErrAuthenticationFailed, err.Error())
	a.Nil(user)
}

func TestAuthenticateByRefreshToken(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)
	a.NoError(core.TruncateDatabaseForTesting(db))

	// initializing test user manager
	um, err := core.NewUserManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	// initializing access manager
	am, err := auth.NewAuthenticator(nil, um, nil)
	a.NoError(err)
	a.NotNil(am)

	// setting authenticator logger
	al, err := util.DefaultLogger(true, "")
	a.NoError(err)
	a.NotNil(al)
	a.NoError(am.SetLogger(al))

	// using ULID as a random password
	testpass := util.NewULID().String()

	// creating test user
	testuser, err := um.CreateWithPassword(
		"testuser",
		"testuser@example.com",
		testpass,
		map[string]string{},
	)
	a.NoError(err)
	a.NotNil(testuser)

	// ====================================================================================
	// normal case
	// ====================================================================================
	user, err := am.Authenticate(testuser.Username, testpass, auth.NewRequestInfo(nil))
	a.NoError(err)
	a.NotNil(user)
	a.True(reflect.DeepEqual(testuser, user))

	// ====================================================================================
	// wrong username
	// ====================================================================================
	user, err = am.Authenticate("wrongusername", testpass, auth.NewRequestInfo(nil))
	a.EqualError(core.ErrUserNotFound, err.Error())
	a.Nil(user)

	// ====================================================================================
	// wrong password
	// ====================================================================================
	user, err = am.Authenticate(testuser.Username, "wrongpass", auth.NewRequestInfo(nil))
	a.EqualError(auth.ErrAuthenticationFailed, err.Error())
	a.Nil(user)
}

func TestDestroySession(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)
	a.NoError(core.TruncateDatabaseForTesting(db))

	// initializing test user manager
	um, err := core.NewUserManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	// initializing access manager
	am, err := auth.NewAuthenticator(nil, um, auth.NewDefaultRegistryBackend())
	a.NoError(err)
	a.NotNil(am)

	// setting authenticator logger
	al, err := util.DefaultLogger(true, "")
	a.NoError(err)
	a.NotNil(al)
	a.NoError(am.SetLogger(al))

	// using ULID as a random password
	testpass := util.NewULID().String()

	// creating user
	testuser, err := um.CreateWithPassword(
		"testuser",
		"testuser@example.com",
		testpass,
		map[string]string{},
	)
	a.NoError(err)
	a.NotNil(testuser)

	// creating another user
	testuser2, err := um.CreateWithPassword(
		"testuser2",
		"testuser2@example.com",
		testpass,
		map[string]string{},
	)
	a.NoError(err)
	a.NotNil(testuser)

	// different RequestInfos
	correctMD := &auth.RequestMetadata{
		IP:        net.IPv4(127, 0, 0, 1),
		UserAgent: "correct user-agent",
	}

	wrongIP := &auth.RequestMetadata{
		IP:        net.IPv4(127, 0, 0, 2),
		UserAgent: "correct user-agent",
	}

	wrongUserAgent := &auth.RequestMetadata{
		IP:        net.IPv4(127, 0, 0, 1),
		UserAgent: "wrong user-agent",
	}

	// authentication is not necessary for this test, just keeps things consistent
	user, err := am.Authenticate(testuser.Username, testpass, correctMD)
	a.NoError(err)
	a.NotNil(user)
	a.True(reflect.DeepEqual(testuser, user))

	// generating token trinity for the user with correct request metadata
	tokenTrinity, err := am.GenerateTokenTrinity(user, correctMD)
	a.NoError(err)
	a.NotNil(tokenTrinity)

	// fetching session from the backend
	bs, err := am.GetSession(tokenTrinity.SessionToken)
	a.NoError(err)
	a.NotNil(bs)
	a.Equal(user.ID, bs.UserID)

	// ====================================================================================
	// testing the actual session destruction
	// ====================================================================================
	// first attempting to destroy session with a wrong user
	// this session was created for "testuser", attempting to destroy
	// by "testuser2"
	a.EqualError(
		am.DestroySession(testuser2, bs.Token, correctMD),
		auth.ErrWrongUser.Error(),
	)

	// correct user but wrong IP
	a.EqualError(
		am.DestroySession(testuser, bs.Token, wrongIP),
		auth.ErrWrongIP.Error(),
	)

	// correct user but wrong user agent
	a.EqualError(
		am.DestroySession(testuser, bs.Token, wrongUserAgent),
		auth.ErrWrongUserAgent.Error(),
	)

	spew.Dump(tokenTrinity)
	spew.Dump(bs)

	// and finally, everything should be correct
	a.NoError(am.DestroySession(testuser, bs.Token, correctMD))

	// ====================================================================================
	// making sure that session doesn't exist anymore and its
	// corresponding access token is revoked properly
	// ====================================================================================
	s, err := am.GetSession(bs.Token)
	a.EqualError(err, auth.ErrSessionNotFound.Error())
	a.Nil(s)

	// checking whether the access token is revoked
	a.True(am.IsRevoked(s.AccessTokenID))
}
