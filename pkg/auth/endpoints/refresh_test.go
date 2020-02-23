package endpoints_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"

	"github.com/agubarev/hometown/pkg/auth"
	"github.com/agubarev/hometown/pkg/auth/endpoints"
	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestHandleRefreshToken(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// initializing test user manager
	um, err := user.NewManagerForTesting(db)
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
	testuser, err := um.CreateUser(
		"testuser",
		"testuser@example.com",
		testpass,
		testReusableUserinfo,
	)
	a.NoError(err)
	a.NotNil(testuser)

	gm, err := um.GroupManager()
	a.NoError(err)
	a.NotNil(gm)

	// creating groups and roles for testing
	g1, err := gm.Create(group.GKGroup, "group_1", "Group 1", nil)
	a.NoError(err)
	a.NotNil(g1)

	g2, err := gm.Create(group.GKGroup, "group_2", "Group 2", nil)
	a.NoError(err)
	a.NotNil(g1)

	g3, err := gm.Create(group.GKGroup, "group_3", "Group 3 (sub-group of Group 2)", g2)
	a.NoError(err)
	a.NotNil(g1)

	r1, err := gm.Create(group.GKRole, "role_1", "Role 1", nil)
	a.NoError(err)
	a.NotNil(g1)

	r2, err := gm.Create(group.GKRole, "role_2", "Role 2", nil)
	a.NoError(err)
	a.NotNil(g1)

	// adding test user to every role and a group
	g1.AddMember(testuser)
	g2.AddMember(testuser)
	g3.AddMember(testuser)
	r1.AddMember(testuser)
	r2.AddMember(testuser)

	// ====================================================================================
	// wrong IP case
	// NOTE: request's RemoteAddr is empty when testing, so validation
	// can be failed by specifying any IP, i.e. 127.0.0.1
	// ====================================================================================
	user, err := am.Authenticate(testuser.Username, testpass, auth.NewRequestInfo(nil))
	a.NoError(err)
	a.NotNil(user)
	a.True(reflect.DeepEqual(user, testuser))

	tp, err := am.GenerateTokenTrinity(user, auth.NewRequestInfo(nil))
	a.NoError(err)
	a.NotNil(tp)

	req, err := http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBufferString(tp.RefreshToken))
	a.NoError(err)

	req = req.WithContext(context.WithValue(context.Background(), auth.CKUserManager, um))
	req = req.WithContext(context.WithValue(context.Background(), auth.CKAuthenticator, am))

	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// triggering the handler
	endpoints.HandleRefreshToken(rr, req)

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
	user, err = am.UserFromToken(string(rtp.AccessToken))
	a.Error(err)
	a.Nil(user)

	// ====================================================================================
	// invalid refresh token case
	// ====================================================================================
	user, err = am.Authenticate(testuser.Username, testpass, auth.NewRequestInfo(nil))
	a.NoError(err)
	a.NotNil(user)
	a.True(reflect.DeepEqual(user, testuser))

	req, err = http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBufferString("wrong refresh token"))
	a.NoError(err)

	req = req.WithContext(context.WithValue(context.Background(), auth.CKUserManager, um))
	req = req.WithContext(context.WithValue(context.Background(), auth.CKAuthenticator, am))

	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	// triggering the handler
	endpoints.HandleRefreshToken(rr, req)

	resp = rr.Result()
	a.Equal(http.StatusUnauthorized, resp.StatusCode)

	// response body is a JWT token
	rbody, err = ioutil.ReadAll(resp.Body)
	a.NoError(err)
	a.NotEmpty(rbody)

	rtp = auth.TokenTrinity{}
	a.NoError(json.Unmarshal(rbody, &rtp))
	a.Empty(rtp.AccessToken)
	a.Empty(rtp.RefreshToken)

	// obtaining an owner of this token
	user, err = am.UserFromToken(string(rtp.AccessToken))
	a.Error(err)
	a.Nil(user)

	// ====================================================================================
	// normal case
	// ====================================================================================
	user, err = am.Authenticate(testuser.Username, testpass, auth.NewRequestInfo(nil))
	a.NoError(err)
	a.NotNil(user)
	a.True(reflect.DeepEqual(user, testuser))

	tp, err = am.GenerateTokenTrinity(user, auth.NewRequestInfo(nil))
	a.NoError(err)
	a.NotNil(tp)

	req, err = http.NewRequest("POST", "/api/v1/auth/refresh", bytes.NewBufferString(tp.RefreshToken))
	a.NoError(err)

	req = req.WithContext(context.WithValue(context.Background(), auth.CKUserManager, um))
	req = req.WithContext(context.WithValue(context.Background(), auth.CKAuthenticator, am))

	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	// triggering the handler
	endpoints.HandleRefreshToken(rr, req)

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
	user, err = am.UserFromToken(string(rtp.AccessToken))
	a.NoError(err)
	a.NotNil(user)
	a.Equal(testuser, user)
}
