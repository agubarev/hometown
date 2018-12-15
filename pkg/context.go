package internal

import (
	"gitlab.com/agubarev/hometown/pkg/user"
	"go.etcd.io/bbolt"
)

// Hometown is the main context
type Hometown struct {
	db       *bbolt.DB
	users    *user.Manager
	sessions *session.SessionManager
	roles    *role.RoleManager
	groups   *group.GroupManager
}
