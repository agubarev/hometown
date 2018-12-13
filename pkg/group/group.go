package group

/*
// package errors
var (
	ErrGroupIsNil     = errors.New("group is nil")
	ErrEmptyGroupName = errors.New("empty group name")
	ErrGroupUserIsNil = errors.New("user is nil")
)

// Group represents a user group
type Group struct {
	Name    string `json:"name"`
	members map[string]*User
}

// NewGroup initializing a new group struct
func NewGroup(name string) *Group {
	return &Group{
		Name:    name,
		members: make(map[string]*User),
	}
}

// IsMember checks whether user belongs to this group
func (group *Group) IsMember(user *User) bool {
	if _, ok := group.members[user.Username]; ok {
		return true
	}

	return false
}

// AddUser adding user to the group
func (group *Group) AddUser(user *User) error {
	if user == nil {
		return ErrGroupUserIsNil
	}

	// add member to group
	group.members[user.Username] = user

	// add group to user
	user.Groups[group.Name] = group

	return nil
}

// RemoveUser removing user from the group
func (group *Group) RemoveUser(user *User) error {
	if user == nil {
		return ErrGroupUserIsNil
	}

	delete(group.members, user.Username)
	delete(user.Groups, group.Name)

	return nil
}
*/
