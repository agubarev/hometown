package user

import "errors"

// errors
var (
	// access policy
	ErrAccessDenied   = errors.New("user access denied")
	ErrNoViewRight    = errors.New("user is not allowed to view this")
	ErrNilAssignor    = errors.New("assignor user is nil")
	ErrNilAssignee    = errors.New("assignee user is nil")
	ErrExcessOfRights = errors.New("assignor is attempting to set the rights that excess his own")
	ErrSameUser       = errors.New("assignor and assignee is the same user")
	ErrNilParent      = errors.New("parent is nil")
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
	ErrNilGroup       = errors.New("group is nil")
	ErrGroupNotFound  = errors.New("group not found")
	ErrUnknownUser    = errors.New("unknown user")
	ErrNilContainer   = errors.New("container is nil")
	ErrAlreadyMember  = errors.New("already a member")
)
