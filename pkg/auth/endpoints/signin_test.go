package endpoints_test

import (
	"bytes"
	"context"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/agubarev/hometown/pkg/auth"
	"github.com/agubarev/hometown/pkg/auth/endpoints"
	"github.com/agubarev/hometown/pkg/database"
	"github.com/agubarev/hometown/pkg/user"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/stretchr/testify/assert"
)

var testReusableUserinfo = map[string]string{
	"firstname": "Andrei",
	"lastname":  "Gubarev",
}

func TestSignin(t *testing.T) {
	a := assert.New(t)

	// obtaining and truncating a test database
	db, err := database.ForTesting()
	a.NoError(err)
	a.NotNil(db)

	// initializing test user manager
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

	// creating test user
	testuser, err := um.CreateWithPassword(
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
	g1, err := gm.Create(user.GKGroup, "group_1", "Group 1", nil)
	a.NoError(err)
	a.NotNil(g1)

	g2, err := gm.Create(user.GKGroup, "group_2", "Group 2", nil)
	a.NoError(err)
	a.NotNil(g1)

	g3, err := gm.Create(user.GKGroup, "group_3", "Group 3 (sub-group of Group 2)", g2)
	a.NoError(err)
	a.NotNil(g1)

	r1, err := gm.Create(user.GKRole, "role_1", "Role 1", nil)
	a.NoError(err)
	a.NotNil(g1)

	r2, err := gm.Create(user.GKRole, "role_2", "Role 2", nil)
	a.NoError(err)
	a.NotNil(g1)

	// adding test user to every role and a group
	g1.AddMember(testuser)
	g2.AddMember(testuser)
	g3.AddMember(testuser)
	r1.AddMember(testuser)
	r2.AddMember(testuser)

	// ====================================================================================
	// wrong password
	// ====================================================================================
	body, err := json.Marshal(auth.UserCredentials{
		Username: testuser.Username,
		Password: "wrongpass",
	})

	req, err := http.NewRequest("POST", "/api/v1/auth/signin", bytes.NewBuffer(body))
	a.NoError(err)

	req = req.WithContext(context.WithValue(context.Background(), auth.CKUserManager, um))
	req = req.WithContext(context.WithValue(context.Background(), auth.CKAuthenticator, am))

	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// triggering the handler
	endpoints.HandleSignin(rr, req)

	resp := rr.Result()
	a.Equal(http.StatusUnauthorized, resp.StatusCode)

	// ====================================================================================
	// empty username
	// ====================================================================================
	body, err = json.Marshal(auth.UserCredentials{
		Username: "",
		Password: testpass,
	})

	req, err = http.NewRequest("POST", "/api/v1/signin", bytes.NewBuffer(body))
	a.NoError(err)

	req = req.WithContext(context.WithValue(context.Background(), auth.CKUserManager, um))
	req = req.WithContext(context.WithValue(context.Background(), auth.CKAuthenticator, am))

	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	// triggering the handler
	endpoints.HandleSignin(rr, req)

	resp = rr.Result()
	a.Equal(http.StatusBadRequest, resp.StatusCode)

	// ====================================================================================
	// empty password
	// ====================================================================================
	body, err = json.Marshal(auth.UserCredentials{
		Username: testuser.Username,
		Password: "",
	})

	req, err = http.NewRequest("POST", "/api/v1/signin", bytes.NewBuffer(body))
	a.NoError(err)

	req = req.WithContext(context.WithValue(context.Background(), auth.CKUserManager, um))
	req = req.WithContext(context.WithValue(context.Background(), auth.CKAuthenticator, am))

	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	// triggering the handler
	endpoints.HandleSignin(rr, req)

	resp = rr.Result()
	a.Equal(http.StatusBadRequest, resp.StatusCode)

	// ====================================================================================
	// non-existing user
	// ====================================================================================
	body, err = json.Marshal(auth.UserCredentials{
		Username: "wronguser",
		Password: testpass,
	})

	req, err = http.NewRequest("POST", "/api/v1/signin", bytes.NewBuffer(body))
	a.NoError(err)

	req = req.WithContext(context.WithValue(context.Background(), auth.CKUserManager, um))
	req = req.WithContext(context.WithValue(context.Background(), auth.CKAuthenticator, am))

	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	// triggering the handler
	endpoints.HandleSignin(rr, req)

	resp = rr.Result()
	a.Equal(http.StatusUnauthorized, resp.StatusCode)

	// ====================================================================================
	// normal case
	// ====================================================================================
	body, err = json.Marshal(auth.UserCredentials{
		Username: testuser.Username,
		Password: testpass,
	})

	req, err = http.NewRequest("POST", "/api/v1/signin", bytes.NewBuffer(body))
	a.NoError(err)

	req = req.WithContext(context.WithValue(context.Background(), auth.CKUserManager, um))
	req = req.WithContext(context.WithValue(context.Background(), auth.CKAuthenticator, am))

	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()

	// triggering the handler
	endpoints.HandleSignin(rr, req)

	resp = rr.Result()
	a.Equal(http.StatusOK, resp.StatusCode)

	// response body is a JWT token
	rbody, err := ioutil.ReadAll(resp.Body)
	a.NoError(err)
	a.NotEmpty(rbody)

	rtp := auth.TokenTrinity{}
	a.NoError(json.Unmarshal(rbody, &rtp))
	a.NotEmpty(rtp.AccessToken)
	a.NotEmpty(rtp.RefreshToken)

	// obtaining an owner of this token
	user, err := am.UserFromToken(string(rtp.AccessToken))
	a.NoError(err)
	a.NotNil(user)
	a.Equal(testuser, user)
}
