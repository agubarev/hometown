package database

import "github.com/pkg/errors"

var (
	ErrDuplicateEntry = errors.New("duplicate entry")
)
