package user

import (
	"github.com/oklog/ulid"
)

// Role defines a set of permissions
type Role struct {
	ID     ulid.ULID `json:"id"`
	Domain *Domain   `json:"-"`
	Parent *Role     `json:"parent"`
	Name   string    `json:"name"`
}
