package user

import "errors"

// errors
var (
	ErrUserExists     = errors.New("user already exists")
	ErrNilUser        = errors.New("user is nil")
	ErrNilStore       = errors.New("store is nil")
	ErrUsernameTaken  = errors.New("username is already taken")
	ErrEmailTaken     = errors.New("email is already taken")
	ErrGroupIsNil     = errors.New("group is nil")
	ErrEmptyGroupName = errors.New("empty group name")
	ErrGroupUserIsNil = errors.New("user is nil")
	ErrNilRole        = errors.New("role is nil")
	ErrNoName         = errors.New("empty role name")
	ErrNilDB          = errors.New("database is nil")
	ErrIndexNotFound  = errors.New("index not found")
	ErrUserNotFound   = errors.New("user not found")
	ErrEmailNotFound  = errors.New("email not found")
	ErrInvalidID      = errors.New("invalid ID")
	ErrBucketNotFound = errors.New("bucket not found")
	ErrNilDomain      = errors.New("domain is nil")
)
