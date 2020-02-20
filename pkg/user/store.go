package user

import (
	"context"
	"sync"

	"github.com/gocraft/dbr/v2"
	"github.com/r3labs/diff"
)

// Store represents a user storage backend contract
type Store interface {
	// user
	CreateUser(ctx context.Context, u *User) (_ *User, err error)
	BulkCreateUser(ctx context.Context, us []*User) (_ []*User, err error)
	FetchUserByID(ctx context.Context, id int) (u *User, err error)
	FetchUserByEmailAddr(ctx context.Context, addr TEmailAddr) (u *User, err error)
	FetchUserByPhoneNumber(ctx context.Context, number TPhoneNumber) (u *User, err error)
	FetchUserByKey(ctx context.Context, key string, value interface{}) (u *User, err error)
	UpdateUser(ctx context.Context, u *User, changelog diff.Changelog) (_ *User, err error)
	DeleteUserByID(ctx context.Context, id int) (err error)

	// emails
	CreateEmail(ctx context.Context, e *Email) (_ *Email, err error)
	BulkCreateEmail(ctx context.Context, es []*Email) (_ []*Email, err error)
	FetchPrimaryEmailByUser(ctx context.Context, u *User) (e *Email, err error)
	FetchEmailsByUser(ctx context.Context, u *User) (es []*Email, err error)
	UpdateEmail(ctx context.Context, e *Email, changelog diff.Changelog) (_ *Email, err error)
	DeleteEmailByAddr(ctx context.Context, u *User, addr TEmailAddr) (err error)

	// phones
	CreatePhone(ctx context.Context, e *Phone) (_ *Phone, err error)
	BulkCreatePhone(ctx context.Context, es []*Phone) (_ []*Phone, err error)
	FetchPrimaryPhoneByUser(ctx context.Context, u *User) (e *Phone, err error)
	FetchPhonesByUser(ctx context.Context, u *User) (es []*Phone, err error)
	UpdatePhone(ctx context.Context, e *Phone, changelog diff.Changelog) (_ *Phone, err error)
	DeletePhoneByNumber(ctx context.Context, u *User) (err error)

	// profile
	CreateProfile(ctx context.Context, p *Profile) (_ *Profile, err error)
	BulkCreateProfile(ctx context.Context, ps []*Profile) (_ []*Profile, err error)
	FetchProfileByUser(ctx context.Context, u *User) (p *Profile, err error)
	UpdateProfile(ctx context.Context, p *Profile, changelog diff.Changelog) (_ *Profile, err error)
	DeleteProfileByUser(ctx context.Context, u *User) (err error)
}

// UserMySQLStore is a default implementation for the MySQL backend
type MySQLStore struct {
	connection *dbr.Connection
}

// memoryStore is a simple in-memory store backend intended for use by tests only
type memoryStore struct {
	users   []*User
	userMap map[int]*User

	globalCounter   int64
	contractorMutex sync.RWMutex
}

// NewMySQLStore is mostly to be used by tests
func NewMySQLStore(conn *dbr.Connection) (Store, err error) {
	if conn == nil {
		return nil, ErrNilStore
	}

	s := &MySQLStore{
		connection: conn,
	}

	return s, nil
}

// NewMemoryStore is mostly to be used by tests
func NewMemoryStore() Store {
	return &memoryStore{
		users:   make([]*User, 10),
		userMap: make(map[int]*User, 10),
	}
}
