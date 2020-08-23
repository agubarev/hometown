package device

import "github.com/pkg/errors"

var (
	ErrInvalidDeviceID       = errors.New("device id is invalid")
	ErrDeviceNotFound        = errors.New("device is not found")
	ErrRelationNotFound      = errors.New("relation is not found")
	ErrRelationAlreadyExists = errors.New("relation already exists")
	ErrNilDatabase           = errors.New("database is nil")
	ErrInvalidAssetID        = errors.New("asset id is invalid")
)
