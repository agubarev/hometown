package auth_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	auth2 "github.com/agubarev/hometown/internal/server/endpoints/auth"
	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/security/auth"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestHandleRefreshToken(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db, err := database.MySQLForTesting()
	a.NoError(err)
	a.NotNil(db)

	// initializing test user manager
	um, ctx, err := user.ManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	// initializing access manager
	am, err := auth.NewAuthenticator(nil, um, auth.NewDefaultRegistryBackend(), auth.NewDefaultConfig())
	a.NoError(err)
	a.NotNil(am)

	// setting authenticator logger
	al, err := util.DefaultLogger(true, "")
	a.NoError(err)
	a.NotNil(al)
	a.NoError(am.SetLogger(al))

	// using ULID as a random password
	testpass := []byte(util.NewULID().String())

	// creating test user
	testuser, err := user.CreateTestUser(ctx, um, "testuser", "testuser@hometown.local", testpass)
	a.NoError(err)
	a.NotNil(testuser)

	gs, err := group.NewMySQLStore(db)
	a.NoError(err)
	a.NotNil(gs)

	gm, err := group.NewManager(ctx, gs)
	a.NoError(err)
	a.NotNil(gm)

	// creating groups and roles for testing
	g1, err := gm.Create(ctx, group.FGroup, 0, "group_1", "Group 1")
	a.NoError(err)
	a.NotNil(g1)

	g2, err := gm.Create(ctx, group.FGroup, 0, "group_2", "Group 2")
	a.NoError(err)
	a.NotNil(g1)

	g3, err := gm.Create(ctx, group.FGroup, g2.ID, "group_3", "Group 3 (sub-group of Group 2)")
	a.NoError(err)
	a.NotNil(g1)

	r1, err := gm.Create(ctx, group.FRole, 0, "role_1", "Role 1")
	a.NoError(err)
	a.NotNil(g1)

	r2, err := gm.Create(ctx, group.FRole, 0, "role_2", "Role 2")
	a.NoError(err)
	a.NotNil(g1)

	// adding test user to every role and a group
	a.NoError(gm.CreateRelation(ctx, g1.ID, testuser.ID))
	a.NoError(gm.CreateRelation(ctx, g2.ID, testuser.ID))
	a.NoError(gm.CreateRelation(ctx, g3.ID, testuser.ID))
	a.NoError(gm.CreateRelation(ctx, r1.ID, testuser.ID))
	a.NoError(gm.CreateRelation(ctx, r2.ID, testuser.ID))

	// ====================================================================================
	// wrong IP case
	// NOTE: request's RemoteAddr is empty when testing, so validation
	// can be failed by specifying any IP, i.e. 127.0.0.1
	// ====================================================================================
	/*
		u, err := am.Authenticate(ctx, testuser.Username, testpass, auth.NewRequestInfo(nil))
		a.NoError(err)
		a.NotNil(u)
		a.True(reflect.DeepEqual(u.Essential, testuser.Essential))

		tt, err := am.GenerateTokenTrinity(ctx, u, auth.NewRequestInfo(nil))
		a.NoError(err)
		a.NotNil(tt)

		req, err := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBufferString(tt.RefreshToken))
		a.NoError(err)

		req = req.WithContext(context.WithValue(context.Background(), auth.CKUserManager, um))
		req = req.WithContext(context.WithValue(context.Background(), auth.CKAuthenticator, am))

		req.Header.Set("Content-Type", "application/json")
		rr := httptest.NewRecorder()

		// triggering the handler
		endpoints.HandleRefreshToken(rr, req)

		resp := rr.Result()
		spew.Dump(resp.Body)
		a.Equal(http.StatusUnauthorized, resp.StatusCode)

		// response body is a JWT token
		rbody, err := ioutil.ReadAll(resp.Body)
		a.NoError(err)
		a.NotEmpty(rbody)

		rtp := auth.TokenTrinity{}
		a.NoError(json.Unmarshal(rbody, &rtp))
		a.Empty(rtp.AccessToken)
		a.Empty(rtp.RefreshToken)

		// obtaining an owner of this token
		u, err = am.UserFromToken(rtp.AccessToken)
		a.Error(err)
		a.Zero(u.SubjectID)
	*/

	// ====================================================================================
	// invalid refresh token case
	// ====================================================================================
	u, err := am.Authenticate(ctx, testuser.Username, testpass, auth.NewRequestInfo(nil))
	a.NoError(err)
	a.NotNil(u)
	a.True(reflect.DeepEqual(u.Essential, testuser.Essential))

	req, err := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBufferString("wrong refresh token"))
	a.NoError(err)

	req = req.WithContext(context.WithValue(context.Background(), auth.CKUserManager, um))
	req = req.WithContext(context.WithValue(context.Background(), auth.CKAuthenticator, am))

	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// triggering the handler
	auth2.HandleRefreshToken(rr, req)

	resp := rr.Result()
	a.Equal(http.StatusUnauthorized, resp.StatusCode)

	// response body is a JWT token
	rbody, err := ioutil.ReadAll(resp.Body)
	a.NoError(err)
	a.NotEmpty(rbody)

	rtp := auth.TokenTrinity{}
	a.NoError(json.Unmarshal(rbody, &rtp))
	a.Empty(rtp.AccessToken)
	a.Empty(rtp.RefreshToken)

	// obtaining an owner of this token
	u, err = am.UserFromToken(ctx, rtp.AccessToken)
	a.Error(err)
	a.Zero(u.ID)

	// ====================================================================================
	// normal case
	// ====================================================================================
	u, err = am.Authenticate(ctx, testuser.Username, testpass, auth.NewRequestInfo(nil))
	a.NoError(err)
	a.NotNil(u)
	a.True(reflect.DeepEqual(u.Essential, testuser.Essential))

	tt, err := am.GenerateTokenTrinity(ctx, u, auth.NewRequestInfo(nil))
	a.NoError(err)
	a.NotNil(tt)

	req, err = http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBufferString(tt.RefreshToken))
	a.NoError(err)

	req = req.WithContext(context.WithValue(context.Background(), auth.CKUserManager, um))
	req = req.WithContext(context.WithValue(context.Background(), auth.CKAuthenticator, am))

	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	// triggering the handler
	auth2.HandleRefreshToken(rr, req)

	resp = rr.Result()
	a.Equal(http.StatusOK, resp.StatusCode)

	// response body is a JWT token
	rbody, err = ioutil.ReadAll(resp.Body)
	a.NoError(err)
	a.NotEmpty(rbody)

	rtp = auth.TokenTrinity{}
	a.NoError(json.Unmarshal(rbody, &rtp))
	a.NotEmpty(rtp.SessionToken)
	a.NotEmpty(rtp.AccessToken)
	a.NotEmpty(rtp.RefreshToken)

	// obtaining an owner of this token
	u, err = am.UserFromToken(ctx, rtp.AccessToken)
	a.NoError(err)
	a.NotNil(u)
	a.True(reflect.DeepEqual(testuser.Essential, u.Essential))
	a.Equal(testuser.ID, u.ID)
}
