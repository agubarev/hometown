package usermanager

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"gitlab.com/agubarev/hometown/util"
	"go.etcd.io/bbolt"
)

func getUserStoreList() []UserStore {
	stores := make([]UserStore, 0)

	return stores
}

func TestUserStoreCreate(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := NewBoltStore(db, nil)
	a.NoError(err)
	a.NotNil(s)

	svc, err := NewUserService(s)
	a.NoError(err)
	a.NotNil(svc)

	unsavedUser, err := NewUser("testuser", "test@example.com")
	a.NoError(err)
	a.NotNil(unsavedUser)

	newUser, err := svc.Create(context.Background(), unsavedUser)
	a.NoError(err)
	a.NotNil(newUser)
	a.Equal("testuser", newUser.Username)
	a.Equal("test@example.com", newUser.Email)
	a.True(reflect.DeepEqual(unsavedUser, newUser))
}

func TestUserStorePutUser(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := NewBoltStore(db, nil)
	a.NoError(err)
	a.NotNil(s)

	newuser, err := NewUser("testuser", "test@example.com")
	a.NoError(err)
	a.NotNil(newuser)

	err = s.PutUser(context.Background(), newuser)
	a.NoError(err)
}

func TestUserStoreGetters(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := NewBoltStore(db, nil)
	a.NoError(err)
	a.NotNil(s)

	newuser, err := NewUser("testuser", "test@example.com")
	a.NoError(err)
	a.NotNil(newuser)

	// storing
	err = s.PutUser(context.Background(), newuser)
	a.NoError(err)

	// retrieving by ID
	u, err := s.GetUserByID(context.Background(), newuser.ID)
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	// retrieving by username index
	u, err = s.GetUserByIndex(context.Background(), "username", "testuser")
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	// retrieving by email index
	u, err = s.GetUserByIndex(context.Background(), "email", "test@example.com")
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	// retrieving by a non-existing index
	u, err = s.GetUserByIndex(context.Background(), "no such index", "absent value")
	a.Error(err)
	a.Nil(u)
}

func TestUserStoreDelete(t *testing.T) {
	t.Parallel()
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := NewBoltStore(db, nil)
	a.NoError(err)
	a.NotNil(s)

	newuser, err := NewUser("testuser", "test@example.com")
	a.NotNil(newuser)

	//---------------------------------------------------------------------------
	// storing and retrieving to make sure it exists
	//---------------------------------------------------------------------------

	err = s.PutUser(context.Background(), newuser)
	a.NoError(err)

	u, err := s.GetUserByID(context.Background(), newuser.ID)
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	u, err = s.GetUserByIndex(context.Background(), "username", "testuser")
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	u, err = s.GetUserByIndex(context.Background(), "email", "test@example.com")
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	//---------------------------------------------------------------------------
	// deleting and attempting to retrieve to make sure it's gone
	// along with all its indexes
	//---------------------------------------------------------------------------

	err = s.DeleteUser(context.Background(), u.ID)
	a.NoError(err)

	u, err = s.GetUserByID(context.Background(), newuser.ID)
	a.EqualError(err, ErrUserNotFound.Error())
	a.Nil(u)

	u, err = s.GetUserByIndex(context.Background(), "username", "testuser")
	a.EqualError(err, ErrUserNotFound.Error())
	a.Nil(u)

	u, err = s.GetUserByIndex(context.Background(), "email", "test@example.com")
	a.EqualError(err, ErrUserNotFound.Error())
	a.Nil(u)
}

//---------------------------------------------------------------------------
// benchmarks
//---------------------------------------------------------------------------

func BenchmarkUserStorePutUser(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewBoltStore(db, NewUserStoreCache(1000))
	var newuser *User

	for n := 0; n < b.N; n++ {
		newuser, err = NewUser("testuser", "test@example.com")
		if err != nil {
			panic(err)
		}

		err = s.PutUser(context.Background(), newuser)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGetByID(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewBoltStore(db, nil)
	newuser, err := NewUser("testuser", "test@example.com")
	err = s.PutUser(context.Background(), newuser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetUserByID(context.Background(), newuser.ID)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGetByIDWithCaching(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewBoltStore(db, NewUserStoreCache(1000))
	newuser, err := NewUser("testuser", "test@example.com")
	err = s.PutUser(context.Background(), newuser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetUserByID(context.Background(), newuser.ID)
		if err != nil {
			panic(err)
		}
	}
}
func BenchmarkUserStoreGetByUsername(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewBoltStore(db, nil)
	newuser, err := NewUser("testuser", "test@example.com")
	err = s.PutUser(context.Background(), newuser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetUserByIndex(context.Background(), "username", newuser.Username)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGetByUsernameWithCaching(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewBoltStore(db, NewUserStoreCache(1000))
	newuser, err := NewUser("testuser", "test@example.com")
	err = s.PutUser(context.Background(), newuser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetUserByIndex(context.Background(), "username", newuser.Username)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGetByEmail(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewBoltStore(db, nil)
	newuser, err := NewUser("testuser", "test@example.com")
	err = s.PutUser(context.Background(), newuser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetUserByIndex(context.Background(), "email", newuser.Email)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkUserStoreGetByEmailWithCaching(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewBoltStore(db, NewUserStoreCache(1000))
	newuser, err := NewUser("testuser", "test@example.com")
	err = s.PutUser(context.Background(), newuser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetUserByIndex(context.Background(), "email", newuser.Email)
		if err != nil {
			panic(err)
		}
	}
}
