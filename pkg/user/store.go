package user

import (
	"context"

	"github.com/google/uuid"
	"github.com/r3labs/diff"
)

// Store represents a user storage backend contract
type Store interface {
	// user
	CreateUser(ctx context.Context, u User) (_ User, err error)
	BulkCreateUser(ctx context.Context, us []User) (_ []User, err error)
	FetchUserByID(ctx context.Context, id uuid.UUID) (u User, err error)
	FetchUserByUsername(ctx context.Context, username string) (u User, err error)
	FetchUserByEmailAddr(ctx context.Context, addr string) (u User, err error)
	FetchUserByPhoneNumber(ctx context.Context, number string) (u User, err error)
	UpdateUser(ctx context.Context, u User, changelog diff.Changelog) (_ User, err error)
	DeleteUserByID(ctx context.Context, id uuid.UUID) (err error)

	// emails
	CreateEmail(ctx context.Context, e Email) (_ Email, err error)
	BulkCreateEmail(ctx context.Context, es []Email) (_ []Email, err error)
	FetchPrimaryEmailByUserID(ctx context.Context, userID uuid.UUID) (e Email, err error)
	FetchEmailByAddr(ctx context.Context, addr string) (e Email, err error)
	FetchEmailsByUserID(ctx context.Context, userID uuid.UUID) (es []Email, err error)
	UpdateEmail(ctx context.Context, e Email, changelog diff.Changelog) (_ Email, err error)
	DeleteEmailByAddr(ctx context.Context, userID uuid.UUID, addr string) (err error)

	// phones
	CreatePhone(ctx context.Context, p Phone) (_ Phone, err error)
	BulkCreatePhone(ctx context.Context, ps []Phone) (_ []Phone, err error)
	FetchPrimaryPhoneByUserID(ctx context.Context, userID uuid.UUID) (p Phone, err error)
	FetchPhoneByNumber(ctx context.Context, number string) (p Phone, err error)
	FetchPhonesByUserID(ctx context.Context, userID uuid.UUID) (ps []Phone, err error)
	UpdatePhone(ctx context.Context, p Phone, changelog diff.Changelog) (_ Phone, err error)
	DeletePhoneByNumber(ctx context.Context, userID uuid.UUID, number string) (err error)

	// profile
	CreateProfile(ctx context.Context, p Profile) (_ Profile, err error)
	BulkCreateProfile(ctx context.Context, ps []Profile) (_ []Profile, err error)
	FetchProfileByUserID(ctx context.Context, userID uuid.UUID) (p Profile, err error)
	UpdateProfile(ctx context.Context, p Profile, changelog diff.Changelog) (_ Profile, err error)
	DeleteProfileByUserID(ctx context.Context, userID uuid.UUID) (err error)
}
