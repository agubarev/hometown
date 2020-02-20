package user

import "errors"

// errors
var (
	ErrNoInputData               = errors.New("no input data")
	ErrNilManager                = errors.New("user manager is nil")
	ErrUserExists                = errors.New("user already exists")
	ErrNilUser                   = errors.New("user is nil")
	ErrNilStore                  = errors.New("user store is nil")
	ErrUsernameTaken             = errors.New("username is already taken")
	ErrEmailTaken                = errors.New("email is already taken")
	ErrIndexNotFound             = errors.New("index not found")
	ErrRelationNotFound          = errors.New("relation not found")
	ErrUserNotFound              = errors.New("user not found")
	ErrEmailNotFound             = errors.New("email not found")
	ErrUnknownUser               = errors.New("unknown user")
	ErrNilContainer              = errors.New("container is nil")
	ErrShortPassword             = errors.New("password is too short")
	ErrLongPassword              = errors.New("password is too long")
	ErrUnsafePassword            = errors.New("password is too unsafe")
	ErrNilUserIndex              = errors.New("user index is nil")
	ErrObjectIsNew               = errors.New("object is new")
	ErrObjectIsNotNew            = errors.New("object is not new")
	ErrZeroID                    = errors.New("object has no id")
	ErrNonZeroID                 = errors.New("object id is not zero")
	ErrNothingChanged            = errors.New("nothing has changed")
	ErrUnknownIndex              = errors.New("unknown index")
	ErrInsufficientDataToHashKey = errors.New("insufficient data to hash a key")
	ErrUserAlreadyConfirmed      = errors.New("user is already confirmed")
	ErrNilPasswordManager        = errors.New("password manager is nil")
	ErrUserPasswordNotEligible   = errors.New("user is not eligible for password assignment")
)
