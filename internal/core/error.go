package core

import "errors"

// errors
var (
	ErrNilUserManager     = errors.New("user manager is nil")
	ErrIndexNotFound      = errors.New("index not found")
	ErrRelationNotFound   = errors.New("relation not found")
	ErrUserNotFound       = errors.New("user not found")
	ErrEmailNotFound      = errors.New("email not found")
	ErrInvalidID          = errors.New("invalid ObjectID")
	ErrNilDominion        = errors.New("dominion is nil")
	ErrDuplicateDomain    = errors.New("found a duplicate domain")
	ErrNilDomain          = errors.New("domain is nil")
	ErrNilUserDomain      = errors.New("user domain is nil")
	ErrDomainNotFound     = errors.New("domain not found")
	ErrUnknownUser        = errors.New("unknown user")
	ErrNilContainer       = errors.New("container is nil")
	ErrNilPasswordManager = errors.New("password manager is nil")
	ErrNilLogger          = errors.New("logger is nil")
)
