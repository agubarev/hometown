package user

import (
	"errors"

	"github.com/oklog/ulid"
)

// package errors
var (
	ErrNilRole = errors.New("role is nil")
	ErrNoName  = errors.New("empty role name")
)

// Role defines a set of permissions
type Role struct {
	ID     ulid.ULID      `json:"id"`
	Domain *domain.Domain `json:"-"`
	Parent *Role          `json:"parent"`
	Name   string         `json:"name"`
}
