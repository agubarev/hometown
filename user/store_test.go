package user

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.etcd.io/bbolt"
)

func TestNewDefaultStore(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NotNil(db)
	a.NoError(err)
	defer db.Close()

	s, err := NewDefaultStore(db, nil)
	a.NotNil(s)
	a.NoError(err)
}

func TestStoreCreate(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := NewDefaultStore(db, nil)
	a.NoError(err)
	a.NotNil(s)

	svc, err := NewDefaultService(s)
	a.NoError(err)
	a.NotNil(svc)

	unsavedUser := NewUser("testuser", "test@example.com")
	a.NotNil(unsavedUser)

	newUser, err := svc.Create(context.Background(), unsavedUser)
	a.NoError(err)
	a.NotNil(newUser)
	a.Equal("testuser", newUsername)
	a.Equal("test@example.com", newEmail)
	a.True(reflect.DeepEqual(unsavedUser, newUser))
}

func TestStorePut(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := NewDefaultStore(db, nil)
	a.NoError(err)
	a.NotNil(s)

	newuser := NewUser("testuser", "test@example.com")
	a.NotNil(newuser)

	err = s.Put(context.Background(), newuser)
	a.NoError(err)
}

func TestStoreGetters(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := NewDefaultStore(db, nil)
	a.NoError(err)
	a.NotNil(s)

	newuser := NewUser("testuser", "test@example.com")
	a.NotNil(newuser)

	// storing
	err = s.Put(context.Background(), newuser)
	a.NoError(err)

	// retrieving by ID
	u, err := s.GetByID(context.Background(), newID)
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	// retrieving by username index
	u, err = s.GetByIndex(context.Background(), "username", "testuser")
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	// retrieving by email index
	u, err = s.GetByIndex(context.Background(), "email", "test@example.com")
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	// retrieving by a non-existing index
	u, err = s.GetByIndex(context.Background(), "no such index", "absent value")
	a.Error(err)
	a.Nil(u)
}

func TestStoreDelete(t *testing.T) {
	a := assert.New(t)

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	a.NoError(err)
	a.NotNil(db)
	defer db.Close()

	s, err := NewDefaultStore(db, nil)
	a.NoError(err)
	a.NotNil(s)

	newuser := NewUser("testuser", "test@example.com")
	a.NotNil(newuser)

	//---------------------------------------------------------------------------
	// storing and retrieving to make sure it exists
	//---------------------------------------------------------------------------

	err = s.Put(context.Background(), newuser)
	a.NoError(err)

	u, err := s.GetByID(context.Background(), newID)
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	u, err = s.GetByIndex(context.Background(), "username", "testuser")
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	u, err = s.GetByIndex(context.Background(), "email", "test@example.com")
	a.NoError(err)
	a.NotNil(u)
	a.Equal("testuser", u.Username)
	a.Equal("test@example.com", u.Email)

	//---------------------------------------------------------------------------
	// deleting and attempting to retrieve to make sure it's gone
	// along with all its indexes
	//---------------------------------------------------------------------------

	err = s.Delete(context.Background(), u.ID)
	a.NoError(err)

	u, err = s.GetByID(context.Background(), newID)
	a.EqualError(err, ErrUserNotFound.Error())
	a.Nil(u)

	u, err = s.GetByIndex(context.Background(), "username", "testuser")
	a.EqualError(err, ErrUserNotFound.Error())
	a.Nil(u)

	u, err = s.GetByIndex(context.Background(), "email", "test@example.com")
	a.EqualError(err, ErrUserNotFound.Error())
	a.Nil(u)
}

//---------------------------------------------------------------------------
// benchmarks
//---------------------------------------------------------------------------

func BenchmarkStorePut(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewDefaultStore(db, NewDefaultStoreCache(1000))
	var newuser *User

	for n := 0; n < b.N; n++ {
		newuser = NewUser("testuser", "test@example.com")
		err = s.Put(context.Background(), newuser)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkStoreGetByID(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewDefaultStore(db, nil)
	newuser := NewUser("testuser", "test@example.com")
	err = s.Put(context.Background(), newuser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetByID(context.Background(), newuser.ID)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkStoreGetByIDWithCaching(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewDefaultStore(db, NewDefaultStoreCache(1000))
	newuser := NewUser("testuser", "test@example.com")
	err = s.Put(context.Background(), newuser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetByID(context.Background(), newuser.ID)
		if err != nil {
			panic(err)
		}
	}
}
func BenchmarkStoreGetByUsername(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewDefaultStore(db, nil)
	newuser := NewUser("testuser", "test@example.com")
	err = s.Put(context.Background(), newuser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetByIndex(context.Background(), "username", newuser.Username)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkStoreGetByUsernameWithCaching(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewDefaultStore(db, NewDefaultStoreCache(1000))
	newuser := NewUser("testuser", "test@example.com")
	err = s.Put(context.Background(), newuser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetByIndex(context.Background(), "username", newuser.Username)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkStoreGetByEmail(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewDefaultStore(db, nil)
	newuser := NewUser("testuser", "test@example.com")
	err = s.Put(context.Background(), newuser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetByIndex(context.Background(), "email", newuser.Email)
		if err != nil {
			panic(err)
		}
	}
}

func BenchmarkStoreGetByEmailWithCaching(b *testing.B) {
	b.ReportAllocs()

	db, err := bbolt.Open(fmt.Sprintf("/tmp/hometown-%s.dat", util.NewULID()), 0600, nil)
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s, err := NewDefaultStore(db, NewDefaultStoreCache(1000))
	newuser := NewUser("testuser", "test@example.com")
	err = s.Put(context.Background(), newuser)
	if err != nil {
		panic(err)
	}
	for n := 0; n < b.N; n++ {
		_, err = s.GetByIndex(context.Background(), "email", newuser.Email)
		if err != nil {
			panic(err)
		}
	}
}
