package internal

import (
	"go.etcd.io/bbolt"
)

// Hometown is the main context
type Hometown struct {
	db       *bbolt.DB
	users    *user.Manager
	sessions *user.SessionManager
	groups   *user.GroupManager
	roles    *user.RoleManager
}
