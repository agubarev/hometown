package role

import (
	"errors"

	"gitlab.com/agubarev/hometown/pkg/user"
)

// package errors
var (
	ErrNilRole = errors.New("role is nil")
	ErrNoName  = errors.New("empty role name")
)

// Role represents a user role
type Role struct {
	Name    string `json:"name"`
	members map[string]*user.User
}

// NewRole initializing a new role struct
func NewRole(name string) *Role {
	return &Role{
		Name:    name,
		members: make(map[string]*User),
	}
}

// IsMember checks whether user belongs to this role
func (role *Role) IsMember(user *User) bool {
	if _, ok := role.members[user.Username]; ok {
		return true
	}

	return false
}

// AddUser adding user to the group
func (role *Role) AddUser(user *User) error {
	if user == nil {
		return user.ErrNilUser
	}

	// add member to role
	role.members[user.Username] = user

	// add role to user
	user.Roles[role.Name] = role

	return nil
}

// RemoveUser removing user from the group
func (role *Role) RemoveUser(user *User) error {
	if user == nil {
		return ErrUserIsNil
	}

	delete(role.members, user.Username)
	delete(user.Roles, role.Name)

	return nil
}
