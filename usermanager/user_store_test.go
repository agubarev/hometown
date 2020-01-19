package usermanager_test

import (
	"fmt"
	"testing"

	"github.com/agubarev/hometown/util"

	"github.com/agubarev/hometown/usermanager"

	"github.com/stretchr/testify/assert"
)

var testUserinfo = map[string]string{
	"firstname": "Andrei",
	"lastname":  "Gubarev",
}

func TestUserStorePut(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	// truncating test database tables
	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	s, err := usermanager.NewUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	newUser, err := usermanager.NewUser("testuser", "test@example.com", testUserinfo)
	a.NoError(err)
	a.NotNil(newUser)

	user, err := s.Put(newUser)
	a.NoError(err)
	a.NotNil(user)

	// retrieving to make sure everything is set properly
	u, err := s.FetchByID(newUser.ID)
	a.NoError(err)
	a.NotNil(u)

	u, err = s.FetchByKey("username", "testuser")
	a.NoError(err)
	a.NotNil(u)

	u, err = s.FetchByKey("email", "test@example.com")
	a.NoError(err)
	a.NotNil(u)
}

func TestUserStoreGetters(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	newUser, err := usermanager.NewUser("testuser", "test@example.com", testUserinfo)
	a.NoError(err)
	a.NotNil(newUser)

	s, err := usermanager.NewUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	// storing
	user, err := s.Put(newUser)
	a.NoError(err)
	a.NotNil(user)

	// retrieving by ID
	u, err := s.FetchByID(newUser.ID)
	a.NoError(err)
	a.NotNil(u)
	a.Equal(newUser.Username, u.Username)
	a.Equal(newUser.Email, u.Email)

	// retrieving by username index
	u, err = s.FetchByKey("username", newUser.Username)
	a.NoError(err)
	a.NotNil(u)
	a.Equal(newUser.Username, u.Username)
	a.Equal(newUser.Email, u.Email)

	// retrieving by email index
	u, err = s.FetchByKey("email", newUser.Email)
	a.NoError(err)
	a.NotNil(u)
	a.Equal(newUser.Username, u.Username)
	a.Equal(newUser.Email, u.Email)

	// retrieving by a non-existing index
	u, err = s.FetchByKey("no such index", "absent value")
	a.EqualError(err, usermanager.ErrUnknownIndex.Error())
	a.Nil(u)
}

func TestUserStoreGetAll(t *testing.T) {
	a := assert.New(t)

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	s, err := usermanager.NewUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	testUsers := make(map[int64]*usermanager.User, 5)
	for i := 0; i < 5; i++ {
		// NOTE: using numbers in names here only for the
		// store testing purpose, normally only letters are allowed in the name
		u, err := usermanager.NewUser(fmt.Sprintf("testuser%d", i), fmt.Sprintf("testuser%d@example.com", i), testUserinfo)
		u.Firstname = fmt.Sprintf("Andrei %d", i)
		u.Lastname = fmt.Sprintf("Gubarev %d", i)
		u.Middlename = fmt.Sprintf("Anatolievich %d", i)

		a.NotNil(u)
		a.NoError(err)

		u, err = s.Put(u)
		a.NoError(err)

		testUsers[u.ID] = u
	}

	loadedUsers, err := s.FetchAll()
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

	db, err := usermanager.DatabaseForTesting()
	a.NoError(err)
	a.NotNil(db)

	a.NoError(usermanager.TruncateDatabaseForTesting(db))

	newUser, err := usermanager.NewUser("testuser", "test@example.com", testUserinfo)
	a.NoError(err)
	a.NotNil(newUser)

	s, err := usermanager.NewUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	// storing and retrieving to make sure it exists
	user, err := s.Put(newUser)
	a.NoError(err)
	a.NotNil(user)

	// retrieving by ID
	u, err := s.FetchByID(newUser.ID)
	a.NoError(err)
	a.NotNil(u)
	a.Equal(newUser.Username, u.Username)
	a.Equal(newUser.Email, u.Email)

	// deleting and attempting to retrieve to make sure it's gone
	// along with all its indexes
	err = s.Delete(u.ID)
	a.NoError(err)

	u, err = s.FetchByID(newUser.ID)
	a.EqualError(err, usermanager.ErrUserNotFound.Error())
	a.Nil(u)

	u, err = s.FetchByKey("username", "testuser")
	a.EqualError(err, usermanager.ErrUserNotFound.Error())
	a.Nil(u)

	u, err = s.FetchByKey("email", "test@example.com")
	a.EqualError(err, usermanager.ErrUserNotFound.Error())
	a.Nil(u)
}

//---------------------------------------------------------------------------
// benchmarks
//---------------------------------------------------------------------------

func BenchmarkUserStorePutUser(b *testing.B) {
	b.ReportAllocs()

	db, err := usermanager.DatabaseForTesting()
	if err != nil {
		panic(err)
	}

	err = usermanager.TruncateDatabaseForTesting(db)
	if err != nil {
		panic(err)
	}

	s, err := usermanager.NewUserStore(db)

	for n := 0; n < b.N; n++ {
		newUser, err := usermanager.NewUser(util.NewULID().String(), fmt.Sprintf("%s@example.com", util.NewULID().String()), testUserinfo)
		if err != nil {
			panic(err)
		}

		_, err = s.Put(newUser)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGet(b *testing.B) {
	b.ReportAllocs()

	db, err := usermanager.DatabaseForTesting()
	if err != nil {
		panic(err)
	}

	err = usermanager.TruncateDatabaseForTesting(db)
	if err != nil {
		panic(err)
	}

	s, err := usermanager.NewUserStore(db)
	newUser, err := usermanager.NewUser("testuser", "test@example.com", testUserinfo)

	newUser, err = s.Put(newUser)
	if err != nil {
		panic(err)
	}

	for n := 0; n < b.N; n++ {
		_, err = s.FetchByID(newUser.ID)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGetByUsername(b *testing.B) {
	b.ReportAllocs()

	db, err := usermanager.DatabaseForTesting()
	if err != nil {
		panic(err)
	}

	err = usermanager.TruncateDatabaseForTesting(db)
	if err != nil {
		panic(err)
	}

	s, err := usermanager.NewUserStore(db)
	newUser, err := usermanager.NewUser("testuser", "test@example.com", testUserinfo)

	newUser, err = s.Put(newUser)
	if err != nil {
		panic(err)
	}

	for n := 0; n < b.N; n++ {
		_, err = s.FetchByKey("username", newUser.Username)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGetByEmail(b *testing.B) {
	b.ReportAllocs()

	db, err := usermanager.DatabaseForTesting()
	if err != nil {
		panic(err)
	}

	err = usermanager.TruncateDatabaseForTesting(db)
	if err != nil {
		panic(err)
	}

	s, err := usermanager.NewUserStore(db)
	newUser, err := usermanager.NewUser("testuser", "test@example.com", testUserinfo)

	newUser, err = s.Put(newUser)
	if err != nil {
		panic(err)
	}

	for n := 0; n < b.N; n++ {
		_, err = s.FetchByKey("email", newUser.Email)
		if err != nil {
			panic(err)
		}
	}
}
