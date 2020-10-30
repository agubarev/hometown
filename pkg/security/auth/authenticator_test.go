package auth_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/agubarev/hometown/pkg/client"
	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/security/auth"
	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/google/uuid"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/assert"
)

func TestAuthenticator_AuthenticateUserByPasswordNormal(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db := database.PostgreSQLForTesting(nil)
	a.NotNil(db)

	// initializing test user manager
	userManager, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(userManager)

	clientManager := client.NewManager(client.NewMemoryStore())
	a.NotNil(clientManager)

	// initializing accesspolicy manager
	authenticator, err := auth.NewAuthenticator(
		nil,
		userManager,
		clientManager,
		auth.NewDefaultRegistryBackend(),
		auth.DefaultOptions(),
	)
	a.NoError(err)
	a.NotNil(authenticator)

	// setting authenticator logger
	al, err := util.DefaultLogger(true, "")
	a.NoError(err)
	a.NotNil(al)
	a.NoError(authenticator.SetLogger(al))

	// generating test password
	testpass := password.NewRaw(32, 3, password.GFDefault)

	// creating test user
	testuser, err := user.CreateTestUser(ctx, userManager, "testuser", "testuser@hometown.local", testpass)
	a.NoError(err)
	a.NotNil(testuser)

	// normal case
	u, err := authenticator.AuthenticateUserByPassword(ctx, testuser.Username, testpass, auth.NewRequestMetadata(nil))
	a.NoError(err)
	a.NotNil(u)
	a.Equal(testuser.ID, u.ID)
	a.True(reflect.DeepEqual(testuser.Essential, u.Essential))

	// wrong username
	u, err = authenticator.AuthenticateUserByPassword(ctx, "wrongusername", testpass, auth.NewRequestMetadata(nil))
	a.EqualError(user.ErrUserNotFound, errors.Cause(err).Error())

	// wrong password
	u, err = authenticator.AuthenticateUserByPassword(ctx, testuser.Username, []byte("wrongpass"), auth.NewRequestMetadata(nil))
	a.EqualError(auth.ErrAuthenticationFailed, errors.Cause(err).Error())

	// empty password
	u, err = authenticator.AuthenticateUserByPassword(ctx, testuser.Username, []byte(""), auth.NewRequestMetadata(nil))
	a.EqualError(auth.ErrAuthenticationFailed, errors.Cause(err).Error())

	// nil password (just in case)
	u, err = authenticator.AuthenticateUserByPassword(ctx, testuser.Username, nil, auth.NewRequestMetadata(nil))
	a.EqualError(auth.ErrAuthenticationFailed, errors.Cause(err).Error())
}

func TestAuthenticator_AuthenticateSuspendedUserByPassword(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db := database.PostgreSQLForTesting(nil)
	a.NotNil(db)

	// initializing test user manager
	userManager, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(userManager)

	clientManager := client.NewManager(client.NewMemoryStore())
	a.NotNil(clientManager)

	// initializing accesspolicy manager
	authenticator, err := auth.NewAuthenticator(
		nil,
		userManager,
		clientManager,
		auth.NewDefaultRegistryBackend(),
		auth.DefaultOptions(),
	)
	a.NoError(err)
	a.NotNil(authenticator)

	// setting authenticator logger
	al, err := util.DefaultLogger(true, "")
	a.NoError(err)
	a.NotNil(al)
	a.NoError(authenticator.SetLogger(al))

	// generating test password
	testpass := password.NewRaw(32, 3, password.GFDefault)

	// creating test user
	testuser, err := user.CreateTestUser(ctx, userManager, "testuser", "testuser@hometown.local", testpass)
	a.NoError(err)
	a.NotNil(testuser)

	// suspending
	a.NoError(userManager.SuspendUser(ctx, testuser.ID, "suspended by test", time.Now().Add(24*time.Hour)))

	// normal case
	_, err = authenticator.AuthenticateUserByPassword(ctx, testuser.Username, testpass, auth.NewRequestMetadata(nil))
	a.Error(err)
	a.EqualError(err, auth.ErrUserSuspended.Error())

	// wrong username
	_, err = authenticator.AuthenticateUserByPassword(ctx, "wrongusername", testpass, auth.NewRequestMetadata(nil))
	a.EqualError(user.ErrUserNotFound, errors.Cause(err).Error())

	// wrong password
	_, err = authenticator.AuthenticateUserByPassword(ctx, testuser.Username, []byte("wrongpass"), auth.NewRequestMetadata(nil))
	a.EqualError(auth.ErrUserSuspended, errors.Cause(err).Error())

	// empty password
	_, err = authenticator.AuthenticateUserByPassword(ctx, testuser.Username, []byte(""), auth.NewRequestMetadata(nil))
	a.EqualError(auth.ErrUserSuspended, errors.Cause(err).Error())

	// nil password (just in case)
	_, err = authenticator.AuthenticateUserByPassword(ctx, testuser.Username, nil, auth.NewRequestMetadata(nil))
	a.EqualError(auth.ErrUserSuspended, errors.Cause(err).Error())
}

func TestAuthenticateByRefreshTokenNormalFlow(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db := database.PostgreSQLForTesting(nil)
	a.NotNil(db)

	// initializing test user manager
	userManager, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(userManager)

	passwordManager, err := password.NewManager(password.NewMemoryStore())
	a.NoError(err)
	a.NotNil(passwordManager)

	// initializing client manager
	clientManager := client.NewManager(client.NewMemoryStore())
	a.NotNil(clientManager)
	a.NoError(clientManager.SetPasswordManager(passwordManager))

	// initializing accesspolicy manager
	authenticator, err := auth.NewAuthenticator(
		nil,
		userManager,
		clientManager,
		auth.NewDefaultRegistryBackend(),
		auth.DefaultOptions(),
	)
	a.NoError(err)
	a.NotNil(authenticator)

	// setting authenticator logger
	al, err := util.DefaultLogger(true, "")
	a.NoError(err)
	a.NotNil(al)
	a.NoError(authenticator.SetLogger(al))

	// generating test password
	testpass := password.NewRaw(32, 3, password.GFDefault)

	//creating test user
	testuser, err := user.CreateTestUser(ctx, userManager, "testuser", "testuser@hometown.local", testpass)
	a.NoError(err)
	a.NotNil(testuser)

	// creating confidential client
	clnt, err := clientManager.CreateClient(ctx, "test client", client.FConfidential)
	a.NoError(err)
	a.NotNil(clnt)

	// creating client password
	clientPassword, err := clientManager.CreatePassword(ctx, clnt.ID)
	a.NoError(err)
	a.NotZero(clientPassword)

	// creating test session to obtain a token pair
	session1, tpair1, err := authenticator.CreateSessionWithRefreshToken(
		ctx,
		nil,
		clnt,
		auth.UserIdentity(testuser.ID),
		auth.NewRequestMetadata(nil),
	)
	a.NoError(err)
	a.NotNil(session1)
	a.NotEmpty(tpair1.AccessToken)
	a.NotZero(tpair1.RefreshToken)

	// obtaining current refresh token
	rtok1, err := authenticator.RefreshTokenByHash(ctx, tpair1.RefreshToken)
	a.NoError(err)
	a.True(rtok1.IsActive())
	a.True(rtok1.ExpireAt.After(time.Now()))
	a.False(rtok1.IsExpired())
	a.False(rtok1.IsRevoked())
	a.False(rtok1.IsRotated())

	// authenticating user by refresh token (rotating)
	session2, tpair2, err := authenticator.AuthenticateUserByRefreshToken(
		ctx,
		clnt,
		tpair1.RefreshToken,
		auth.NewRequestMetadata(nil),
	)
	a.NoError(err)
	a.NotNil(session2)
	a.NotEqual(tpair2.AccessToken, tpair1.AccessToken)
	a.NotEqual(tpair2.RefreshToken, tpair1.RefreshToken)
	a.NotEmpty(tpair2.AccessToken)
	a.NotZero(tpair2.RefreshToken)

	// obtaining new refresh token
	rtok2, err := authenticator.RefreshTokenByHash(ctx, tpair2.RefreshToken)
	a.NoError(err)
	a.True(rtok2.IsActive())
	a.True(rtok2.ExpireAt.After(time.Now()))
	a.False(rtok2.IsExpired())
	a.False(rtok2.IsRevoked())
	a.False(rtok2.IsRotated())
	a.Equal(rtok2.LastSessionID, session1.ID)
	a.Equal(rtok2.ParentID, rtok1.ID)

	// obtaining old refresh token
	rtok1, err = authenticator.RefreshTokenByHash(ctx, tpair1.RefreshToken)
	a.NoError(err)
	a.False(rtok1.IsActive())
	a.True(rtok1.ExpireAt.After(time.Now()))
	a.False(rtok1.IsExpired())
	a.False(rtok1.IsRevoked())
	a.True(rtok1.IsRotated())
	a.Equal(rtok1.LastSessionID, session1.ID)
	a.Equal(rtok1.RotatedID, rtok2.ID)
	a.Equal(rtok1.ParentID, uuid.Nil)
}

func TestAuthenticateByRevokedRefreshToken(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db := database.PostgreSQLForTesting(nil)
	a.NotNil(db)

	// initializing test user manager
	userManager, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(userManager)

	passwordManager, err := password.NewManager(password.NewMemoryStore())
	a.NoError(err)
	a.NotNil(passwordManager)

	// initializing client manager
	clientManager := client.NewManager(client.NewMemoryStore())
	a.NotNil(clientManager)
	a.NoError(clientManager.SetPasswordManager(passwordManager))

	// initializing accesspolicy manager
	authenticator, err := auth.NewAuthenticator(
		nil,
		userManager,
		clientManager,
		auth.NewDefaultRegistryBackend(),
		auth.DefaultOptions(),
	)
	a.NoError(err)
	a.NotNil(authenticator)

	// setting authenticator logger
	al, err := util.DefaultLogger(true, "")
	a.NoError(err)
	a.NotNil(al)
	a.NoError(authenticator.SetLogger(al))

	// generating test password
	testpass := password.NewRaw(32, 3, password.GFDefault)

	//creating test user
	testuser, err := user.CreateTestUser(ctx, userManager, "testuser", "testuser@hometown.local", testpass)
	a.NoError(err)
	a.NotNil(testuser)

	// creating confidential client
	clnt, err := clientManager.CreateClient(ctx, "test client", client.FConfidential)
	a.NoError(err)
	a.NotNil(clnt)

	// creating client password
	clientPassword, err := clientManager.CreatePassword(ctx, clnt.ID)
	a.NoError(err)
	a.NotZero(clientPassword)

	// creating test session to obtain a token pair
	session1, tpair1, err := authenticator.CreateSessionWithRefreshToken(
		ctx,
		nil,
		clnt,
		auth.UserIdentity(testuser.ID),
		auth.NewRequestMetadata(nil),
	)
	a.NoError(err)
	a.NotNil(session1)
	a.NotEmpty(tpair1.AccessToken)
	a.NotZero(tpair1.RefreshToken)

	// obtaining current refresh token
	rtok1, err := authenticator.RefreshTokenByHash(ctx, tpair1.RefreshToken)
	a.NoError(err)
	a.True(rtok1.IsActive())
	a.True(rtok1.ExpireAt.After(time.Now()))
	a.False(rtok1.IsExpired())
	a.False(rtok1.IsRevoked())
	a.False(rtok1.IsRotated())

	// revoking refresh token
	a.NoError(authenticator.RevokeRefreshToken(ctx, rtok1.Hash, "revoked by test"))

	// authenticating user by refresh token (rotating)
	session2, tpair2, err := authenticator.AuthenticateUserByRefreshToken(
		ctx,
		clnt,
		tpair1.RefreshToken,
		auth.NewRequestMetadata(nil),
	)
	a.Error(err)
	a.EqualError(errors.Cause(err), auth.ErrRefreshTokenRevoked.Error())
	a.Nil(session2)
	a.Empty(tpair2.AccessToken)
	a.Zero(tpair2.RefreshToken)
}

/*
func TestRevokeSession(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db := database.PostgreSQLConnection(nil)
	a.NotNil(db)

	// initializing test user manager
	userManager, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(userManager)

	clientManager := client.NewManager(client.NewMemoryStore())
	a.NotNil(clientManager)

	// initializing accesspolicy manager
	am, err := auth.NewAuthenticator(
		nil,
		userManager,
		clientManager,
		auth.NewDefaultRegistryBackend(),
		auth.DefaultOptions(),
	)
	a.NoError(err)
	a.NotNil(am)

	// setting authenticator logger
	al, err := util.DefaultLogger(true, "")
	a.NoError(err)
	a.NotNil(al)
	a.NoError(am.SetLogger(al))

	// generating test password
	testpass := password.NewRaw(32, 3, password.GFDefault)

	// creating user
	testuser, err := user.CreateTestUser(ctx, userManager, "testuser", "testuser@hometown.local", testpass)
	a.NoError(err)
	a.NotNil(testuser)

	// creating another user
	testuser2, err := user.CreateTestUser(ctx, userManager, "testuser2", "testuser2@hometown.local", testpass)
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
	u, err := am.AuthenticateUserByPassword(ctx, testuser.Username, testpass, correctMD)
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
	// first attempting to destroy session with a wrong user
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
*/
