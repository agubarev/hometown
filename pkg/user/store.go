package user

import (
	"context"

	"github.com/agubarev/hometown/pkg/util/bytearray"
	"github.com/google/uuid"
)

// Store represents a user storage backend contract
type Store interface {
	// user
	UpsertUser(ctx context.Context, u User) (_ User, err error)
	FetchUserByID(ctx context.Context, id uuid.UUID) (u User, err error)
	FetchUserByUsername(ctx context.Context, username bytearray.ByteString32) (u User, err error)
	FetchUserByEmailAddr(ctx context.Context, addr bytearray.ByteString256) (u User, err error)
	FetchUserByPhoneNumber(ctx context.Context, number bytearray.ByteString16) (u User, err error)
	DeleteUserByID(ctx context.Context, id uuid.UUID) (err error)

	// emails
	UpsertEmail(ctx context.Context, e Email) (_ Email, err error)
	FetchPrimaryEmailByUserID(ctx context.Context, userID uuid.UUID) (e Email, err error)
	FetchEmailByAddr(ctx context.Context, addr bytearray.ByteString256) (e Email, err error)
	FetchEmailsByUserID(ctx context.Context, userID uuid.UUID) (es []Email, err error)
	DeleteEmailByAddr(ctx context.Context, userID uuid.UUID, addr bytearray.ByteString256) (err error)

	// phones
	UpsertPhone(ctx context.Context, p Phone) (_ Phone, err error)
	FetchPrimaryPhoneByUserID(ctx context.Context, userID uuid.UUID) (p Phone, err error)
	FetchPhoneByNumber(ctx context.Context, number bytearray.ByteString16) (p Phone, err error)
	FetchPhonesByUserID(ctx context.Context, userID uuid.UUID) (ps []Phone, err error)
	DeletePhoneByNumber(ctx context.Context, userID uuid.UUID, number bytearray.ByteString16) (err error)

	// profile
	UpsertProfile(ctx context.Context, p Profile) (_ Profile, err error)
	FetchProfileByUserID(ctx context.Context, userID uuid.UUID) (p Profile, err error)
	DeleteProfileByUserID(ctx context.Context, userID uuid.UUID) (err error)
}
