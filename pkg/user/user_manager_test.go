package user_test

import (
	context2 "context"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/agubarev/hometown/internal/core"
	"github.com/agubarev/hometown/pkg/accesspolicy"
	"github.com/agubarev/hometown/pkg/group"
	"github.com/agubarev/hometown/pkg/password"
	"github.com/agubarev/hometown/pkg/token"
	"github.com/agubarev/hometown/pkg/util"
	"github.com/stretchr/testify/assert"
)

func TestUserManagerNew(t *testing.T) {
	a := assert.New(t)

	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(core.TruncateDatabaseForTesting(db))

	userStore, err := core.NewUserStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	groupStore, err := group.NewGroupStore(db)
	a.NoError(err)
	a.NotNil(groupStore)

	tokenStore, err := token.NewTokenStore(db)
	a.NoError(err)
	a.NotNil(tokenStore)

	aps, err := accesspolicy.NewDefaultAccessPolicyStore(db)
	a.NoError(err)
	a.NotNil(aps)

	passwordStore, err := password.NewPasswordStore(db)
	a.NoError(err)
	a.NotNil(passwordStore)

	passwordManager, err := password.NewDefaultManager(passwordStore)
	a.NoError(err)
	a.NotNil(passwordManager)

	groupContainer, err := group.NewGroupManager(groupStore)
	a.NoError(err)
	a.NotNil(groupContainer)

	tokenContainer, err := token.NewTokenManager(tokenStore)
	a.NoError(err)
	a.NotNil(tokenContainer)

	userManager, err := core.NewUserManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	a.NoError(userManager.SetGroupManager(groupContainer))
	a.NoError(userManager.SetTokenManager(tokenContainer))
	a.NoError(userManager.Init())
}

func TestUserManagerCreate(t *testing.T) {
	a := assert.New(t)

	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)
	a.NoError(core.TruncateDatabaseForTesting(db))

	userStore, err := core.NewUserStore(db)
	a.NoError(err)
	a.NotNil(userStore)

	groupStore, err := group.NewGroupStore(db)
	a.NoError(err)
	a.NotNil(groupStore)

	tokenStore, err := token.NewTokenStore(db)
	a.NoError(err)
	a.NotNil(tokenStore)

	aps, err := accesspolicy.NewDefaultAccessPolicyStore(db)
	a.NoError(err)
	a.NotNil(aps)

	passwordStore, err := password.NewPasswordStore(db)
	a.NoError(err)
	a.NotNil(passwordStore)

	passwordManager, err := password.NewDefaultManager(passwordStore)
	a.NoError(err)
	a.NotNil(passwordManager)

	groupContainer, err := group.NewGroupManager(groupStore)
	a.NoError(err)
	a.NotNil(groupContainer)

	tokenContainer, err := token.NewTokenManager(tokenStore)
	a.NoError(err)
	a.NotNil(tokenContainer)

	userManager, err := core.NewUserManager(userStore, nil)
	a.NoError(err)
	a.NotNil(userManager)

	a.NoError(userManager.SetGroupManager(groupContainer))
	a.NoError(userManager.SetTokenManager(tokenContainer))
	a.NoError(userManager.Init())

	u1, err := userManager.Create("testuser1")
	a.NoError(err)
	a.NotNil(u1)

	u2, err := userManager.Create("testuser2")
	a.NoError(err)
	a.NotNil(u2)

	u3, err := userManager.Create("testuser3")
	a.NoError(err)
	a.NotNil(u3)

	u4, err := userManager.Create("testuser4")
	a.Error(err)
	a.Nil(u4)

	userContainer, err := userManager.Container()
	a.NoError(err)
	a.NotNil(userContainer)

	// container must contain only 3 runtime user objects at this point
	a.Len(userContainer.List(nil), 3)

	//---------------------------------------------------------------------------
	// checking store
	//---------------------------------------------------------------------------
	testUsers := map[int64]*user.User{
		u1.ID: u1,
		u2.ID: u2,
		u3.ID: u3,
	}

	storedUsers, err := userStore.GetUsers(context2.TODO())
	a.NoError(err)
	a.Len(storedUsers, 3)

	for _, su := range storedUsers {
		a.Equal(testUsers[su.ID].Username, su.Username)
		a.Equal(testUsers[su.ID].Email, su.Email)
		a.Equal(testUsers[su.ID].Firstname, su.Firstname)
		a.Equal(testUsers[su.ID].Lastname, su.Lastname)
		a.Equal(testUsers[su.ID].Middlename, su.Middlename)
		a.Equal(testUsers[su.ID].EmailConfirmedAt, su.EmailConfirmedAt)
	}
}

func TestUserManagerInit(t *testing.T) {
	a := assert.New(t)

	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)
	a.NoError(core.TruncateDatabaseForTesting(db))

	um, err := core.NewUserManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	//---------------------------------------------------------------------------
	// creating test groups and roles
	//---------------------------------------------------------------------------
	gm, err := um.GroupManager()
	a.NoError(err)
	a.NotNil(gm)

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

	//---------------------------------------------------------------------------
	// make sure the container is fresh and empty
	// NOTE: at this point the store is newly created and must be empty
	//---------------------------------------------------------------------------
	uc, err := um.Container()
	a.NoError(err)
	a.NotNil(uc)

	a.Len(uc.List(nil), 0)

	//---------------------------------------------------------------------------
	// creating and storing test users
	//---------------------------------------------------------------------------

	// will just reuse the same user info data for each new user
	validUserinfo := map[string]string{
		"firstname": "Андрей",
		"lastname":  "Губарев",
	}

	u1, err := um.Create("testuser1")
	a.NoError(err)
	a.NotNil(u1)

	u2, err := um.Create("testuser2")
	a.NoError(err)
	a.NotNil(u2)

	u3, err := um.Create("testuser3")
	a.NoError(err)
	a.NotNil(u3)

	// checking container
	a.Len(uc.List(nil), 3)

	//---------------------------------------------------------------------------
	// assigning users to groups and roles
	//---------------------------------------------------------------------------
	// user 1 to group 1
	a.NoError(g1.AddMember(u1))

	// user 2 and 3 to group 2
	a.NoError(g2.AddMember(u2))
	a.NoError(g2.AddMember(u3))

	// user 3 to group 3
	a.NoError(g3.AddMember(u3))

	// user 1 and 2 to role 1
	a.NoError(r1.AddMember(u1))
	a.NoError(r1.AddMember(u2))

	// user 3 to role 2
	a.NoError(r2.AddMember(u3))

	//---------------------------------------------------------------------------
	// reinitializing the user manager and expecting all of its required,
	// previously created users, groups, roles, tokens, policies etc. to be loaded
	//---------------------------------------------------------------------------
	um, err = core.NewUserManagerForTesting(db)
	a.NoError(err)
	a.NotNil(um)

	// checking whether the users are reloaded
	a.Len(uc.List(nil), 3)

	//---------------------------------------------------------------------------
	// checking groups
	//---------------------------------------------------------------------------
	gm, err = um.GroupManager()
	a.NoError(err)
	a.NotNil(gm)

	// validating user 1
	u1, err = um.GetByKey("id", u1.ID)
	a.NoError(err)
	a.NotNil(u1)
	a.True(g1.IsMember(u1))
	a.False(g2.IsMember(u1))
	a.False(g3.IsMember(u1))
	a.True(r1.IsMember(u1))
	a.False(r2.IsMember(u1))

	// validating user 2
	u2, err = um.GetByKey("id", u2.ID)
	a.NoError(err)
	a.NotNil(u2)
	a.False(g1.IsMember(u2))
	a.True(g2.IsMember(u2))
	a.False(g3.IsMember(u2))
	a.True(r1.IsMember(u2))
	a.False(r2.IsMember(u2))

	// validating user 3
	u3, err = um.GetByKey("id", u3.ID)
	a.NoError(err)
	a.NotNil(u3)
	a.False(g1.IsMember(u3))
	a.True(g2.IsMember(u3))
	a.True(g3.IsMember(u3))
	a.False(r1.IsMember(u3))
	a.True(r2.IsMember(u3))
}

func BenchmarkUserManagerCreate(b *testing.B) {
	b.ReportAllocs()

	db, err := core.DatabaseForTesting()
	if err != nil {
		panic(err)
	}

	core.TruncateDatabaseForTesting(db)

	um, err := core.NewUserManagerForTesting(db)
	if err != nil {
		panic(err)
	}

	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	// creating 100 users
	for i := 0; i < 100; i++ {
		username := util.NewULID().String()
		email := fmt.Sprintf("%s@example.com", username)

		_, err := um.Create(username)
		if err != nil {
			panic(err)
		}
	}

	c, err := um.Container()
	if err != nil {
		panic(err)
	}

	for i := 0; i < b.N; i++ {
		_, err := c.GetByID(1 + r.Int63n(int64(99)))
		if err != nil {
			panic(err)
		}
	}
}
