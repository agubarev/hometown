package password

import "github.com/pkg/errors"

var (
	ErrNilOwnerID       = errors.New("owner id is zero")
	ErrZeroKind         = errors.New("password kind is zero")
	ErrNilPasswordStore = errors.New("password store is nil")
	ErrEmptyPassword    = errors.New("empty password is forbidden")
	ErrPasswordNotFound = errors.New("password not found")
	ErrShortPassword    = errors.New("password is too short")
	ErrLongPassword     = errors.New("password is too long")
	ErrUnsafePassword   = errors.New("password is too unsafe")
	ErrInfeasibleSafety = errors.New("password safety is infeasible with such length and score")
)
