package user_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/agubarev/hometown/pkg/util"

	"github.com/agubarev/hometown/internal/core"

	"github.com/stretchr/testify/assert"
)

var testUserinfo = map[string]string{
	"firstname": "Andrei",
	"lastname":  "Gubarev",
}

func TestUserStorePut(t *testing.T) {
	a := assert.New(t)

	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	// truncating test database tables
	a.NoError(core.TruncateDatabaseForTesting(db))

	s, err := core.NewUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	newUser, err := user.UserNew("testuser", "test@example.com", testUserinfo)
	a.NoError(err)
	a.NotNil(newUser)

	user, err := s.Update(context.TODO(), newUser)
	a.NoError(err)
	a.NotNil(user)

	// retrieving to make sure everything is set properly
	u, err := s.GetUserByID(context.TODO(), newUser.ID)
	a.NoError(err)
	a.NotNil(u)

	u, err = s.GetUserByKey(context.TODO(), "username", "testuser")
	a.NoError(err)
	a.NotNil(u)

	u, err = s.GetUserByKey(context.TODO(), "email", "test@example.com")
	a.NoError(err)
	a.NotNil(u)
}

func TestUserStoreGetters(t *testing.T) {
	a := assert.New(t)

	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(core.TruncateDatabaseForTesting(db))

	newUser, err := user.UserNew("testuser", "test@example.com", testUserinfo)
	a.NoError(err)
	a.NotNil(newUser)

	s, err := core.NewUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	// storing
	user, err := s.Update(context.TODO(), newUser)
	a.NoError(err)
	a.NotNil(user)

	// retrieving by ID
	u, err := s.GetUserByID(context.TODO(), newUser.ID)
	a.NoError(err)
	a.NotNil(u)
	a.Equal(newUser.Username, u.Username)
	a.Equal(newUser.Email, u.Email)

	// retrieving by username index
	u, err = s.GetUserByKey(context.TODO(), "username", newUser.Username)
	a.NoError(err)
	a.NotNil(u)
	a.Equal(newUser.Username, u.Username)
	a.Equal(newUser.Email, u.Email)

	// retrieving by email index
	u, err = s.GetUserByKey(context.TODO(), "email", newUser.Email)
	a.NoError(err)
	a.NotNil(u)
	a.Equal(newUser.Username, u.Username)
	a.Equal(newUser.Email, u.Email)

	// retrieving by a non-existing index
	u, err = s.GetUserByKey(context.TODO(), "no such index", "absent value")
	a.EqualError(err, core.ErrUnknownIndex.Error())
	a.Nil(u)
}

func TestUserStoreGetAll(t *testing.T) {
	a := assert.New(t)

	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(core.TruncateDatabaseForTesting(db))

	s, err := core.NewUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	testUsers := make(map[int64]*user.User, 5)
	for i := 0; i < 5; i++ {
		// NOTE: using numbers in names here only for the
		// store testing purpose, normally only letters are allowed in the name
		u, err := user.UserNew(fmt.Sprintf("testuser%d", i), fmt.Sprintf("testuser%d@example.com", i), testUserinfo)
		u.Firstname = fmt.Sprintf("Andrei %d", i)
		u.Lastname = fmt.Sprintf("Gubarev %d", i)
		u.Middlename = fmt.Sprintf("Anatolievich %d", i)

		a.NotNil(u)
		a.NoError(err)

		u, err = s.Update(context.TODO(), u)
		a.NoError(err)

		testUsers[u.ID] = u
	}

	loadedUsers, err := s.GetUsers(context.TODO())
	a.NoError(err)
	a.Len(loadedUsers, 5)

	for _, u := range loadedUsers {
		a.Equal(u.ID, testUsers[u.ID].ID)
		a.Equal(u.Username, testUsers[u.ID].Username)
		a.Equal(u.Email, testUsers[u.ID].Email)
		a.Equal(u.Firstname, testUsers[u.ID].Firstname)
		a.Equal(u.Lastname, testUsers[u.ID].Lastname)
		a.Equal(u.Middlename, testUsers[u.ID].Middlename)
	}
}

func TestUserStoreDelete(t *testing.T) {
	a := assert.New(t)

	db, err := core.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(core.TruncateDatabaseForTesting(db))

	newUser, err := user.UserNew("testuser", "test@example.com", testUserinfo)
	a.NoError(err)
	a.NotNil(newUser)

	s, err := core.NewUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	// storing and retrieving to make sure it exists
	user, err := s.Update(context.TODO(), newUser)
	a.NoError(err)
	a.NotNil(user)

	// retrieving by ID
	u, err := s.GetUserByID(context.TODO(), newUser.ID)
	a.NoError(err)
	a.NotNil(u)
	a.Equal(newUser.Username, u.Username)
	a.Equal(newUser.Email, u.Email)

	// deleting and attempting to retrieve to make sure it's gone
	// along with all its indexes
	err = s.DeleteByID(context.TODO(), u.ID)
	a.NoError(err)

	u, err = s.GetUserByID(context.TODO(), newUser.ID)
	a.EqualError(err, core.ErrUserNotFound.Error())
	a.Nil(u)

	u, err = s.GetUserByKey(context.TODO(), "username", "testuser")
	a.EqualError(err, core.ErrUserNotFound.Error())
	a.Nil(u)

	u, err = s.GetUserByKey(context.TODO(), "email", "test@example.com")
	a.EqualError(err, core.ErrUserNotFound.Error())
	a.Nil(u)
}

//---------------------------------------------------------------------------
// benchmarks
//---------------------------------------------------------------------------

func BenchmarkUserStorePutUser(b *testing.B) {
	b.ReportAllocs()

	db, err := core.DatabaseForTesting()
	if err != nil {
		panic(err)
	}

	err = core.TruncateDatabaseForTesting(db)
	if err != nil {
		panic(err)
	}

	s, err := core.NewUserStore(db)

	for n := 0; n < b.N; n++ {
		newUser, err := user.UserNew(util.NewULID().String(), fmt.Sprintf("%s@example.com", util.NewULID().String()), testUserinfo)
		if err != nil {
			panic(err)
		}

		_, err = s.Update(context.TODO(), newUser)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGet(b *testing.B) {
	b.ReportAllocs()

	db, err := core.DatabaseForTesting()
	if err != nil {
		panic(err)
	}

	err = core.TruncateDatabaseForTesting(db)
	if err != nil {
		panic(err)
	}

	s, err := core.NewUserStore(db)
	newUser, err := user.UserNew("testuser", "test@example.com", testUserinfo)

	newUser, err = s.Update(context.TODO(), newUser)
	if err != nil {
		panic(err)
	}

	for n := 0; n < b.N; n++ {
		_, err = s.GetUserByID(context.TODO(), newUser.ID)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGetByUsername(b *testing.B) {
	b.ReportAllocs()

	db, err := core.DatabaseForTesting()
	if err != nil {
		panic(err)
	}

	err = core.TruncateDatabaseForTesting(db)
	if err != nil {
		panic(err)
	}

	s, err := core.NewUserStore(db)
	newUser, err := user.UserNew("testuser", "test@example.com", testUserinfo)

	newUser, err = s.Update(context.TODO(), newUser)
	if err != nil {
		panic(err)
	}

	for n := 0; n < b.N; n++ {
		_, err = s.GetUserByKey(context.TODO(), "username", newUser.Username)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGetByEmail(b *testing.B) {
	b.ReportAllocs()

	db, err := core.DatabaseForTesting()
	if err != nil {
		panic(err)
	}

	err = core.TruncateDatabaseForTesting(db)
	if err != nil {
		panic(err)
	}

	s, err := core.NewUserStore(db)
	newUser, err := user.UserNew("testuser", "test@example.com", testUserinfo)

	newUser, err = s.Update(context.TODO(), newUser)
	if err != nil {
		panic(err)
	}

	for n := 0; n < b.N; n++ {
		_, err = s.GetUserByKey(context.TODO(), "email", newUser.Email)
		if err != nil {
			panic(err)
		}
	}
}
