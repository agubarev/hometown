package endpoints_test

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agubarev/hometown/pkg/client"
	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/security/auth"
	"github.com/agubarev/hometown/pkg/security/auth/provider/endpoints"
	"github.com/agubarev/hometown/pkg/security/password"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/go-chi/chi"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestMiddlewareAuthenticatorNormal(t *testing.T) {
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

	// injecting authenticator into the context
	ctx = context.WithValue(ctx, auth.CKAuthenticator, authenticator)

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

	// creating confidential client
	clnt, err := clientManager.CreateClient(ctx, "test client", client.FConfidential)
	a.NoError(err)
	a.NotNil(clnt)

	// creating client password
	clientPassword, err := clientManager.CreatePassword(ctx, clnt.ID)
	a.NoError(err)
	a.NotZero(clientPassword)

	// creating test session to obtain a token pair
	session, tpair, err := authenticator.CreateSessionWithRefreshToken(
		ctx,
		uuid.New(),
		nil,
		clnt,
		auth.UserIdentity(testuser.ID),
		auth.NewRequestMetadata(nil),
	)
	a.NoError(err)
	a.NotNil(session)
	a.NotEmpty(tpair.AccessToken)
	a.NotZero(tpair.RefreshToken)

	// creating code verifier and challenge
	codeVerifier := "secret phrase"
	method := "s256"

	h := sha256.New()
	h.Write([]byte(codeVerifier))
	codeChallenge := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// creating authorization code
	code, err := authenticator.CreateAuthorizationCode(
		ctx,
		auth.PKCEChallenge{
			Challenge: codeChallenge,
			Method:    method,
		},
		tpair,
	)
	a.NoError(err)
	a.NotEmpty(code)

	tpair, err = authenticator.ExchangeAuthorizationCode(ctx, code, codeVerifier)
	a.NoError(err)
	a.Equal(tpair.AccessToken, tpair.AccessToken)
	a.Equal(tpair.RefreshToken, tpair.RefreshToken)

	// test request
	req, err := http.NewRequest("GET", "/protected", nil)
	a.NoError(err)

	// setting up test request
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tpair.AccessToken))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// initializing new router for testing
	router := chi.NewRouter()
	router.Use(endpoints.Authenticator(func(r *http.Request) (atok string, err error) {
		return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), nil
	}))

	// target endpoint that must be reached after passing
	// the middleware which authenticates by access token
	router.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		a.NoError(err)
	})

	// processing test request
	router.ServeHTTP(rr, req)

	resp := rr.Result()
	a.Equal(http.StatusOK, resp.StatusCode)

	rbody, err := ioutil.ReadAll(resp.Body)
	a.NoError(err)
	a.NotEmpty(rbody)
	a.Equal([]byte("ok"), rbody)
}

func TestMiddlewareAuthenticatorExpiredSession(t *testing.T) {
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

	// injecting authenticator into the context
	ctx = context.WithValue(ctx, auth.CKAuthenticator, authenticator)

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

	// creating confidential client
	clnt, err := clientManager.CreateClient(ctx, "test client", client.FConfidential)
	a.NoError(err)
	a.NotNil(clnt)

	// creating client password
	clientPassword, err := clientManager.CreatePassword(ctx, clnt.ID)
	a.NoError(err)
	a.NotZero(clientPassword)

	// creating test session to obtain a token pair
	session, tpair, err := authenticator.CreateSessionWithRefreshToken(
		ctx,
		uuid.New(),
		nil,
		clnt,
		auth.UserIdentity(testuser.ID),
		auth.NewRequestMetadata(nil),
	)
	a.NoError(err)
	a.NotNil(session)
	a.NotEmpty(tpair.AccessToken)
	a.NotZero(tpair.RefreshToken)

	// creating code verifier and challenge
	codeVerifier := "secret phrase"
	method := "s256"

	h := sha256.New()
	h.Write([]byte(codeVerifier))
	codeChallenge := base64.URLEncoding.EncodeToString(h.Sum(nil))

	// creating authorization code
	code, err := authenticator.CreateAuthorizationCode(
		ctx,
		auth.PKCEChallenge{
			Challenge: codeChallenge,
			Method:    method,
		},
		tpair,
	)
	a.NoError(err)
	a.NotEmpty(code)

	tpair, err = authenticator.ExchangeAuthorizationCode(ctx, code, codeVerifier)
	a.NoError(err)
	a.Equal(tpair.AccessToken, tpair.AccessToken)
	a.Equal(tpair.RefreshToken, tpair.RefreshToken)

	// test request
	req, err := http.NewRequest("GET", "/protected", nil)
	a.NoError(err)

	// setting up test request
	req = req.WithContext(ctx)
	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", tpair.AccessToken))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// initializing new router for testing
	router := chi.NewRouter()
	router.Use(endpoints.Authenticator(func(r *http.Request) (atok string, err error) {
		return strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer "), nil
	}))

	// target endpoint that must be reached after passing
	// the middleware which authenticates by access token
	router.Get("/protected", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, err := w.Write([]byte("ok"))
		a.NoError(err)
	})

	// expiring session
	a.NoError(authenticator.RevokeSession(ctx, session.ID, auth.SRevokedBySystem, "revoked by test"))

	// processing test request
	router.ServeHTTP(rr, req)

	resp := rr.Result()
	a.Equal(http.StatusUnauthorized, resp.StatusCode)

	rbody, err := ioutil.ReadAll(resp.Body)
	a.NoError(err)
	a.Empty(rbody)
}
