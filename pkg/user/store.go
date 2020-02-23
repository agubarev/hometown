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
	FetchUserByUsername(ctx context.Context, username TUsername) (u *User, err error)
	FetchUserByEmailAddr(ctx context.Context, addr TEmailAddr) (u *User, err error)
	FetchUserByPhoneNumber(ctx context.Context, number TPhoneNumber) (u *User, err error)
	UpdateUser(ctx context.Context, u *User, changelog diff.Changelog) (_ *User, err error)
	DeleteUserByID(ctx context.Context, id int) (err error)

	// emails
	CreateEmail(ctx context.Context, e *Email) (_ *Email, err error)
	BulkCreateEmail(ctx context.Context, es []*Email) (_ []*Email, err error)
	FetchPrimaryEmailByUserID(ctx context.Context, userID int) (e *Email, err error)
	FetchEmailByAddr(ctx context.Context, addr TEmailAddr) (e *Email, err error)
	FetchEmailsByUserID(ctx context.Context, userID int) (es []*Email, err error)
	UpdateEmail(ctx context.Context, e *Email, changelog diff.Changelog) (_ *Email, err error)
	DeleteEmailByAddr(ctx context.Context, userID int, addr TEmailAddr) (err error)

	// phones
	CreatePhone(ctx context.Context, p *Phone) (_ *Phone, err error)
	BulkCreatePhone(ctx context.Context, ps []*Phone) (_ []*Phone, err error)
	FetchPrimaryPhoneByUserID(ctx context.Context, userID int) (p *Phone, err error)
	FetchPhoneByNumber(ctx context.Context, number TPhoneNumber) (p *Phone, err error)
	FetchPhonesByUserID(ctx context.Context, userID int) (ps []*Phone, err error)
	UpdatePhone(ctx context.Context, p *Phone, changelog diff.Changelog) (_ *Phone, err error)
	DeletePhoneByNumber(ctx context.Context, userID int, number TPhoneNumber) (err error)

	// profile
	CreateProfile(ctx context.Context, p *Profile) (_ *Profile, err error)
	BulkCreateProfile(ctx context.Context, ps []*Profile) (_ []*Profile, err error)
	FetchProfileByUserID(ctx context.Context, userID int) (p *Profile, err error)
	UpdateProfile(ctx context.Context, p *Profile, changelog diff.Changelog) (_ *Profile, err error)
	DeleteProfileByUserID(ctx context.Context, userID int) (err error)
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
func NewMySQLStore(conn *dbr.Connection) (Store, error) {
	if conn == nil {
		return nil, ErrNilStore
	}

	s := &MySQLStore{
		connection: conn,
	}

	return s, nil
}

/*
// NewMemoryStore is mostly to be used by tests
func NewMemoryStore() Store {
	return &memoryStore{
		users:   make([]*User, 10),
		userMap: make(map[int]*User, 10),
	}
}
*/
