package usermanager

import "errors"

// errors
var (
	ErrNilUserManager             = errors.New("user manager is nil")
	ErrNilPasswordStore           = errors.New("password store is nil")
	ErrEmptyPassword              = errors.New("empty password is forbidden")
	ErrPasswordNotFound           = errors.New("password not found")
	ErrEmptyDominion              = errors.New("no domains were found")
	ErrSuperDomainExists          = errors.New("super domain already exists")
	ErrSuperDomainNotFound        = errors.New("super domain not found")
	ErrSuperuserNotFound          = errors.New("super user does not exist")
	ErrNilAccessPolicy            = errors.New("access policy is nil")
	ErrNilRightsRoster            = errors.New("rights roster is nil")
	ErrAccessDenied               = errors.New("user access denied")
	ErrNoViewRight                = errors.New("user is not allowed to view this")
	ErrNilAssignor                = errors.New("assignor user is nil")
	ErrNilAssignee                = errors.New("assignee user is nil")
	ErrExcessOfRights             = errors.New("assignor is attempting to set the rights that excess his own")
	ErrSameUser                   = errors.New("assignor and assignee is the same user")
	ErrNilParent                  = errors.New("parent is nil")
	ErrUserExists                 = errors.New("user already exists")
	ErrNilUserContainer           = errors.New("user container is nil")
	ErrNilUser                    = errors.New("user is nil")
	ErrNilUserStore               = errors.New("user store is nil")
	ErrNilGroupStore              = errors.New("group store is nil")
	ErrGroupAlreadyRegistered     = errors.New("group is already registered")
	ErrUsernameTaken              = errors.New("username is already taken")
	ErrEmailTaken                 = errors.New("email is already taken")
	ErrEmptyGroupName             = errors.New("empty group name")
	ErrDuplicateParent            = errors.New("duplicate parent")
	ErrGroupKindMismatch          = errors.New("group kinds mismatch")
	ErrInvalidGroupKind           = errors.New("invalid group kind")
	ErrNilRole                    = errors.New("role is nil")
	ErrNoName                     = errors.New("empty role name")
	ErrNilAccessPolicyStore       = errors.New("access policy store is nil")
	ErrAccessPolicyNotFound       = errors.New("access policy not found")
	ErrNilDB                      = errors.New("database is nil")
	ErrIndexNotFound              = errors.New("index not found")
	ErrRelationNotFound           = errors.New("relation not found")
	ErrUserNotFound               = errors.New("user not found")
	ErrEmailNotFound              = errors.New("email not found")
	ErrInvalidID                  = errors.New("invalid ID")
	ErrBucketNotFound             = errors.New("bucket not found")
	ErrNilDominion                = errors.New("dominion is nil")
	ErrDuplicateDomain            = errors.New("found a duplicate domain")
	ErrNilDomain                  = errors.New("domain is nil")
	ErrNilUserDomain              = errors.New("user domain is nil")
	ErrDomainNotFound             = errors.New("domain not found")
	ErrNilGroup                   = errors.New("group is nil")
	ErrNilGroupContainer          = errors.New("group container is nil")
	ErrGroupNotFound              = errors.New("group not found")
	ErrUnknownUser                = errors.New("unknown user")
	ErrNilContainer               = errors.New("container is nil")
	ErrNotMember                  = errors.New("user is not a member")
	ErrAlreadyMember              = errors.New("already a member")
	ErrCircuitedParent            = errors.New("circuited parenting")
	ErrCircuitCheckTimeout        = errors.New("circuit check timed out")
	ErrDomainDirectoryNotFound    = errors.New("domain directory is not found")
	ErrDomainDirectoryNotReadable = errors.New("domain directory is not readable")
	ErrDomainDirectoryNotWritable = errors.New("domain directory is not writable")
	ErrShortPassword              = errors.New("password is too short")
	ErrLongPassword               = errors.New("password is too long")
	ErrUnsafePassword             = errors.New("password is too unsafe")
	ErrNilUserIndex               = errors.New("user index is nil")
	ErrNilPasswordManager         = errors.New("password manager is nil")
)
