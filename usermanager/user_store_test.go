package usermanager_test

import (
	"os"
	"testing"

	"gitlab.com/agubarev/hometown/usermanager"

	"github.com/stretchr/testify/assert"
)

func TestUserStorePut(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)
	a.NoError(err)
	a.NotNil(db)

	s, err := usermanager.NewDefaultUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	newUser, err := usermanager.NewUser("testuser", "test@example.com")
	a.NoError(err)
	a.NotNil(newUser)

	a.NoError(s.Put(newUser))

	// retrieving to make sure everything is set properly
	u, err := s.Get(newUser.ID)
	a.NoError(err)
	a.NotNil(u)

	u, err = s.GetByIndex("username", "testuser")
	a.NoError(err)
	a.NotNil(u)

	u, err = s.GetByIndex("email", "test@example.com")
	a.NoError(err)
	a.NotNil(u)
}

func TestUserStoreGetters(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)
	a.NoError(err)
	a.NotNil(db)

	newUser, err := usermanager.NewUser("testuser", "test@example.com")
	a.NoError(err)
	a.NotNil(newUser)

	s, err := usermanager.NewDefaultUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	// storing
	a.NoError(s.Put(newUser))

	// retrieving by ID
	u, err := s.Get(newUser.ID)
	a.NoError(err)
	a.NotNil(u)
	a.Equal(newUser.Username, u.Username)
	a.Equal(newUser.Email, u.Email)

	// retrieving by username index
	u, err = s.GetByIndex("username", newUser.Username)
	a.NoError(err)
	a.NotNil(u)
	a.Equal(newUser.Username, u.Username)
	a.Equal(newUser.Email, u.Email)

	// retrieving by email index
	u, err = s.GetByIndex("email", newUser.Email)
	a.NoError(err)
	a.NotNil(u)
	a.Equal(newUser.Username, u.Username)
	a.Equal(newUser.Email, u.Email)

	// retrieving by a non-existing index
	u, err = s.GetByIndex("no such index", "absent value")
	a.EqualError(err, usermanager.ErrUserNotFound.Error())
	a.Nil(u)
}

func TestUserStoreDelete(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)
	a.NoError(err)
	a.NotNil(db)

	newUser, err := usermanager.NewUser("testuser", "test@example.com")
	a.NoError(err)
	a.NotNil(newUser)

	s, err := usermanager.NewDefaultUserStore(db)
	a.NoError(err)
	a.NotNil(s)

	// storing and retrieving to make sure it exists
	a.NoError(s.Put(newUser))

	// retrieving by ID
	u, err := s.Get(newUser.ID)
	a.NoError(err)
	a.NotNil(u)
	a.Equal(newUser.Username, u.Username)
	a.Equal(newUser.Email, u.Email)

	// deleting and attempting to retrieve to make sure it's gone
	// along with all its indexes
	err = s.Delete(u.ID)
	a.NoError(err)

	u, err = s.Get(newUser.ID)
	a.EqualError(err, usermanager.ErrUserNotFound.Error())
	a.Nil(u)

	u, err = s.GetByIndex("username", "testuser")
	a.EqualError(err, usermanager.ErrUserNotFound.Error())
	a.Nil(u)

	u, err = s.GetByIndex("email", "test@example.com")
	a.EqualError(err, usermanager.ErrUserNotFound.Error())
	a.Nil(u)
}

//---------------------------------------------------------------------------
// benchmarks
//---------------------------------------------------------------------------

func BenchmarkUserStorePutUser(b *testing.B) {
	b.ReportAllocs()

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)

	s, err := usermanager.NewDefaultUserStore(db)
	newUser, err := usermanager.NewUser("testuser", "test@example.com")
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {

		err = s.Put(newUser)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGet(b *testing.B) {
	b.ReportAllocs()

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)

	s, err := usermanager.NewDefaultUserStore(db)
	newUser, err := usermanager.NewUser("testuser", "test@example.com")
	err = s.Put(newUser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.Get(newUser.ID)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGetByUsername(b *testing.B) {
	b.ReportAllocs()

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)

	s, err := usermanager.NewDefaultUserStore(db)
	newUser, err := usermanager.NewUser("testuser", "test@example.com")
	err = s.Put(newUser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetByIndex("username", newUser.Username)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGetByEmail(b *testing.B) {
	b.ReportAllocs()

	db, dbPath, err := usermanager.CreateRandomBadgerDB()
	defer os.RemoveAll(dbPath)

	s, err := usermanager.NewDefaultUserStore(db)
	newUser, err := usermanager.NewUser("testuser", "test@example.com")
	err = s.Put(newUser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetByIndex("email", newUser.Email)
		if err != nil {
			panic(err)
		}
	}
}
