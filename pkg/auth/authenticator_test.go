package auth_test

import (
	"net"
	"reflect"
	"testing"

	"github.com/agubarev/hometown/pkg/auth"
	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/user"

	"github.com/agubarev/hometown/pkg/util"
	"github.com/davecgh/go-spew/spew"
	"github.com/stretchr/testify/assert"
)

func TestAuthenticate(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// initializing test u manager
	um, err := user.NewUserManagerForTesting(db)
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

	// creating test u
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
	u, err := am.Authenticate(testuser.Username, testpass, auth.NewRequestInfo(nil))
	a.NoError(err)
	a.NotNil(u)
	a.True(reflect.DeepEqual(testuser, u))

	// ====================================================================================
	// wrong username
	// ====================================================================================
	u, err = am.Authenticate("wrongusername", testpass, auth.NewRequestInfo(nil))
	a.EqualError(user.ErrUserNotFound, err.Error())
	a.Nil(u)

	// ====================================================================================
	// wrong password
	// ====================================================================================
	u, err = am.Authenticate(testuser.Username, "wrongpass", auth.NewRequestInfo(nil))
	a.EqualError(auth.ErrAuthenticationFailed, err.Error())
	a.Nil(u)
}

func TestAuthenticateByRefreshToken(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// initializing test u manager
	um, err := user.NewUserManagerForTesting(db)
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

	// creating test u
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
	u, err := am.Authenticate(testuser.Username, testpass, auth.NewRequestInfo(nil))
	a.NoError(err)
	a.NotNil(u)
	a.True(reflect.DeepEqual(testuser, u))

	// ====================================================================================
	// wrong username
	// ====================================================================================
	u, err = am.Authenticate("wrongusername", testpass, auth.NewRequestInfo(nil))
	a.EqualError(user.ErrUserNotFound, err.Error())
	a.Nil(u)

	// ====================================================================================
	// wrong password
	// ====================================================================================
	u, err = am.Authenticate(testuser.Username, "wrongpass", auth.NewRequestInfo(nil))
	a.EqualError(auth.ErrAuthenticationFailed, err.Error())
	a.Nil(u)
}

func TestDestroySession(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// initializing test u manager
	um, err := user.NewUserManagerForTesting(db)
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

	// creating u
	testuser, err := um.CreateWithPassword(
		"testuser",
		"testuser@example.com",
		testpass,
		map[string]string{},
	)
	a.NoError(err)
	a.NotNil(testuser)

	// creating another u
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
	u, err := am.Authenticate(testuser.Username, testpass, correctMD)
	a.NoError(err)
	a.NotNil(u)
	a.True(reflect.DeepEqual(testuser, u))

	// generating token trinity for the u with correct request metadata
	tokenTrinity, err := am.GenerateTokenTrinity(u, correctMD)
	a.NoError(err)
	a.NotNil(tokenTrinity)

	// fetching session from the backend
	bs, err := am.GetSession(tokenTrinity.SessionToken)
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
		am.DestroySession(testuser2, bs.Token, correctMD),
		auth.ErrWrongUser.Error(),
	)

	// correct u but wrong IP
	a.EqualError(
		am.DestroySession(testuser, bs.Token, wrongIP),
		auth.ErrWrongIP.Error(),
	)

	// correct u but wrong u agent
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
